/**
 * wsStore — 管理全局单例 WebSocket 连接
 *
 * 职责：
 * 1. 维护一个 WS 连接，应用启动后由 AppLayout 调用 connect()
 * 2. 接收服务端帧，按 cmd 分发给订阅者
 * 3. 管理未读数（unread_init / chat 帧触发更新）
 * 4. 管理好友在线状态（online / offline 帧触发更新）
 */

import { create } from 'zustand';
import type {
  WsFrame, WsChatPush, WsChatAck, WsErrorPush,
  WsOnlinePush, WsUnreadInit,
} from '../types';

// ─── 消息订阅机制 ──────────────────────────────────────────────────────────────

type ChatListener = (msg: WsChatPush) => void;
type AckListener = (ack: WsChatAck) => void;

const chatListeners = new Map<number, Set<ChatListener>>(); // key = peer_uid

export function subscribeChatMessages(peerUid: number, fn: ChatListener) {
  if (!chatListeners.has(peerUid)) chatListeners.set(peerUid, new Set());
  chatListeners.get(peerUid)!.add(fn);
  return () => chatListeners.get(peerUid)?.delete(fn);
}

const ackListeners = new Map<number, Set<AckListener>>(); // key = peer_uid

export function subscribeChatAck(peerUid: number, fn: AckListener) {
  if (!ackListeners.has(peerUid)) ackListeners.set(peerUid, new Set());
  ackListeners.get(peerUid)!.add(fn);
  return () => ackListeners.get(peerUid)?.delete(fn);
}

// ─── Store 定义 ───────────────────────────────────────────────────────────────

interface WsState {
  ws: WebSocket | null;
  connected: boolean;
  /** 未读数 map：{ [peer_uid: string]: count } */
  unread: Record<string, number>;
  /** 在线好友 uid set */
  onlineUids: Set<number>;

  connect: (sessionId: string) => void;
  disconnect: () => void;
  sendChat: (toUid: number, content: string) => boolean;
  clearUnread: (peerUid: number) => void;
  setUnread: (peerUid: number, count: number) => void;
}

export const useWsStore = create<WsState>()((set, get) => ({
  ws: null,
  connected: false,
  unread: {},
  onlineUids: new Set(),

  connect(sessionId: string) {
    const existing = get().ws;
    if (existing && existing.readyState <= WebSocket.OPEN) return; // 已连接

    const url = `${location.protocol === 'https:' ? 'wss' : 'ws'}://${location.host}/ws?session_id=${sessionId}&device=web`;
    const socket = new WebSocket(url);

    socket.onopen = () => {
      set({ connected: true });
      // 客户端心跳：每 25s 发一次 __ping__（服务端心跳是 30s）
      const hb = setInterval(() => {
        if (socket.readyState === WebSocket.OPEN) {
          socket.send('__ping__');
        } else {
          clearInterval(hb);
        }
      }, 25000);
    };

    socket.onclose = () => {
      set({ connected: false, ws: null });
    };

    socket.onerror = () => {
      set({ connected: false });
    };

    socket.onmessage = (event) => {
      handleFrame(event.data, set, get);
    };

    set({ ws: socket });
  },

  disconnect() {
    const { ws } = get();
    ws?.close();
    set({ ws: null, connected: false });
  },

  sendChat(toUid, content) {
    const { ws, connected } = get();
    if (!ws || !connected) return false;
    ws.send(JSON.stringify({ cmd: 'chat', data: { to_uid: toUid, content } }));
    return true;
  },

  clearUnread(peerUid) {
    set((s) => ({ unread: { ...s.unread, [String(peerUid)]: 0 } }));
  },

  setUnread(peerUid, count) {
    set((s) => ({ unread: { ...s.unread, [String(peerUid)]: count } }));
  },
}));

// ─── 帧分发 ───────────────────────────────────────────────────────────────────

function handleFrame(
  raw: string,
  set: (partial: Partial<WsState> | ((s: WsState) => Partial<WsState>)) => void,
  get: () => WsState,
) {
  let frame: WsFrame;
  try {
    frame = JSON.parse(raw);
  } catch {
    return;
  }

  switch (frame.cmd) {
    case 'chat': {
      const push = frame.data as WsChatPush;
      // 触发对应对话的监听器
      chatListeners.get(push.from_uid)?.forEach((fn) => fn(push));
      // 更新未读数
      if (!chatListeners.get(push.from_uid)?.size) {
        // 窗口没打开才加未读
        set((s) => ({
          unread: {
            ...s.unread,
            [String(push.from_uid)]: (s.unread[String(push.from_uid)] ?? 0) + 1,
          },
        }));
      }
      break;
    }

    case 'chat_ack': {
      const ack = frame.data as WsChatAck;
      ackListeners.get(ack.to_uid)?.forEach((fn) => fn(ack));
      break;
    }

    case 'unread_init': {
      const data = frame.data as WsUnreadInit;
      set({ unread: { ...data } });
      break;
    }

    case 'online': {
      const push = frame.data as WsOnlinePush;
      set((s) => {
        const next = new Set(s.onlineUids);
        next.add(push.uid);
        return { onlineUids: next };
      });
      break;
    }

    case 'offline': {
      const push = frame.data as WsOnlinePush;
      set((s) => {
        const next = new Set(s.onlineUids);
        next.delete(push.uid);
        return { onlineUids: next };
      });
      break;
    }

    case 'error': {
      const err = frame.data as WsErrorPush;
      console.warn('[ws] error from server:', err);
      break;
    }
  }
}
