package message

import (
	"context"
	"fmt"
	"strconv"

	"github.com/redis/go-redis/v9"
)

// redisUnreadStore 基于 Redis Hash 的未读计数实现。
//
// 数据结构：
//
//	key:   unread:{toUID}
//	field: {fromUID}
//	value: count (string，由 HINCRBY 维护)
//
// 优点：单次 HGETALL 拿到全部对话未读数，O(n) where n=好友数。
type redisUnreadStore struct {
	rdb *redis.Client
}

// NewUnreadStore 返回 Redis 实现的 UnreadStore。
func NewUnreadStore(rdb *redis.Client) UnreadStore {
	return &redisUnreadStore{rdb: rdb}
}

func unreadKey(toUID int64) string {
	return fmt.Sprintf("unread:%d", toUID)
}

func (s *redisUnreadStore) Incr(ctx context.Context, toUID, fromUID int64) error {
	field := strconv.FormatInt(fromUID, 10)
	return s.rdb.HIncrBy(ctx, unreadKey(toUID), field, 1).Err()
}

func (s *redisUnreadStore) Clear(ctx context.Context, toUID, fromUID int64) error {
	field := strconv.FormatInt(fromUID, 10)
	return s.rdb.HSet(ctx, unreadKey(toUID), field, 0).Err()
}

func (s *redisUnreadStore) GetAll(ctx context.Context, uid int64) (map[int64]int64, error) {
	raw, err := s.rdb.HGetAll(ctx, unreadKey(uid)).Result()
	if err != nil {
		return nil, fmt.Errorf("unread.GetAll: %w", err)
	}
	result := make(map[int64]int64, len(raw))
	for k, v := range raw {
		fromUID, _ := strconv.ParseInt(k, 10, 64)
		count, _ := strconv.ParseInt(v, 10, 64)
		result[fromUID] = count
	}
	return result, nil
}
