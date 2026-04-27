// Package server 定义应用级上下文和各模块 handler 的聚合入口。
// AppContext 持有所有基础设施依赖（DB、Redis），各 handler 从这里取用。
package server

import (
	"database/sql"

	"github.com/redis/go-redis/v9"

	"github.com/hui0882/chatroom/internal/friend"
	"github.com/hui0882/chatroom/internal/message"
	"github.com/hui0882/chatroom/internal/session"
	"github.com/hui0882/chatroom/internal/user"
	"github.com/hui0882/chatroom/internal/ws"
	"github.com/hui0882/chatroom/pkg/config"
)

// AppContext 聚合所有模块的 handler，路由注册时通过它拿到具体 handler。
type AppContext struct {
	Health  *HealthHandler
	User    *user.Handler
	Friend  *friend.Handler
	Message *message.Handler
	WS      *ws.Handler
	Hub     *ws.Hub
	Session *session.Manager
}

// NewAppContext 初始化 AppContext，注入共享依赖
func NewAppContext(cfg *config.Config, db *sql.DB, rdb *redis.Client) *AppContext {
	// Session 管理
	sm := session.NewManager(rdb, cfg.Session.TTL)

	// WebSocket Hub（先用 nil handler，后面 dispatcher wiring 后 SetHandler）
	hub := ws.NewHub(nil)
	go hub.Run()

	// 用户模块
	userRepo := user.NewRepository(db)
	userSvc := user.NewService(userRepo, sm, hub)
	userHandler := user.NewHandler(userSvc)

	// 好友模块
	friendRepo := friend.NewRepository(db)
	friendSvc := friend.NewService(friendRepo, userRepo)
	friendHandler := friend.NewHandler(friendSvc)

	// 消息模块
	pgMsgRepo := message.NewRepository(db)
	cacheMsgRepo := message.NewCacheRepository(pgMsgRepo, rdb)
	unreadStore := message.NewUnreadStore(rdb)
	msgSvc := message.NewService(cacheMsgRepo, unreadStore, friendSvc)
	msgHandler := message.NewHandler(msgSvc)

	// WS Dispatcher（消息路由核心）
	dispatcher := message.NewDispatcher(hub, msgSvc, friendRepo)
	hub.SetHandler(dispatcher.Handle)
	hub.SetConnectHooks(dispatcher.OnConnect, dispatcher.OnDisconnect)

	// WebSocket handler，使用 session manager 验证
	wsValidator := ws.BuildSessionValidator(sm)
	wsHandler := ws.NewHandler(hub, wsValidator)

	return &AppContext{
		Health:  NewHealthHandler(cfg, db, rdb),
		User:    userHandler,
		Friend:  friendHandler,
		Message: msgHandler,
		WS:      wsHandler,
		Hub:     hub,
		Session: sm,
	}
}
