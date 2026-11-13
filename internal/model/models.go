package model

import "time"

// ===== 角色 =====
const (
	RoleApplicant = "APPLICANT"
	RoleApprover  = "APPROVER"
	RoleOperator  = "OPERATOR"
)

// ===== 申请状态 =====
const (
	AppDraft     = "DRAFT"
	AppSubmitted = "SUBMITTED"
	AppApproved  = "APPROVED"
	AppRejected  = "REJECTED"
)

// ===== 活动状态机 =====
const (
	ActApproved      = "APPROVED"       // 已批准，待选试点
	ActPiloting      = "PILOTING"       // 试点中
	ActPilotPassed   = "PILOT_PASSED"   // 试点通过，可推广
	ActRollingOut    = "ROLLING_OUT"    // 推广中
	ActPendingVerify = "PENDING_VERIFY" // 推广完成，待核查完成
	ActCompleted     = "COMPLETED"      // 全部成功完成
	ActPaused        = "PAUSED"         // 试点失败/超时
	ActStopped       = "STOPPED"        // 已停止
	ActRestoring     = "RESTORING"      // 恢复中
	ActRestored      = "RESTORED"       // 恢复完成
)

// 终态集合
var ActivityTerminal = map[string]bool{
	ActCompleted: true,
	ActRestored:  true,
	ActStopped:   true,
}

// 未结束状态（用于唯一性约束）：除终态外都算未结束
func IsActiveActivityStatus(s string) bool {
	return !ActivityTerminal[s]
}

// ===== 版本批次 =====
const (
	BatchPilot   = "PILOT"
	BatchRollout = "ROLLOUT"
	BatchRestore = "RESTORE"
)

// ===== 单车版本状态 =====
const (
	VerPublished  = "PUBLISHED"  // 已发布（车端可取）
	VerFetched    = "FETCHED"    // 已拉取
	VerSuccess    = "SUCCESS"    // 生效成功
	VerFailed     = "FAILED"     // 应用失败
	VerTimeout    = "TIMEOUT"    // 超时未确认
	VerSuperseded = "SUPERSEDED" // 被更高版本替代，不再下发
)

// ===== 回执结果 =====
const (
	ReceiptSuccess = "SUCCESS"
	ReceiptFailed  = "FAILED"
)

// 失败原因枚举
const (
	FailApplyError       = "APPLY_ERROR"
	FailUnsupportedValue = "UNSUPPORTED_VALUE"
	FailStorageError     = "STORAGE_ERROR"
	FailTimeoutApply     = "TIMEOUT_APPLY"
)

// User 系统用户（会话令牌来源）
type User struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Username  string    `gorm:"uniqueIndex;size:64" json:"username"`
	Password  string    `gorm:"size:128" json:"-"`
	Name      string    `gorm:"size:64" json:"name"`
	Role      string    `gorm:"size:32;index" json:"role"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// Fleet 车队
type Fleet struct {
	ID        string    `gorm:"primaryKey;size:32" json:"id"`
	Name      string    `gorm:"size:64" json:"name"`
	CreatedAt time.Time `json:"createdAt"`
}

// Vehicle 模拟车辆（车端令牌来源）
type Vehicle struct {
	ID                  string `gorm:"primaryKey;size:32" json:"id"`
	FleetID             string `gorm:"size:32;index" json:"fleetId"`
	Name                string `gorm:"size:64" json:"name"`
	Secret              string `gorm:"size:128" json:"-"` // 车端登录凭证种子
	PrevInterval        int    `json:"prevInterval"`      // 活动前的原日志间隔
	EffectiveInterval   int    `json:"effectiveInterval"` // 全局生效值（仅 SUCCESS 回执更新，单调不减）
	EffectiveVersionNo  int    `json:"effectiveVersionNo"`
	CreatedAt           time.Time `json:"createdAt"`
}

// Application 日志策略申请
type Application struct {
	ID              string `gorm:"primaryKey;size:32" json:"id"`
	FleetID         string `gorm:"size:32;index" json:"fleetId"`
	ApplicantID     uint   `gorm:"index" json:"applicantId"`
	Reason          string `gorm:"size:500" json:"reason"`
	NewInterval     int    `json:"newInterval"`
	TargetVehicles  string `gorm:"type:text" json:"-"` // JSON 数组，固定车队全部 5 台
	Status          string `gorm:"size:32;index" json:"status"`
	SubmitCount     int    `json:"submitCount"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

// ApplicationSnapshot 历次提交快照（退回再提交时并存）
type ApplicationSnapshot struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	ApplicationID  string    `gorm:"size:32;index" json:"applicationId"`
	Reason         string    `gorm:"size:500" json:"reason"`
	NewInterval    int       `json:"newInterval"`
	TargetVehicles string    `gorm:"type:text" json:"targetVehicles"`
	SubmittedAt    time.Time `json:"submittedAt"`
}

// Activity 活动（状态机主体）
type Activity struct {
	ID               string `gorm:"primaryKey;size:32" json:"id"`
	ApplicationID    string `gorm:"size:32;index" json:"applicationId"`
	FleetID          string `gorm:"size:32;index" json:"fleetId"`
	NewInterval      int    `json:"newInterval"` // 批准后的冻结值
	Status           string `gorm:"size:32;index" json:"status"`
	Version          int    `json:"-"` // 乐观锁
	PublishedVehicles string `gorm:"type:text" json:"-"` // 停止时固化的已发布车辆集合
	NotSentVehicles   string `gorm:"type:text" json:"-"` // 停止时固化的未发布车辆集合
	FullyRestored    bool   `json:"fullyRestored"`
	RestoreApproved  bool   `json:"restoreApproved"` // 恢复是否已获负责人批准
	StopReason       string `gorm:"size:500" json:"stopReason,omitempty"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

// ConfigVersion 每车每版本（关键字段创建后不可更新；状态/计数/时间戳可变）
type ConfigVersion struct {
	ID               uint      `gorm:"primaryKey" json:"id"`
	ActivityID       string    `gorm:"size:32;index:idx_act_veh;index:idx_act_ver" json:"activityId"`
	VehicleID        string    `gorm:"size:32;index:idx_act_veh" json:"vehicleId"`
	VersionNo        int       `gorm:"index:idx_act_ver" json:"versionNo"` // 按车维度递增
	ExpectedInterval int       `json:"expectedInterval"`
	Batch            string    `gorm:"size:32" json:"batch"`
	Status           string    `gorm:"size:32" json:"status"`
	FailReason       string    `gorm:"size:32" json:"failReason,omitempty"`
	TimedOut         bool      `json:"timedOut"`         // 是否曾超时（迟到回执后仍保留）
	TimeoutAt        *time.Time `json:"timeoutAt,omitempty"` // 超时判定时刻
	PublishedAt      *time.Time `json:"publishedAt,omitempty"`
	FetchedAt        *time.Time `json:"fetchedAt,omitempty"`
	ReceiptAt        *time.Time `json:"receiptAt,omitempty"`
	DuplicateCount   int       `json:"duplicateCount"`
	RequestID        string    `gorm:"size:64" json:"requestId"`
	CreatedAt        time.Time `json:"createdAt"`
}

// AuditLog 审计（不可篡改）
type AuditLog struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	ActivityID     string    `gorm:"size:32;index" json:"activityId,omitempty"`
	ApplicationID  string    `gorm:"size:32;index" json:"applicationId,omitempty"`
	OperatorID     uint      `json:"operatorId"`
	OperatorRole   string    `gorm:"size:32" json:"operatorRole"`
	Action         string    `gorm:"size:64" json:"action"`
	TargetVehicle  string    `gorm:"size:32" json:"targetVehicle,omitempty"`
	BeforeValue    string    `gorm:"size:64" json:"beforeValue,omitempty"`
	AfterValue     string    `gorm:"size:64" json:"afterValue,omitempty"`
	RequestID      string    `gorm:"size:64;index" json:"requestId"`
	CreatedAt      time.Time `json:"createdAt"`
}

// Assumption 假设登记（题目未明确事项）
type Assumption struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Code         string    `gorm:"size:32;index" json:"code"`
	Description  string    `gorm:"type:text" json:"description"`
	Implemented  bool      `json:"implemented"`
	Status       string    `gorm:"size:32" json:"status"` // 待确认 / 已确认
	ConfirmedBy  string    `gorm:"size:64" json:"confirmedBy,omitempty"`
	ConfirmedAt  *time.Time `json:"confirmedAt,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
}
