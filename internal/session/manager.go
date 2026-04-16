// Package session 提供基于 Redis 的 Session 管理。
// Session ID 为 UUID，存入 Redis Hash，有效期由 config.Session.TTL 控制（默认 7 天）。
//
// Redis Key 格式：session:{session_id}
// Hash 字段：
//
//	uid    int64  用户 ID
//	role   string 用户角色（user / admin）
package session

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const keyPrefix = "session:"

// ErrSessionNotFound 表示 Session 不存在或已过期。
var ErrSessionNotFound = errors.New("session not found or expired")

// Info 是存储在 Redis 中的 Session 内容。
type Info struct {
	SessionID string
	UserID    int64
	Role      string
}

// Manager 负责 Session 的创建、读取、删除。
type Manager struct {
	rdb *redis.Client
	ttl time.Duration
}

// NewManager 创建 Session 管理器。ttlSeconds 对应 config.Session.TTL。
func NewManager(rdb *redis.Client, ttlSeconds int) *Manager {
	return &Manager{
		rdb: rdb,
		ttl: time.Duration(ttlSeconds) * time.Second,
	}
}

// Create 为指定用户创建一个新 Session，返回 Session ID。
func (m *Manager) Create(ctx context.Context, uid int64, role string) (string, error) {
	sid := uuid.NewString()
	key := keyPrefix + sid
	pipe := m.rdb.Pipeline()
	pipe.HSet(ctx, key,
		"uid", uid,
		"role", role,
	)
	pipe.Expire(ctx, key, m.ttl)
	if _, err := pipe.Exec(ctx); err != nil {
		return "", fmt.Errorf("session create: %w", err)
	}
	return sid, nil
}

// Get 读取 Session；不存在或过期时返回 ErrSessionNotFound。
func (m *Manager) Get(ctx context.Context, sid string) (*Info, error) {
	key := keyPrefix + sid
	vals, err := m.rdb.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("session get: %w", err)
	}
	if len(vals) == 0 {
		return nil, ErrSessionNotFound
	}
	uid, err := strconv.ParseInt(vals["uid"], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("session parse uid: %w", err)
	}
	return &Info{
		SessionID: sid,
		UserID:    uid,
		Role:      vals["role"],
	}, nil
}

// Delete 删除 Session（退出登录）。
func (m *Manager) Delete(ctx context.Context, sid string) error {
	return m.rdb.Del(ctx, keyPrefix+sid).Err()
}

// DeleteAll 删除某个用户的所有 Session（用于封禁/强制下线）。
// 实现方案：扫描 session:* 并比对 uid 字段。
// 注意：在连接数极多的场景下应改用 SCAN 游标；当前实现适合中小规模。
func (m *Manager) DeleteAll(ctx context.Context, uid int64) error {
	var cursor uint64
	uidStr := strconv.FormatInt(uid, 10)
	for {
		keys, next, err := m.rdb.Scan(ctx, cursor, keyPrefix+"*", 100).Result()
		if err != nil {
			return fmt.Errorf("session scan: %w", err)
		}
		for _, key := range keys {
			v, err := m.rdb.HGet(ctx, key, "uid").Result()
			if err != nil {
				continue
			}
			if v == uidStr {
				_ = m.rdb.Del(ctx, key)
			}
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}
	return nil
}
