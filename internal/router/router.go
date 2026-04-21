package router

import (
	"github.com/gin-gonic/gin"

	"github.com/hui0882/chatroom/internal/middleware"
	"github.com/hui0882/chatroom/internal/server"
	"github.com/hui0882/chatroom/internal/ws"
	"github.com/hui0882/chatroom/pkg/config"
)

// Setup 注册所有路由并返回配置好的 gin.Engine
func Setup(cfg *config.Config, app *server.AppContext) *gin.Engine {
	if cfg.App.Debug {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()

	// 全局中间件
	r.Use(middleware.Recovery())
	r.Use(middleware.RequestLogger(cfg.App.Debug))

	// ── 健康检查 ──────────────────────────────────────────────
	r.GET("/health", app.Health.Check)

	// ── WebSocket 测试接口（无需认证，开发调试用）────────────
	r.GET("/websocket_test", ws.TestEchoHandler)

	// ── WebSocket 流式输出测试（无需认证，模拟 AI 流式回复）─
	r.GET("/websocket_stream", ws.StreamEchoHandler)

	// ── WebSocket 正式连接（需登录 Session）────────────────────
	r.GET("/ws", app.WS.Serve)

	// ── API v1 ────────────────────────────────────────────────
	v1 := r.Group("/api/v1")
	{
		// 无需鉴权
		auth := v1.Group("/auth")
		{
			auth.POST("/register", app.User.Register)
			auth.POST("/login", app.User.Login)
			auth.POST("/logout", middleware.Auth(app.Session), app.User.Logout)
		}

		// 需要登录
		userGroup := v1.Group("/user", middleware.Auth(app.Session))
		{
			userGroup.GET("/me", app.User.Me)
			userGroup.PUT("/password", app.User.ChangePassword)
		}

		// 管理员专属接口
		admin := v1.Group("/admin",
			middleware.Auth(app.Session),
			middleware.AdminOnly(),
		)
		{
			admin.GET("/users", app.User.AdminListUsers)
			admin.POST("/users/:id/ban", app.User.AdminBanUser)
			admin.POST("/users/:id/unban", app.User.AdminUnbanUser)
			admin.DELETE("/users/:id", app.User.AdminDeleteUser)
			admin.POST("/users/:id/restore", app.User.AdminRestoreUser)
			admin.POST("/users/:id/reset-password", app.User.AdminResetPassword)
			admin.POST("/users/:id/kick", app.User.AdminKickUser)
		}
	}

	return r
}
