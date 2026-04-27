package friend

import "errors"

// ─── 领域错误 ─────────────────────────────────────────────────────────────────

var (
	ErrNotFound       = errors.New("friend: not found")
	ErrAlreadyFriends = errors.New("friend: already friends")
	ErrRequestExists  = errors.New("friend: request already exists")
	ErrSelfOperation  = errors.New("friend: cannot operate on yourself")
	ErrNotPending     = errors.New("friend: request is not pending")
)

// ─── 数据模型 ─────────────────────────────────────────────────────────────────

// Request 好友申请记录
type Request struct {
	ID        int64  `json:"id"`
	FromUID   int64  `json:"from_uid"`
	ToUID     int64  `json:"to_uid"`
	Message   string `json:"message"`
	Status    string `json:"status"` // pending / accepted / rejected / cancelled
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`

	// 联表冗余字段，查询时填充
	FromUsername string `json:"from_username,omitempty"`
	FromNickname string `json:"from_nickname,omitempty"`
	FromAvatar   string `json:"from_avatar,omitempty"`
	ToUsername   string `json:"to_username,omitempty"`
	ToNickname   string `json:"to_nickname,omitempty"`
}

// Friend 好友条目
type Friend struct {
	UID       int64  `json:"uid"` // 对方 ID
	Username  string `json:"username"`
	Nickname  string `json:"nickname"`
	Avatar    string `json:"avatar_url"`
	Remark    string `json:"remark"` // 备注名
	CreatedAt string `json:"created_at"`
}
