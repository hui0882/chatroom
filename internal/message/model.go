package message

import (
	"context"
	"errors"
	"time"
)

// ─── 错误定义 ─────────────────────────────────────────────────────────────────

var (
	ErrNotFriend    = errors.New("not friends")
	ErrInvalidInput = errors.New("invalid input")
)

// ─── 数据模型 ─────────────────────────────────────────────────────────────────

// MsgType 消息类型
type MsgType string

const (
	MsgTypeText MsgType = "text"
)

// Message 一条私聊消息
type Message struct {
	ID        int64     `json:"id"`
	FromUID   int64     `json:"from_uid"`
	ToUID     int64     `json:"to_uid"`
	Content   string    `json:"content"`
	MsgType   MsgType   `json:"msg_type"`
	CreatedAt time.Time `json:"created_at"`
}

// ─── Repository 接口（存储层抽象，方便替换实现）────────────────────────────────

// Repository 定义消息存储操作。
// 具体实现：pgRepository（PostgreSQL）、cacheRepository（Redis 包装层）
type Repository interface {
	// Save 持久化一条消息，返回带 ID 和 CreatedAt 的完整消息
	Save(ctx context.Context, msg *Message) (*Message, error)

	// ListHistory 分页查询两人对话历史（游标翻页，倒序）
	// cursor=0 表示从最新消息开始；返回的消息按 id DESC 排列
	ListHistory(ctx context.Context, uid1, uid2 int64, cursor int64, limit int) ([]*Message, error)
}

// ─── UnreadStore 接口（未读计数抽象）─────────────────────────────────────────

// UnreadStore 管理未读消息计数，当前实现基于 Redis Hash。
type UnreadStore interface {
	// Incr 给 toUID 的来自 fromUID 的未读数 +1
	Incr(ctx context.Context, toUID, fromUID int64) error

	// Clear 清零 toUID 来自 fromUID 的未读数（打开会话时调用）
	Clear(ctx context.Context, toUID, fromUID int64) error

	// GetAll 获取 uid 所有对话的未读数 map[fromUID]count
	GetAll(ctx context.Context, uid int64) (map[int64]int64, error)
}
