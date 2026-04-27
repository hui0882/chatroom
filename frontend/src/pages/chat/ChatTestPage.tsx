import { useState, useRef, useEffect, useCallback } from 'react';
import {
  Card, Input, Button, Tag, Typography, Divider, Space, Tabs, Badge
} from 'antd';
import { SendOutlined, ApiOutlined, DisconnectOutlined } from '@ant-design/icons';
import { useWebSocket, type WsStatus } from '../../hooks/useWebSocket';

const { Text, Paragraph } = Typography;
const { TextArea } = Input;

// ─── 普通回显 Tab ─────────────────────────────────────────────────────────────

interface EchoMessage {
  dir: 'sent' | 'recv';
  text: string;
  time: string;
}

function EchoTab() {
  const [status, setStatus] = useState<WsStatus>('closed');
  const [messages, setMessages] = useState<EchoMessage[]>([]);
  const [input, setInput] = useState('');
  const listRef = useRef<HTMLDivElement>(null);

  const addMsg = (dir: 'sent' | 'recv', text: string) => {
    setMessages((prev) => [
      ...prev,
      { dir, text, time: new Date().toLocaleTimeString() },
    ]);
  };

  const { connect, disconnect, send } = useWebSocket({
    url: `ws://${location.host}/websocket_test`,
    onMessage: (data) => addMsg('recv', data),
    onStatusChange: setStatus,
  });

  // 自动滚动到底部
  useEffect(() => {
    listRef.current?.scrollTo({ top: listRef.current.scrollHeight, behavior: 'smooth' });
  }, [messages]);

  const handleSend = () => {
    if (!input.trim()) return;
    send(input);
    addMsg('sent', input);
    setInput('');
  };

  const statusColor: Record<WsStatus, string> = {
    connecting: 'processing',
    open: 'success',
    closed: 'default',
    error: 'error',
  };
  const statusText: Record<WsStatus, string> = {
    connecting: '连接中',
    open: '已连接',
    closed: '未连接',
    error: '连接错误',
  };

  return (
    <Space direction="vertical" style={{ width: '100%' }}>
      <Space>
        <Badge status={statusColor[status] as 'processing' | 'success' | 'default' | 'error'} />
        <Text>{statusText[status]}</Text>
        {status === 'closed' || status === 'error' ? (
          <Button type="primary" size="small" icon={<ApiOutlined />} onClick={connect}>
            连接
          </Button>
        ) : (
          <Button size="small" icon={<DisconnectOutlined />} onClick={disconnect}>
            断开
          </Button>
        )}
      </Space>

      <div
        ref={listRef}
        style={{
          height: 340,
          overflowY: 'auto',
          border: '1px solid #f0f0f0',
          borderRadius: 8,
          padding: '8px 12px',
          background: '#fafafa',
        }}
      >
        {messages.length === 0 && (
          <Text type="secondary" style={{ display: 'block', textAlign: 'center', marginTop: 120 }}>
            连接后发送消息，服务器会返回倒序内容
          </Text>
        )}
        {messages.map((m, i) => (
          <div
            key={i}
            style={{
              display: 'flex',
              justifyContent: m.dir === 'sent' ? 'flex-end' : 'flex-start',
              marginBottom: 8,
            }}
          >
            <div
              style={{
                maxWidth: '70%',
                padding: '6px 12px',
                borderRadius: m.dir === 'sent' ? '12px 2px 12px 12px' : '2px 12px 12px 12px',
                background: m.dir === 'sent' ? '#1677ff' : '#fff',
                color: m.dir === 'sent' ? '#fff' : '#000',
                boxShadow: '0 1px 4px rgba(0,0,0,0.08)',
                wordBreak: 'break-all',
              }}
            >
              <div style={{ fontSize: 14 }}>{m.text}</div>
              <div style={{ fontSize: 10, opacity: 0.6, textAlign: 'right', marginTop: 2 }}>
                {m.time}
              </div>
            </div>
          </div>
        ))}
      </div>

      <Space.Compact style={{ width: '100%' }}>
        <Input
          value={input}
          onChange={(e) => setInput(e.target.value)}
          onPressEnter={handleSend}
          placeholder="输入消息，支持中文，回车发送"
          disabled={status !== 'open'}
        />
        <Button
          type="primary"
          icon={<SendOutlined />}
          onClick={handleSend}
          disabled={status !== 'open'}
        >
          发送
        </Button>
      </Space.Compact>
    </Space>
  );
}

// ─── 流式输出 Tab ─────────────────────────────────────────────────────────────

function StreamTab() {
  const [status, setStatus] = useState<WsStatus>('closed');
  const [input, setInput] = useState('');
  const [streaming, setStreaming] = useState(false);
  const [streamText, setStreamText] = useState('');
  const [history, setHistory] = useState<{ input: string; output: string }[]>([]);
  const bufRef = useRef('');

  const handleMessage = useCallback((raw: string) => {
    try {
      const chunk = JSON.parse(raw) as { type: string; content?: string };
      if (chunk.type === 'chunk' && chunk.content) {
        bufRef.current += chunk.content;
        setStreamText(bufRef.current);
      } else if (chunk.type === 'done') {
        setHistory((prev) => [
          ...prev,
          { input: prev[prev.length - 1]?.input ?? '', output: bufRef.current },
        ]);
        bufRef.current = '';
        setStreamText('');
        setStreaming(false);
      }
    } catch {
      // 非 JSON 帧忽略
    }
  }, []);

  const { connect, disconnect, send } = useWebSocket({
    url: `ws://${location.host}/websocket_stream`,
    onMessage: handleMessage,
    onStatusChange: setStatus,
  });

  const handleSend = () => {
    if (!input.trim() || streaming) return;
    setHistory((prev) => [...prev, { input, output: '' }]);
    bufRef.current = '';
    setStreamText('');
    setStreaming(true);
    send(input);
    setInput('');
  };

  const statusColor: Record<WsStatus, 'processing' | 'success' | 'default' | 'error'> = {
    connecting: 'processing',
    open: 'success',
    closed: 'default',
    error: 'error',
  };

  return (
    <Space direction="vertical" style={{ width: '100%' }}>
      <Space>
        <Badge status={statusColor[status]} />
        <Text>{status === 'open' ? '已连接' : status === 'connecting' ? '连接中' : '未连接'}</Text>
        {status !== 'open' ? (
          <Button type="primary" size="small" icon={<ApiOutlined />} onClick={connect}>
            连接
          </Button>
        ) : (
          <Button size="small" icon={<DisconnectOutlined />} onClick={disconnect}>
            断开
          </Button>
        )}
        {streaming && <Tag color="blue">输出中…</Tag>}
      </Space>

      {/* 历史记录 */}
      {history.map((h, i) => (
        <Card key={i} size="small" style={{ background: '#f9f9f9' }}>
          <Text type="secondary" style={{ fontSize: 12 }}>发送：</Text>
          <Paragraph style={{ margin: '2px 0 8px' }}>{h.input}</Paragraph>
          <Divider style={{ margin: '4px 0' }} />
          <Text type="secondary" style={{ fontSize: 12 }}>回复：</Text>
          <Paragraph style={{ margin: '2px 0 0', wordBreak: 'break-all' }}>
            {h.output || <Text type="secondary">（空）</Text>}
          </Paragraph>
        </Card>
      ))}

      {/* 当前流式输出 */}
      {streaming && (
        <Card size="small" style={{ background: '#e6f4ff', borderColor: '#91caff' }}>
          <Text type="secondary" style={{ fontSize: 12 }}>正在输出：</Text>
          <Paragraph style={{ margin: '4px 0 0', wordBreak: 'break-all', minHeight: 24 }}>
            {streamText}
            <span style={{ animation: 'blink 1s infinite', marginLeft: 2 }}>▌</span>
          </Paragraph>
        </Card>
      )}

      <TextArea
        value={input}
        onChange={(e) => setInput(e.target.value)}
        placeholder="输入内容，服务器将以 20 字/秒流式反向输出"
        autoSize={{ minRows: 2, maxRows: 5 }}
        disabled={status !== 'open' || streaming}
        onPressEnter={(e) => {
          if (!e.shiftKey) {
            e.preventDefault();
            handleSend();
          }
        }}
      />
      <Button
        type="primary"
        icon={<SendOutlined />}
        onClick={handleSend}
        disabled={status !== 'open' || streaming || !input.trim()}
        block
      >
        发送（Enter 快捷键，Shift+Enter 换行）
      </Button>
    </Space>
  );
}

// ─── 主页面 ───────────────────────────────────────────────────────────────────

export default function ChatTestPage() {
  return (
    <Card
      title="WebSocket 调试"
      style={{ maxWidth: 660, margin: '0 auto' }}
    >
      <Tabs
        defaultActiveKey="echo"
        items={[
          { key: 'echo', label: '回显测试', children: <EchoTab /> },
          { key: 'stream', label: '流式输出', children: <StreamTab /> },
        ]}
      />
    </Card>
  );
}
