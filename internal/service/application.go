package service

import (
	"encoding/json"
	"errors"
	"sort"
	"strings"

	"gin-demo/internal/model"
	"gin-demo/pkg/clock"
	"gin-demo/pkg/db"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// CreateAppRequest 创建申请请求
type CreateAppRequest struct {
	FleetID        string   `json:"fleetId"`
	Reason         string   `json:"reason"`
	NewInterval    int      `json:"newInterval"`
	TargetVehicles []string `json:"targetVehicles"`
}

// CreateApplication BR-01 申请完整性校验
func CreateApplication(uid uint, role, rid string, req CreateAppRequest) (*model.Application, error) {
	if role != model.RoleApplicant {
		return nil, errors.New("仅研发申请人可创建申请")
	}
	if strings.TrimSpace(req.Reason) == "" || len([]rune(req.Reason)) > 500 {
		return nil, errors.New("VALIDATION_FAILED:reason 非空且≤500字")
	}
	if req.NewInterval < 10 || req.NewInterval > 300 {
		return nil, errors.New("VALIDATION_FAILED:newInterval 必须为 10–300 的整数")
	}
	if req.FleetID == "" {
		return nil, errors.New("VALIDATION_FAILED:fleetId 必填")
	}
	// 目标车辆必须恰好为车队全部车辆
	var fleetVehicles []model.Vehicle
	if err := db.DB.Where("fleet_id = ?", req.FleetID).Find(&fleetVehicles).Error; err != nil {
		return nil, err
	}
	if len(fleetVehicles) == 0 {
		return nil, errors.New("VALIDATION_FAILED:车队不存在或没有车辆")
	}
	want := make([]string, 0, len(fleetVehicles))
	for _, v := range fleetVehicles {
		want = append(want, v.ID)
	}
	sort.Strings(want)
	got := append([]string{}, req.TargetVehicles...)
	sort.Strings(got)
	if strings.Join(want, ",") != strings.Join(got, ",") {
		return nil, errors.New("VALIDATION_FAILED:目标车辆必须为该车队全部车辆，禁止部分选择")
	}

	tvs, _ := json.Marshal(want)
	app := model.Application{
		ID:             genID("APP"),
		FleetID:        req.FleetID,
		ApplicantID:    uid,
		Reason:         strings.TrimSpace(req.Reason),
		NewInterval:    req.NewInterval,
		TargetVehicles: string(tvs),
		Status:         model.AppDraft,
		SubmitCount:    0,
	}
	if err := db.DB.Create(&app).Error; err != nil {
		return nil, err
	}
	_ = WriteAudit(db.DB, rid, uid, role, "CREATE_APPLICATION", "", app.ID, "", "", "")
	return &app, nil
}

// SubmitApplication 提交申请（记录快照）
func SubmitApplication(uid uint, role, rid, appID string) (*model.Application, error) {
	if role != model.RoleApplicant {
		return nil, errors.New("仅研发申请人可提交")
	}
	var app model.Application
	if err := db.DB.Where("id = ? AND applicant_id = ?", appID, uid).First(&app).Error; err != nil {
		return nil, errors.New("申请不存在或无权限")
	}
	if app.Status != model.AppDraft && app.Status != model.AppRejected {
		return nil, errors.New("当前状态不可提交")
	}
	if err := db.DB.Transaction(func(tx *gorm.DB) error {
		app.Status = model.AppSubmitted
		app.SubmitCount++
		if err := tx.Save(&app).Error; err != nil {
			return err
		}
		snap := model.ApplicationSnapshot{
			ApplicationID:  app.ID,
			Reason:         app.Reason,
			NewInterval:    app.NewInterval,
			TargetVehicles: app.TargetVehicles,
			SubmittedAt:    clock.Now(),
		}
		if err := tx.Create(&snap).Error; err != nil {
			return err
		}
		return WriteAudit(tx, rid, uid, role, "SUBMIT_APPLICATION", "", app.ID, "", "", "")
	}); err != nil {
		return nil, err
	}
	return &app, nil
}

// RejectApplication BR-02/03 退回（必须填理由，禁止自审）
func RejectApplication(uid uint, role, rid, appID, reason string) (*model.Application, error) {
	if role != model.RoleApprover {
		return nil, errors.New("仅审批负责人可退回")
	}
	if strings.TrimSpace(reason) == "" {
		return nil, errors.New("REASON_REQUIRED:退回必须填写理由")
	}
	var app model.Application
	if err := db.DB.Where("id = ?", appID).First(&app).Error; err != nil {
		return nil, errors.New("申请不存在")
	}
	if app.ApplicantID == uid {
		return nil, errors.New("SELF_APPROVAL_FORBIDDEN:不能审批/退回自己的申请")
	}
	if app.Status != model.AppSubmitted {
		return nil, errors.New("当前状态不可退回")
	}
	if err := db.DB.Transaction(func(tx *gorm.DB) error {
		app.Status = model.AppRejected
		if err := tx.Save(&app).Error; err != nil {
			return err
		}
		return WriteAudit(tx, rid, uid, role, "REJECT_APPLICATION", "", app.ID, "", "", reason)
	}); err != nil {
		return nil, err
	}
	return &app, nil
}

// ApproveApplication BR-02/04/05 批准 → 创建活动并冻结参数
func ApproveApplication(uid uint, role, rid, appID string) (*model.Activity, error) {
	if role != model.RoleApprover {
		return nil, errors.New("仅审批负责人可批准")
	}
	var app model.Application
	if err := db.DB.Where("id = ?", appID).First(&app).Error; err != nil {
		return nil, errors.New("申请不存在")
	}
	if app.ApplicantID == uid {
		return nil, errors.New("SELF_APPROVAL_FORBIDDEN:不能审批自己的申请")
	}
	if app.Status != model.AppSubmitted {
		return nil, errors.New("当前状态不可批准")
	}
	// 并发安全：加锁检查车队是否存在未结束活动（INV-01 / BR-05 / BR-06）
	var act model.Activity
	err := db.DB.Transaction(func(tx *gorm.DB) error {
		// 锁住该车队最后一条活动以串行化并发批准
		var lockRow model.Activity
		tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("fleet_id = ?", app.FleetID).Order("created_at desc").Limit(1).
			Find(&lockRow)
		var cnt int64
		if err := tx.Model(&model.Activity{}).
			Where("fleet_id = ? AND status IN ?", app.FleetID, activeStatusSlice()).
			Count(&cnt).Error; err != nil {
			return err
		}
		if cnt > 0 {
			return errors.New("FLEET_HAS_ACTIVE_ACTIVITY:车队存在未结束活动，无法批准新申请")
		}
		// 创建活动
		act = model.Activity{
			ID:          genID("ACT"),
			ApplicationID: app.ID,
			FleetID:     app.FleetID,
			NewInterval: app.NewInterval, // 冻结
			Status:      model.ActApproved,
			Version:     1,
		}
		if err := tx.Create(&act).Error; err != nil {
			return err
		}
		app.Status = model.AppApproved
		if err := tx.Save(&app).Error; err != nil {
			return err
		}
		return WriteAudit(tx, rid, uid, role, "APPROVE_APPLICATION", act.ID, app.ID, "", "", "")
	})
	if err != nil {
		return nil, err
	}
	return &act, nil
}

// activeStatusSlice 未结束活动状态集合（用于 INV-01 / BR-05 / BR-06 唯一性约束）
func activeStatusSlice() []string {
	return []string{
		model.ActApproved, model.ActPiloting, model.ActPilotPassed,
		model.ActRollingOut, model.ActPendingVerify, model.ActPaused,
		model.ActRestoring,
	}
}

// UpdateApplication BR-03/04 修改申请（仅 DRAFT/REJECTED 可改；批准后冻结）
func UpdateApplication(uid uint, role, rid, appID, reason string, newInterval int) (*model.Application, error) {
	if role != model.RoleApplicant {
		return nil, errors.New("仅研发申请人可修改申请")
	}
	var app model.Application
	if err := db.DB.Where("id = ? AND applicant_id = ?", appID, uid).First(&app).Error; err != nil {
		return nil, errors.New("申请不存在或无权限")
	}
	if app.Status == model.AppApproved || app.Status == model.AppSubmitted {
		return nil, errors.New("PARAM_FROZEN:参数已冻结，批准后不可修改")
	}
	if reason != "" {
		if strings.TrimSpace(reason) == "" || len([]rune(reason)) > 500 {
			return nil, errors.New("VALIDATION_FAILED:reason 非空且≤500字")
		}
		app.Reason = strings.TrimSpace(reason)
	}
	if newInterval != 0 {
		if newInterval < 10 || newInterval > 300 {
			return nil, errors.New("VALIDATION_FAILED:newInterval 必须为 10–300 的整数")
		}
		app.NewInterval = newInterval
	}
	if err := db.DB.Save(&app).Error; err != nil {
		return nil, err
	}
	_ = WriteAudit(db.DB, rid, uid, role, "UPDATE_APPLICATION", "", app.ID, "", "", "")
	return &app, nil
}

// GetApplication 详情（含快照 diff）
func GetApplication(appID string) (*model.Application, []model.ApplicationSnapshot, error) {
	var app model.Application
	if err := db.DB.Where("id = ?", appID).First(&app).Error; err != nil {
		return nil, nil, err
	}
	var snaps []model.ApplicationSnapshot
	db.DB.Where("application_id = ?", appID).Order("submitted_at asc").Find(&snaps)
	return &app, snaps, nil
}

// parseTargetVehicles 解析目标车辆 JSON
func parseTargetVehicles(s string) []string {
	var v []string
	_ = json.Unmarshal([]byte(s), &v)
	return v
}
