// Package user 实现用户注册、登录、信息管理及管理员操作等功能。
package user

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

// ─── 领域模型 ───────────────────────────────────────────────────────────────

// User 对应 users 表
type User struct {
	ID           int64          `json:"id"`
	Username     string         `json:"username"`
	PasswordHash string         `json:"-"`
	Nickname     string         `json:"nickname"`
	AvatarURL    string         `json:"avatar_url"`
	Bio          string         `json:"bio"`
	Gender       string         `json:"gender"` // unknown / male / female
	Age          *int           `json:"age,omitempty"`
	Role         string         `json:"role"`   // user / admin
	Status       string         `json:"status"` // active / banned / deleted
	BanReason    string         `json:"ban_reason,omitempty"`
	BanUntil     *time.Time     `json:"ban_until,omitempty"`
	OpenID       sql.NullString `json:"-"` // 微信 openid，预留
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    *time.Time     `json:"deleted_at,omitempty"`
}

// IsBanned 检查用户当前是否处于封禁状态。
func (u *User) IsBanned() bool {
	if u.Status != "banned" {
		return false
	}
	// 有截止时间且已过期 → 视为未封禁（业务层应定时清理，这里做兜底）
	if u.BanUntil != nil && time.Now().After(*u.BanUntil) {
		return false
	}
	return true
}

// IsDeleted 判断是否逻辑删除。
func (u *User) IsDeleted() bool {
	return u.Status == "deleted"
}

// ─── Repository 接口 ────────────────────────────────────────────────────────

// Repository 定义用户数据访问接口，方便后续替换实现或 Mock。
type Repository interface {
	// 创建用户，返回插入后的完整 User（含 ID）
	Create(ctx context.Context, u *User) (*User, error)
	// 按 ID 查询
	FindByID(ctx context.Context, id int64) (*User, error)
	// 按用户名查询
	FindByUsername(ctx context.Context, username string) (*User, error)
	// 更新用户基础信息
	Update(ctx context.Context, u *User) error
	// 封禁用户
	Ban(ctx context.Context, id int64, reason string, until *time.Time) error
	// 解封用户
	Unban(ctx context.Context, id int64) error
	// 逻辑删除用户
	SoftDelete(ctx context.Context, id int64) error
	// 恢复逻辑删除（admin 专用）
	Restore(ctx context.Context, id int64) error
	// 更新密码 hash
	UpdatePassword(ctx context.Context, id int64, hash string) error
	// 分页查询用户列表（admin）
	List(ctx context.Context, filter ListFilter) ([]*User, int64, error)
}

// ListFilter 查询列表的过滤/分页参数
type ListFilter struct {
	Status   string // "" = 不过滤
	Keyword  string // 按用户名/昵称模糊搜索
	Page     int    // 从 1 开始
	PageSize int    // 每页数量，默认 20
}

// ─── 常见错误 ───────────────────────────────────────────────────────────────

var (
	ErrNotFound       = errors.New("user not found")
	ErrUsernameExists = errors.New("username already exists")
)
