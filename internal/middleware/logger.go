package middleware

import (
	"bytes"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/hui0882/chatroom/pkg/logger"
	"github.com/hui0882/chatroom/pkg/response"
)

// RequestLogger 记录每条 HTTP 请求的基本信息。
// debug=true 时额外记录请求 Body（最多读取 4KB，避免超大请求撑爆内存）。
func RequestLogger(debug bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		log := logger.L()

		// 调试模式：读取并记录请求 Body，再还原供后续 handler 使用
		var bodySnippet string
		if debug && c.Request.Body != nil {
			const maxBodyLog = 4 * 1024 // 4 KB
			raw, err := io.ReadAll(io.LimitReader(c.Request.Body, maxBodyLog))
			if err == nil {
				bodySnippet = string(raw)
				c.Request.Body = io.NopCloser(bytes.NewReader(raw))
			}
		}

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		fields := []zap.Field{
			zap.String("method", c.Request.Method),
			zap.String("path", c.FullPath()),
			zap.String("raw_query", c.Request.URL.RawQuery),
			zap.Int("status", status),
			zap.Duration("latency", latency),
			zap.String("client_ip", c.ClientIP()),
		}

		if debug {
			fields = append(fields,
				zap.String("user_agent", c.Request.UserAgent()),
				zap.String("request_body", bodySnippet),
			)
		}

		switch {
		case status >= http.StatusInternalServerError:
			log.Error("request", fields...)
		case status >= http.StatusBadRequest:
			log.Warn("request", fields...)
		default:
			log.Info("request", fields...)
		}
	}
}

// Recovery 捕获 panic，记录错误日志并返回 500
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				logger.L().Error("panic recovered",
					zap.Any("error", err),
					zap.String("path", c.Request.URL.Path),
				)
				response.FailWithStatus(c, http.StatusInternalServerError,
					response.CodeInternalError, "internal server error")
				c.Abort()
			}
		}()
		c.Next()
	}
}
