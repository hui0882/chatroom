import { useEffect, useRef, useCallback } from 'react';

export type WsStatus = 'connecting' | 'open' | 'closed' | 'error';

export interface UseWebSocketOptions {
  url: string;
  onMessage: (data: string) => void;
  onStatusChange?: (status: WsStatus) => void;
  /** 客户端心跳间隔（ms），默认 25000。设 0 禁用。
   *  注意：服务端每 30s 发一次 ping，浏览器 WS 会自动回 pong，
   *  这里的心跳是客户端主动发送文本心跳包（可选），用于穿越某些代理。
   */
  heartbeatInterval?: number;
}

/**
 * useWebSocket
 *
 * 封装 WebSocket 连接，含：
 * - 服务端 ping/pong 心跳自动响应（浏览器原生行为，无需额外处理）
 * - 可选的客户端文本心跳包（发送 "__ping__"，服务端收到后会回显，可忽略）
 * - 连接断开后不自动重连（由上层决定是否重连）
 */
export function useWebSocket({
  url,
  onMessage,
  onStatusChange,
  heartbeatInterval = 25_000,
}: UseWebSocketOptions) {
  const wsRef = useRef<WebSocket | null>(null);
  const heartbeatRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const statusRef = useRef<WsStatus>('closed');

  const setStatus = useCallback(
    (s: WsStatus) => {
      statusRef.current = s;
      onStatusChange?.(s);
    },
    [onStatusChange]
  );

  const clearHeartbeat = () => {
    if (heartbeatRef.current) {
      clearInterval(heartbeatRef.current);
      heartbeatRef.current = null;
    }
  };

  const connect = useCallback(() => {
    if (wsRef.current && wsRef.current.readyState < WebSocket.CLOSING) {
      return; // 已连接或正在连接
    }

    setStatus('connecting');
    const ws = new WebSocket(url);
    wsRef.current = ws;

    ws.onopen = () => {
      setStatus('open');

      if (heartbeatInterval > 0) {
        heartbeatRef.current = setInterval(() => {
          if (ws.readyState === WebSocket.OPEN) {
            ws.send('__ping__');
          }
        }, heartbeatInterval);
      }
    };

    ws.onmessage = (e) => {
      // 忽略服务端对 __ping__ 的回显
      if (typeof e.data === 'string' && e.data === '__gnip__') return;
      onMessage(e.data);
    };

    ws.onerror = () => {
      setStatus('error');
      clearHeartbeat();
    };

    ws.onclose = () => {
      setStatus('closed');
      clearHeartbeat();
    };
  }, [url, onMessage, heartbeatInterval, setStatus]);

  const disconnect = useCallback(() => {
    clearHeartbeat();
    wsRef.current?.close();
  }, []);

  const send = useCallback((data: string) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(data);
      return true;
    }
    return false;
  }, []);

  // 组件卸载时关闭连接
  useEffect(() => {
    return () => {
      clearHeartbeat();
      wsRef.current?.close();
    };
  }, []);

  return { connect, disconnect, send, statusRef };
}
