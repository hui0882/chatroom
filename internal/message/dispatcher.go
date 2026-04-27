package message

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"go.uber.org/zap"

	"github.com/hui0882/chatroom/internal/friend"
	"github.com/hui0882/chatroom/internal/ws"
	"github.com/hui0882/chatroom/pkg/logger"
)

// ─── WS 协议帧定义 ────────────────────────────────────────────────────────────

// InboundFrame 客户端发来的 WS 帧
// {"cmd":"chat","data":{...}}
type InboundFrame struct {
	Cmd  string          `json:"cmd"`
	Data json.RawMessage `json:"data"`
}

// ChatPayload cmd=chat 时的 data 结构
type ChatPayload struct {
	ToUID   int64  `json:"to_uid"`
	Content string `json:"content"`
}

// OutboundFrame 服务端推送给客户端的帧
type OutboundFrame struct {
	Cmd  string `json:"cmd"`
	Data any    `json:"data"`
}

// ─── 具体推送帧 ───────────────────────────────────────────────────────────────

// ChatPush cmd=chat 推送，接收方收到新消息
type ChatPush struct {
	ID        int64     `json:"id"`
	FromUID   int64     `json:"from_uid"`
	ToUID     int64     `json:"to_uid"`
	Content   string    `json:"content"`
	MsgType   MsgType   `json:"msg_type"`
	CreatedAt time.Time `json:"created_at"`
}

// ChatAck cmd=chat_ack 推送，发送方收到服务器确认（含服务端分配的 msg ID）
type ChatAck struct {
	ID        int64     `json:"id"`
	ToUID     int64     `json:"to_uid"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

// ErrorPush cmd=error 推送，告知客户端操作失败
type ErrorPush struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

// OnlinePush cmd=online / cmd=offline 推送，通知好友上下线
type OnlinePush struct {
	UID int64 `json:"uid"`
}

// UnreadPush cmd=unread_init，连接后推送全量未读数
// map[string(peer_uid)]count  （JSON key 必须是 string）
type UnreadPush map[string]int64

// ─── Dispatcher ───────────────────────────────────────────────────────────────

// Dispatcher 是 ws.MessageHandler 的具体实现，负责解析 WS 帧并调用 Service。
type Dispatcher struct {
	hub        *ws.Hub
	svc        *Service
	friendRepo friend.Repository
}

// NewDispatcher 创建 Dispatcher。
// friendRepo 用于在线状态广播时查询好友列表。
func NewDispatcher(hub *ws.Hub, svc *Service, friendRepo friend.Repository) *Dispatcher {
	return &Dispatcher{hub: hub, svc: svc, friendRepo: friendRepo}
}

// Handle 实现 ws.MessageHandler 签名，由 Hub 的事件循环调用。
func (d *Dispatcher) Handle(c *ws.Client, data []byte) {
	// 心跳帧直接忽略
	if string(data) == "__ping__" {
		return
	}

	var frame InboundFrame
	if err := json.Unmarshal(data, &frame); err != nil {
		d.sendError(c, 1001, "invalid json")
		return
	}

	switch frame.Cmd {
	case "chat":
		d.handleChat(c, frame.Data)
	default:
		d.sendError(c, 1002, "unknown cmd: "+frame.Cmd)
	}
}

// handleChat 处理私聊消息发送
func (d *Dispatcher) handleChat(c *ws.Client, raw json.RawMessage) {
	var payload ChatPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		d.sendError(c, 1001, "invalid chat payload")
		return
	}
	if payload.ToUID <= 0 || payload.Content == "" {
		d.sendError(c, 1001, "to_uid and content are required")
		return
	}

	ctx := context.Background()
	saved, err := d.svc.SendMessage(ctx, c.UserID, payload.ToUID, payload.Content)
	if err != nil {
		if errors.Is(err, ErrNotFriend) {
			d.sendError(c, 4030, "you are not friends")
		} else if errors.Is(err, ErrInvalidInput) {
			d.sendError(c, 1001, "invalid input")
		} else {
			logger.L().Error("ws send message failed",
				zap.Int64("from", c.UserID),
				zap.Int64("to", payload.ToUID),
				zap.Error(err),
			)
			d.sendError(c, 5000, "server error")
		}
		return
	}

	// 1. 给接收方推送新消息
	push := OutboundFrame{
		Cmd: "chat",
		Data: ChatPush{
			ID:        saved.ID,
			FromUID:   saved.FromUID,
			ToUID:     saved.ToUID,
			Content:   saved.Content,
			MsgType:   saved.MsgType,
			CreatedAt: saved.CreatedAt,
		},
	}
	d.sendToUser(payload.ToUID, push)

	// 2. 给发送方回 ack（告知消息已落库，含服务端 ID）
	ack := OutboundFrame{
		Cmd: "chat_ack",
		Data: ChatAck{
			ID:        saved.ID,
			ToUID:     payload.ToUID,
			Content:   saved.Content,
			CreatedAt: saved.CreatedAt,
		},
	}
	d.sendJSONToClient(c, ack)
}

// ─── 连接生命周期钩子 ─────────────────────────────────────────────────────────

// OnConnect 在用户 WebSocket 连接建立后调用：
//  1. 推送全量未读数
//  2. 通知好友"上线了"
func (d *Dispatcher) OnConnect(c *ws.Client) {
	// 推送未读数
	d.PushUnreadInit(c)
	// 广播上线状态给好友
	d.broadcastOnlineStatus(c.UserID, true)
}

// OnDisconnect 在用户 WebSocket 连接断开后调用：广播离线状态给好友。
func (d *Dispatcher) OnDisconnect(c *ws.Client) {
	// 如果用户还有其他设备在线，不广播离线
	if d.hub.IsOnline(c.UserID) {
		return
	}
	d.broadcastOnlineStatus(c.UserID, false)
}

// broadcastOnlineStatus 向该用户的好友广播在/离线状态。
func (d *Dispatcher) broadcastOnlineStatus(uid int64, online bool) {
	ctx := context.Background()
	friends, err := d.friendRepo.ListFriends(ctx, uid)
	if err != nil || len(friends) == 0 {
		return
	}

	cmd := "online"
	if !online {
		cmd = "offline"
	}
	frame := OutboundFrame{Cmd: cmd, Data: OnlinePush{UID: uid}}
	b, _ := json.Marshal(frame)

	targets := make([]int64, 0, len(friends))
	for _, f := range friends {
		targets = append(targets, f.UID)
	}
	d.hub.BroadcastToUsers(b, uid, targets...)
}

// PushUnreadInit 在用户 WebSocket 连接建立后，推送全量未读数。
func (d *Dispatcher) PushUnreadInit(c *ws.Client) {
	ctx := context.Background()
	counts, err := d.svc.GetUnread(ctx, c.UserID)
	if err != nil {
		return
	}
	// 转成 map[string]int64（JSON key 必须是 string）
	m := make(UnreadPush, len(counts))
	for k, v := range counts {
		m[int64Key(k)] = v
	}
	frame := OutboundFrame{Cmd: "unread_init", Data: m}
	d.sendJSONToClient(c, frame)
}

// ─── 辅助方法 ─────────────────────────────────────────────────────────────────

func (d *Dispatcher) sendError(c *ws.Client, code int, msg string) {
	frame := OutboundFrame{Cmd: "error", Data: ErrorPush{Code: code, Msg: msg}}
	d.sendJSONToClient(c, frame)
}

func (d *Dispatcher) sendJSONToClient(c *ws.Client, v any) {
	b, err := json.Marshal(v)
	if err != nil {
		return
	}
	select {
	case c.Send <- b:
	default:
		logger.L().Warn("ws send queue full, drop frame", zap.Int64("uid", c.UserID))
	}
}

func (d *Dispatcher) sendToUser(toUID int64, v any) {
	b, err := json.Marshal(v)
	if err != nil {
		return
	}
	d.hub.SendToUser(toUID, b)
}

// int64Key 将 int64 转为字符串，用作 JSON map key
func int64Key(n int64) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	digits := make([]byte, 0, 20)
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	if neg {
		digits = append([]byte{'-'}, digits...)
	}
	return string(digits)
}
