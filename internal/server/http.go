package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/hui0882/chatroom/pkg/config"
	"github.com/hui0882/chatroom/pkg/logger"
)

// HTTPServer 封装 net/http.Server，支持优雅关闭
type HTTPServer struct {
	srv *http.Server
}

// NewHTTPServer 创建 HTTP Server
func NewHTTPServer(cfg *config.AppConfig, handler http.Handler) *HTTPServer {
	return &HTTPServer{
		srv: &http.Server{
			Addr:         fmt.Sprintf(":%d", cfg.Port),
			Handler:      handler,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
	}
}

// Start 在协程中启动，阻塞直到 ListenAndServe 返回
func (s *HTTPServer) Start() error {
	logger.L().Info("HTTP server starting", zap.String("addr", s.srv.Addr))
	if err := s.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

// Shutdown 优雅关闭，等待已有请求处理完毕（最多 10 秒）
func (s *HTTPServer) Shutdown() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	logger.L().Info("HTTP server shutting down...")
	if err := s.srv.Shutdown(ctx); err != nil {
		logger.L().Error("HTTP server shutdown error", zap.Error(err))
	} else {
		logger.L().Info("HTTP server stopped")
	}
}
