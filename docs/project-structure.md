# 项目目录结构说明

## 顶层结构

```
chatroom/
├── cmd/
│   └── server/
│       └── main.go              # 程序入口，初始化并启动服务
├── configs/
│   ├── config.yaml              # 开发环境配置
│   └── config.prod.yaml         # 生产环境配置（不提交到仓库）
├── docs/                        # 设计文档（本目录）
├── internal/                    # 核心业务代码（外部不可 import）
│   ├── bootstrap/               # 应用初始化（DB/Redis/MQ 连接）
│   ├── middleware/              # Gin 中间件
│   ├── module/                  # 业务模块（核心）
│   │   ├── auth/
│   │   ├── user/
│   │   ├── friend/
│   │   ├── chat/
│   │   ├── group/
│   │   ├── gateway/
│   │   └── notify/
│   ├── router/                  # 路由注册
│   └── server/                  # HTTP Server + WebSocket Server 启动
├── pkg/                         # 可复用的工具包
│   ├── config/                  # 配置加载
│   ├── db/                      # PostgreSQL 连接池封装
│   ├── redis/                   # Redis 客户端封装
│   ├── mq/                      # RocketMQ 生产者/消费者封装
│   ├── session/                 # Session 管理
│   ├── response/                # 统一响应结构
│   ├── logger/                  # 日志封装（zap）
│   ├── crypto/                  # 密码 bcrypt 工具
│   └── snowflake/               # 雪花 ID 生成器（后续替换 BIGSERIAL）
├── scripts/
│   ├── migrate.go               # 数据库迁移入口
│   └── sql/
│       └── init.sql             # 初始建表 SQL
├── docker-compose.yaml          # 本地开发基础设施
├── Dockerfile                   # 应用镜像构建
├── go.mod
├── go.sum
└── README.md
```

---

## 模块内部结构（以 `module/chat` 为例）

每个业务模块遵循统一的三层结构：

```
internal/module/chat/
├── handler.go       # HTTP Handler 或 WS 命令处理（输入校验、调用 service）
├── service.go       # 业务逻辑层（调用 repo、MQ、Redis）
├── repository.go    # 数据访问层（封装 SQL 和 Redis 操作）
├── model.go         # 本模块的 DTO / 请求响应结构体
└── consumer.go      # MQ 消费者（如有）
```

> **模块间通信规则**：
> - 禁止跨模块直接调用 `repository`，只能调用对方的 `service` 接口（通过 interface 解耦）
> - 需要异步处理或广播的场景，通过 RocketMQ 发消息而不是直接调用

---

## 各模块职责说明

### `module/auth`

- 注册：校验参数 → bcrypt 加密密码 → 写入 `users` 表
- 登录：查用户 → 校验密码 → 生成 session_id → 写入 Redis → 返回 Cookie
- 登出：删除 Redis Session
- 微信登录（预留）：OAuth 回调 → 查/创建用户（通过 openid）→ 返回 Session

### `module/user`

- 用户信息 CRUD
- 用户搜索（用户名/手机号/邮箱模糊搜索）
- AI 推荐接口（预留，读 `user_behavior_logs`）

### `module/friend`

- 好友申请的发起、接受、拒绝
- 好友列表查询
- 拉黑管理
- 好友申请接受时：创建单聊 `conversation` 记录

### `module/gateway`

- WebSocket 连接注册与管理（本地连接池 `map[uid]*websocket.Conn`）
- 连接建立时：验证 Session，写 Redis `user:{uid}:node` 和 `node:{node_id}:conns`
- 断开时：清理 Redis 中的路由信息
- 接收客户端消息，按 `cmd` 分发到对应的处理器
- 提供 `Push(uid, msg)` 方法供其他消费者调用

### `module/chat`

- 单聊/群聊消息发送（生产到 MQ）
- 消息持久化消费者（`MessagePersistConsumer`）
- 消息投递消费者（`MessageDeliverConsumer`）
- ACK 消费者（`MessageAckConsumer`）
- 消息历史分页查询（Redis 缓存 → PostgreSQL 兜底）
- 消息撤回

### `module/group`

- 群组 CRUD
- 群成员管理（邀请、踢出、角色修改、禁言）
- 群公告管理
- 群变更事件生产到 `chatroom_notify` Topic

### `module/notify`

- 消费 `chatroom_notify` Topic
- 判断目标用户是否在线，在线则调用 `gateway.Push` 推送
- 离线消息落库（预留通知中心）

---

## 配置文件结构（config.yaml）

```yaml
app:
  name: chatroom
  node_id: "node1"          # 多节点时各节点唯一 ID
  port: 8080

db:
  dsn: "host=localhost port=5432 user=chatroom password=xxx dbname=chatroom sslmode=disable"
  max_open_conns: 50
  max_idle_conns: 10

redis:
  addr: "localhost:6379"
  password: ""
  db: 0
  pool_size: 20

rocketmq:
  name_server: "localhost:9876"
  producer_group: "chatroom_producer"
  retry_times: 3

session:
  ttl: 604800                # 7天，单位秒

log:
  level: "info"
  output: "stdout"            # stdout | file

oss:
  endpoint: ""                # 对象存储地址（预留）
  bucket: ""
```

---

## Docker Compose 基础设施说明

`docker-compose.yaml` 启动以下服务：

| 服务 | 端口 | 说明 |
|------|------|------|
| `postgres` | 5432 | PostgreSQL 15，数据卷持久化 |
| `redis` | 6379 | Redis 7，数据卷持久化 |
| `rocketmq-namesrv` | 9876 | RocketMQ NameServer |
| `rocketmq-broker` | 10911 | RocketMQ Broker |
| `rocketmq-dashboard` | 8081 | RocketMQ 控制台（可选） |

应用本体通过 `go run` 启动，不放入 docker-compose，方便本地热重载开发。
