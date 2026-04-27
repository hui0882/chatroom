package friend

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/hui0882/chatroom/internal/user"
)

// Service 好友业务逻辑层
type Service struct {
	repo     Repository
	userRepo user.Repository
}

func NewService(repo Repository, userRepo user.Repository) *Service {
	return &Service{repo: repo, userRepo: userRepo}
}

// SearchUser 按用户名搜索（排除自己）
func (s *Service) SearchUser(ctx context.Context, selfUID int64, keyword string) (*user.User, error) {
	u, err := s.userRepo.FindByUsername(ctx, keyword)
	if err != nil {
		return nil, ErrNotFound
	}
	if u.ID == selfUID {
		return nil, ErrSelfOperation
	}
	if u.Status != "active" {
		return nil, ErrNotFound
	}
	return u, nil
}

// SendRequest 发送好友申请
func (s *Service) SendRequest(ctx context.Context, fromUID, toUID int64, message string) (*Request, error) {
	if fromUID == toUID {
		return nil, ErrSelfOperation
	}
	// 已经是好友
	if ok, _ := s.repo.IsFriend(ctx, fromUID, toUID); ok {
		return nil, ErrAlreadyFriends
	}
	// 对方是否已向我发过申请（互发时直接接受）
	if rev, err := s.repo.FindRequest(ctx, toUID, fromUID); err == nil && rev.Status == "pending" {
		if err := s.acceptRequest(ctx, rev); err != nil {
			return nil, err
		}
		return rev, nil
	}
	return s.repo.CreateRequest(ctx, fromUID, toUID, message)
}

// CancelRequest 撤回申请（发送方操作）
func (s *Service) CancelRequest(ctx context.Context, fromUID, requestID int64) error {
	req, err := s.repo.FindRequestByID(ctx, requestID)
	if err != nil {
		return ErrNotFound
	}
	if req.FromUID != fromUID {
		return ErrNotFound
	}
	if req.Status != "pending" {
		return ErrNotPending
	}
	return s.repo.UpdateRequestStatus(ctx, requestID, "cancelled")
}

// AcceptRequest 接受申请（接收方操作）
func (s *Service) AcceptRequest(ctx context.Context, toUID, requestID int64) error {
	req, err := s.repo.FindRequestByID(ctx, requestID)
	if err != nil {
		return ErrNotFound
	}
	if req.ToUID != toUID {
		return ErrNotFound
	}
	if req.Status != "pending" {
		return ErrNotPending
	}
	return s.acceptRequest(ctx, req)
}

func (s *Service) acceptRequest(ctx context.Context, req *Request) error {
	if err := s.repo.UpdateRequestStatus(ctx, req.ID, "accepted"); err != nil {
		return fmt.Errorf("update request: %w", err)
	}
	return s.repo.AddFriends(ctx, req.FromUID, req.ToUID)
}

// RejectRequest 拒绝申请（接收方操作）
func (s *Service) RejectRequest(ctx context.Context, toUID, requestID int64) error {
	req, err := s.repo.FindRequestByID(ctx, requestID)
	if err != nil {
		return ErrNotFound
	}
	if req.ToUID != toUID {
		return ErrNotFound
	}
	if req.Status != "pending" {
		return ErrNotPending
	}
	return s.repo.UpdateRequestStatus(ctx, requestID, "rejected")
}

// ListFriends 获取好友列表
func (s *Service) ListFriends(ctx context.Context, uid int64) ([]*Friend, error) {
	return s.repo.ListFriends(ctx, uid)
}

// DeleteFriend 删除好友（双向）
func (s *Service) DeleteFriend(ctx context.Context, uid, friendUID int64) error {
	if uid == friendUID {
		return ErrSelfOperation
	}
	return s.repo.RemoveFriend(ctx, uid, friendUID)
}

// ListReceivedRequests 收到的待处理申请
func (s *Service) ListReceivedRequests(ctx context.Context, uid int64) ([]*Request, error) {
	return s.repo.ListReceivedRequests(ctx, uid)
}

// ListSentRequests 发出的待处理申请
func (s *Service) ListSentRequests(ctx context.Context, uid int64) ([]*Request, error) {
	return s.repo.ListSentRequests(ctx, uid)
}

// IsFriend 判断是否已是好友
func (s *Service) IsFriend(ctx context.Context, uid, friendUID int64) (bool, error) {
	return s.repo.IsFriend(ctx, uid, friendUID)
}

// 确保 sql 包被引用（repository 用到，service 间接依赖）
var _ = sql.ErrNoRows
