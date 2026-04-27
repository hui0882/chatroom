package friend

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/hui0882/chatroom/internal/middleware"
	"github.com/hui0882/chatroom/pkg/response"
)

// Handler HTTP 处理层
type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// ─── GET /api/v1/users/search?username=xxx ────────────────────────────────────

func (h *Handler) SearchUser(c *gin.Context) {
	keyword := c.Query("username")
	if keyword == "" {
		response.Fail(c, response.CodeInvalidParam, "username is required")
		return
	}
	info := middleware.MustGetSession(c)
	u, err := h.svc.SearchUser(c.Request.Context(), info.UserID, keyword)
	if err != nil {
		if errors.Is(err, ErrSelfOperation) {
			response.Fail(c, response.CodeInvalidParam, "cannot search yourself")
			return
		}
		response.Fail(c, response.CodeNotFound, "user not found")
		return
	}

	// 查询是否已是好友
	isFriend, _ := h.svc.IsFriend(c.Request.Context(), info.UserID, u.ID)

	response.OK(c, gin.H{
		"id":         u.ID,
		"username":   u.Username,
		"nickname":   u.Nickname,
		"avatar_url": u.AvatarURL,
		"gender":     u.Gender,
		"is_friend":  isFriend,
	})
}

// ─── POST /api/v1/friends/requests ───────────────────────────────────────────

func (h *Handler) SendRequest(c *gin.Context) {
	var req struct {
		ToUID   int64  `json:"to_uid" binding:"required"`
		Message string `json:"message"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeInvalidParam, err.Error())
		return
	}
	info := middleware.MustGetSession(c)
	result, err := h.svc.SendRequest(c.Request.Context(), info.UserID, req.ToUID, req.Message)
	if err != nil {
		switch {
		case errors.Is(err, ErrSelfOperation):
			response.Fail(c, response.CodeInvalidParam, "cannot add yourself")
		case errors.Is(err, ErrAlreadyFriends):
			response.Fail(c, response.CodeInvalidParam, "already friends")
		default:
			response.Fail(c, response.CodeInternalError, err.Error())
		}
		return
	}
	response.OK(c, result)
}

// ─── POST /api/v1/friends/requests/:id/cancel ────────────────────────────────

func (h *Handler) CancelRequest(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		return
	}
	info := middleware.MustGetSession(c)
	if err := h.svc.CancelRequest(c.Request.Context(), info.UserID, id); err != nil {
		response.Fail(c, response.CodeNotFound, err.Error())
		return
	}
	response.OK(c, nil)
}

// ─── POST /api/v1/friends/requests/:id/accept ────────────────────────────────

func (h *Handler) AcceptRequest(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		return
	}
	info := middleware.MustGetSession(c)
	if err := h.svc.AcceptRequest(c.Request.Context(), info.UserID, id); err != nil {
		response.Fail(c, response.CodeNotFound, err.Error())
		return
	}
	response.OK(c, nil)
}

// ─── POST /api/v1/friends/requests/:id/reject ────────────────────────────────

func (h *Handler) RejectRequest(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		return
	}
	info := middleware.MustGetSession(c)
	if err := h.svc.RejectRequest(c.Request.Context(), info.UserID, id); err != nil {
		response.Fail(c, response.CodeNotFound, err.Error())
		return
	}
	response.OK(c, nil)
}

// ─── GET /api/v1/friends ──────────────────────────────────────────────────────

func (h *Handler) ListFriends(c *gin.Context) {
	info := middleware.MustGetSession(c)
	list, err := h.svc.ListFriends(c.Request.Context(), info.UserID)
	if err != nil {
		response.Fail(c, response.CodeInternalError, err.Error())
		return
	}
	if list == nil {
		list = []*Friend{}
	}
	response.OK(c, list)
}

// ─── DELETE /api/v1/friends/:id ───────────────────────────────────────────────

func (h *Handler) DeleteFriend(c *gin.Context) {
	friendUID, err := parseID(c)
	if err != nil {
		return
	}
	info := middleware.MustGetSession(c)
	if err := h.svc.DeleteFriend(c.Request.Context(), info.UserID, friendUID); err != nil {
		response.Fail(c, response.CodeInternalError, err.Error())
		return
	}
	response.OK(c, nil)
}

// ─── GET /api/v1/friends/requests/received ───────────────────────────────────

func (h *Handler) ListReceivedRequests(c *gin.Context) {
	info := middleware.MustGetSession(c)
	list, err := h.svc.ListReceivedRequests(c.Request.Context(), info.UserID)
	if err != nil {
		response.Fail(c, response.CodeInternalError, err.Error())
		return
	}
	if list == nil {
		list = []*Request{}
	}
	response.OK(c, list)
}

// ─── GET /api/v1/friends/requests/sent ───────────────────────────────────────

func (h *Handler) ListSentRequests(c *gin.Context) {
	info := middleware.MustGetSession(c)
	list, err := h.svc.ListSentRequests(c.Request.Context(), info.UserID)
	if err != nil {
		response.Fail(c, response.CodeInternalError, err.Error())
		return
	}
	if list == nil {
		list = []*Request{}
	}
	response.OK(c, list)
}

// ─── 辅助 ─────────────────────────────────────────────────────────────────────

func parseID(c *gin.Context) (int64, error) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, response.CodeInvalidParam, "invalid id")
	}
	return id, err
}
