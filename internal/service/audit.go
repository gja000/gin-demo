package service

import (
	"gin-demo/internal/model"

	"gorm.io/gorm"
)

// WriteAudit 写入审计（不可篡改，所有关键动作均记录）
func WriteAudit(tx *gorm.DB, rid string, operatorID uint, operatorRole, action, activityID, applicationID, targetVehicle, before, after string) error {
	rec := model.AuditLog{
		ActivityID:    activityID,
		ApplicationID: applicationID,
		OperatorID:    operatorID,
		OperatorRole:  operatorRole,
		Action:        action,
		TargetVehicle: targetVehicle,
		BeforeValue:   before,
		AfterValue:    after,
		RequestID:     rid,
	}
	return tx.Create(&rec).Error
}

// uidToUint 将字符串 uid 转 uint
func uidToUint(s string) uint {
	var n uint
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + uint(c-'0')
		}
	}
	return n
}
