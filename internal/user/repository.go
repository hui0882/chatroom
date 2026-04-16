package user

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// pgRepository 是基于 PostgreSQL 的 Repository 实现。
type pgRepository struct {
	db *sql.DB
}

// NewRepository 创建 pgRepository。
func NewRepository(db *sql.DB) Repository {
	return &pgRepository{db: db}
}

// ─── Create ─────────────────────────────────────────────────────────────────

func (r *pgRepository) Create(ctx context.Context, u *User) (*User, error) {
	const q = `
		INSERT INTO users
			(username, password_hash, nickname, avatar_url, bio, gender, age, role, status)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,'active')
		RETURNING id, created_at, updated_at`

	row := r.db.QueryRowContext(ctx, q,
		u.Username, u.PasswordHash, u.Nickname,
		u.AvatarURL, u.Bio, u.Gender, u.Age, u.Role,
	)
	created := *u
	if err := row.Scan(&created.ID, &created.CreatedAt, &created.UpdatedAt); err != nil {
		if strings.Contains(err.Error(), "unique") || strings.Contains(err.Error(), "duplicate") {
			return nil, ErrUsernameExists
		}
		return nil, fmt.Errorf("user create: %w", err)
	}
	return &created, nil
}

// ─── FindByID ────────────────────────────────────────────────────────────────

func (r *pgRepository) FindByID(ctx context.Context, id int64) (*User, error) {
	const q = `SELECT ` + userColumns + ` FROM users WHERE id=$1`
	return r.scanOne(r.db.QueryRowContext(ctx, q, id))
}

// ─── FindByUsername ──────────────────────────────────────────────────────────

func (r *pgRepository) FindByUsername(ctx context.Context, username string) (*User, error) {
	const q = `SELECT ` + userColumns + ` FROM users WHERE username=$1`
	return r.scanOne(r.db.QueryRowContext(ctx, q, username))
}

// ─── Update ──────────────────────────────────────────────────────────────────

func (r *pgRepository) Update(ctx context.Context, u *User) error {
	const q = `
		UPDATE users SET
			nickname=$2, avatar_url=$3, bio=$4, gender=$5, age=$6, updated_at=NOW()
		WHERE id=$1 AND status != 'deleted'`
	_, err := r.db.ExecContext(ctx, q,
		u.ID, u.Nickname, u.AvatarURL, u.Bio, u.Gender, u.Age,
	)
	return err
}

// ─── Ban ─────────────────────────────────────────────────────────────────────

func (r *pgRepository) Ban(ctx context.Context, id int64, reason string, until *time.Time) error {
	const q = `
		UPDATE users SET status='banned', ban_reason=$2, ban_until=$3, updated_at=NOW()
		WHERE id=$1 AND status='active'`
	res, err := r.db.ExecContext(ctx, q, id, reason, until)
	if err != nil {
		return fmt.Errorf("ban user: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// ─── Unban ───────────────────────────────────────────────────────────────────

func (r *pgRepository) Unban(ctx context.Context, id int64) error {
	const q = `
		UPDATE users SET status='active', ban_reason='', ban_until=NULL, updated_at=NOW()
		WHERE id=$1 AND status='banned'`
	_, err := r.db.ExecContext(ctx, q, id)
	return err
}

// ─── SoftDelete ──────────────────────────────────────────────────────────────

func (r *pgRepository) SoftDelete(ctx context.Context, id int64) error {
	const q = `
		UPDATE users SET status='deleted', deleted_at=NOW(), updated_at=NOW()
		WHERE id=$1 AND status != 'deleted'`
	res, err := r.db.ExecContext(ctx, q, id)
	if err != nil {
		return fmt.Errorf("soft delete user: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// ─── Restore ─────────────────────────────────────────────────────────────────

func (r *pgRepository) Restore(ctx context.Context, id int64) error {
	const q = `
		UPDATE users SET status='active', deleted_at=NULL, updated_at=NOW()
		WHERE id=$1 AND status='deleted'`
	_, err := r.db.ExecContext(ctx, q, id)
	return err
}

// ─── UpdatePassword ──────────────────────────────────────────────────────────

func (r *pgRepository) UpdatePassword(ctx context.Context, id int64, hash string) error {
	const q = `UPDATE users SET password_hash=$2, updated_at=NOW() WHERE id=$1`
	_, err := r.db.ExecContext(ctx, q, id, hash)
	return err
}

// ─── List ────────────────────────────────────────────────────────────────────

func (r *pgRepository) List(ctx context.Context, f ListFilter) ([]*User, int64, error) {
	if f.PageSize <= 0 {
		f.PageSize = 20
	}
	if f.Page <= 0 {
		f.Page = 1
	}
	offset := (f.Page - 1) * f.PageSize

	where := []string{}
	args := []interface{}{}
	idx := 1

	if f.Status != "" {
		where = append(where, fmt.Sprintf("status=$%d", idx))
		args = append(args, f.Status)
		idx++
	}
	if f.Keyword != "" {
		where = append(where, fmt.Sprintf("(username ILIKE $%d OR nickname ILIKE $%d)", idx, idx))
		args = append(args, "%"+f.Keyword+"%")
		idx++
	}

	whereClause := ""
	if len(where) > 0 {
		whereClause = "WHERE " + strings.Join(where, " AND ")
	}

	// 总数
	var total int64
	countQ := fmt.Sprintf("SELECT COUNT(*) FROM users %s", whereClause)
	if err := r.db.QueryRowContext(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count users: %w", err)
	}

	// 分页查询
	listArgs := append(args, f.PageSize, offset)
	listQ := fmt.Sprintf(
		`SELECT %s FROM users %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,
		userColumns, whereClause, idx, idx+1,
	)
	rows, err := r.db.QueryContext(ctx, listQ, listArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		u, err := r.scanRow(rows)
		if err != nil {
			return nil, 0, err
		}
		users = append(users, u)
	}
	return users, total, rows.Err()
}

// ─── 内部辅助 ─────────────────────────────────────────────────────────────────

// userColumns 列表与 scanRow 的 Scan 顺序保持一致。
const userColumns = `id, username, password_hash, nickname, avatar_url, bio, gender, age,
	role, status, ban_reason, ban_until, openid, created_at, updated_at, deleted_at`

type rowScanner interface {
	Scan(dest ...interface{}) error
}

func (r *pgRepository) scanOne(row *sql.Row) (*User, error) {
	u, err := r.scanRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return u, nil
}

func (r *pgRepository) scanRow(row rowScanner) (*User, error) {
	var u User
	var banReason sql.NullString
	var banUntil sql.NullTime
	var deletedAt sql.NullTime
	var age sql.NullInt32

	err := row.Scan(
		&u.ID, &u.Username, &u.PasswordHash,
		&u.Nickname, &u.AvatarURL, &u.Bio, &u.Gender, &age,
		&u.Role, &u.Status,
		&banReason, &banUntil,
		&u.OpenID,
		&u.CreatedAt, &u.UpdatedAt, &deletedAt,
	)
	if err != nil {
		return nil, err
	}

	if banReason.Valid {
		u.BanReason = banReason.String
	}
	if banUntil.Valid {
		t := banUntil.Time
		u.BanUntil = &t
	}
	if deletedAt.Valid {
		t := deletedAt.Time
		u.DeletedAt = &t
	}
	if age.Valid {
		v := int(age.Int32)
		u.Age = &v
	}
	return &u, nil
}
