import { Layout, Menu, Button, Avatar, Typography, Space, message, Badge } from 'antd';
import {
  MessageOutlined, SettingOutlined, UserOutlined, LogoutOutlined,
  TeamOutlined,
} from '@ant-design/icons';
import { useState, useEffect } from 'react';
import { useAuthStore } from '../../store/authStore';
import { useWsStore } from '../../store/wsStore';
import { authApi } from '../../api/auth';
import ChatTestPage from '../../pages/chat/ChatTestPage';
import AdminPage from '../../pages/admin/AdminPage';
import ContactPage from '../../pages/contact/ContactPage';

const { Header, Content, Sider } = Layout;
const { Text } = Typography;

type PageKey = 'chat' | 'contact' | 'admin';

export default function AppLayout() {
  const { user, clearAuth, sessionId } = useAuthStore();
  const { connect, disconnect, unread } = useWsStore();
  const [page, setPage] = useState<PageKey>('chat');

  // 建立 WebSocket 连接
  useEffect(() => {
    if (sessionId) {
      connect(sessionId);
    }
    return () => {
      disconnect();
    };
  }, [sessionId, connect, disconnect]);

  // 计算联系人总未读数
  const totalUnread = Object.values(unread).reduce((sum, n) => sum + n, 0);
  const unreadDisplay = totalUnread === 0 ? undefined : totalUnread > 99 ? '99+' : totalUnread;

  const handleLogout = async () => {
    try {
      await authApi.logout();
    } catch {
      // 忽略
    }
    disconnect();
    clearAuth();
    message.success('已退出登录');
  };

  const menuItems = [
    { key: 'chat', icon: <MessageOutlined />, label: 'WS 调试' },
    {
      key: 'contact',
      icon: <TeamOutlined />,
      label: (
        <Badge count={unreadDisplay} offset={[12, 0]} size="small">
          联系人
        </Badge>
      ),
    },
    ...(user?.role === 'admin'
      ? [{ key: 'admin', icon: <SettingOutlined />, label: '用户管理' }]
      : []),
  ];

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Sider theme="light" width={200} style={{ borderRight: '1px solid #f0f0f0' }}>
        <div
          style={{
            padding: '16px',
            fontWeight: 700,
            fontSize: 18,
            letterSpacing: 1,
            borderBottom: '1px solid #f0f0f0',
          }}
        >
          💬 Chatroom
        </div>
        <Menu
          mode="inline"
          selectedKeys={[page]}
          items={menuItems}
          onClick={({ key }) => setPage(key as PageKey)}
          style={{ borderRight: 0 }}
        />
      </Sider>

      <Layout>
        <Header
          style={{
            background: '#fff',
            padding: '0 24px',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'flex-end',
            borderBottom: '1px solid #f0f0f0',
          }}
        >
          <Space>
            <Avatar icon={<UserOutlined />} style={{ background: '#1677ff' }} />
            <Text strong>{user?.nickname}</Text>
            {user?.role === 'admin' && (
              <Text type="secondary" style={{ fontSize: 12 }}>（管理员）</Text>
            )}
            <Button
              type="text"
              icon={<LogoutOutlined />}
              onClick={handleLogout}
            >
              退出
            </Button>
          </Space>
        </Header>

        <Content style={{ padding: 24, background: '#f5f5f5' }}>
          {page === 'chat' && <ChatTestPage />}
          {page === 'contact' && <ContactPage />}
          {page === 'admin' && user?.role === 'admin' && <AdminPage />}
        </Content>
      </Layout>
    </Layout>
  );
}
