package db

import (
	"gin-demo/config"
	"gin-demo/internal/model"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func InitMySQL() error {
	c := config.Get()
	db, err := gorm.Open(mysql.Open(c.MySQLDSN), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return err
	}
	DB = db
	// 自动迁移（生产可改为版本化迁移）
	if err := db.AutoMigrate(
		&model.User{},
		&model.Fleet{},
		&model.Vehicle{},
		&model.Application{},
		&model.ApplicationSnapshot{},
		&model.Activity{},
		&model.ConfigVersion{},
		&model.AuditLog{},
		&model.Assumption{},
	); err != nil {
		return err
	}
	return nil
}
