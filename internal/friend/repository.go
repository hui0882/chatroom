package friend

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// Repository 定义好友模块数据操作接口
type Repository interface {
	// 申请相关
	CreateRequest(ctx context.Context, fromUID, toUID int64, message string) (*Request, error)
	FindRequest(ctx context.Context, fromUID, toUID int64) (*Request, error)
	FindRequestByID(ctx context.Context, id int64) (*Request, error)
	UpdateRequestStatus(ctx context.Context, id int64, status string) error
	ListReceivedRequests(ctx context.Context, toUID int64) ([]*Request, error)
	ListSentRequests(ctx context.Context, fromUID int64) ([]*Request, error)

	// 好友关系
	AddFriends(ctx context.Context, uid1, uid2 int64) error
	RemoveFriend(ctx context.Context, uid, friendUID int64) error
	IsFriend(ctx context.Context, uid, friendUID int64) (bool, error)
	ListFriends(ctx context.Context, uid int64) ([]*Friend, error)
}

type pgRepository struct{ db *sql.DB }

func NewRepository(db *sql.DB) Repository { return &pgRepository{db: db} }

// ─── 申请 ─────────────────────────────────────────────────────────────────────

func (r *pgRepository) CreateRequest(ctx context.Context, fromUID, toUID int64, message string) (*Request, error) {
	const q = `
		INSERT INTO friend_requests (from_uid, to_uid, message)
		VALUES ($1, $2, $3)
		ON CONFLICT (from_uid, to_uid) DO UPDATE
			SET status='pending', message=EXCLUDED.message, updated_at=NOW()
		RETURNING id, from_uid, to_uid, message, status, created_at, updated_at`

	var req Request
	err := r.db.QueryRowContext(ctx, q, fromUID, toUID, message).Scan(
		&req.ID, &req.FromUID, &req.ToUID, &req.Message,
		&req.Status, &req.CreatedAt, &req.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	return &req, nil
}

func (r *pgRepository) FindRequest(ctx context.Context, fromUID, toUID int64) (*Request, error) {
	const q = `SELECT id, from_uid, to_uid, message, status, created_at, updated_at
		FROM friend_requests WHERE from_uid=$1 AND to_uid=$2`
	var req Request
	err := r.db.QueryRowContext(ctx, q, fromUID, toUID).Scan(
		&req.ID, &req.FromUID, &req.ToUID, &req.Message,
		&req.Status, &req.CreatedAt, &req.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &req, nil
}

func (r *pgRepository) FindRequestByID(ctx context.Context, id int64) (*Request, error) {
	const q = `SELECT id, from_uid, to_uid, message, status, created_at, updated_at
		FROM friend_requests WHERE id=$1`
	var req Request
	err := r.db.QueryRowContext(ctx, q, id).Scan(
		&req.ID, &req.FromUID, &req.ToUID, &req.Message,
		&req.Status, &req.CreatedAt, &req.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &req, nil
}

func (r *pgRepository) UpdateRequestStatus(ctx context.Context, id int64, status string) error {
	const q = `UPDATE friend_requests SET status=$2, updated_at=NOW() WHERE id=$1`
	_, err := r.db.ExecContext(ctx, q, id, status)
	return err
}

func (r *pgRepository) ListReceivedRequests(ctx context.Context, toUID int64) ([]*Request, error) {
	const q = `
		SELECT fr.id, fr.from_uid, fr.to_uid, fr.message, fr.status, fr.created_at, fr.updated_at,
		       u.username, u.nickname, u.avatar_url
		FROM friend_requests fr
		JOIN users u ON u.id = fr.from_uid
		WHERE fr.to_uid=$1 AND fr.status='pending'
		ORDER BY fr.created_at DESC`

	rows, err := r.db.QueryContext(ctx, q, toUID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRequests(rows, true)
}

func (r *pgRepository) ListSentRequests(ctx context.Context, fromUID int64) ([]*Request, error) {
	const q = `
		SELECT fr.id, fr.from_uid, fr.to_uid, fr.message, fr.status, fr.created_at, fr.updated_at,
		       u.username, u.nickname, u.avatar_url
		FROM friend_requests fr
		JOIN users u ON u.id = fr.to_uid
		WHERE fr.from_uid=$1 AND fr.status='pending'
		ORDER BY fr.created_at DESC`

	rows, err := r.db.QueryContext(ctx, q, fromUID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRequests(rows, false)
}

func scanRequests(rows *sql.Rows, isReceived bool) ([]*Request, error) {
	var list []*Request
	for rows.Next() {
		var req Request
		var username, nickname, avatar string
		if err := rows.Scan(
			&req.ID, &req.FromUID, &req.ToUID, &req.Message,
			&req.Status, &req.CreatedAt, &req.UpdatedAt,
			&username, &nickname, &avatar,
		); err != nil {
			return nil, err
		}
		if isReceived {
			req.FromUsername = username
			req.FromNickname = nickname
			req.FromAvatar = avatar
		} else {
			req.ToUsername = username
			req.ToNickname = nickname
		}
		list = append(list, &req)
	}
	return list, rows.Err()
}

// ─── 好友关系 ─────────────────────────────────────────────────────────────────

func (r *pgRepository) AddFriends(ctx context.Context, uid1, uid2 int64) error {
	const q = `
		INSERT INTO friends (uid, friend_uid) VALUES ($1,$2), ($2,$1)
		ON CONFLICT DO NOTHING`
	_, err := r.db.ExecContext(ctx, q, uid1, uid2)
	return err
}

func (r *pgRepository) RemoveFriend(ctx context.Context, uid, friendUID int64) error {
	const q = `DELETE FROM friends WHERE (uid=$1 AND friend_uid=$2) OR (uid=$2 AND friend_uid=$1)`
	_, err := r.db.ExecContext(ctx, q, uid, friendUID)
	return err
}

func (r *pgRepository) IsFriend(ctx context.Context, uid, friendUID int64) (bool, error) {
	const q = `SELECT 1 FROM friends WHERE uid=$1 AND friend_uid=$2`
	var x int
	err := r.db.QueryRowContext(ctx, q, uid, friendUID).Scan(&x)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return err == nil, err
}

func (r *pgRepository) ListFriends(ctx context.Context, uid int64) ([]*Friend, error) {
	const q = `
		SELECT u.id, u.username, u.nickname, u.avatar_url, f.remark, f.created_at
		FROM friends f
		JOIN users u ON u.id = f.friend_uid
		WHERE f.uid=$1 AND u.status='active'
		ORDER BY f.created_at DESC`

	rows, err := r.db.QueryContext(ctx, q, uid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*Friend
	for rows.Next() {
		var f Friend
		if err := rows.Scan(&f.UID, &f.Username, &f.Nickname, &f.Avatar, &f.Remark, &f.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, &f)
	}
	_ = strings.TrimSpace("") // keep import
	return list, rows.Err()
}
