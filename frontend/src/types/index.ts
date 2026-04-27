// ─── 通用响应结构 ─────────────────────────────────────────────────────────────

export interface ApiResponse<T = null> {
  code: number;
  msg: string;
  data: T;
}

// ─── 用户 ─────────────────────────────────────────────────────────────────────

export interface User {
  id: number;
  username: string;
  nickname: string;
  avatar_url: string;
  bio: string;
  gender: 'unknown' | 'male' | 'female';
  age?: number;
  status: 'active' | 'banned' | 'deleted';
  role: 'user' | 'admin';
  created_at: string;
}

export interface AdminUser extends User {
  role: 'user' | 'admin';
  ban_reason?: string;
  ban_until?: string;
  deleted_at?: string;
}

// ─── 认证 ─────────────────────────────────────────────────────────────────────

export interface LoginResult {
  session_id: string;
  user: User;
}

export interface RegisterInput {
  username: string;
  password: string;
  nickname: string;
  gender?: 'unknown' | 'male' | 'female';
  age?: number;
}

// ─── WebSocket 流式帧 ─────────────────────────────────────────────────────────

export interface StreamChunk {
  type: 'chunk' | 'done' | 'error';
  content?: string;
  error?: string;
}

// ─── 管理员 ───────────────────────────────────────────────────────────────────

export interface AdminListResult {
  total: number;
  list: AdminUser[];
}

export interface BanInput {
  reason: string;
  ban_until?: string; // ISO8601，不传=永久封禁
}

// ─── 好友 ─────────────────────────────────────────────────────────────────────

export interface FriendRequest {
  id: number;
  from_uid: number;
  to_uid: number;
  message: string;
  status: 'pending' | 'accepted' | 'rejected' | 'cancelled';
  created_at: string;
  updated_at: string;
  // 收到的申请带发送方信息
  from_username?: string;
  from_nickname?: string;
  from_avatar?: string;
  // 发出的申请带接收方信息
  to_username?: string;
  to_nickname?: string;
}

export interface Friend {
  uid: number;
  username: string;
  nickname: string;
  avatar_url: string;
  remark: string;
  created_at: string;
}

export interface UserSearchResult {
  id: number;
  username: string;
  nickname: string;
  avatar_url: string;
  gender: string;
  is_friend: boolean;
}

// ─── 消息 ─────────────────────────────────────────────────────────────────────

export type MsgType = 'text';

export interface ChatMessage {
  id: number;
  from_uid: number;
  to_uid: number;
  content: string;
  msg_type: MsgType;
  created_at: string;
}

export interface HistoryResult {
  list: ChatMessage[];
  next_cursor: number; // 0 表示没有更多
}

// ─── WebSocket 帧协议 ─────────────────────────────────────────────────────────

export interface WsFrame<T = unknown> {
  cmd: string;
  data: T;
}

export interface WsChatPush {
  id: number;
  from_uid: number;
  to_uid: number;
  content: string;
  msg_type: MsgType;
  created_at: string;
}

export interface WsChatAck {
  id: number;
  to_uid: number;
  content: string;
  created_at: string;
}

export interface WsErrorPush {
  code: number;
  msg: string;
}

export interface WsOnlinePush {
  uid: number;
}

// unread_init: { [peer_uid: string]: number }
export type WsUnreadInit = Record<string, number>;
