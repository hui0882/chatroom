package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"

	"github.com/hui0882/chatroom/internal/bootstrap"
	"github.com/hui0882/chatroom/internal/router"
	"github.com/hui0882/chatroom/internal/server"
	"github.com/hui0882/chatroom/pkg/logger"
)

func main() {
	// ── 命令行参数 ──────────────────────────────────────────────
	cfgPath := flag.String("config", "config.json", "path to config.json")
	flag.Parse()

	// ── 初始化基础设施 ──────────────────────────────────────────
	app, err := bootstrap.Init(*cfgPath)
	if err != nil {
		log.Fatalf("bootstrap failed: %v", err)
	}
	defer app.Close()

	l := logger.L()
	l.Info("chatroom starting",
		zap.String("node", app.Config.App.NodeID),
		zap.Bool("debug", app.Config.App.Debug),
	)

	// ── 构建路由 ────────────────────────────────────────────────
	appCtx := server.NewAppContext(app.Config, app.DB, app.Redis)
	engine := router.Setup(app.Config, appCtx)

	// ── 启动 HTTP Server ────────────────────────────────────────
	httpSrv := server.NewHTTPServer(&app.Config.App, engine)

	// 在独立 goroutine 中监听
	go func() {
		if err := httpSrv.Start(); err != nil {
			l.Fatal("HTTP server error", zap.Error(err))
		}
	}()

	// ── 等待退出信号 ────────────────────────────────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	l.Info("received shutdown signal", zap.String("signal", sig.String()))

	// ── 优雅关闭 ────────────────────────────────────────────────
	httpSrv.Shutdown()
	l.Info("chatroom stopped")
}
