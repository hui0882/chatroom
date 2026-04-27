import http from './http';
import type { Friend, FriendRequest, UserSearchResult } from '../types';

export const friendApi = {
  // 搜索用户
  searchUser: (username: string) =>
    http.get<{ code: number; data: UserSearchResult }>('/users/search', { params: { username } }),

  // 发送好友申请
  sendRequest: (to_uid: number, message?: string) =>
    http.post<{ code: number; data: FriendRequest }>('/friends/requests', { to_uid, message: message ?? '' }),

  // 撤回申请
  cancelRequest: (id: number) =>
    http.post(`/friends/requests/${id}/cancel`),

  // 接受申请
  acceptRequest: (id: number) =>
    http.post(`/friends/requests/${id}/accept`),

  // 拒绝申请
  rejectRequest: (id: number) =>
    http.post(`/friends/requests/${id}/reject`),

  // 收到的申请列表
  listReceived: () =>
    http.get<{ code: number; data: FriendRequest[] }>('/friends/requests/received'),

  // 发出的申请列表
  listSent: () =>
    http.get<{ code: number; data: FriendRequest[] }>('/friends/requests/sent'),

  // 好友列表
  listFriends: () =>
    http.get<{ code: number; data: Friend[] }>('/friends'),

  // 删除好友
  deleteFriend: (uid: number) =>
    http.delete(`/friends/${uid}`),
};
