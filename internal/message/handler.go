package message

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/hui0882/chatroom/internal/middleware"
)

// Handler 处理消息相关 HTTP 请求。
type Handler struct {
	svc *Service
}

// NewHandler 创建消息 Handler。
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// ListHistory GET /api/v1/messages/:peer_uid
// 查询与指定用户的聊天历史（游标翻页）
// Query: cursor=<msg_id>（不传或传 0 从最新开始），limit=<n>（默认 50，最大 100）
func (h *Handler) ListHistory(c *gin.Context) {
	sess := middleware.MustGetSession(c)
	if sess == nil {
		return
	}
	selfUID := sess.UserID

	peerUID, err := strconv.ParseInt(c.Param("peer_uid"), 10, 64)
	if err != nil || peerUID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1001, "msg": "invalid peer_uid"})
		return
	}

	cursor, _ := strconv.ParseInt(c.Query("cursor"), 10, 64)
	limit, _ := strconv.Atoi(c.Query("limit"))
	if limit <= 0 {
		limit = 50
	}

	msgs, err := h.svc.ListHistory(c.Request.Context(), selfUID, peerUID, cursor, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 5000, "msg": "内部错误"})
		return
	}
	if msgs == nil {
		msgs = []*Message{}
	}

	// 计算下一页游标（最后一条消息的 ID，即 list 末尾最小 ID）
	var nextCursor int64
	if len(msgs) == limit {
		nextCursor = msgs[len(msgs)-1].ID
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"list":        msgs,
			"next_cursor": nextCursor, // 0 表示没有更多了
		},
	})
}

// GetUnread GET /api/v1/messages/unread
// 获取当前用户所有对话的未读计数
func (h *Handler) GetUnread(c *gin.Context) {
	sess := middleware.MustGetSession(c)
	if sess == nil {
		return
	}

	counts, err := h.svc.GetUnread(c.Request.Context(), sess.UserID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 5000, "msg": "内部错误"})
		return
	}
	if counts == nil {
		counts = map[int64]int64{}
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": counts})
}

// ClearUnread POST /api/v1/messages/unread/:peer_uid/clear
// 清零与指定用户会话的未读数（打开聊天窗口时调用）
func (h *Handler) ClearUnread(c *gin.Context) {
	sess := middleware.MustGetSession(c)
	if sess == nil {
		return
	}

	peerUID, err := strconv.ParseInt(c.Param("peer_uid"), 10, 64)
	if err != nil || peerUID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1001, "msg": "invalid peer_uid"})
		return
	}

	if err := h.svc.ClearUnread(c.Request.Context(), sess.UserID, peerUID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 5000, "msg": "内部错误"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "msg": "ok"})
}

// 确保 errors 包被引用
var _ = errors.New
