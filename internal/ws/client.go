// Package ws 实现 WebSocket 连接管理。
// Client 代表一个已认证的 WebSocket 连接，持有写队列和元数据。
package ws

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"github.com/hui0882/chatroom/pkg/logger"
)

const (
	// 写超时
	writeWait = 10 * time.Second
	// 读超时：客户端必须在此时间内发送 pong，否则断开连接
	// 设为 35s，给 30s ping 间隔留 5s 容差
	pongWait = 35 * time.Second
	// ping 发送间隔，必须 < pongWait
	pingPeriod = 30 * time.Second
	// 最大单帧消息字节数：64 KB
	maxMessageSize = 64 * 1024
)

// Client 表示一个 WebSocket 连接。
type Client struct {
	Hub    *Hub
	Conn   *websocket.Conn
	Send   chan []byte // 待发往客户端的消息队列
	UserID int64
	Device string // pc / mobile / web 等，可扩展多设备
	mu     sync.Mutex
	closed bool
}

// NewClient 创建 Client 并注册到 Hub。
func NewClient(hub *Hub, conn *websocket.Conn, userID int64, device string) *Client {
	return &Client{
		Hub:    hub,
		Conn:   conn,
		Send:   make(chan []byte, 256),
		UserID: userID,
		Device: device,
	}
}

// ReadPump 持续从 WebSocket 读取帧并交给 Hub 处理，退出时从 Hub 注销。
// 必须在独立 goroutine 中运行。
func (c *Client) ReadPump() {
	defer func() {
		c.Hub.Unregister <- c
		c.safeClose()
	}()

	c.Conn.SetReadLimit(maxMessageSize)
	_ = c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		return c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		_, msg, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err,
				websocket.CloseGoingAway,
				websocket.CloseAbnormalClosure,
			) {
				logger.L().Warn("ws read error",
					zap.Int64("uid", c.UserID),
					zap.Error(err),
				)
			}
			break
		}
		c.Hub.Inbound <- &Message{Client: c, Data: msg}
	}
}

// WritePump 持续将 Send 队列中的帧写入 WebSocket，同时发送心跳 ping。
// 必须在独立 goroutine 中运行。
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.safeClose()
	}()

	for {
		select {
		case msg, ok := <-c.Send:
			_ = c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Hub 关闭了此 channel
				_ = c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.Conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				logger.L().Warn("ws write error",
					zap.Int64("uid", c.UserID),
					zap.Error(err),
				)
				return
			}

		case <-ticker.C:
			_ = c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// safeClose 保证 Conn 只被关闭一次。
func (c *Client) safeClose() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.closed {
		c.closed = true
		_ = c.Conn.Close()
	}
}
