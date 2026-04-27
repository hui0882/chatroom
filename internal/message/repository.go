package message

import (
	"context"
	"database/sql"
	"fmt"
)

// pgRepository 基于 PostgreSQL 的消息存储实现。
type pgRepository struct {
	db *sql.DB
}

// NewRepository 返回 PostgreSQL 实现。
func NewRepository(db *sql.DB) Repository {
	return &pgRepository{db: db}
}

// Save 插入一条消息并返回包含 ID 和 CreatedAt 的完整记录。
func (r *pgRepository) Save(ctx context.Context, msg *Message) (*Message, error) {
	const q = `
		INSERT INTO messages (from_uid, to_uid, content, msg_type)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at
	`
	row := r.db.QueryRowContext(ctx, q,
		msg.FromUID, msg.ToUID, msg.Content, string(msg.MsgType),
	)
	saved := *msg
	if err := row.Scan(&saved.ID, &saved.CreatedAt); err != nil {
		return nil, fmt.Errorf("message.Save: %w", err)
	}
	return &saved, nil
}

// ListHistory 游标翻页查询两人会话历史，倒序返回（最新的在前）。
// cursor=0 时从最新消息开始；返回 id < cursor 的记录。
func (r *pgRepository) ListHistory(ctx context.Context, uid1, uid2 int64, cursor int64, limit int) ([]*Message, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	// 利用复合索引：LEAST/GREATEST 保证两人顺序无关
	var q string
	var rows *sql.Rows
	var err error

	if cursor == 0 {
		q = `
			SELECT id, from_uid, to_uid, content, msg_type, created_at
			FROM messages
			WHERE LEAST(from_uid, to_uid) = $1
			  AND GREATEST(from_uid, to_uid) = $2
			  AND deleted_at IS NULL
			ORDER BY id DESC
			LIMIT $3
		`
		rows, err = r.db.QueryContext(ctx, q,
			min64(uid1, uid2), max64(uid1, uid2), limit,
		)
	} else {
		q = `
			SELECT id, from_uid, to_uid, content, msg_type, created_at
			FROM messages
			WHERE LEAST(from_uid, to_uid) = $1
			  AND GREATEST(from_uid, to_uid) = $2
			  AND id < $3
			  AND deleted_at IS NULL
			ORDER BY id DESC
			LIMIT $4
		`
		rows, err = r.db.QueryContext(ctx, q,
			min64(uid1, uid2), max64(uid1, uid2), cursor, limit,
		)
	}
	if err != nil {
		return nil, fmt.Errorf("message.ListHistory: %w", err)
	}
	defer rows.Close()

	var msgs []*Message
	for rows.Next() {
		m := &Message{}
		var msgType string
		if err := rows.Scan(&m.ID, &m.FromUID, &m.ToUID, &m.Content, &msgType, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("message.ListHistory scan: %w", err)
		}
		m.MsgType = MsgType(msgType)
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}

func min64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func max64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
