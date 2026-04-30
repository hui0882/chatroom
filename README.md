# Chatroom

基于 Go 实现的高可用即时通讯系统，支持单聊、群聊、WebSocket 实时通信，并具备水平扩展能力。

## 技术栈

| 层次 | 技术选型 |
|------|---------|
| Web 框架 | Gin |
| 实时通信 | WebSocket（gorilla/websocket） |
| 数据库 | PostgreSQL |
| 缓存 / 会话 / 跨节点路由 | Redis |
| 消息队列 | RocketMQ |
| 认证方式 | Session（存储于 Redis） |
| 部署方式 | Docker Compose |

## 文档索引

| 文档 | 说明 |
|------|------|
| [docs/architecture.md](docs/architecture.md) | 整体架构设计、模块划分、水平扩展方案 |
| [docs/database.md](docs/database.md) | PostgreSQL 数据库表结构设计 |
| [docs/api.md](docs/api.md) | RESTful HTTP API 接口设计 |
| [docs/websocket-protocol.md](docs/websocket-protocol.md) | WebSocket 消息协议设计 |
| [docs/project-structure.md](docs/project-structure.md) | 项目目录结构说明 |
| [docs/mq-topics.md](docs/mq-topics.md) | RocketMQ Topic 与消息流转设计 |
| [PLAN.md](PLAN.md) | 项目开发规划与进度跟踪 |

## 快速开始

### 1. 环境准备

确保已安装以下组件：
- Go 1.24+
- PostgreSQL 15+
- Redis 7+

### 2. 配置文件

在项目根目录创建 `config.json` 文件，配置格式如下：

```json
{
  "app": {
    "name": "chatroom",
    "node_id": "node-1",
    "port": 8080,
    "debug": true
  },
  "db": {
    "host": "localhost",
    "port": 5432,
    "user": "postgres",
    "password": "your_password_here",
    "dbname": "chatroom",
    "sslmode": "disable",
    "max_open_conns": 100,
    "max_idle_conns": 10
  },
  "redis": {
    "addr": "localhost:6379",
    "password": "your_redis_password",
    "db": 0,
    "pool_size": 100
  },
  "rocketmq": {
    "name_server": "localhost:9876",
    "producer_group": "chatroom_producer",
    "retry_times": 3
  },
  "session": {
    "ttl": 604800
  },
  "log": {
    "level": "debug",
    "output": "stdout",
    "dir": "logs"
  },
  "oss": {
    "endpoint": "",
    "bucket": "",
    "access_key": "",
    "secret_key": ""
  }
}
```

### 3. 数据库初始化

```bash
# 连接到 PostgreSQL
psql -U postgres -d chatroom

# 执行初始化脚本
\i scripts/sql/001_init_users.sql
\i scripts/sql/002_friends.sql
\i scripts/sql/003_messages.sql
```

### 4. 启动服务

```bash
# 编译并运行
go build -o /tmp/chatroom ./cmd/server/main.go && /tmp/chatroom

# 或者直接运行
go run cmd/server/main.go
```

## 配置说明

### PostgreSQL 配置

| 配置项 | 说明 | 默认值 |
|--------|------|--------|
| `db.host` | PostgreSQL 服务器地址 | `localhost` |
| `db.port` | PostgreSQL 端口 | `5432` |
| `db.user` | 数据库用户名 | `postgres` |
| `db.password` | 数据库密码 | （必填） |
| `db.dbname` | 数据库名称 | `chatroom` |
| `db.sslmode` | SSL 模式 | `disable` |
| `db.max_open_conns` | 最大打开连接数 | `100` |
| `db.max_idle_conns` | 最大空闲连接数 | `10` |

### Redis 配置

| 配置项 | 说明 | 默认值 |
|--------|------|--------|
| `redis.addr` | Redis 地址（host:port） | `localhost:6379` |
| `redis.password` | Redis 密码 | （必填） |
| `redis.db` | Redis 数据库编号 | `0` |
| `redis.pool_size` | 连接池大小 | `100` |

### 应用配置

| 配置项 | 说明 | 默认值 |
|--------|------|--------|
| `app.name` | 应用名称 | `chatroom` |
| `app.node_id` | 节点 ID（多节点部署时唯一） | `node-1` |
| `app.port` | HTTP 服务端口 | `8080` |
| `app.debug` | 调试模式（记录请求日志） | `true` |
| `session.ttl` | Session 过期时间（秒） | `604800`（7天） |
| `log.level` | 日志级别（debug/info/warn/error） | `info` |
| `log.output` | 日志输出方式（stdout/file） | `stdout` |
| `log.dir` | 日志文件目录（output=file 时生效） | `logs` |

## 功能概览

### 已完成功能

- **账号体系**：用户名注册登录，Session 认证，管理员权限
- **用户管理**：封禁/解封、软删除/恢复、密码重置、踢出在线用户
- **好友系统**：好友申请/接受/拒绝、好友列表、删除好友、用户搜索
- **消息系统**：私聊消息发送、历史记录查询、未读消息管理
- **WebSocket**：实时双向通信、多设备支持、流式输出测试

### 待开发功能

- **群聊**：群组管理、成员权限、群公告
- **消息增强**：离线消息、消息 ACK、MQ 解耦
- **文件上传**：图片/文件/音频上传（OSS 集成）
- **微信登录**：微信扫码登录集成

## API 接口

### 认证接口

```
POST   /api/v1/auth/register         用户注册
POST   /api/v1/auth/login            用户登录
POST   /api/v1/auth/logout           用户登出（需登录）
```

### 用户接口

```
GET    /api/v1/user/me               获取当前用户信息（需登录）
PUT    /api/v1/user/password         修改密码（需登录）
GET    /api/v1/users/search          搜索用户（需登录）
```

### 好友接口

```
GET    /api/v1/friends               获取好友列表（需登录）
DELETE /api/v1/friends/:id           删除好友（需登录）
POST   /api/v1/friends/requests      发送好友申请（需登录）
GET    /api/v1/friends/requests/received  获取收到的申请（需登录）
GET    /api/v1/friends/requests/sent      获取发出的申请（需登录）
POST   /api/v1/friends/requests/:id/cancel  撤回申请（需登录）
POST   /api/v1/friends/requests/:id/accept  接受申请（需登录）
POST   /api/v1/friends/requests/:id/reject  拒绝申请（需登录）
```

### 消息接口

```
GET    /api/v1/messages/unread       获取所有未读数（需登录）
POST   /api/v1/messages/unread/:peer_uid/clear  清零未读（需登录）
GET    /api/v1/messages/:peer_uid    获取聊天历史（需登录）
```

### WebSocket 接口

```
GET    /ws?session_id=<sid>&device=<type>  WebSocket 正式连接（需 Session）
GET    /websocket_test                     WebSocket 测试（无需认证）
GET    /websocket_stream                   WebSocket 流式测试（无需认证）
```

### 管理员接口

```
GET    /api/v1/admin/users           获取用户列表（需管理员）
POST   /api/v1/admin/users/:id/ban   封禁用户（需管理员）
POST   /api/v1/admin/users/:id/unban 解封用户（需管理员）
DELETE /api/v1/admin/users/:id       删除用户（需管理员）
POST   /api/v1/admin/users/:id/restore      恢复用户（需管理员）
POST   /api/v1/admin/users/:id/reset-password  重置密码（需管理员）
POST   /api/v1/admin/users/:id/kick         踢出用户（需管理员）
```

## 项目结构

```
chatroom/
├── cmd/server/           # 程序入口
├── internal/             # 业务模块
│   ├── bootstrap/        # 启动引导
│   ├── friend/           # 好友模块
│   ├── message/          # 消息模块
│   ├── middleware/        # 中间件
│   ├── router/           # 路由注册
│   ├── server/           # HTTP 服务
│   ├── session/          # Session 管理
│   ├── user/             # 用户模块
│   └── ws/               # WebSocket 模块
├── pkg/                  # 公共包
│   ├── config/           # 配置加载
│   ├── db/               # PostgreSQL 连接
│   ├── logger/           # 日志系统
│   ├── redis/            # Redis 连接
│   └── response/         # 统一响应
├── docs/                 # 设计文档
├── scripts/sql/          # 数据库脚本
└── frontend/             # 前端代码（Vue 3）
```

## 开发指南

### 分支策略

```
main          生产就绪代码，只接受 PR 合入
dev           日常开发分支，各 Phase 功能在此集成
feat/<name>   功能分支，从 dev 切出，完成后 PR 到 dev
fix/<name>    缺陷修复分支
```

### Commit 规范

```
feat:     新功能
fix:      缺陷修复
refactor: 重构（不改变外部行为）
test:     测试相关
docs:     文档变更
chore:    构建/依赖/配置等杂项
```

## 许可证

本项目采用 MIT 许可证。
