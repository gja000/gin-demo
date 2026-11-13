package service

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strconv"
	"strings"
	"time"

	"gin-demo/config"
	"gin-demo/internal/model"
	"gin-demo/pkg/clock"
	"gin-demo/pkg/db"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// PullConfigResponse 车端拉取配置返回
type PullConfigResponse struct {
	HasConfig bool   `json:"hasConfig"`
	Code      string `json:"code,omitempty"` // NO_PUBLISHED_VERSION
	VersionID string `json:"versionId,omitempty"`
	VersionNo int    `json:"versionNo,omitempty"`
	ExpectedInterval int `json:"expectedInterval,omitempty"`
	Batch     string `json:"batch,omitempty"`
}

// PullConfig BR-11/13 车端拉取面向自己的最新已发布版本
func PullConfig(vid, rid, activityID string) (*PullConfigResponse, error) {
	var veh model.Vehicle
	if err := db.DB.Where("id = ?", vid).First(&veh).Error; err != nil {
		return nil, errors.New("车辆不存在")
	}
	if activityID == "" {
		// 取该车队当前未结束活动
		var act model.Activity
		if err := db.DB.Where("fleet_id = ? AND status IN ?", veh.FleetID, activeStatusSlice()).
			Order("created_at desc").First(&act).Error; err != nil {
			return &PullConfigResponse{HasConfig: false, Code: "NO_PUBLISHED_VERSION"}, nil
		}
		activityID = act.ID
	}
	// 最新版本（非 SUPERSEDED）
	var cv model.ConfigVersion
	err := db.DB.Where("activity_id = ? AND vehicle_id = ? AND status <> ?", activityID, vid, model.VerSuperseded).
		Order("version_no desc").First(&cv).Error
	if err != nil {
		return &PullConfigResponse{HasConfig: false, Code: "NO_PUBLISHED_VERSION"}, nil
	}
	// PUBLISHED -> FETCHED
	if cv.Status == model.VerPublished {
		if err := db.DB.Transaction(func(tx *gorm.DB) error {
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", cv.ID).First(&cv).Error; err != nil {
				return err
			}
			if cv.Status == model.VerPublished {
				now := clock.Now()
				cv.Status = model.VerFetched
				cv.FetchedAt = &now
				if err := tx.Save(&cv).Error; err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
			return nil, err
		}
	}
	return &PullConfigResponse{
		HasConfig:       true,
		VersionID:       itoaU(uint(cv.ID)),
		VersionNo:       cv.VersionNo,
		ExpectedInterval: cv.ExpectedInterval,
		Batch:           cv.Batch,
	}, nil
}

// ReceiptRequest 车端回执请求
type ReceiptRequest struct {
	VersionID  string `json:"versionId"`
	Result     string `json:"result"` // SUCCESS / FAILED
	FailReason string `json:"failReason,omitempty"`
}

// ReceiptResult 回执处理结果
type ReceiptResult struct {
	Status         string `json:"status"`
	Ignored        bool   `json:"ignored,omitempty"`
	Duplicate      bool   `json:"duplicate,omitempty"`
	DuplicateCount int    `json:"duplicateCount,omitempty"`
	Code           string `json:"code,omitempty"`
	Message        string `json:"message,omitempty"`
}

// SubmitReceipt BR-14~17 车端回执处理
func SubmitReceipt(vid, rid string, req ReceiptRequest) (*ReceiptResult, error) {
	cvID := atoiU(req.VersionID)
	var cv model.ConfigVersion
	if err := db.DB.Where("id = ?", cvID).First(&cv).Error; err != nil {
		// 未知版本
		_ = WriteAudit(db.DB, rid, 0, "VEHICLE", "RECEIPT_REJECTED", "", "", vid, "", digest(req))
		return &ReceiptResult{Code: "RECEIPT_REJECTED", Message: "未知版本"}, errors.New("RECEIPT_REJECTED:未知版本")
	}
	// 归属校验：必须属于该车辆
	var veh model.Vehicle
	db.DB.Where("id = ?", cv.VehicleID).First(&veh)
	if cv.VehicleID != vid {
		_ = WriteAudit(db.DB, rid, 0, "VEHICLE", "RECEIPT_REJECTED", cv.ActivityID, "", vid, "", digest(req))
		return &ReceiptResult{Code: "RECEIPT_REJECTED", Message: "非本车版本"}, errors.New("RECEIPT_REJECTED:非本车版本")
	}

	stale := false
	dup := false
	err := db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", cv.ID).First(&cv).Error; err != nil {
			return err
		}
		now := clock.Now()
		if req.Result == model.ReceiptSuccess {
			// 幂等：已 SUCCESS 的重复回执
			if cv.Status == model.VerSuccess {
				cv.DuplicateCount++
				dup = true
				if err := tx.Save(&cv).Error; err != nil {
					return err
				}
				return nil
			}
			// 单调生效：旧版本回执不得覆盖
			if cv.VersionNo < veh.EffectiveVersionNo {
				stale = true
				_ = WriteAudit(tx, rid, 0, "VEHICLE", "STALE_RECEIPT_IGNORED", cv.ActivityID, "", vid, "", "")
				return nil
			}
			cv.Status = model.VerSuccess
			cv.FailReason = ""
			cv.ReceiptAt = &now
			if err := tx.Save(&cv).Error; err != nil {
				return err
			}
			// 更新车辆生效值（仅更高版本）
			if cv.VersionNo >= veh.EffectiveVersionNo {
				veh.EffectiveInterval = cv.ExpectedInterval
				veh.EffectiveVersionNo = cv.VersionNo
				if err := tx.Save(&veh).Error; err != nil {
					return err
				}
			}
			_ = WriteAudit(tx, rid, 0, "VEHICLE", "RECEIPT_SUCCESS", cv.ActivityID, "", vid, "", strconv.Itoa(cv.ExpectedInterval))
		} else if req.Result == model.ReceiptFailed {
			if !validFailReason(req.FailReason) {
				return errors.New("VALIDATION_FAILED:failReason 必须为 APPLY_ERROR/UNSUPPORTED_VALUE/STORAGE_ERROR/TIMEOUT_APPLY")
			}
			cv.Status = model.VerFailed
			cv.FailReason = req.FailReason
			cv.ReceiptAt = &now
			// 失败不改生效值
			if err := tx.Save(&cv).Error; err != nil {
				return err
			}
			_ = WriteAudit(tx, rid, 0, "VEHICLE", "RECEIPT_FAILED", cv.ActivityID, "", vid, "", req.FailReason)
		} else {
			return errors.New("VALIDATION_FAILED:result 必须为 SUCCESS/FAILED")
		}
		return nil
	})
	if err != nil {
		if strings.HasPrefix(err.Error(), "RECEIPT_REJECTED") {
			return &ReceiptResult{Code: "RECEIPT_REJECTED", Message: err.Error()}, err
		}
		return buildReceiptResult(&cv), err
	}
	// 旧回执：返回明确拒绝码，生效值不变
	if stale {
		return &ReceiptResult{Status: cv.Status, Code: "STALE_RECEIPT_IGNORED", Ignored: true}, nil
	}
	res := buildReceiptResult(&cv)
	if dup {
		res.Code = "DUPLICATE_RECEIPT"
		res.Duplicate = true
	}
	// 推进活动状态机（门禁）
	_ = ReevaluateActivity(cv.ActivityID)
	return res, nil
}

func buildReceiptResult(cv *model.ConfigVersion) *ReceiptResult {
	return &ReceiptResult{
		Status:         cv.Status,
		Duplicate:      cv.DuplicateCount > 0,
		DuplicateCount: cv.DuplicateCount,
	}
}

func validFailReason(r string) bool {
	switch r {
	case model.FailApplyError, model.FailUnsupportedValue, model.FailStorageError, model.FailTimeoutApply:
		return true
	}
	return false
}

func digest(req ReceiptRequest) string {
	b := sha256.Sum256([]byte(req.VersionID + "|" + req.Result + "|" + req.FailReason))
	return hex.EncodeToString(b[:])[:16]
}

// TimeoutScan BR-19 定时扫描超时版本（默认5秒一轮）
func TimeoutScan() error {
	timeout := config.Get().TimeoutDuration()
	cutoff := clock.Now().Add(-timeout)
	var versions []model.ConfigVersion
	if err := db.DB.Where("status IN ? AND published_at IS NOT NULL AND published_at < ?",
		[]string{model.VerPublished, model.VerFetched}, cutoff).Find(&versions).Error; err != nil {
		return err
	}
	for _, cv := range versions {
		func() {
			_ = db.DB.Transaction(func(tx *gorm.DB) error {
				var cur model.ConfigVersion
				if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", cv.ID).First(&cur).Error; err != nil {
					return err
				}
				if cur.Status != model.VerPublished && cur.Status != model.VerFetched {
					return nil
				}
				now := clock.Now()
				cur.Status = model.VerTimeout
				cur.TimedOut = true
				cur.TimeoutAt = &now
				if err := tx.Save(&cur).Error; err != nil {
					return err
				}
				_ = WriteAudit(tx, "scan", 0, "SYSTEM", "TIMEOUT_JUDGE", cur.ActivityID, "", cur.VehicleID, "", "")
				return nil
			})
			_ = ReevaluateActivity(cv.ActivityID)
		}()
	}
	return nil
}

// StartTimeoutScanner 启动后台超时扫描
func StartTimeoutScanner() {
	interval := config.Get().ScanIntervalSeconds
	if interval < 1 {
		interval = 5
	}
	go func() {
		ticker := time.NewTicker(time.Duration(interval) * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			_ = TimeoutScan()
		}
	}()
}

func itoaU(u uint) string { return itoa64(uint64(u)) }
func atoiU(s string) uint {
	var n uint
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + uint(c-'0')
		}
	}
	return n
}
