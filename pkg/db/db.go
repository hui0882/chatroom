package db

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver

	"github.com/hui0882/chatroom/pkg/config"
)

// Init 初始化 PostgreSQL 连接池并返回 *sql.DB
func Init(cfg *config.DBConfig) (*sql.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode,
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %w", err)
	}

	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(30 * time.Minute)
	db.SetConnMaxIdleTime(10 * time.Minute)

	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("db ping: %w", err)
	}

	return db, nil
}
