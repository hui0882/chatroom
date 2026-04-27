package message

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	// 每个对话在 Redis 里最多缓存多少条消息
	cacheMaxLen int64 = 100
	// 缓存 key 的 TTL
	cacheTTL = 7 * 24 * time.Hour
)

// cacheKey 返回两人会话的 Redis ZSet key（uid 小的在前保证唯一性）
func cacheKey(uid1, uid2 int64) string {
	a, b := min64(uid1, uid2), max64(uid1, uid2)
	return fmt.Sprintf("msg:conv:%d:%d", a, b)
}

// CacheRepository 在 pgRepository 外包一层 Redis ZSet 缓存。
//
// ZSet 结构：
//
//	key:    msg:conv:{min_uid}:{max_uid}
//	score:  msg.ID（保证有序且可做游标）
//	member: "{id}|{from_uid}|{to_uid}|{content}|{msg_type}|{unix_sec}"
//	        （content 中的 "|" 转义为 "\p"，"\" 转义为 "\\"）
type CacheRepository struct {
	pg  Repository
	rdb *redis.Client
}

// NewCacheRepository 包装 pgRepository，加 Redis 缓存。
func NewCacheRepository(pg Repository, rdb *redis.Client) Repository {
	return &CacheRepository{pg: pg, rdb: rdb}
}

// Save 先写 DB，成功后异步写入 Redis 缓存（写失败不影响主流程）。
func (c *CacheRepository) Save(ctx context.Context, msg *Message) (*Message, error) {
	saved, err := c.pg.Save(ctx, msg)
	if err != nil {
		return nil, err
	}
	// 异步写 Redis，不阻塞响应
	go c.addToCache(saved)
	return saved, nil
}

func (c *CacheRepository) addToCache(msg *Message) {
	ctx := context.Background()
	key := cacheKey(msg.FromUID, msg.ToUID)
	member := encodeMsg(msg)
	z := redis.Z{Score: float64(msg.ID), Member: member}

	pipe := c.rdb.Pipeline()
	pipe.ZAdd(ctx, key, z)
	// 只保留最新的 cacheMaxLen 条（删掉 score 最小的旧记录）
	pipe.ZRemRangeByRank(ctx, key, 0, -(cacheMaxLen + 1))
	pipe.Expire(ctx, key, cacheTTL)
	_, _ = pipe.Exec(ctx)
}

// ListHistory 先查 Redis ZSet，命中则直接返回；否则 fallback 到 DB 并预热缓存。
func (c *CacheRepository) ListHistory(ctx context.Context, uid1, uid2 int64, cursor int64, limit int) ([]*Message, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	// 尝试从 Redis 读
	msgs, err := c.fromCache(ctx, uid1, uid2, cursor, limit)
	if err == nil && len(msgs) > 0 {
		return msgs, nil
	}

	// fallback: 从 DB 读
	msgs, err = c.pg.ListHistory(ctx, uid1, uid2, cursor, limit)
	if err != nil {
		return nil, err
	}
	// 预热：首页时把这批消息写入 Redis
	if cursor == 0 {
		go c.warmCache(uid1, uid2, msgs)
	}
	return msgs, nil
}

func (c *CacheRepository) fromCache(ctx context.Context, uid1, uid2, cursor int64, limit int) ([]*Message, error) {
	key := cacheKey(uid1, uid2)

	var maxScore string
	if cursor == 0 {
		maxScore = "+inf"
	} else {
		maxScore = "(" + strconv.FormatInt(cursor, 10) // exclusive: 不含 cursor 本身
	}
	const minScore = "-inf"

	// ZRevRangeByScore: 从高 score（最新）到低 score（最旧）取 limit 条
	opt := &redis.ZRangeBy{
		Min:    minScore,
		Max:    maxScore,
		Offset: 0,
		Count:  int64(limit),
	}
	members, err := c.rdb.ZRevRangeByScoreWithScores(ctx, key, opt).Result()
	if err != nil || len(members) == 0 {
		return nil, fmt.Errorf("cache miss")
	}

	msgs := make([]*Message, 0, len(members))
	for _, z := range members {
		member, ok := z.Member.(string)
		if !ok {
			return nil, fmt.Errorf("invalid cache member type")
		}
		m, err := decodeMsg(member)
		if err != nil {
			return nil, err
		}
		msgs = append(msgs, m)
	}
	return msgs, nil
}

func (c *CacheRepository) warmCache(uid1, uid2 int64, msgs []*Message) {
	if len(msgs) == 0 {
		return
	}
	ctx := context.Background()
	key := cacheKey(uid1, uid2)
	zs := make([]redis.Z, 0, len(msgs))
	for _, m := range msgs {
		zs = append(zs, redis.Z{Score: float64(m.ID), Member: encodeMsg(m)})
	}
	pipe := c.rdb.Pipeline()
	pipe.ZAdd(ctx, key, zs...)
	pipe.ZRemRangeByRank(ctx, key, 0, -(cacheMaxLen + 1))
	pipe.Expire(ctx, key, cacheTTL)
	_, _ = pipe.Exec(ctx)
}

// ─── 编解码 ───────────────────────────────────────────────────────────────────

// encodeMsg 将消息序列化为 "|" 分隔的字符串存入 ZSet member。
// 格式：{id}|{from_uid}|{to_uid}|{content}|{msg_type}|{unix_nano}
func encodeMsg(m *Message) string {
	return fmt.Sprintf("%d|%d|%d|%s|%s|%d",
		m.ID, m.FromUID, m.ToUID,
		escapePipe(m.Content),
		string(m.MsgType),
		m.CreatedAt.UnixNano(),
	)
}

func decodeMsg(s string) (*Message, error) {
	// 最多分 6 段（content 已转义，不含 "|"）
	parts := splitN(s, '|', 6)
	if len(parts) != 6 {
		return nil, fmt.Errorf("decode: invalid format, got %d parts", len(parts))
	}
	id, _ := strconv.ParseInt(parts[0], 10, 64)
	from, _ := strconv.ParseInt(parts[1], 10, 64)
	to, _ := strconv.ParseInt(parts[2], 10, 64)
	content := unescapePipe(parts[3])
	msgType := MsgType(parts[4])
	nano, _ := strconv.ParseInt(parts[5], 10, 64)

	return &Message{
		ID:        id,
		FromUID:   from,
		ToUID:     to,
		Content:   content,
		MsgType:   msgType,
		CreatedAt: time.Unix(0, nano).UTC(),
	}, nil
}

// escapePipe 转义 content 中的 "\" 和 "|" 以防破坏编码格式
func escapePipe(s string) string {
	result := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '\\':
			result = append(result, '\\', '\\')
		case '|':
			result = append(result, '\\', 'p')
		default:
			result = append(result, s[i])
		}
	}
	return string(result)
}

func unescapePipe(s string) string {
	result := make([]byte, 0, len(s))
	i := 0
	for i < len(s) {
		if s[i] == '\\' && i+1 < len(s) {
			switch s[i+1] {
			case 'p':
				result = append(result, '|')
				i += 2
				continue
			case '\\':
				result = append(result, '\\')
				i += 2
				continue
			}
		}
		result = append(result, s[i])
		i++
	}
	return string(result)
}

// splitN 在字节 sep 上最多分割为 n 份（最后一份不再拆分）
func splitN(s string, sep byte, n int) []string {
	parts := make([]string, 0, n)
	start := 0
	count := 0
	for i := 0; i < len(s) && count < n-1; i++ {
		if s[i] == sep {
			parts = append(parts, s[start:i])
			start = i + 1
			count++
		}
	}
	parts = append(parts, s[start:])
	return parts
}
