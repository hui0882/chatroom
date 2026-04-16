package server

import (
	"context"
	"database/sql"
	"net/http"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"github.com/hui0882/chatroom/pkg/config"
	"github.com/hui0882/chatroom/pkg/response"
)

// HealthHandler 处理 /health 接口
type HealthHandler struct {
	cfg *config.Config
	db  *sql.DB
	rdb *redis.Client
}

func NewHealthHandler(cfg *config.Config, db *sql.DB, rdb *redis.Client) *HealthHandler {
	return &HealthHandler{cfg: cfg, db: db, rdb: rdb}
}

// healthData 响应体
type healthData struct {
	Status    string            `json:"status"`
	Node      string            `json:"node"`
	Timestamp string            `json:"timestamp"`
	Uptime    string            `json:"uptime"`
	GoVersion string            `json:"go_version"`
	Services  map[string]string `json:"services"`
}

var startTime = time.Now()

// Check godoc
//
//	@Summary     健康检查
//	@Description 检查服务本身及 PostgreSQL、Redis 的连通性
//	@Tags        system
//	@Produce     json
//	@Success     200 {object} response.R{data=healthData}
//	@Router      /health [get]
func (h *HealthHandler) Check(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	services := make(map[string]string, 2)

	// 检查 PostgreSQL
	if err := h.db.PingContext(ctx); err != nil {
		services["postgresql"] = "unhealthy: " + err.Error()
	} else {
		services["postgresql"] = "healthy"
	}

	// 检查 Redis
	if err := h.rdb.Ping(ctx).Err(); err != nil {
		services["redis"] = "unhealthy: " + err.Error()
	} else {
		services["redis"] = "healthy"
	}

	// 判断整体状态
	overall := "ok"
	httpStatus := http.StatusOK
	for _, v := range services {
		if v != "healthy" {
			overall = "degraded"
			httpStatus = http.StatusServiceUnavailable
			break
		}
	}

	data := healthData{
		Status:    overall,
		Node:      h.cfg.App.NodeID,
		Timestamp: time.Now().Format(time.RFC3339),
		Uptime:    time.Since(startTime).Round(time.Second).String(),
		GoVersion: runtime.Version(),
		Services:  services,
	}

	c.JSON(httpStatus, response.R{Code: response.CodeOK, Msg: "ok", Data: data})
}
