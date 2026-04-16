package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/hui0882/chatroom/internal/session"
	"github.com/hui0882/chatroom/pkg/response"
)

const (
	// SessionKey 是从 Cookie 或 Header 中读取 Session ID 的键名
	SessionCookieName = "sessionId"
	SessionHeaderName = "X-Session-Id"

	// ContextKeySession 是注入到 gin.Context 中的键名
	ContextKeySession = "session"
)

// Auth 是登录鉴权中间件，验证 Session 合法性并将 session.Info 注入 Context。
// 同时支持 Cookie 和 X-Session-Id Header，优先取 Cookie。
func Auth(sm *session.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		sid := extractSessionID(c)
		if sid == "" {
			response.FailWithStatus(c, http.StatusUnauthorized, response.CodeUnauthorized, "please login first")
			c.Abort()
			return
		}

		info, err := sm.Get(c.Request.Context(), sid)
		if err != nil {
			response.FailWithStatus(c, http.StatusUnauthorized, response.CodeUnauthorized, "session expired or invalid")
			c.Abort()
			return
		}

		c.Set(ContextKeySession, info)
		c.Next()
	}
}

// AdminOnly 在 Auth 之后使用，限制接口仅管理员可访问。
func AdminOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		info := MustGetSession(c)
		if info.Role != "admin" {
			response.FailWithStatus(c, http.StatusForbidden, response.CodeForbidden, "admin only")
			c.Abort()
			return
		}
		c.Next()
	}
}

// MustGetSession 从 Context 中取出 session.Info，调用前必须已经过 Auth 中间件。
func MustGetSession(c *gin.Context) *session.Info {
	v, _ := c.Get(ContextKeySession)
	return v.(*session.Info)
}

// extractSessionID 按优先级提取 Session ID：Cookie > Header
func extractSessionID(c *gin.Context) string {
	if cookie, err := c.Cookie(SessionCookieName); err == nil && cookie != "" {
		return cookie
	}
	return c.GetHeader(SessionHeaderName)
}
