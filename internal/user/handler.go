package user

import (
	"errors"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/hui0882/chatroom/internal/middleware"
	"github.com/hui0882/chatroom/pkg/response"
)

// Handler 处理用户相关的 HTTP 请求。
type Handler struct {
	svc *Service
}

// NewHandler 创建 Handler。
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// ─── POST /api/v1/auth/register ──────────────────────────────────────────────

func (h *Handler) Register(c *gin.Context) {
	var in RegisterInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, response.CodeInvalidParam, err.Error())
		return
	}

	u, err := h.svc.Register(c.Request.Context(), in)
	if err != nil {
		if errors.Is(err, ErrUsernameExists) {
			response.Fail(c, response.CodeUserExists, "username already exists")
			return
		}
		response.Fail(c, response.CodeInvalidParam, err.Error())
		return
	}

	response.OK(c, toPublicUser(u))
}

// ─── POST /api/v1/auth/login ─────────────────────────────────────────────────

func (h *Handler) Login(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeInvalidParam, err.Error())
		return
	}

	result, err := h.svc.Login(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			response.Fail(c, response.CodeWrongPassword, "wrong username or password")
			return
		}
		// 封禁信息直接返回（已包含原因和截止时间）
		response.Fail(c, response.CodeUserBanned, err.Error())
		return
	}

	// 同时设置 Cookie（Web 端自动携带）
	c.SetCookie(
		middleware.SessionCookieName,
		result.SessionID,
		7*24*3600, // 7 天
		"/",
		"",    // domain
		false, // secure（生产环境应为 true）
		true,  // httpOnly
	)

	response.OK(c, gin.H{
		"session_id": result.SessionID,
		"user":       toPublicUser(result.User),
	})
}

// ─── POST /api/v1/auth/logout ────────────────────────────────────────────────

func (h *Handler) Logout(c *gin.Context) {
	info := middleware.MustGetSession(c)
	_ = h.svc.Logout(c.Request.Context(), info.SessionID)

	c.SetCookie(middleware.SessionCookieName, "", -1, "/", "", false, true)
	response.OK(c, nil)
}

// ─── GET /api/v1/user/me ─────────────────────────────────────────────────────

func (h *Handler) Me(c *gin.Context) {
	info := middleware.MustGetSession(c)
	u, err := h.svc.GetByID(c.Request.Context(), info.UserID)
	if err != nil {
		response.Fail(c, response.CodeNotFound, "user not found")
		return
	}
	response.OK(c, toPublicUser(u))
}

// ─── PUT /api/v1/user/password ───────────────────────────────────────────────

func (h *Handler) ChangePassword(c *gin.Context) {
	var req struct {
		OldPassword string `json:"old_password" binding:"required"`
		NewPassword string `json:"new_password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeInvalidParam, err.Error())
		return
	}
	info := middleware.MustGetSession(c)
	if err := h.svc.ChangePassword(c.Request.Context(), info.UserID, req.OldPassword, req.NewPassword); err != nil {
		response.Fail(c, response.CodeInvalidParam, err.Error())
		return
	}
	response.OK(c, nil)
}

// ─── 管理员接口 ──────────────────────────────────────────────────────────────

// GET /api/v1/admin/users
func (h *Handler) AdminListUsers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	filter := ListFilter{
		Status:   c.Query("status"),
		Keyword:  c.Query("keyword"),
		Page:     page,
		PageSize: pageSize,
	}
	users, total, err := h.svc.ListUsers(c.Request.Context(), filter)
	if err != nil {
		response.Fail(c, response.CodeInternalError, err.Error())
		return
	}
	list := make([]gin.H, 0, len(users))
	for _, u := range users {
		list = append(list, toAdminUser(u))
	}
	response.OK(c, gin.H{"total": total, "list": list})
}

// POST /api/v1/admin/users/:id/ban
func (h *Handler) AdminBanUser(c *gin.Context) {
	uid, err := parseUID(c)
	if err != nil {
		return
	}
	var in BanInput
	if err := c.ShouldBindJSON(&in); err != nil {
		response.Fail(c, response.CodeInvalidParam, err.Error())
		return
	}
	if err := h.svc.BanUser(c.Request.Context(), uid, in); err != nil {
		if errors.Is(err, ErrNotFound) {
			response.Fail(c, response.CodeNotFound, "user not found")
			return
		}
		response.Fail(c, response.CodeInternalError, err.Error())
		return
	}
	response.OK(c, nil)
}

// POST /api/v1/admin/users/:id/unban
func (h *Handler) AdminUnbanUser(c *gin.Context) {
	uid, err := parseUID(c)
	if err != nil {
		return
	}
	if err := h.svc.UnbanUser(c.Request.Context(), uid); err != nil {
		response.Fail(c, response.CodeInternalError, err.Error())
		return
	}
	response.OK(c, nil)
}

// DELETE /api/v1/admin/users/:id
func (h *Handler) AdminDeleteUser(c *gin.Context) {
	uid, err := parseUID(c)
	if err != nil {
		return
	}
	if err := h.svc.DeleteUser(c.Request.Context(), uid); err != nil {
		if errors.Is(err, ErrNotFound) {
			response.Fail(c, response.CodeNotFound, "user not found")
			return
		}
		response.Fail(c, response.CodeInternalError, err.Error())
		return
	}
	response.OK(c, nil)
}

// POST /api/v1/admin/users/:id/restore
func (h *Handler) AdminRestoreUser(c *gin.Context) {
	uid, err := parseUID(c)
	if err != nil {
		return
	}
	if err := h.svc.RestoreUser(c.Request.Context(), uid); err != nil {
		response.Fail(c, response.CodeInternalError, err.Error())
		return
	}
	response.OK(c, nil)
}

// POST /api/v1/admin/users/:id/reset-password
func (h *Handler) AdminResetPassword(c *gin.Context) {
	uid, err := parseUID(c)
	if err != nil {
		return
	}
	var req struct {
		NewPassword string `json:"new_password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, response.CodeInvalidParam, err.Error())
		return
	}
	if err := h.svc.ResetPassword(c.Request.Context(), uid, req.NewPassword); err != nil {
		response.Fail(c, response.CodeInvalidParam, err.Error())
		return
	}
	response.OK(c, nil)
}

// POST /api/v1/admin/users/:id/kick
func (h *Handler) AdminKickUser(c *gin.Context) {
	uid, err := parseUID(c)
	if err != nil {
		return
	}
	if err := h.svc.KickUser(c.Request.Context(), uid); err != nil {
		response.Fail(c, response.CodeInternalError, err.Error())
		return
	}
	response.OK(c, nil)
}

// ─── 辅助 ─────────────────────────────────────────────────────────────────────

func parseUID(c *gin.Context) (int64, error) {
	uid, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.Fail(c, response.CodeInvalidParam, "invalid user id")
	}
	return uid, err
}

// PublicUser 是对外暴露的用户信息（隐藏敏感字段）
type PublicUser struct {
	ID        int64     `json:"id"`
	Username  string    `json:"username"`
	Nickname  string    `json:"nickname"`
	AvatarURL string    `json:"avatar_url"`
	Bio       string    `json:"bio"`
	Gender    string    `json:"gender"`
	Age       *int      `json:"age,omitempty"`
	Role      string    `json:"role"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

func toPublicUser(u *User) PublicUser {
	return PublicUser{
		ID:        u.ID,
		Username:  u.Username,
		Nickname:  u.Nickname,
		AvatarURL: u.AvatarURL,
		Bio:       u.Bio,
		Gender:    u.Gender,
		Age:       u.Age,
		Role:      u.Role,
		Status:    u.Status,
		CreatedAt: u.CreatedAt,
	}
}

// toAdminUser 包含管理员可见的额外字段
func toAdminUser(u *User) gin.H {
	m := gin.H{
		"id":         u.ID,
		"username":   u.Username,
		"nickname":   u.Nickname,
		"avatar_url": u.AvatarURL,
		"gender":     u.Gender,
		"age":        u.Age,
		"role":       u.Role,
		"status":     u.Status,
		"created_at": u.CreatedAt,
	}
	if u.BanReason != "" {
		m["ban_reason"] = u.BanReason
	}
	if u.BanUntil != nil {
		m["ban_until"] = u.BanUntil
	}
	if u.DeletedAt != nil {
		m["deleted_at"] = u.DeletedAt
	}
	return m
}
