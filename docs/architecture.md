# 架构设计

## 1. 整体架构图

```
                        ┌─────────────────────────────────────┐
                        │           客户端（App / Web）          │
                        └────────────┬──────────────┬──────────┘
                                     │ HTTP          │ WebSocket
                                     ▼              ▼
                        ┌────────────────────────────────────────┐
                        │           负载均衡（Nginx / LB）          │
                        └──────┬─────────────────┬───────────────┘
                               │                 │
                    ┌──────────▼──────┐  ┌───────▼──────────┐
                    │  Chatroom Node1  │  │  Chatroom Node2  │  ← 可水平扩展
                    │  (Go + Gin)     │  │  (Go + Gin)      │
                    └──────┬──────────┘  └──────┬───────────┘
                           │                    │
                           └──────────┬─────────┘
                                      │
              ┌────────────┬──────────┼──────────┬─────────────┐
              │            │          │          │             │
      ┌───────▼──┐  ┌──────▼───┐  ┌──▼───────┐  │    ┌────────▼──────┐
      │ Redis     │  │PostgreSQL│  │RocketMQ  │  │    │  对象存储（OSS）  │
      │(Session/  │  │(持久化)  │  │(消息队列) │  │    │(图片/文件/音频)  │
      │ 连接路由/ │  │          │  │          │  │    └───────────────┘
      │ 消息缓存) │  │          │  │          │  │
      └───────────┘  └──────────┘  └──────────┘  │
                                                  └── 微信OAuth服务（后续）
```

## 2. 模块划分（Modular Monolith）

项目采用**模块化单体**架构：代码按业务域拆分为独立模块，每个模块有自己的 handler/service/repository 层，模块间通过接口调用或 MQ 解耦，未来可按需拆为独立微服务。

| 模块 | 职责 |
|------|------|
| `module/auth` | 注册、登录、登出，Session 管理，微信 OAuth 预留 |
| `module/user` | 用户信息 CRUD，头像上传，好友关系，联系人搜索 |
| `module/chat` | 单聊消息收发，离线消息投递，消息 ACK |
| `module/group` | 群组 CRUD，成员管理，群角色权限，群公告 |
| `module/gateway` | WebSocket 连接管理，心跳，消息路由 |
| `module/notify` | 系统通知，好友申请通知（消费 MQ 消息） |

## 3. WebSocket 连接路由（多节点方案）

多节点部署时，不同用户的 WebSocket 连接分布在不同节点上，需要跨节点消息投递：

```
发送方 ──▶ Node1(gateway) ──▶ 写入 RocketMQ topic:msg.deliver
                                        │
                                        ▼
                              Node1/Node2/NodeN 均消费该 Topic
                                        │
                              各节点查询 Redis: user:{uid}:node
                              判断目标用户是否连在本节点
                                        │
                              ┌─────────▼─────────┐
                              │ 在本节点 → 直接推送  │
                              │ 不在 → 忽略，由对应  │
                              │ 节点处理            │
                              └────────────────────┘
```

**Redis 连接路由键设计：**

```
user:{uid}:node     → "node1"           # 该用户连接在哪个节点
user:{uid}:session  → session_id        # 当前 Session
node:{node_id}:conns → Set{uid1,uid2}   # 某节点上的所有连接用户
```

## 4. Session 管理

- 登录成功后服务端生成 `session_id`（UUID），存入 Redis，TTL 默认 7 天
- 每次请求携带 `Cookie: session_id=xxx` 或 `Header: X-Session-Id: xxx`
- WebSocket 握手时同样验证 Session
- Redis Key: `session:{session_id}` → `{uid, username, login_at, ...}`

## 5. 水平扩展注意事项

| 组件 | 扩展策略 |
|------|---------|
| Chatroom 节点 | 无状态（Session 在 Redis），直接增加节点数即可 |
| Redis | 使用 Redis Cluster 或哨兵模式保证高可用 |
| PostgreSQL | 读写分离，消息表按会话 ID 分区（后期） |
| RocketMQ | 增加 Broker 节点，消费者组水平扩展 |

## 6. 消息流转全链路

```
客户端发送消息
     │
     ▼
WebSocket Gateway（接收，校验 Session）
     │
     ▼
写入 RocketMQ Topic: msg.persist（持久化）
     │
     ├──▶ Consumer: MessagePersistConsumer
     │         └── 写入 PostgreSQL messages 表
     │         └── 写入 Redis 近期消息缓存（ZSet，score=seq_id）
     │
     └──▶ 写入 RocketMQ Topic: msg.deliver（投递）
              │
              ▼
         Consumer: MessageDeliverConsumer（各节点）
              │
              ├── 目标在线 → 通过 WebSocket 推送，等待 ACK
              │
              └── 目标离线 → 更新 Redis 未读数，等待下次登录拉取
```

## 7. AI 联系人推荐预留设计

- `users` 表保留足够的用户属性字段（地区、标签等）
- 消息行为数据（与谁聊天频率、群共同好友数）写入 `user_behavior_log` 表
- 后续 AI 服务可直接读取该表离线计算推荐，或通过 RocketMQ 消费行为事件实时计算
- API 预留 `GET /api/v1/users/recommend` 接口，初期返回空列表
