package ws

import (
	"sync"

	"go.uber.org/zap"

	"github.com/hui0882/chatroom/pkg/logger"
)

// Message 封装了一帧入站消息：来源 Client + 原始字节。
type Message struct {
	Client *Client
	Data   []byte
}

// MessageHandler 是消息处理函数，由上层业务注册进来。
// Hub 对每条入站消息调用它，业务逻辑可在此处解析 cmd 字段并分发。
type MessageHandler func(c *Client, data []byte)

// Hub 集中管理所有 Client 的注册/注销及消息广播。
// 单进程内所有 WebSocket 连接都通过同一个 Hub。
type Hub struct {
	// 所有在线连接：uid → map[device]*Client（允许多设备）
	clients map[int64]map[string]*Client
	mu      sync.RWMutex

	Register   chan *Client
	Unregister chan *Client
	Inbound    chan *Message

	handler MessageHandler
}

// NewHub 创建并返回一个新的 Hub。
func NewHub(handler MessageHandler) *Hub {
	return &Hub{
		clients:    make(map[int64]map[string]*Client),
		Register:   make(chan *Client, 64),
		Unregister: make(chan *Client, 64),
		Inbound:    make(chan *Message, 512),
		handler:    handler,
	}
}

// Run 开始事件循环，应在独立 goroutine 中调用（程序生命周期内持续运行）。
func (h *Hub) Run() {
	for {
		select {
		case c := <-h.Register:
			h.mu.Lock()
			if h.clients[c.UserID] == nil {
				h.clients[c.UserID] = make(map[string]*Client)
			}
			// 如果同设备类型已有旧连接，先踢掉
			if old, ok := h.clients[c.UserID][c.Device]; ok {
				close(old.Send)
			}
			h.clients[c.UserID][c.Device] = c
			h.mu.Unlock()
			logger.L().Info("ws client registered",
				zap.Int64("uid", c.UserID),
				zap.String("device", c.Device),
			)

		case c := <-h.Unregister:
			h.mu.Lock()
			if devices, ok := h.clients[c.UserID]; ok {
				if cur, ok := devices[c.Device]; ok && cur == c {
					delete(devices, c.Device)
					close(c.Send)
				}
				if len(devices) == 0 {
					delete(h.clients, c.UserID)
				}
			}
			h.mu.Unlock()
			logger.L().Info("ws client unregistered",
				zap.Int64("uid", c.UserID),
				zap.String("device", c.Device),
			)

		case msg := <-h.Inbound:
			if h.handler != nil {
				h.handler(msg.Client, msg.Data)
			}
		}
	}
}

// SendToUser 将消息推送给指定用户的所有设备。
func (h *Hub) SendToUser(uid int64, data []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, c := range h.clients[uid] {
		select {
		case c.Send <- data:
		default:
			logger.L().Warn("ws send queue full, dropping message",
				zap.Int64("uid", uid),
				zap.String("device", c.Device),
			)
		}
	}
}

// KickUser 强制断开指定用户的所有 WebSocket 连接（管理员踢出）。
func (h *Hub) KickUser(uid int64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if devices, ok := h.clients[uid]; ok {
		for _, c := range devices {
			close(c.Send)
		}
		delete(h.clients, uid)
	}
}

// OnlineCount 返回当前在线连接数（设备维度）。
func (h *Hub) OnlineCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	total := 0
	for _, devices := range h.clients {
		total += len(devices)
	}
	return total
}

// IsOnline 判断某用户是否有活跃连接。
func (h *Hub) IsOnline(uid int64) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	_, ok := h.clients[uid]
	return ok
}
