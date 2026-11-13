package handler

import (
	"net/http"
	"strings"

	"gin-demo/config"
	"gin-demo/internal/auth"
	"gin-demo/internal/middleware"
	"gin-demo/internal/model"
	"gin-demo/internal/service"
	"gin-demo/pkg/db"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// SetupRoutes 注册全部路由
func SetupRoutes(r *gin.Engine) {
	api := r.Group("/api")
	{
		api.POST("/login", login)
		api.POST("/vehicle-agent/login", vehicleLogin)

		// 用户接口（会话令牌，忽略请求体角色字段）
		authed := api.Group("")
		authed.Use(middleware.UserAuth())
		{
			authed.GET("/me", me)
			authed.GET("/assumptions", listAssumptions)

			// 申请
			authed.POST("/applications", middleware.UserAuth(model.RoleApplicant), createApplication)
			authed.PATCH("/applications/:id", middleware.UserAuth(model.RoleApplicant), updateApplication)
			authed.POST("/applications/:id/submit", middleware.UserAuth(model.RoleApplicant), submitApplication)
			authed.POST("/applications/:id/reject", middleware.UserAuth(model.RoleApprover), rejectApplication)
			authed.POST("/applications/:id/approve", middleware.UserAuth(model.RoleApprover), approveApplication)
			authed.GET("/applications", listApplications)
			authed.GET("/applications/:id", getApplication)

			// 活动
			authed.GET("/activities", listActivities)
			authed.GET("/activities/:id", getActivity)
			authed.GET("/activities/:id/vehicles", getActivityVehicles)
			authed.GET("/activities/:id/overview", getActivityOverview)
			authed.GET("/activities/:id/timeline", getActivityTimeline)
			authed.GET("/activities/:id/audit", getActivityAudit)

			authed.POST("/activities/:id/pilot", middleware.UserAuth(model.RoleOperator), startPilot)
			authed.POST("/activities/:id/rollout", middleware.UserAuth(model.RoleOperator), rollout)
			authed.POST("/activities/:id/stop", middleware.UserAuth(model.RoleApprover, model.RoleOperator), stopActivity)
			authed.POST("/activities/:id/restore", middleware.UserAuth(model.RoleOperator), requestRestore)
			authed.POST("/activities/:id/restore/approve", middleware.UserAuth(model.RoleApprover), approveRestore)
			authed.POST("/activities/:id/restore/execute", middleware.UserAuth(model.RoleOperator), executeRestore)
			authed.POST("/activities/:id/complete", middleware.UserAuth(model.RoleApprover), completeActivity)
			authed.POST("/activities/:id/restore/complete", middleware.UserAuth(model.RoleApprover), completeRestore)

			// 审计
			authed.GET("/audit", listAudit)

			// 开发环境数据重置（受 ENABLE_RESET 门控）
			authed.POST("/dev/reset", devReset)
		}

		// 车端接口（与用户令牌完全隔离）
		veh := api.Group("/vehicle-agent")
		veh.Use(middleware.VehicleAuth())
		{
			veh.GET("/config", pullConfig)
			veh.POST("/receipts", submitReceipt)
		}
	}
}

// ---- 鉴权 ----
func login(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "BAD_REQUEST", "message": err.Error()})
		return
	}
	var u model.User
	if err := db.DB.Where("username = ?", req.Username).First(&u).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": "UNAUTHORIZED", "message": "账号或密码错误"})
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": "UNAUTHORIZED", "message": "账号或密码错误"})
		return
	}
	token, err := auth.GenerateUserToken(u.ID, u.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "ERROR", "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user":  gin.H{"id": u.ID, "username": u.Username, "name": u.Name, "role": u.Role},
	})
}

func vehicleLogin(c *gin.Context) {
	var req struct {
		VehicleID string `json:"vehicleId"`
		Secret    string `json:"secret"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "BAD_REQUEST", "message": err.Error()})
		return
	}
	var v model.Vehicle
	if err := db.DB.Where("id = ? AND secret = ?", req.VehicleID, req.Secret).First(&v).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"code": "UNAUTHORIZED", "message": "车辆或凭证错误"})
		return
	}
	token, err := auth.GenerateVehicleToken(v.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": "ERROR", "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": token, "vehicleId": v.ID})
}

func me(c *gin.Context) {
	uid := middleware.CurrentUID(c)
	role := middleware.CurrentRole(c)
	var u model.User
	if err := db.DB.Where("id = ?", uid).First(&u).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": "NOT_FOUND", "message": "用户不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": u.ID, "username": u.Username, "name": u.Name, "role": role})
}

func listAssumptions(c *gin.Context) {
	var list []model.Assumption
	db.DB.Order("id asc").Find(&list)
	c.JSON(http.StatusOK, list)
}

// ---- 申请 ----
func createApplication(c *gin.Context) {
	var req service.CreateAppRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "BAD_REQUEST", "message": err.Error()})
		return
	}
	uid := service.UIDFromString(middleware.CurrentUID(c))
	app, err := service.CreateApplication(uid, middleware.CurrentRole(c), rid(c), req)
	if err != nil {
		apiError(c, err)
		return
	}
	c.JSON(http.StatusOK, app)
}

func updateApplication(c *gin.Context) {
	var req struct {
		Reason      string `json:"reason"`
		NewInterval int    `json:"newInterval"`
	}
	c.ShouldBindJSON(&req)
	uid := service.UIDFromString(middleware.CurrentUID(c))
	app, err := service.UpdateApplication(uid, middleware.CurrentRole(c), rid(c), c.Param("id"), req.Reason, req.NewInterval)
	if err != nil {
		apiError(c, err)
		return
	}
	c.JSON(http.StatusOK, app)
}

func submitApplication(c *gin.Context) {
	uid := service.UIDFromString(middleware.CurrentUID(c))
	app, err := service.SubmitApplication(uid, middleware.CurrentRole(c), rid(c), c.Param("id"))
	if err != nil {
		apiError(c, err)
		return
	}
	c.JSON(http.StatusOK, app)
}

func rejectApplication(c *gin.Context) {
	var req struct {
		Reason string `json:"reason"`
	}
	c.ShouldBindJSON(&req)
	uid := service.UIDFromString(middleware.CurrentUID(c))
	app, err := service.RejectApplication(uid, middleware.CurrentRole(c), rid(c), c.Param("id"), req.Reason)
	if err != nil {
		apiError(c, err)
		return
	}
	c.JSON(http.StatusOK, app)
}

func approveApplication(c *gin.Context) {
	uid := service.UIDFromString(middleware.CurrentUID(c))
	act, err := service.ApproveApplication(uid, middleware.CurrentRole(c), rid(c), c.Param("id"))
	if err != nil {
		apiError(c, err)
		return
	}
	c.JSON(http.StatusOK, act)
}

func listApplications(c *gin.Context) {
	role := middleware.CurrentRole(c)
	uid := service.UIDFromString(middleware.CurrentUID(c))
	q := db.DB.Model(&model.Application{})
	if role == model.RoleApplicant {
		q = q.Where("applicant_id = ?", uid)
	}
	var list []model.Application
	q.Order("created_at desc").Find(&list)
	c.JSON(http.StatusOK, list)
}

func getApplication(c *gin.Context) {
	app, snaps, err := service.GetApplication(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": "NOT_FOUND", "message": "申请不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"application": app, "snapshots": snaps})
}

// ---- 活动 ----
func listActivities(c *gin.Context) {
	var list []model.Activity
	db.DB.Order("created_at desc").Find(&list)
	c.JSON(http.StatusOK, list)
}

func getActivity(c *gin.Context) {
	var act model.Activity
	if err := db.DB.Where("id = ?", c.Param("id")).First(&act).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": "NOT_FOUND", "message": "活动不存在"})
		return
	}
	states, _ := service.GetVehicleStates(act.ID, act.FleetID)
	overview := service.ComputeOverview(states)
	c.JSON(http.StatusOK, gin.H{"activity": act, "states": states, "overview": overview})
}

func getActivityVehicles(c *gin.Context) {
	var act model.Activity
	if err := db.DB.Where("id = ?", c.Param("id")).First(&act).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": "NOT_FOUND", "message": "活动不存在"})
		return
	}
	states, err := service.GetVehicleStates(act.ID, act.FleetID)
	if err != nil {
		apiError(c, err)
		return
	}
	c.JSON(http.StatusOK, states)
}

func getActivityOverview(c *gin.Context) {
	var act model.Activity
	if err := db.DB.Where("id = ?", c.Param("id")).First(&act).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": "NOT_FOUND", "message": "活动不存在"})
		return
	}
	states, _ := service.GetVehicleStates(act.ID, act.FleetID)
	c.JSON(http.StatusOK, service.ComputeOverview(states))
}

func getActivityTimeline(c *gin.Context) {
	vid := c.Query("vehicleId")
	var act model.Activity
	if err := db.DB.Where("id = ?", c.Param("id")).First(&act).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": "NOT_FOUND", "message": "活动不存在"})
		return
	}
	var versions []model.ConfigVersion
	q := db.DB.Where("activity_id = ?", act.ID)
	if vid != "" {
		q = q.Where("vehicle_id = ?", vid)
	}
	q.Order("vehicle_id asc, version_no asc").Find(&versions)
	c.JSON(http.StatusOK, versions)
}

func getActivityAudit(c *gin.Context) {
	var list []model.AuditLog
	db.DB.Where("activity_id = ?", c.Param("id")).Order("created_at asc").Find(&list)
	c.JSON(http.StatusOK, list)
}

func startPilot(c *gin.Context) {
	var req struct {
		VehicleIds []string `json:"vehicleIds"`
	}
	c.ShouldBindJSON(&req)
	uid := service.UIDFromString(middleware.CurrentUID(c))
	act, err := service.StartPilot(uid, middleware.CurrentRole(c), rid(c), c.Param("id"), req.VehicleIds)
	if err != nil {
		apiError(c, err)
		return
	}
	c.JSON(http.StatusOK, act)
}

func rollout(c *gin.Context) {
	uid := service.UIDFromString(middleware.CurrentUID(c))
	act, err := service.Rollout(uid, middleware.CurrentRole(c), rid(c), c.Param("id"))
	if err != nil {
		apiError(c, err)
		return
	}
	c.JSON(http.StatusOK, act)
}

func stopActivity(c *gin.Context) {
	var req struct {
		Reason string `json:"reason"`
	}
	c.ShouldBindJSON(&req)
	uid := service.UIDFromString(middleware.CurrentUID(c))
	act, err := service.Stop(uid, middleware.CurrentRole(c), rid(c), c.Param("id"), req.Reason)
	if err != nil {
		apiError(c, err)
		return
	}
	c.JSON(http.StatusOK, act)
}

func requestRestore(c *gin.Context) {
	uid := service.UIDFromString(middleware.CurrentUID(c))
	act, err := service.RequestRestore(uid, middleware.CurrentRole(c), rid(c), c.Param("id"))
	if err != nil {
		apiError(c, err)
		return
	}
	c.JSON(http.StatusOK, act)
}

func approveRestore(c *gin.Context) {
	uid := service.UIDFromString(middleware.CurrentUID(c))
	act, err := service.ApproveRestore(uid, middleware.CurrentRole(c), rid(c), c.Param("id"))
	if err != nil {
		apiError(c, err)
		return
	}
	c.JSON(http.StatusOK, act)
}

func executeRestore(c *gin.Context) {
	uid := service.UIDFromString(middleware.CurrentUID(c))
	act, err := service.ExecuteRestore(uid, middleware.CurrentRole(c), rid(c), c.Param("id"))
	if err != nil {
		apiError(c, err)
		return
	}
	c.JSON(http.StatusOK, act)
}

func completeActivity(c *gin.Context) {
	uid := service.UIDFromString(middleware.CurrentUID(c))
	act, err := service.CompleteActivity(uid, middleware.CurrentRole(c), rid(c), c.Param("id"))
	if err != nil {
		apiError(c, err)
		return
	}
	c.JSON(http.StatusOK, act)
}

func completeRestore(c *gin.Context) {
	uid := service.UIDFromString(middleware.CurrentUID(c))
	act, err := service.CompleteRestore(uid, middleware.CurrentRole(c), rid(c), c.Param("id"))
	if err != nil {
		apiError(c, err)
		return
	}
	c.JSON(http.StatusOK, act)
}

// ---- 车端 ----
func pullConfig(c *gin.Context) {
	vid := middleware.CurrentVID(c)
	actID := c.Query("activityId")
	res, err := service.PullConfig(vid, rid(c), actID)
	if err != nil {
		apiError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func submitReceipt(c *gin.Context) {
	var req service.ReceiptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": "BAD_REQUEST", "message": err.Error()})
		return
	}
	vid := middleware.CurrentVID(c)
	res, err := service.SubmitReceipt(vid, rid(c), req)
	if err != nil {
		// 业务码（RECEIPT_REJECTED 等）仍以 200 返回业务结构，便于前端处理
		code := http.StatusOK
		if res != nil && res.Code != "" {
			c.JSON(code, res)
			return
		}
		apiError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func listAudit(c *gin.Context) {
	action := c.Query("action")
	vehicle := c.Query("vehicleId")
	q := db.DB.Model(&model.AuditLog{})
	if action != "" {
		q = q.Where("action = ?", action)
	}
	if vehicle != "" {
		q = q.Where("target_vehicle = ?", vehicle)
	}
	var list []model.AuditLog
	q.Order("created_at asc").Find(&list)
	c.JSON(http.StatusOK, list)
}

// devReset 仅开发环境：清空活动相关数据并将车辆生效值回到前值
func devReset(c *gin.Context) {
	if !config.Get().EnableReset {
		c.JSON(http.StatusForbidden, gin.H{"code": "FORBIDDEN", "message": "未启用重置（ENABLE_RESET=1）"})
		return
	}
	db.DB.Exec("DELETE FROM audit_logs")
	db.DB.Exec("DELETE FROM config_versions")
	db.DB.Exec("DELETE FROM activities")
	db.DB.Exec("DELETE FROM application_snapshots")
	db.DB.Exec("DELETE FROM applications")
	db.DB.Exec("UPDATE vehicles SET effective_interval = prev_interval, effective_version_no = 0")
	c.JSON(http.StatusOK, gin.H{"code": "OK", "message": "已重置开发数据"})
}

// ---- 工具 ----
func rid(c *gin.Context) string {
	v, _ := c.Get("requestId")
	s, _ := v.(string)
	return s
}

func apiError(c *gin.Context, err error) {
	msg := err.Error()
	status := http.StatusBadRequest
	code := "ERROR"
	switch {
	case strings.HasPrefix(msg, "VALIDATION_FAILED"):
		status = http.StatusBadRequest
		code = "VALIDATION_FAILED"
	case strings.HasPrefix(msg, "SELF_APPROVAL_FORBIDDEN"):
		status = http.StatusForbidden
		code = "SELF_APPROVAL_FORBIDDEN"
	case strings.HasPrefix(msg, "REASON_REQUIRED"):
		status = http.StatusBadRequest
		code = "REASON_REQUIRED"
	case strings.HasPrefix(msg, "FLEET_HAS_ACTIVE_ACTIVITY"):
		status = http.StatusConflict
		code = "FLEET_HAS_ACTIVE_ACTIVITY"
	case strings.HasPrefix(msg, "PILOT_NOT_PASSED"):
		status = http.StatusConflict
		code = "PILOT_NOT_PASSED"
	case strings.HasPrefix(msg, "PILOT_SIZE_INVALID"):
		status = http.StatusBadRequest
		code = "PILOT_SIZE_INVALID"
	case strings.HasPrefix(msg, "NOT_ALL_CONFIRMED"):
		status = http.StatusConflict
		code = "NOT_ALL_CONFIRMED"
	case strings.HasPrefix(msg, "UNCONFIRMED_REMAINS"):
		status = http.StatusConflict
		code = "UNCONFIRMED_REMAINS"
	case strings.HasPrefix(msg, "NO_AFFECTED_VEHICLE"):
		status = http.StatusConflict
		code = "NO_AFFECTED_VEHICLE"
	case strings.HasPrefix(msg, "RESTORE_NOT_APPROVED"):
		status = http.StatusForbidden
		code = "RESTORE_NOT_APPROVED"
	case strings.HasPrefix(msg, "RECEIPT_REJECTED"):
		status = http.StatusForbidden
		code = "RECEIPT_REJECTED"
	case strings.HasPrefix(msg, "PARAM_FROZEN"):
		status = http.StatusForbidden
		code = "PARAM_FROZEN"
	}
	c.JSON(status, gin.H{"code": code, "message": msg})
}
