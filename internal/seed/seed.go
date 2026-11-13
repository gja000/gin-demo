package seed

import (
	"gin-demo/internal/model"
	"gin-demo/pkg/db"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// Run 初始化基础数据（幂等）
func Run() error {
	if err := seedFleet(); err != nil {
		return err
	}
	if err := seedVehicles(); err != nil {
		return err
	}
	if err := seedUsers(); err != nil {
		return err
	}
	if err := seedAssumptions(); err != nil {
		return err
	}
	return nil
}

func seedFleet() error {
	return db.DB.Where("id = ?", "FLEET-A").FirstOrCreate(&model.Fleet{
		ID:   "FLEET-A",
		Name: "测试车队 A",
	}).Error
}

func seedVehicles() error {
	// prevInterval: 演示恢复分支时各车前值可不同（V1/V2=60，V3~V5=90）
	specs := []struct {
		id  string
		pre int
	}{
		{"V1", 60}, {"V2", 60}, {"V3", 90}, {"V4", 90}, {"V5", 90},
	}
	for _, s := range specs {
		v := model.Vehicle{
			ID:                s.id,
			FleetID:           "FLEET-A",
			Name:              "车辆 " + s.id,
			Secret:            "secret-" + s.id, // 演示用固定种子，生产应由安全渠道下发
			PrevInterval:      s.pre,
			EffectiveInterval: s.pre,
			EffectiveVersionNo: 0,
		}
		// 首次创建；已存在则跳过（不覆盖用户可能修改的生效值）
		var cnt int64
		db.DB.Model(&model.Vehicle{}).Where("id = ?", s.id).Count(&cnt)
		if cnt == 0 {
			if err := db.DB.Create(&v).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func seedUsers() error {
	users := []struct {
		username string
		role     string
		name     string
	}{
		{"applicant", model.RoleApplicant, "研发申请人"},
		{"approver", model.RoleApprover, "审批负责人"},
		{"operator", model.RoleOperator, "发布运营"},
	}
	for _, u := range users {
		hash, err := bcrypt.GenerateFromPassword([]byte("123456"), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		err = db.DB.Where("username = ?", u.username).FirstOrCreate(&model.User{
			Username: u.username,
			Password: string(hash),
			Name:     u.name,
			Role:     u.role,
		}).Error
		if err != nil {
			return err
		}
	}
	return nil
}

// seedAssumptions 登记题目未明确事项的初始实现口径（AGENTS.md 第 10 节）
func seedAssumptions() error {
	items := []model.Assumption{
		{Code: "ASM-MEEgle", Description: "无真实 meegle 工作项编号；feat/fix 提交使用占位编号 #PLACEHOLDER-09，已偏离 AGENTS-git.md §3.4 红线，待评审确认。", Implemented: true, Status: "待确认"},
		{Code: "ASM-STOP-CHAIN", Description: "停止语义：运营发起停止由负责人批准确认；负责人亦可直接决定停止。停止仅阻止未发布部分下发，文案统一为『已停止，车端仍保留已下发配置』。", Implemented: true, Status: "待确认"},
		{Code: "ASM-PILOT-SIZE", Description: "试点固定恰好 2 台，运营从 5 台中显式选择。", Implemented: true, Status: "待确认"},
		{Code: "ASM-NO-RESEND", Description: "不提供单车单独重发；回执按版本幂等处理。", Implemented: true, Status: "待确认"},
		{Code: "ASM-LATE-RECEIPT", Description: "迟到回执对当前期望版本更新该车生效值与时间线；已停止活动中迟到回执仅记录时间线，不改变活动状态。", Implemented: true, Status: "待确认"},
	}
	for _, a := range items {
		var cnt int64
		db.DB.Model(&model.Assumption{}).Where("code = ?", a.Code).Count(&cnt)
		if cnt == 0 {
			if err := db.DB.Create(&a).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

var _ = gorm.ErrRecordNotFound
