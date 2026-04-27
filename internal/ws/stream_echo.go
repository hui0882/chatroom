// Package ws 提供 /websocket_stream 流式测试接口。
// 该接口无需登录，模拟 AI 流式输出：将收到的文本消息按 Unicode 码点倒序，
// 然后以约 20 字/秒的速度逐字符推送回客户端，每帧携带一个字符。
// 客户端收到 {"type":"done"} 帧表示本次输出结束。
// 支持 ping/pong 心跳：服务端每 30s 发一次 ping，35s 内没有 pong 则断开。
package ws

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"github.com/hui0882/chatroom/pkg/logger"
)

// streamRate 每秒输出字符数，20 字/秒 → 每字间隔 50ms
const streamRate = 20

var streamUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 4096,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// streamChunk 是流式输出的单帧结构。
type streamChunk struct {
	Type    string `json:"type"`              // "chunk" | "done" | "error"
	Content string `json:"content,omitempty"` // 当 type=="chunk" 时携带本帧字符
	Error   string `json:"error,omitempty"`   // 当 type=="error" 时携带错误描述
}

// StreamEchoHandler 处理 /websocket_stream 连接。
// 流程：
//  1. 等待客户端发送一条文本消息
//  2. 将该消息按 Unicode 码点倒序
//  3. 以 20 字/秒速度逐字符推送 {"type":"chunk","content":"X"} 帧
//  4. 全部推送完毕后发送 {"type":"done"} 帧
//  5. 回到步骤 1，继续等待下一条消息
func StreamEchoHandler(c *gin.Context) {
	conn, err := streamUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logger.L().Error("ws_stream upgrade failed", zap.Error(err))
		return
	}
	defer conn.Close()

	logger.L().Info("ws_stream client connected",
		zap.String("remote", c.Request.RemoteAddr),
	)

	// 配置读超时与 pong 处理
	conn.SetReadLimit(maxMessageSize)
	_ = conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	// ping 定时器（独立 goroutine，不阻塞读循环）
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()
	go func() {
		for range ticker.C {
			_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}()

	interval := time.Duration(float64(time.Second) / float64(streamRate))

	for {
		msgType, data, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err,
				websocket.CloseGoingAway,
				websocket.CloseAbnormalClosure,
			) {
				logger.L().Warn("ws_stream read error", zap.Error(err))
			}
			break
		}

		// 只处理文本帧；二进制帧原路返回
		if msgType != websocket.TextMessage {
			_ = conn.WriteMessage(msgType, data)
			continue
		}

		// 按 rune 倒序
		runes := []rune(reverseRunes(string(data)))

		// 逐字符流式推送
		for _, r := range runes {
			chunk := streamChunk{Type: "chunk", Content: string(r)}
			frame, _ := json.Marshal(chunk)
			_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := conn.WriteMessage(websocket.TextMessage, frame); err != nil {
				logger.L().Warn("ws_stream write error", zap.Error(err))
				goto cleanup
			}
			time.Sleep(interval)
		}

		// 发送结束帧
		done := streamChunk{Type: "done"}
		doneFrame, _ := json.Marshal(done)
		_ = conn.SetWriteDeadline(time.Now().Add(writeWait))
		if err := conn.WriteMessage(websocket.TextMessage, doneFrame); err != nil {
			logger.L().Warn("ws_stream write done error", zap.Error(err))
			break
		}
	}

cleanup:
	logger.L().Info("ws_stream client disconnected",
		zap.String("remote", c.Request.RemoteAddr),
	)
}
