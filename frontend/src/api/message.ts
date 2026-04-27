import http from './http';
import type { HistoryResult } from '../types';

export const messageApi = {
  // 查询聊天历史（游标翻页）
  listHistory: (peerUid: number, cursor = 0, limit = 50) =>
    http.get<{ code: number; data: HistoryResult }>(`/messages/${peerUid}`, {
      params: { cursor, limit },
    }),

  // 获取全部对话未读数
  getUnread: () =>
    http.get<{ code: number; data: Record<string, number> }>('/messages/unread'),

  // 清零指定对话未读数
  clearUnread: (peerUid: number) =>
    http.post(`/messages/unread/${peerUid}/clear`),
};
