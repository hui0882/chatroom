package message

import (
	"context"
	"fmt"

	"github.com/hui0882/chatroom/internal/friend"
)

// Service 封装消息业务逻辑。
type Service struct {
	repo      Repository
	unread    UnreadStore
	friendSvc *friend.Service
}

// NewService 创建消息 Service。
// friendSvc 用于发送前校验双方是否是好友。
func NewService(repo Repository, unread UnreadStore, friendSvc *friend.Service) *Service {
	return &Service{
		repo:      repo,
		unread:    unread,
		friendSvc: friendSvc,
	}
}

// SendMessage 发送私聊消息（已经是好友才允许）。
// 返回完整消息（含 ID 和 CreatedAt）。
func (s *Service) SendMessage(ctx context.Context, fromUID, toUID int64, content string) (*Message, error) {
	if content == "" {
		return nil, ErrInvalidInput
	}

	// 校验好友关系
	ok, err := s.friendSvc.IsFriend(ctx, fromUID, toUID)
	if err != nil {
		return nil, fmt.Errorf("check friend: %w", err)
	}
	if !ok {
		return nil, ErrNotFriend
	}

	msg := &Message{
		FromUID: fromUID,
		ToUID:   toUID,
		Content: content,
		MsgType: MsgTypeText,
	}

	saved, err := s.repo.Save(ctx, msg)
	if err != nil {
		return nil, err
	}

	// 未读计数 +1（给接收方）
	_ = s.unread.Incr(ctx, toUID, fromUID)

	return saved, nil
}

// ListHistory 分页查询两人聊天历史。
func (s *Service) ListHistory(ctx context.Context, selfUID, peerUID int64, cursor int64, limit int) ([]*Message, error) {
	// 清零自己看 peer 的未读数
	_ = s.unread.Clear(ctx, selfUID, peerUID)
	return s.repo.ListHistory(ctx, selfUID, peerUID, cursor, limit)
}

// GetUnread 获取 uid 的所有对话未读数。
func (s *Service) GetUnread(ctx context.Context, uid int64) (map[int64]int64, error) {
	return s.unread.GetAll(ctx, uid)
}

// ClearUnread 清零 uid 来自 peerUID 的未读数（打开会话窗口时调用）。
func (s *Service) ClearUnread(ctx context.Context, uid, peerUID int64) error {
	return s.unread.Clear(ctx, uid, peerUID)
}
