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

// StartPilot BR-07/09 选择试点车（恰好2台）并生成试点版本
func StartPilot(uid uint, role, rid, actID string, vehicleIDs []string) (*model.Activity, error) {
	if role != model.RoleOperator {
		return nil, errors.New("仅发布运营可选择试点")
	}
	sort.Strings(vehicleIDs)
	if len(vehicleIDs) != 2 {
		return nil, errors.New("PILOT_SIZE_INVALID:试点必须恰好 2 台")
	}
	var act model.Activity
	if err := db.DB.Where("id = ?", actID).First(&act).Error; err != nil {
		return nil, errors.New("活动不存在")
	}
	// 幂等：若已存在试点版本则返回现状（BR-09）
	var cnt int64
	db.DB.Model(&model.ConfigVersion{}).Where("activity_id = ? AND batch = ?", actID, model.BatchPilot).Count(&cnt)
	if cnt > 0 {
		return &act, nil
	}
	if act.Status != model.ActApproved {
		return nil, errors.New("当前状态不可启动试点")
	}
	// 校验车辆属于该车队
	if err := validateFleetVehicles(act.FleetID, vehicleIDs); err != nil {
		return nil, err
	}
	if err := db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", actID).First(&act).Error; err != nil {
			return err
		}
		if act.Status != model.ActApproved {
			return errors.New("状态已变更")
		}
		now := clock.Now()
		for _, vid := range vehicleIDs {
			cv := model.ConfigVersion{
				ActivityID:       actID,
				VehicleID:        vid,
				VersionNo:        1,
				ExpectedInterval: act.NewInterval,
				Batch:            model.BatchPilot,
				Status:           model.VerPublished,
				PublishedAt:      &now,
				RequestID:        rid,
			}
			if err := tx.Create(&cv).Error; err != nil {
				return err
			}
		}
		act.Status = model.ActPiloting
		act.Version++
		if err := tx.Save(&act).Error; err != nil {
			return err
		}
		return WriteAudit(tx, rid, uid, role, "START_PILOT", actID, "", "", strings.Join(vehicleIDs, ","), "")
	}); err != nil {
		return nil, err
	}
	return &act, nil
}

// Rollout BR-08/09 推广剩余车辆
func Rollout(uid uint, role, rid, actID string) (*model.Activity, error) {
	if role != model.RoleOperator {
		return nil, errors.New("仅发布运营可推广")
	}
	var act model.Activity
	if err := db.DB.Where("id = ?", actID).First(&act).Error; err != nil {
		return nil, errors.New("活动不存在")
	}
	// 幂等：已存在推广版本则直接返回当前状态（BR-09）
	var rollCnt int64
	db.DB.Model(&model.ConfigVersion{}).Where("activity_id = ? AND batch = ?", actID, model.BatchRollout).Count(&rollCnt)
	if rollCnt > 0 {
		return &act, nil
	}
	if act.Status != model.ActPilotPassed {
		return nil, errors.New("PILOT_NOT_PASSED:试点未通过，禁止推广")
	}
	if err := db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", actID).First(&act).Error; err != nil {
			return err
		}
		if act.Status != model.ActPilotPassed {
			return errors.New("PILOT_NOT_PASSED")
		}
		// 试点车
		var pilot []model.ConfigVersion
		tx.Where("activity_id = ? AND batch = ?", actID, model.BatchPilot).Find(&pilot)
		pilotSet := map[string]bool{}
		for _, p := range pilot {
			pilotSet[p.VehicleID] = true
		}
		// 车队全部车
		var vehicles []model.Vehicle
		tx.Where("fleet_id = ?", act.FleetID).Find(&vehicles)
		now := clock.Now()
		for _, v := range vehicles {
			if pilotSet[v.ID] {
				continue
			}
			cv := model.ConfigVersion{
				ActivityID:       actID,
				VehicleID:        v.ID,
				VersionNo:        1,
				ExpectedInterval: act.NewInterval,
				Batch:            model.BatchRollout,
				Status:           model.VerPublished,
				PublishedAt:      &now,
				RequestID:        rid,
			}
			if err := tx.Create(&cv).Error; err != nil {
				return err
			}
		}
		act.Status = model.ActRollingOut
		act.Version++
		if err := tx.Save(&act).Error; err != nil {
			return err
		}
		return WriteAudit(tx, rid, uid, role, "ROLLOUT", actID, "", "", "", "")
	}); err != nil {
		return nil, err
	}
	return &act, nil
}

// Stop BR-22/23 停止：固化已发布/未发布集合，仅阻止未发布部分
func Stop(uid uint, role, rid, actID, reason string) (*model.Activity, error) {
	if role != model.RoleApprover && role != model.RoleOperator {
		return nil, errors.New("仅负责人或运营可停止")
	}
	var act model.Activity
	if err := db.DB.Where("id = ?", actID).First(&act).Error; err != nil {
		return nil, errors.New("活动不存在")
	}
	if model.ActivityTerminal[act.Status] {
		return nil, errors.New("活动已终态，不可停止")
	}
	if err := db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", actID).First(&act).Error; err != nil {
			return err
		}
		if model.ActivityTerminal[act.Status] {
			return errors.New("活动已终态")
		}
		// 已发布车辆 = 有 config_version 的车辆
		var published []string
		tx.Table("config_versions").Where("activity_id = ?", actID).
			Pluck("DISTINCT vehicle_id", &published)
		pubSet := map[string]bool{}
		for _, p := range published {
			pubSet[p] = true
		}
		var vehicles []model.Vehicle
		tx.Where("fleet_id = ?", act.FleetID).Find(&vehicles)
		notSent := []string{}
		for _, v := range vehicles {
			if !pubSet[v.ID] {
				notSent = append(notSent, v.ID)
			}
		}
		pubJSON, _ := json.Marshal(published)
		notJSON, _ := json.Marshal(notSent)
		act.PublishedVehicles = string(pubJSON)
		act.NotSentVehicles = string(notJSON)
		act.StopReason = reason
		act.Status = model.ActStopped
		if err := tx.Save(&act).Error; err != nil {
			return err
		}
		return WriteAudit(tx, rid, uid, role, "STOP_ACTIVITY", actID, "", "", "", reason)
	}); err != nil {
		return nil, err
	}
	return &act, nil
}

// RequestRestore 运营发起恢复（仅 STOPPED）
func RequestRestore(uid uint, role, rid, actID string) (*model.Activity, error) {
	if role != model.RoleOperator {
		return nil, errors.New("仅发布运营可发起恢复")
	}
	var act model.Activity
	if err := db.DB.Where("id = ?", actID).First(&act).Error; err != nil {
		return nil, errors.New("活动不存在")
	}
	if act.Status != model.ActStopped {
		return nil, errors.New("仅已停止活动可发起恢复")
	}
	if err := db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", actID).First(&act).Error; err != nil {
			return err
		}
		if act.RestoreApproved {
			return errors.New("恢复已批准，无需重复发起")
		}
		return WriteAudit(tx, rid, uid, role, "REQUEST_RESTORE", actID, "", "", "", "")
	}); err != nil {
		return nil, err
	}
	return &act, nil
}

// ApproveRestore BR-24 负责人批准恢复
func ApproveRestore(uid uint, role, rid, actID string) (*model.Activity, error) {
	if role != model.RoleApprover {
		return nil, errors.New("仅负责人可批准恢复")
	}
	var act model.Activity
	if err := db.DB.Where("id = ?", actID).First(&act).Error; err != nil {
		return nil, errors.New("活动不存在")
	}
	if act.Status != model.ActStopped {
		return nil, errors.New("仅已停止活动可批准恢复")
	}
	if err := db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", actID).First(&act).Error; err != nil {
			return err
		}
		act.RestoreApproved = true
		if err := tx.Save(&act).Error; err != nil {
			return err
		}
		return WriteAudit(tx, rid, uid, role, "APPROVE_RESTORE", actID, "", "", "", "")
	}); err != nil {
		return nil, err
	}
	return &act, nil
}

// ExecuteRestore BR-25/27 执行恢复下发（生成恢复版本）
func ExecuteRestore(uid uint, role, rid, actID string) (*model.Activity, error) {
	if role != model.RoleOperator {
		return nil, errors.New("仅发布运营可执行恢复")
	}
	var act model.Activity
	if err := db.DB.Where("id = ?", actID).First(&act).Error; err != nil {
		return nil, errors.New("活动不存在")
	}
	if act.Status != model.ActStopped {
		return nil, errors.New("仅已停止活动可执行恢复")
	}
	if !act.RestoreApproved {
		return nil, errors.New("RESTORE_NOT_APPROVED:恢复未获负责人批准")
	}
	if err := db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", actID).First(&act).Error; err != nil {
			return err
		}
		// 已发布车辆集合（停止时固化）
		var published []string
		if act.PublishedVehicles != "" {
			_ = json.Unmarshal([]byte(act.PublishedVehicles), &published)
		}
		if len(published) == 0 {
			return errors.New("NO_AFFECTED_VEHICLE:停止时无车辆被发布，无需恢复")
		}
		now := clock.Now()
		for _, vid := range published {
			// 该车历史最大版本号
			var maxNo int
			tx.Table("config_versions").Where("activity_id = ? AND vehicle_id = ?", actID, vid).
				Select("COALESCE(MAX(version_no),0)").Scan(&maxNo)
			// 旧版本置为 SUPERSEDED（不再下发，BR-13）
			tx.Model(&model.ConfigVersion{}).
				Where("activity_id = ? AND vehicle_id = ? AND status IN ?", actID, vid,
					[]string{model.VerPublished, model.VerFetched}).
				Update("status", model.VerSuperseded)
			// 该车现值（前值）= 车辆原 PrevInterval
			var veh model.Vehicle
			tx.Where("id = ?", vid).First(&veh)
			cv := model.ConfigVersion{
				ActivityID:       actID,
				VehicleID:        vid,
				VersionNo:        maxNo + 1,
				ExpectedInterval: veh.PrevInterval,
				Batch:            model.BatchRestore,
				Status:           model.VerPublished,
				PublishedAt:      &now,
				RequestID:        rid,
			}
			if err := tx.Create(&cv).Error; err != nil {
				return err
			}
		}
		act.Status = model.ActRestoring
		act.Version++
		if err := tx.Save(&act).Error; err != nil {
			return err
		}
		return WriteAudit(tx, rid, uid, role, "EXECUTE_RESTORE", actID, "", "", "", "")
	}); err != nil {
		return nil, err
	}
	return &act, nil
}

// CompleteActivity BR-28 完成（全部5台 SUCCESS）
func CompleteActivity(uid uint, role, rid, actID string) (*model.Activity, error) {
	if role != model.RoleApprover {
		return nil, errors.New("仅负责人可判定完成")
	}
	var act model.Activity
	if err := db.DB.Where("id = ?", actID).First(&act).Error; err != nil {
		return nil, errors.New("活动不存在")
	}
	if act.Status == model.ActCompleted {
		return &act, nil
	}
	if act.Status != model.ActPendingVerify {
		return nil, errors.New("活动未处于待核查完成状态")
	}
	states, err := GetVehicleStates(actID, act.FleetID)
	if err != nil {
		return nil, err
	}
	for _, s := range states {
		if s.Status != model.VerSuccess {
			return nil, errors.New("NOT_ALL_CONFIRMED:存在未确认车辆，无法完成")
		}
	}
	if err := db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", actID).First(&act).Error; err != nil {
			return err
		}
		act.Status = model.ActCompleted
		act.Version++
		if err := tx.Save(&act).Error; err != nil {
			return err
		}
		return WriteAudit(tx, rid, uid, role, "COMPLETE_ACTIVITY", actID, "", "", "", "")
	}); err != nil {
		return nil, err
	}
	return &act, nil
}

// CompleteRestore BR-26 恢复完成判定
func CompleteRestore(uid uint, role, rid, actID string) (*model.Activity, error) {
	if role != model.RoleApprover {
		return nil, errors.New("仅负责人可判定恢复完成")
	}
	var act model.Activity
	if err := db.DB.Where("id = ?", actID).First(&act).Error; err != nil {
		return nil, errors.New("活动不存在")
	}
	if act.Status != model.ActRestoring {
		return nil, errors.New("活动未处于恢复中")
	}
	states, err := GetVehicleStates(actID, act.FleetID)
	if err != nil {
		return nil, err
	}
	var published []string
	if act.PublishedVehicles != "" {
		_ = json.Unmarshal([]byte(act.PublishedVehicles), &published)
	}
	unconfirmed := 0
	for _, vid := range published {
		for _, s := range states {
			if s.VehicleID == vid && s.Status != model.VerSuccess {
				unconfirmed++
			}
		}
	}
	if err := db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", actID).First(&act).Error; err != nil {
			return err
		}
		if unconfirmed > 0 {
			act.FullyRestored = false
			// 保持 RESTORING，禁止“全部恢复”
			if err := tx.Save(&act).Error; err != nil {
				return err
			}
			return errors.New("UNCONFIRMED_REMAINS:仍有车辆未确认，不可判定恢复完成")
		}
		act.FullyRestored = true
		act.Status = model.ActRestored
		act.Version++
		if err := tx.Save(&act).Error; err != nil {
			return err
		}
		return WriteAudit(tx, rid, uid, role, "COMPLETE_RESTORE", actID, "", "", "", "")
	}); err != nil {
		return nil, err
	}
	return &act, nil
}

// ReevaluateActivity 根据当前版本状态推进活动状态机（试点/推广门禁）
func ReevaluateActivity(actID string) error {
	var act model.Activity
	if err := db.DB.Where("id = ?", actID).First(&act).Error; err != nil {
		return err
	}
	states, err := GetVehicleStates(actID, act.FleetID)
	if err != nil {
		return err
	}
	return db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", actID).First(&act).Error; err != nil {
			return err
		}
		if act.Status == model.ActPiloting {
			pilotStates := filterByBatch(states, model.BatchPilot)
			if len(pilotStates) == 2 {
				anyBad := false
				allGood := true
				for _, s := range pilotStates {
					if s.Status == model.VerFailed || s.Status == model.VerTimeout {
						anyBad = true
					}
					if s.Status != model.VerSuccess {
						allGood = false
					}
				}
				if anyBad {
					act.Status = model.ActPaused
				} else if allGood {
					act.Status = model.ActPilotPassed
				}
			}
		} else if act.Status == model.ActRollingOut {
			allGood := true
			for _, s := range states {
				if s.Status != model.VerSuccess {
					allGood = false
					break
				}
			}
			if allGood {
				act.Status = model.ActPendingVerify
			}
		}
		return tx.Save(&act).Error
	})
}

func filterByBatch(states []VehicleState, batch string) []VehicleState {
	var out []VehicleState
	for _, s := range states {
		if s.Batch == batch {
			out = append(out, s)
		}
	}
	return out
}

func validateFleetVehicles(fleetID string, vehicleIDs []string) error {
	var vehicles []model.Vehicle
	db.DB.Where("fleet_id = ?", fleetID).Find(&vehicles)
	set := map[string]bool{}
	for _, v := range vehicles {
		set[v.ID] = true
	}
	for _, id := range vehicleIDs {
		if !set[id] {
			return errors.New("VALIDATION_FAILED:包含非本车队车辆")
		}
	}
	return nil
}
