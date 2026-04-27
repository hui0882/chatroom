import { useState, useEffect } from 'react';
import {
  Table, Button, Tag, Space, Input, Select, Modal, Form,
  message, DatePicker, Typography, Popconfirm, Card,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  StopOutlined, CheckCircleOutlined, DeleteOutlined, RedoOutlined,
  KeyOutlined, LogoutOutlined, SearchOutlined,
} from '@ant-design/icons';
import dayjs from 'dayjs';
import { adminApi } from '../../api/admin';
import type { AdminUser } from '../../types';

const { Text } = Typography;

const STATUS_COLOR: Record<string, string> = {
  active: 'success',
  banned: 'error',
  deleted: 'default',
};
const STATUS_TEXT: Record<string, string> = {
  active: '正常',
  banned: '已封禁',
  deleted: '已注销',
};

export default function AdminPage() {
  const [users, setUsers] = useState<AdminUser[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [page, setPage] = useState(1);
  const [keyword, setKeyword] = useState('');
  const [statusFilter, setStatusFilter] = useState<string | undefined>(undefined);

  // 封禁弹窗
  const [banTarget, setBanTarget] = useState<AdminUser | null>(null);
  const [banForm] = Form.useForm();
  const [banLoading, setBanLoading] = useState(false);

  // 重置密码弹窗
  const [pwdTarget, setPwdTarget] = useState<AdminUser | null>(null);
  const [pwdForm] = Form.useForm();
  const [pwdLoading, setPwdLoading] = useState(false);

  const fetchUsers = async (p = page) => {
    setLoading(true);
    try {
      const res = await adminApi.listUsers({
        page: p,
        page_size: 20,
        keyword: keyword || undefined,
        status: statusFilter,
      });
      setUsers(res.data.data.list);
      setTotal(res.data.data.total);
    } catch (e: unknown) {
      message.error((e as Error).message);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchUsers(1);
    setPage(1);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [keyword, statusFilter]);

  const handleBan = async () => {
    const values = await banForm.validateFields();
    if (!banTarget) return;
    setBanLoading(true);
    try {
      await adminApi.banUser(banTarget.id, {
        reason: values.reason,
        ban_until: values.ban_until
          ? dayjs(values.ban_until).toISOString()
          : undefined,
      });
      message.success('封禁成功');
      setBanTarget(null);
      fetchUsers();
    } catch (e: unknown) {
      message.error((e as Error).message);
    } finally {
      setBanLoading(false);
    }
  };

  const handleResetPwd = async () => {
    const values = await pwdForm.validateFields();
    if (!pwdTarget) return;
    setPwdLoading(true);
    try {
      await adminApi.resetPassword(pwdTarget.id, values.new_password);
      message.success('密码已重置');
      setPwdTarget(null);
    } catch (e: unknown) {
      message.error((e as Error).message);
    } finally {
      setPwdLoading(false);
    }
  };

  const columns: ColumnsType<AdminUser> = [
    { title: 'ID', dataIndex: 'id', width: 70 },
    {
      title: '用户名',
      dataIndex: 'username',
      render: (v: string, r: AdminUser) => (
        <Space direction="vertical" size={0}>
          <Text strong>{v}</Text>
          <Text type="secondary" style={{ fontSize: 12 }}>{r.nickname}</Text>
        </Space>
      ),
    },
    { title: '性别', dataIndex: 'gender', width: 70 },
    {
      title: '状态',
      dataIndex: 'status',
      width: 100,
      render: (s: string) => (
        <Tag color={STATUS_COLOR[s] ?? 'default'}>{STATUS_TEXT[s] ?? s}</Tag>
      ),
    },
    {
      title: '角色',
      dataIndex: 'role',
      width: 80,
      render: (r: string) =>
        r === 'admin' ? <Tag color="gold">管理员</Tag> : <Tag>用户</Tag>,
    },
    {
      title: '封禁原因',
      dataIndex: 'ban_reason',
      render: (v: string, r: AdminUser) =>
        v ? (
          <Space direction="vertical" size={0}>
            <Text type="danger">{v}</Text>
            {r.ban_until && (
              <Text type="secondary" style={{ fontSize: 11 }}>
                至 {dayjs(r.ban_until).format('YYYY-MM-DD')}
              </Text>
            )}
          </Space>
        ) : '—',
    },
    {
      title: '注册时间',
      dataIndex: 'created_at',
      render: (v: string) => dayjs(v).format('YYYY-MM-DD HH:mm'),
      width: 150,
    },
    {
      title: '操作',
      width: 240,
      render: (_: unknown, record: AdminUser) => (
        <Space size={4} wrap>
          {record.status === 'active' && (
            <Button
              size="small"
              danger
              icon={<StopOutlined />}
              onClick={() => { setBanTarget(record); banForm.resetFields(); }}
            >封禁</Button>
          )}
          {record.status === 'banned' && (
            <Popconfirm
              title="确认解封？"
              onConfirm={async () => {
                await adminApi.unbanUser(record.id);
                message.success('已解封');
                fetchUsers();
              }}
            >
              <Button size="small" icon={<CheckCircleOutlined />}>解封</Button>
            </Popconfirm>
          )}
          {record.status !== 'deleted' && (
            <Popconfirm
              title="确认注销该用户？历史数据保留。"
              onConfirm={async () => {
                await adminApi.deleteUser(record.id);
                message.success('已注销');
                fetchUsers();
              }}
            >
              <Button size="small" danger icon={<DeleteOutlined />}>注销</Button>
            </Popconfirm>
          )}
          {record.status === 'deleted' && (
            <Popconfirm
              title="确认恢复该账号？"
              onConfirm={async () => {
                await adminApi.restoreUser(record.id);
                message.success('已恢复');
                fetchUsers();
              }}
            >
              <Button size="small" icon={<RedoOutlined />}>恢复</Button>
            </Popconfirm>
          )}
          <Button
            size="small"
            icon={<KeyOutlined />}
            onClick={() => { setPwdTarget(record); pwdForm.resetFields(); }}
          >改密</Button>
          {record.status === 'active' && (
            <Popconfirm
              title="确认踢出该用户的所有在线连接？"
              onConfirm={async () => {
                await adminApi.kickUser(record.id);
                message.success('已踢出');
              }}
            >
              <Button size="small" icon={<LogoutOutlined />}>踢出</Button>
            </Popconfirm>
          )}
        </Space>
      ),
    },
  ];

  return (
    <Card title="用户管理">
      {/* 筛选栏 */}
      <Space style={{ marginBottom: 16 }} wrap>
        <Input
          placeholder="搜索用户名/昵称"
          prefix={<SearchOutlined />}
          allowClear
          style={{ width: 220 }}
          onPressEnter={(e) => setKeyword((e.target as HTMLInputElement).value)}
          onChange={(e) => !e.target.value && setKeyword('')}
        />
        <Select
          placeholder="状态筛选"
          allowClear
          style={{ width: 140 }}
          onChange={(v) => setStatusFilter(v)}
          options={[
            { label: '正常', value: 'active' },
            { label: '封禁', value: 'banned' },
            { label: '已注销', value: 'deleted' },
          ]}
        />
        <Button onClick={() => fetchUsers()} loading={loading}>刷新</Button>
      </Space>

      <Table
        rowKey="id"
        dataSource={users}
        columns={columns}
        loading={loading}
        size="small"
        pagination={{
          total,
          pageSize: 20,
          current: page,
          showTotal: (t) => `共 ${t} 条`,
          onChange: (p) => { setPage(p); fetchUsers(p); },
        }}
      />

      {/* 封禁弹窗 */}
      <Modal
        title={`封禁用户：${banTarget?.username}`}
        open={!!banTarget}
        onOk={handleBan}
        onCancel={() => setBanTarget(null)}
        confirmLoading={banLoading}
        okText="确认封禁"
        okButtonProps={{ danger: true }}
      >
        <Form form={banForm} layout="vertical">
          <Form.Item name="reason" label="封禁原因" rules={[{ required: true }]}>
            <Input placeholder="请输入封禁原因" />
          </Form.Item>
          <Form.Item name="ban_until" label="解封时间（留空=永久封禁）">
            <DatePicker style={{ width: '100%' }} showTime />
          </Form.Item>
        </Form>
      </Modal>

      {/* 重置密码弹窗 */}
      <Modal
        title={`重置密码：${pwdTarget?.username}`}
        open={!!pwdTarget}
        onOk={handleResetPwd}
        onCancel={() => setPwdTarget(null)}
        confirmLoading={pwdLoading}
        okText="确认重置"
      >
        <Form form={pwdForm} layout="vertical">
          <Form.Item
            name="new_password"
            label="新密码"
            rules={[
              { required: true },
              { min: 8, message: '至少 8 位' },
              { pattern: /^(?=.*[a-zA-Z])(?=.*\d).+$/, message: '须含字母和数字' },
            ]}
          >
            <Input.Password placeholder="新密码（至少8位，含字母和数字）" />
          </Form.Item>
        </Form>
      </Modal>
    </Card>
  );
}
