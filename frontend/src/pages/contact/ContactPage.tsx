import { useState, useEffect, useRef, useCallback } from 'react';
import {
  Tabs, Input, Button, List, Avatar, Empty, Badge,
  Popconfirm, message, Tag, Modal, Form, Space, Typography, Spin,
} from 'antd';
import {
  UserOutlined, SearchOutlined, UserAddOutlined,
  DeleteOutlined, CheckOutlined, CloseOutlined,
  MessageOutlined, SendOutlined, LoadingOutlined,
} from '@ant-design/icons';
import { friendApi } from '../../api/friend';
import { messageApi } from '../../api/message';
import type { Friend, FriendRequest, UserSearchResult, ChatMessage } from '../../types';
import { useAuthStore } from '../../store/authStore';
import {
  useWsStore,
  subscribeChatMessages,
  subscribeChatAck,
} from '../../store/wsStore';

const { Text } = Typography;

// ─── 好友列表 Tab ─────────────────────────────────────────────────────────────

function FriendListTab({ onChat }: { onChat: (friend: Friend) => void }) {
  const [friends, setFriends] = useState<Friend[]>([]);
  const [loading, setLoading] = useState(false);
  const { onlineUids, unread } = useWsStore();

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const res = await friendApi.listFriends();
      setFriends(res.data.data ?? []);
    } catch {
      message.error('加载好友列表失败');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { load(); }, [load]);

  const handleDelete = async (uid: number) => {
    try {
      await friendApi.deleteFriend(uid);
      message.success('已删除好友');
      load();
    } catch {
      message.error('删除失败');
    }
  };

  return (
    <List
      loading={loading}
      dataSource={friends}
      locale={{ emptyText: <Empty description="暂无好友，去搜索添加吧" /> }}
      renderItem={(f) => {
        const isOnline = onlineUids.has(f.uid);
        const unreadCount = unread[String(f.uid)] ?? 0;
        return (
          <List.Item
            actions={[
              <Badge count={unreadCount > 100 ? '99+' : unreadCount} key="unread">
                <Button
                  type="link"
                  icon={<MessageOutlined />}
                  onClick={() => onChat(f)}
                >
                  发消息
                </Button>
              </Badge>,
              <Popconfirm
                key="del"
                title="确认删除好友？"
                onConfirm={() => handleDelete(f.uid)}
                okText="删除"
                cancelText="取消"
              >
                <Button type="link" danger icon={<DeleteOutlined />}>删除</Button>
              </Popconfirm>,
            ]}
          >
            <List.Item.Meta
              avatar={
                <Badge dot status={isOnline ? 'success' : 'default'} offset={[-2, 28]}>
                  <Avatar icon={<UserOutlined />} src={f.avatar_url || undefined} />
                </Badge>
              }
              title={f.remark || f.nickname}
              description={
                <Text type="secondary" style={{ fontSize: 12 }}>
                  @{f.username}{f.remark ? `（${f.nickname}）` : ''}
                  {isOnline && <Tag color="success" style={{ marginLeft: 4, fontSize: 10, padding: '0 4px' }}>在线</Tag>}
                </Text>
              }
            />
          </List.Item>
        );
      }}
    />
  );
}

// ─── 好友申请 Tab ─────────────────────────────────────────────────────────────

function RequestsTab() {
  const [received, setReceived] = useState<FriendRequest[]>([]);
  const [sent, setSent] = useState<FriendRequest[]>([]);
  const [loading, setLoading] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const [r1, r2] = await Promise.all([
        friendApi.listReceived(),
        friendApi.listSent(),
      ]);
      setReceived(r1.data.data ?? []);
      setSent(r2.data.data ?? []);
    } catch {
      message.error('加载申请列表失败');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { load(); }, [load]);

  const handleAccept = async (id: number) => {
    try {
      await friendApi.acceptRequest(id);
      message.success('已接受好友申请');
      load();
    } catch {
      message.error('操作失败');
    }
  };

  const handleReject = async (id: number) => {
    try {
      await friendApi.rejectRequest(id);
      message.success('已拒绝');
      load();
    } catch {
      message.error('操作失败');
    }
  };

  const handleCancel = async (id: number) => {
    try {
      await friendApi.cancelRequest(id);
      message.success('已撤回申请');
      load();
    } catch {
      message.error('操作失败');
    }
  };

  return (
    <div>
      <Text strong style={{ display: 'block', marginBottom: 8 }}>
        收到的申请 <Badge count={received.length} style={{ backgroundColor: '#1677ff' }} />
      </Text>
      <List
        loading={loading}
        dataSource={received}
        locale={{ emptyText: <Empty description="暂无待处理申请" image={Empty.PRESENTED_IMAGE_SIMPLE} /> }}
        renderItem={(req) => (
          <List.Item
            actions={[
              <Button
                key="accept"
                type="primary"
                size="small"
                icon={<CheckOutlined />}
                onClick={() => handleAccept(req.id)}
              >
                接受
              </Button>,
              <Button
                key="reject"
                size="small"
                danger
                icon={<CloseOutlined />}
                onClick={() => handleReject(req.id)}
              >
                拒绝
              </Button>,
            ]}
          >
            <List.Item.Meta
              avatar={<Avatar icon={<UserOutlined />} src={req.from_avatar || undefined} />}
              title={req.from_nickname || req.from_username}
              description={
                <Space direction="vertical" size={2}>
                  <Text type="secondary" style={{ fontSize: 12 }}>@{req.from_username}</Text>
                  {req.message && (
                    <Text type="secondary" style={{ fontSize: 12 }}>附言：{req.message}</Text>
                  )}
                </Space>
              }
            />
          </List.Item>
        )}
        style={{ marginBottom: 24 }}
      />

      <Text strong style={{ display: 'block', marginBottom: 8 }}>
        发出的申请
      </Text>
      <List
        loading={loading}
        dataSource={sent}
        locale={{ emptyText: <Empty description="暂无发出的申请" image={Empty.PRESENTED_IMAGE_SIMPLE} /> }}
        renderItem={(req) => (
          <List.Item
            actions={[
              <Popconfirm
                key="cancel"
                title="撤回这条申请？"
                onConfirm={() => handleCancel(req.id)}
                okText="撤回"
                cancelText="取消"
              >
                <Button size="small">撤回</Button>
              </Popconfirm>,
            ]}
          >
            <List.Item.Meta
              avatar={<Avatar icon={<UserOutlined />} />}
              title={req.to_nickname || req.to_username}
              description={
                <Space direction="vertical" size={2}>
                  <Text type="secondary" style={{ fontSize: 12 }}>@{req.to_username}</Text>
                  <Tag color="blue" style={{ fontSize: 11 }}>待对方确认</Tag>
                </Space>
              }
            />
          </List.Item>
        )}
      />
    </div>
  );
}

// ─── 搜索添加 Tab ─────────────────────────────────────────────────────────────

function SearchTab() {
  const [keyword, setKeyword] = useState('');
  const [result, setResult] = useState<UserSearchResult | null>(null);
  const [notFound, setNotFound] = useState(false);
  const [searching, setSearching] = useState(false);
  const [modalOpen, setModalOpen] = useState(false);
  const [sending, setSending] = useState(false);
  const [form] = Form.useForm();

  const handleSearch = async () => {
    if (!keyword.trim()) return;
    setSearching(true);
    setResult(null);
    setNotFound(false);
    try {
      const res = await friendApi.searchUser(keyword.trim());
      setResult(res.data.data);
    } catch {
      setNotFound(true);
    } finally {
      setSearching(false);
    }
  };

  const handleSend = async (values: { message?: string }) => {
    if (!result) return;
    setSending(true);
    try {
      await friendApi.sendRequest(result.id, values.message);
      message.success('申请已发送');
      setModalOpen(false);
      form.resetFields();
      setResult((prev) => prev ? { ...prev, is_friend: true } : prev);
    } catch (e: unknown) {
      const err = e as { response?: { data?: { msg?: string } } };
      message.error(err?.response?.data?.msg ?? '发送失败');
    } finally {
      setSending(false);
    }
  };

  return (
    <div>
      <Space.Compact style={{ width: '100%', marginBottom: 16 }}>
        <Input
          placeholder="输入对方用户名精确搜索"
          value={keyword}
          onChange={(e) => setKeyword(e.target.value)}
          onPressEnter={handleSearch}
          prefix={<SearchOutlined />}
        />
        <Button type="primary" loading={searching} onClick={handleSearch}>
          搜索
        </Button>
      </Space.Compact>

      {notFound && (
        <Empty description="未找到该用户" image={Empty.PRESENTED_IMAGE_SIMPLE} />
      )}

      {result && (
        <List
          dataSource={[result]}
          renderItem={(u) => (
            <List.Item
              actions={[
                u.is_friend ? (
                  <Tag key="friend" color="success">已是好友</Tag>
                ) : (
                  <Button
                    key="add"
                    type="primary"
                    icon={<UserAddOutlined />}
                    onClick={() => setModalOpen(true)}
                  >
                    添加好友
                  </Button>
                ),
              ]}
            >
              <List.Item.Meta
                avatar={<Avatar icon={<UserOutlined />} src={u.avatar_url || undefined} />}
                title={u.nickname}
                description={<Text type="secondary" style={{ fontSize: 12 }}>@{u.username}</Text>}
              />
            </List.Item>
          )}
        />
      )}

      <Modal
        title="发送好友申请"
        open={modalOpen}
        onCancel={() => setModalOpen(false)}
        footer={null}
        destroyOnHidden
      >
        <Form form={form} onFinish={handleSend} layout="vertical">
          <Form.Item label="附言（可选）" name="message">
            <Input.TextArea rows={3} placeholder="输入一句话打个招呼…" maxLength={100} showCount />
          </Form.Item>
          <Form.Item style={{ marginBottom: 0, textAlign: 'right' }}>
            <Space>
              <Button onClick={() => setModalOpen(false)}>取消</Button>
              <Button type="primary" htmlType="submit" loading={sending}>发送申请</Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}

// ─── 聊天窗口 ─────────────────────────────────────────────────────────────────

interface LocalMessage {
  id: number; // 临时消息用负数（pending）
  from_uid: number;
  content: string;
  created_at: string;
  pending?: boolean; // 等待服务端 ack
  failed?: boolean;
}

function ChatWindow({ friend, onClose }: { friend: Friend; onClose: () => void }) {
  const { user } = useAuthStore();
  const { sendChat, clearUnread, connected } = useWsStore();
  const [msgs, setMsgs] = useState<LocalMessage[]>([]);
  const [input, setInput] = useState('');
  const [loadingHistory, setLoadingHistory] = useState(false);
  const [nextCursor, setNextCursor] = useState(0);
  const [hasMore, setHasMore] = useState(true);
  const bottomRef = useRef<HTMLDivElement>(null);
  const pendingIdRef = useRef(-1); // 临时消息 ID（负数递减）

  // 加载历史消息
  const loadHistory = useCallback(async (cursor = 0) => {
    setLoadingHistory(true);
    try {
      const res = await messageApi.listHistory(friend.uid, cursor);
      const { list, next_cursor } = res.data.data;
      const sorted = [...(list ?? [])].reverse(); // API 返回倒序，展示正序
      if (cursor === 0) {
        // 首次加载
        setMsgs(sorted.map(toLocal));
      } else {
        // 加载更多：prepend
        setMsgs((prev) => [...sorted.map(toLocal), ...prev]);
      }
      setNextCursor(next_cursor);
      setHasMore(next_cursor !== 0);
    } finally {
      setLoadingHistory(false);
    }
  }, [friend.uid]);

  useEffect(() => {
    setMsgs([]);
    setHasMore(true);
    loadHistory(0);
    // 清零该对话未读数
    clearUnread(friend.uid);
    messageApi.clearUnread(friend.uid).catch(() => {/* ignore */});
  }, [friend.uid, loadHistory, clearUnread]);

  // 滚动到底部（首次加载 / 新消息）
  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [msgs.length]);

  // 订阅来自好友的新消息
  useEffect(() => {
    const unsub = subscribeChatMessages(friend.uid, (push) => {
      setMsgs((prev) => [...prev, {
        id: push.id,
        from_uid: push.from_uid,
        content: push.content,
        created_at: push.created_at,
      }]);
      // 清零未读（窗口已打开）
      clearUnread(friend.uid);
    });
    return unsub;
  }, [friend.uid, clearUnread]);

  // 订阅发送 ack
  useEffect(() => {
    const unsub = subscribeChatAck(friend.uid, (ack) => {
      setMsgs((prev) => prev.map((m) =>
        m.pending && m.content === ack.content
          ? { ...m, id: ack.id, created_at: ack.created_at, pending: false }
          : m
      ));
    });
    return unsub;
  }, [friend.uid]);

  const handleSend = () => {
    if (!input.trim() || !user) return;
    const content = input.trim();
    setInput('');

    // 乐观更新：先在本地显示 pending 消息
    const tmpId = pendingIdRef.current--;
    setMsgs((prev) => [...prev, {
      id: tmpId,
      from_uid: user.id,
      content,
      created_at: new Date().toISOString(),
      pending: true,
    }]);

    const ok = sendChat(friend.uid, content);
    if (!ok) {
      // WS 未连接，标记失败
      setMsgs((prev) => prev.map((m) => m.id === tmpId ? { ...m, failed: true, pending: false } : m));
    }
  };

  const friendName = friend.remark || friend.nickname;

  return (
    <div style={{
      position: 'fixed', right: 24, bottom: 24,
      width: 360, height: 520,
      background: '#fff', borderRadius: 12,
      boxShadow: '0 8px 32px rgba(0,0,0,0.18)',
      display: 'flex', flexDirection: 'column',
      zIndex: 1000,
    }}>
      {/* 顶栏 */}
      <div style={{
        padding: '12px 16px',
        borderBottom: '1px solid #f0f0f0',
        display: 'flex', alignItems: 'center', justifyContent: 'space-between',
        borderRadius: '12px 12px 0 0',
        background: '#1677ff',
        color: '#fff',
      }}>
        <Space>
          <Avatar size="small" icon={<UserOutlined />} src={friend.avatar_url || undefined} />
          <Text strong style={{ color: '#fff' }}>{friendName}</Text>
          {!connected && <Tag color="warning" style={{ fontSize: 10 }}>连接中…</Tag>}
        </Space>
        <Button type="text" size="small" style={{ color: '#fff' }} onClick={onClose}>✕</Button>
      </div>

      {/* 消息区 */}
      <div style={{ flex: 1, padding: '8px 12px', overflowY: 'auto', background: '#f5f7fa' }}>
        {hasMore && (
          <div style={{ textAlign: 'center', marginBottom: 8 }}>
            <Button
              size="small"
              type="link"
              loading={loadingHistory}
              onClick={() => loadHistory(nextCursor)}
            >
              加载更多
            </Button>
          </div>
        )}
        {loadingHistory && msgs.length === 0 && (
          <div style={{ textAlign: 'center', paddingTop: 40 }}>
            <Spin indicator={<LoadingOutlined />} />
          </div>
        )}
        {msgs.length === 0 && !loadingHistory && (
          <Empty
            description={<span style={{ fontSize: 13, color: '#999' }}>开始聊天吧 👋</span>}
            image={Empty.PRESENTED_IMAGE_SIMPLE}
            style={{ marginTop: 60 }}
          />
        )}
        {msgs.map((m) => {
          const isSelf = m.from_uid === user?.id;
          return (
            <div key={m.id} style={{
              display: 'flex',
              justifyContent: isSelf ? 'flex-end' : 'flex-start',
              marginBottom: 8,
            }}>
              <div style={{
                maxWidth: '75%',
                padding: '8px 12px',
                borderRadius: isSelf ? '12px 2px 12px 12px' : '2px 12px 12px 12px',
                background: isSelf ? '#1677ff' : '#fff',
                color: isSelf ? '#fff' : '#333',
                boxShadow: '0 1px 4px rgba(0,0,0,0.1)',
                fontSize: 14,
                lineHeight: 1.5,
                opacity: m.pending ? 0.65 : 1,
                border: m.failed ? '1px solid #ff4d4f' : 'none',
              }}>
                {m.content}
                {m.pending && (
                  <LoadingOutlined style={{ marginLeft: 4, fontSize: 10, opacity: 0.7 }} />
                )}
                {m.failed && (
                  <span style={{ marginLeft: 4, fontSize: 11, color: '#ff4d4f' }}>发送失败</span>
                )}
              </div>
            </div>
          );
        })}
        <div ref={bottomRef} />
      </div>

      {/* 输入区 */}
      <div style={{ padding: '8px 12px', borderTop: '1px solid #f0f0f0' }}>
        <Space.Compact style={{ width: '100%' }}>
          <Input
            placeholder={connected ? '输入消息，Enter 发送' : '正在连接…'}
            value={input}
            disabled={!connected}
            onChange={(e) => setInput(e.target.value)}
            onPressEnter={handleSend}
            maxLength={500}
          />
          <Button
            type="primary"
            icon={<SendOutlined />}
            disabled={!connected || !input.trim()}
            onClick={handleSend}
          >
            发送
          </Button>
        </Space.Compact>
      </div>
    </div>
  );
}

function toLocal(m: ChatMessage): LocalMessage {
  return {
    id: m.id,
    from_uid: m.from_uid,
    content: m.content,
    created_at: m.created_at,
  };
}

// ─── 主页面 ───────────────────────────────────────────────────────────────────

export default function ContactPage() {
  const [activeChat, setActiveChat] = useState<Friend | null>(null);
  const [activeTab, setActiveTab] = useState('friends');

  const tabs = [
    {
      key: 'friends',
      label: '好友列表',
      children: <FriendListTab onChat={(f) => setActiveChat(f)} />,
    },
    {
      key: 'requests',
      label: '好友申请',
      children: <RequestsTab />,
    },
    {
      key: 'search',
      label: '添加好友',
      children: <SearchTab />,
    },
  ];

  return (
    <div style={{ background: '#fff', borderRadius: 8, padding: 24, minHeight: 500 }}>
      <Tabs
        activeKey={activeTab}
        onChange={setActiveTab}
        items={tabs}
      />

      {activeChat && (
        <ChatWindow
          friend={activeChat}
          onClose={() => setActiveChat(null)}
        />
      )}
    </div>
  );
}
