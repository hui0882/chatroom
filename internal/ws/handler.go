package ws

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"github.com/hui0882/chatroom/internal/session"
	"github.com/hui0882/chatroom/pkg/logger"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	// 生产环境应校验 Origin，这里允许所有（开发阶段）
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Upgrade 将 HTTP 连接升级为 WebSocket，并验证 Session。
// sessionValidator 由外部（auth 中间件）提供，返回 (userID, device, error)。
type SessionValidator func(r *http.Request) (userID int64, device string, err error)

// Handler 封装 WebSocket 升级逻辑。
type Handler struct {
	hub       *Hub
	validator SessionValidator
}

// NewHandler 创建 WebSocket Handler。
func NewHandler(hub *Hub, validator SessionValidator) *Handler {
	return &Handler{hub: hub, validator: validator}
}

// BuildSessionValidator 根据 session.Manager 构造 WS SessionValidator。
// 优先取 URL query param ?session_id=xxx，否则取 Header X-Session-Id。
func BuildSessionValidator(sm *session.Manager) SessionValidator {
	return func(r *http.Request) (int64, string, error) {
		sid := r.URL.Query().Get("session_id")
		if sid == "" {
			sid = r.Header.Get("X-Session-Id")
		}
		if sid == "" {
			// 尝试从 Cookie 读取
			if cookie, err := r.Cookie("sessionId"); err == nil {
				sid = cookie.Value
			}
		}
		if sid == "" {
			return 0, "", errors.New("missing session_id")
		}
		info, err := sm.Get(r.Context(), sid)
		if err != nil {
			return 0, "", err
		}
		// 设备类型：从 query param 取，默认 "web"
		device := r.URL.Query().Get("device")
		if device == "" {
			device = "web"
		}
		return info.UserID, device, nil
	}
}

// Serve 是 WebSocket 连接入口，负责：
// 1. 验证 Session
// 2. 升级连接
// 3. 注册 Client 到 Hub
// 4. 启动 ReadPump / WritePump
func (h *Handler) Serve(c *gin.Context) {
	uid, device, err := h.validator(c.Request)
	if err != nil {
		logger.L().Warn("ws auth failed", zap.Error(err))
		c.JSON(http.StatusUnauthorized, gin.H{"code": 1002, "msg": "unauthorized"})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logger.L().Error("ws upgrade failed", zap.Error(err))
		return
	}

	client := NewClient(h.hub, conn, uid, device)
	h.hub.Register <- client

	go client.WritePump()
	go client.ReadPump()
}
