package user

import (
	"context"
	"fmt"
	"regexp"
	"time"
	"unicode"

	"golang.org/x/crypto/bcrypt"

	"github.com/hui0882/chatroom/internal/session"
	"github.com/hui0882/chatroom/internal/ws"
)

const bcryptCost = bcrypt.DefaultCost

// Service 封装用户相关业务逻辑。
type Service struct {
	repo    Repository
	session *session.Manager
	hub     *ws.Hub
}

// NewService 创建 Service。
func NewService(repo Repository, sm *session.Manager, hub *ws.Hub) *Service {
	return &Service{repo: repo, session: sm, hub: hub}
}

// ─── 注册 ────────────────────────────────────────────────────────────────────

// RegisterInput 注册请求参数
type RegisterInput struct {
	Username string `json:"username" binding:"required,min=3,max=30"`
	Password string `json:"password" binding:"required"`
	Nickname string `json:"nickname" binding:"required,min=1,max=30"`
	Gender   string `json:"gender"` // unknown / male / female，可选
	Age      *int   `json:"age"`    // 可选
}

// Register 注册新用户，返回创建后的用户对象。
func (s *Service) Register(ctx context.Context, in RegisterInput) (*User, error) {
	// 1. 密码强度：至少 8 位，同时包含字母和数字
	if err := validatePassword(in.Password); err != nil {
		return nil, err
	}

	// 2. 用户名格式：3~30 位，只允许字母/数字/下划线
	if !usernameRe.MatchString(in.Username) {
		return nil, fmt.Errorf("username must be 3-30 chars, letters/digits/underscore only")
	}

	// 3. 性别枚举校验
	gender := normalizeGender(in.Gender)

	// 4. 哈希密码
	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcryptCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	u := &User{
		Username:     in.Username,
		PasswordHash: string(hash),
		Nickname:     in.Nickname,
		Gender:       gender,
		Age:          in.Age,
		Role:         "user",
	}
	return s.repo.Create(ctx, u)
}

// ─── 登录 ────────────────────────────────────────────────────────────────────

// LoginResult 登录成功后的返回值
type LoginResult struct {
	SessionID string `json:"session_id"`
	User      *User  `json:"user"`
}

// Login 验证用户名和密码，成功后创建 Session。
func (s *Service) Login(ctx context.Context, username, password string) (*LoginResult, error) {
	u, err := s.repo.FindByUsername(ctx, username)
	if err != nil {
		return nil, ErrNotFound
	}

	// 检查账号状态
	if u.IsDeleted() {
		return nil, fmt.Errorf("account does not exist")
	}
	if u.IsBanned() {
		msg := "account is banned"
		if u.BanReason != "" {
			msg += ": " + u.BanReason
		}
		if u.BanUntil != nil {
			msg += fmt.Sprintf(" (until %s)", u.BanUntil.Format("2006-01-02 15:04:05"))
		}
		return nil, fmt.Errorf(msg)
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return nil, fmt.Errorf("wrong username or password")
	}

	// 创建 Session
	sid, err := s.session.Create(ctx, u.ID, u.Role)
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	return &LoginResult{SessionID: sid, User: u}, nil
}

// ─── 退出登录 ────────────────────────────────────────────────────────────────

func (s *Service) Logout(ctx context.Context, sid string) error {
	return s.session.Delete(ctx, sid)
}

// ─── 获取用户信息 ─────────────────────────────────────────────────────────────

func (s *Service) GetByID(ctx context.Context, id int64) (*User, error) {
	return s.repo.FindByID(ctx, id)
}

// ─── 修改密码 ────────────────────────────────────────────────────────────────

func (s *Service) ChangePassword(ctx context.Context, uid int64, oldPwd, newPwd string) error {
	u, err := s.repo.FindByID(ctx, uid)
	if err != nil {
		return err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(oldPwd)); err != nil {
		return fmt.Errorf("wrong current password")
	}
	if err := validatePassword(newPwd); err != nil {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPwd), bcryptCost)
	if err != nil {
		return err
	}
	return s.repo.UpdatePassword(ctx, uid, string(hash))
}

// ─── 管理员：封禁 ─────────────────────────────────────────────────────────────

// BanInput 封禁参数
type BanInput struct {
	Reason   string     `json:"reason" binding:"required"`
	BanUntil *time.Time `json:"ban_until"` // nil = 永久封禁
}

func (s *Service) BanUser(ctx context.Context, targetUID int64, in BanInput) error {
	if err := s.repo.Ban(ctx, targetUID, in.Reason, in.BanUntil); err != nil {
		return err
	}
	// 强制清除该用户所有 Session
	if err := s.session.DeleteAll(ctx, targetUID); err != nil {
		return fmt.Errorf("delete sessions: %w", err)
	}
	// 踢出 WebSocket 连接
	s.hub.KickUser(targetUID)
	return nil
}

// ─── 管理员：解封 ─────────────────────────────────────────────────────────────

func (s *Service) UnbanUser(ctx context.Context, targetUID int64) error {
	return s.repo.Unban(ctx, targetUID)
}

// ─── 管理员：逻辑删除 ─────────────────────────────────────────────────────────

func (s *Service) DeleteUser(ctx context.Context, targetUID int64) error {
	if err := s.repo.SoftDelete(ctx, targetUID); err != nil {
		return err
	}
	_ = s.session.DeleteAll(ctx, targetUID)
	s.hub.KickUser(targetUID)
	return nil
}

// ─── 管理员：恢复账号 ─────────────────────────────────────────────────────────

func (s *Service) RestoreUser(ctx context.Context, targetUID int64) error {
	return s.repo.Restore(ctx, targetUID)
}

// ─── 管理员：重置密码 ─────────────────────────────────────────────────────────

func (s *Service) ResetPassword(ctx context.Context, targetUID int64, newPwd string) error {
	if err := validatePassword(newPwd); err != nil {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPwd), bcryptCost)
	if err != nil {
		return err
	}
	return s.repo.UpdatePassword(ctx, targetUID, string(hash))
}

// ─── 管理员：强制踢出 WS 连接 ────────────────────────────────────────────────

func (s *Service) KickUser(ctx context.Context, targetUID int64) error {
	s.hub.KickUser(targetUID)
	return s.session.DeleteAll(ctx, targetUID)
}

// ─── 管理员：用户列表 ─────────────────────────────────────────────────────────

func (s *Service) ListUsers(ctx context.Context, f ListFilter) ([]*User, int64, error) {
	return s.repo.List(ctx, f)
}

// ─── 辅助函数 ─────────────────────────────────────────────────────────────────

var usernameRe = regexp.MustCompile(`^[a-zA-Z0-9_]{3,30}$`)

func validatePassword(pwd string) error {
	if len(pwd) < 8 {
		return fmt.Errorf("password must be at least 8 characters")
	}
	hasLetter, hasDigit := false, false
	for _, ch := range pwd {
		if unicode.IsLetter(ch) {
			hasLetter = true
		}
		if unicode.IsDigit(ch) {
			hasDigit = true
		}
	}
	if !hasLetter || !hasDigit {
		return fmt.Errorf("password must contain both letters and digits")
	}
	return nil
}

func normalizeGender(g string) string {
	switch g {
	case "male", "female":
		return g
	default:
		return "unknown"
	}
}
