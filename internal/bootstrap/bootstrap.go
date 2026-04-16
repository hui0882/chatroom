package bootstrap

import (
	"database/sql"
	"fmt"

	"github.com/redis/go-redis/v9"

	"github.com/hui0882/chatroom/pkg/config"
	pkgdb "github.com/hui0882/chatroom/pkg/db"
	"github.com/hui0882/chatroom/pkg/logger"
	pkgredis "github.com/hui0882/chatroom/pkg/redis"
)

// App 持有全局共享资源
type App struct {
	Config *config.Config
	DB     *sql.DB
	Redis  *redis.Client
}

// Init 按顺序初始化所有基础设施，任一失败则返回错误
func Init(cfgPath string) (*App, error) {
	// 1. 加载配置
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	// 2. 初始化日志（其他组件依赖日志，必须最先）
	if err = logger.Init(cfg); err != nil {
		return nil, fmt.Errorf("init logger: %w", err)
	}
	log := logger.L()

	// 3. 初始化 PostgreSQL
	log.Info("connecting to PostgreSQL...")
	db, err := pkgdb.Init(&cfg.DB)
	if err != nil {
		return nil, fmt.Errorf("init db: %w", err)
	}
	log.Info("PostgreSQL connected")

	// 4. 初始化 Redis
	log.Info("connecting to Redis...")
	rdb, err := pkgredis.Init(&cfg.Redis)
	if err != nil {
		return nil, fmt.Errorf("init redis: %w", err)
	}
	log.Info("Redis connected")

	return &App{
		Config: cfg,
		DB:     db,
		Redis:  rdb,
	}, nil
}

// Close 释放所有资源（在程序退出时调用）
func (a *App) Close() {
	if a.DB != nil {
		_ = a.DB.Close()
	}
	if a.Redis != nil {
		_ = a.Redis.Close()
	}
	logger.Sync()
}
