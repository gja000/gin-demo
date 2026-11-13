package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strings"

	"gin-demo/internal/auth"

	"github.com/gin-gonic/gin"
)

const (
	ctxUID  = "uid"
	ctxRole = "role"
	ctxVID  = "vid"
)

// RequestID 注入请求 ID 用于审计与链路追踪
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		rid := c.GetHeader("X-Request-ID")
		if rid == "" {
			rid = newRequestID()
		}
		c.Set("requestId", rid)
		c.Header("X-Request-ID", rid)
		c.Next()
	}
}

// UserAuth 用户鉴权；可传入允许的角色（空表示任意登录用户）
func UserAuth(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractBearer(c)
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": "UNAUTHORIZED", "message": "缺少认证令牌"})
			return
		}
		cl, err := auth.ParseUserToken(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": "UNAUTHORIZED", "message": "令牌无效"})
			return
		}
		if len(roles) > 0 {
			ok := false
			for _, r := range roles {
				if cl.Role == r {
					ok = true
					break
				}
			}
			if !ok {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"code": "FORBIDDEN", "message": "角色无权访问"})
				return
			}
		}
		c.Set(ctxUID, cl.Sub)
		c.Set(ctxRole, cl.Role)
		c.Next()
	}
}

// VehicleAuth 车端鉴权（与用户令牌隔离）
func VehicleAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractBearer(c)
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": "UNAUTHORIZED", "message": "缺少车端令牌"})
			return
		}
		cl, err := auth.ParseVehicleToken(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": "UNAUTHORIZED", "message": "车端令牌无效"})
			return
		}
		c.Set(ctxVID, cl.Sub)
		c.Next()
	}
}

func extractBearer(c *gin.Context) string {
	h := c.GetHeader("Authorization")
	if strings.HasPrefix(h, "Bearer ") {
		return strings.TrimSpace(h[7:])
	}
	return ""
}

func CurrentUID(c *gin.Context) string { v, _ := c.Get(ctxUID); s, _ := v.(string); return s }
func CurrentRole(c *gin.Context) string { v, _ := c.Get(ctxRole); s, _ := v.(string); return s }
func CurrentVID(c *gin.Context) string { v, _ := c.Get(ctxVID); s, _ := v.(string); return s }

func newRequestID() string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return "req-" + hex.EncodeToString(b)
}
