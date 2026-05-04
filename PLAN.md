# 项目开发规划

> **项目**：Chatroom —— 基于 Go 的高可用即时通讯系统  
> **仓库**：github.com/hui0882/chatroom  
> **文档版本**：v0.4.0  
> **最后更新**：2026-04-30  
> **负责人**：hanxiaoxiao

---

## 目录

1. [开发进度总览](#开发进度总览)
2. [项目概述](#1-项目概述)
3. [整体里程碑规划](#2-整体里程碑规划)
4. [当前进度（Phase 1-2 完成 + Phase 3 进行中）](#3-当前进度phase-1-2-完成--phase-3-进行中)
5. [待开发内容](#4-待开发内容)
   - [Phase 2 — 认证与用户（已完成）](#phase-2--认证与用户已完成)
   - [Phase 3 — 消息核心](#phase-3--消息核心)
   - [Phase 4 — 群组](#phase-4--群组)
   - [Phase 5 — 工程质量](#phase-5--工程质量)
   - [Phase 6 — 扩展能力](#phase-6--扩展能力)
6. [技术债与已知风险](#5-技术债与已知风险)
7. [依赖组件状态](#6-依赖组件状态)
8. [开发约定](#7-开发约定)

---

## 开发进度总览

**最后更新**：2026-04-30

### 整体进度

```
Phase 1  基础骨架      ████████████  完成  ✅
Phase 2  认证与用户    ████████████  完成  ✅
Phase 3  消息核心      ████████░░░░  进行中 🔄 (40%)
Phase 4  群组          ░░░░░░░░░░░░  待开发
Phase 5  工程质量      ░░░░░░░░░░░░  待开发
Phase 6  扩展能力      ░░░░░░░░░░░░  待开发
```

**总体完成度**：约 45%（Phase 1-2 完成，Phase 3 进行中）

### 已完成功能清单

#### Phase 1 - 基础骨架 ✅
- [x] 工程目录结构搭建
- [x] 配置加载系统（JSON 配置文件）
- [x] 日志系统（zap，支持文件轮替）
- [x] PostgreSQL 连接池初始化
- [x] Redis 客户端初始化
- [x] HTTP 服务器（Gin，优雅启停）
- [x] 统一响应格式和错误码
- [x] 健康检查接口（`/health`）
- [x] 请求日志中间件
- [x] Panic Recovery 中间件
- [x] 设计文档完成（6 份）

#### Phase 2 - 认证与用户 ✅
- [x] WebSocket 框架（Hub/Client 模型，多设备支持）
- [x] WebSocket 测试接口（字节倒序、流式输出）
- [x] Session 管理（Redis Hash，7 天 TTL）
- [x] 认证中间件（Cookie/Header 读取）
- [x] 管理员权限中间件
- [x] 用户注册（密码规则校验 + bcrypt 加密）
- [x] 用户登录（密码校验 + 封禁检测 + Session 签发）
- [x] 用户登出
- [x] 密码修改
- [x] 管理员功能（封禁/解封/软删除/恢复/重置密码/踢出）
- [x] 好友模块（申请/接受/拒绝/列表/删除/搜索）
- [x] 消息模块（私聊发送/历史查询/未读管理）
- [x] 数据库表设计（users, friends, friend_requests, messages）

#### Phase 3 - 消息核心 🔄 (40%)
- [x] 私聊消息存储（PostgreSQL）
- [x] 未读消息计数（Redis Hash）
- [x] 消息历史查询（游标分页）
- [ ] WebSocket 消息分发（dispatcher）
- [ ] RocketMQ 集成
- [ ] 消息缓存（Redis ZSet）
- [ ] 离线消息推送
- [ ] 消息 ACK 机制
- [ ] 消息撤回

### 技术栈使用情况

| 组件 | 状态 | 说明 |
|------|------|------|
| Go 1.24 | ✅ 已使用 | 主要开发语言 |
| Gin v1.10 | ✅ 已使用 | HTTP 框架 |
| PostgreSQL 15 | ✅ 已使用 | 主数据库，存储用户、好友、消息 |
| Redis 7 | ✅ 已使用 | Session 存储、未读计数 |
| gorilla/websocket | ✅ 已使用 | WebSocket 通信 |
| zap | ✅ 已使用 | 结构化日志 |
| bcrypt | ✅ 已使用 | 密码哈希 |
| RocketMQ | 🔲 待引入 | 消息队列（Phase 3） |

### API 接口统计

- **已实现接口**：22 个
- **认证相关**：3 个（注册、登录、登出）
- **用户相关**：3 个（个人信息、修改密码、搜索）
- **好友相关**：8 个（申请、接受、拒绝、列表、删除等）
- **消息相关**：3 个（历史、未读、清零）
- **管理员相关**：7 个（用户管理）
- **WebSocket**：3 个（正式连接、测试、流式测试）
- **健康检查**：1 个

### 数据库表统计

- **users**：用户表（10 个字段，4 个索引）
- **friends**：好友关系表（双向存储）
- **friend_requests**：好友申请表（含状态管理）
- **messages**：消息表（支持私聊）

### 下一步开发重点

1. **Phase 3 完成**（预计 2-3 周）
   - WebSocket 消息分发器
   - RocketMQ 集成
   - 消息缓存和离线推送

2. **Phase 4 群组功能**（预计 2 周）
   - 群组 CRUD
   - 成员管理
   - 群消息

3. **Phase 5 工程质量**（预计 1-2 周）
   - 单元测试
   - Docker Compose 部署
   - 数据库迁移工具

---

## 1. 项目概述

Chatroom 是一个面向生产环境的即时通讯后端系统，采用 **模块化单体（Modular Monolith）** 架构，后续可按需水平拆分为独立微服务。核心特性：

- WebSocket 实时双向通信，支持多节点水平扩展
- 单聊 / 群聊，含离线消息、消息 ACK、历史分页
- 账号体系（注册/登录/Session），预留微信扫码登录
- 以 Redis 为中心化连接路由，RocketMQ 解耦各模块
- PostgreSQL 持久化，Redis 缓存近期消息
- 预留 AI 联系人推荐扩展点

**技术栈**：Go 1.24 · Gin v1.10 · PostgreSQL 15 · Redis 7 · RocketMQ · zap · gorilla/websocket

---

## 2. 整体里程碑规划

```
Phase 1  基础骨架      ████████████  完成  ✅
Phase 2  认证与用户    ████████████  完成  ✅
Phase 3  消息核心      ████████░░░░  进行中 🔄
Phase 4  群组          ░░░░░░░░░░░░  待开发
Phase 5  工程质量      ░░░░░░░░░░░░  待开发
Phase 6  扩展能力      ░░░░░░░░░░░░  待开发
```

| 阶段 | 主要交付物 | 状态 |
|------|-----------|------|
| Phase 1 | 工程骨架、基础设施连接、日志、/health 接口 | ✅ 已完成 |
| Phase 2 | WebSocket 框架、注册登录、Session 中间件、用户管理、管理员操作、好友模块、消息基础 | ✅ 已完成 |
| Phase 3 | 单聊/群聊消息、离线消息、ACK、MQ 接入 | 🔄 进行中（约 40%）|
| Phase 4 | 群组管理、成员权限、群公告 | 🔲 未开始 |
| Phase 5 | 单元测试、集成测试、数据库迁移脚本、Docker Compose | 🔲 未开始 |
| Phase 6 | 微信登录、文件上传（OSS）、AI 推荐接口 | 🔲 未开始 |

---

## 3. 当前进度（Phase 1-2 完成 + Phase 3 进行中）

### 3.1 Phase 1 已完成内容

#### 设计文档（`docs/`）
| 文档 | 说明 |
|------|------|
| `docs/architecture.md` | 整体架构图、多节点路由方案、消息全链路、AI 预留设计 |
| `docs/database.md` | 8 张 PostgreSQL 表 DDL + Redis Key 设计 |
| `docs/api.md` | RESTful 接口定义（持续同步，当前 v0.3.0）|
| `docs/websocket-protocol.md` | WebSocket 帧格式、全部 cmd 命令定义 |
| `docs/mq-topics.md` | 5 个 RocketMQ Topic 生产/消费设计 |
| `docs/project-structure.md` | 目录结构规范、模块职责 |

#### 工程骨架（代码）

| 模块 | 文件 | 说明 |
|------|------|------|
| 配置加载 | `pkg/config/config.go` | JSON 配置解析，含 debug 开关 |
| 日志系统 | `pkg/logger/logger.go` | zap 封装，日志按日期 + 启动序号轮替，调试模式记录请求 Body |
| 统一响应 | `pkg/response/response.go` | 统一 JSON 响应结构与错误码定义 |
| PostgreSQL | `pkg/db/db.go` | 连接池初始化，Ping 检测 |
| Redis | `pkg/redis/redis.go` | 客户端初始化，Ping 检测 |
| 启动引导 | `internal/bootstrap/bootstrap.go` | 顺序初始化所有基础设施，统一 Close |
| HTTP 中间件 | `internal/middleware/logger.go` | 请求日志（debug 模式记录 Body）+ panic Recovery |
| HTTP Server | `internal/server/http.go` | 优雅启停（10s 超时） |
| 应用上下文 | `internal/server/app.go` | AppContext 聚合所有 Handler |
| 健康检查 | `internal/server/health.go` | `GET /health`，含 PG/Redis 探活 |
| 路由注册 | `internal/router/router.go` | 全局中间件挂载，路由分组 |
| 程序入口 | `cmd/server/main.go` | flag 解析、信号监听、优雅退出 |

---

### 3.2 Phase 2 已完成内容

#### WebSocket 框架（`internal/ws/`）

| 文件 | 说明 |
|------|------|
| `internal/ws/hub.go` | Hub：`map[int64]map[string]*Client`（uid→device→Client），Register/Unregister/Inbound channel，Run() 事件循环，SendToUser，KickUser |
| `internal/ws/client.go` | Client：ReadPump / WritePump goroutine 对，ping/pong 心跳（pongWait=60s，pingPeriod=54s），safeClose 保证连接关闭幂等 |
| `internal/ws/handler.go` | SessionValidator 类型，Handler.Serve() 负责升级连接，BuildSessionValidator() 优先取 URL query → Header → Cookie |
| `internal/ws/test_echo.go` | `GET /websocket_test` 无需认证，按 Unicode 码点（rune）倒序后返回，支持中文 ✅ |
| `internal/ws/stream_echo.go` | `GET /websocket_stream` 无需认证，模拟 AI 流式输出：倒序文本后以 20 字/秒逐字符推送 JSON chunk 帧，结束后发 done 帧 ✅ |

多设备策略：同 uid 不同 device 可并存；同 uid + 同 device 重连时关闭旧连接 Send 通道，触发旧连接 WritePump 优雅退出。

#### Session 管理（`internal/session/`）

| 文件 | 说明 |
|------|------|
| `internal/session/manager.go` | Manager：Redis Hash 存储（key=`session:{uuid}`，fields=uid/role），TTL=7天；Create / Get / Delete / DeleteAll（SCAN 全量清除指定用户所有 Session）|

#### 认证中间件（`internal/middleware/`）

| 文件 | 说明 |
|------|------|
| `internal/middleware/auth.go` | Auth(sm)：优先读 Cookie `sessionId`，后取 Header `X-Session-Id`，验证后注入 `session.Info` 到 Context；AdminOnly()：校验 role=="admin" |

#### 用户模块（`internal/user/`）

| 文件 | 说明 |
|------|------|
| `internal/user/model.go` | User 结构体（含 IsBanned/IsDeleted 方法），Repository 接口，ListFilter，哨兵错误 ErrNotFound / ErrUsernameExists |
| `internal/user/repository.go` | pgRepository：Create（RETURNING id+时间戳，唯一约束 → ErrUsernameExists），FindByUsername，FindByID，List（动态 WHERE + 分页），Ban，Unban，SoftDelete，Restore，UpdatePassword |
| `internal/user/service.go` | 注册（密码规则校验 + bcrypt 加密），登录（密码校验 + 封禁检测 + Session 签发），登出，改密，BanUser / UnbanUser / DeleteUser / RestoreUser / ResetPassword / KickUser（均含 session.DeleteAll + hub.KickUser） |
| `internal/user/handler.go` | 全部 HTTP Handler，Login 同时写 7 天 HttpOnly Cookie，toPublicUser / toAdminUser 视图转换 |

#### 好友模块（`internal/friend/`）

| 文件 | 说明 |
|------|------|
| `internal/friend/model.go` | Friend / Request 结构体，Repository 接口，哨兵错误定义 |
| `internal/friend/repository.go` | pgRepository：CreateRequest / FindRequest / UpdateRequestStatus / AddFriends / RemoveFriend / ListFriends / IsFriend |
| `internal/friend/service.go` | 好友申请（含互发自动接受）、撤回、接受、拒绝、列表、删除、搜索用户 |
| `internal/friend/handler.go` | HTTP Handler，处理好友相关请求 |

#### 消息模块（`internal/message/`）

| 文件 | 说明 |
|------|------|
| `internal/message/model.go` | Message 结构体，Repository 接口，UnreadStore 接口，消息类型常量 |
| `internal/message/repository.go` | pgRepository：Save / ListHistory（游标分页） |
| `internal/message/redis_unread.go` | RedisUnreadStore：未读计数（Redis Hash，key=`unread:{uid}`） |
| `internal/message/service.go` | 发送消息（校验好友关系）、历史查询（含清零未读）、未读管理 |
| `internal/message/handler.go` | HTTP Handler：GET /messages/:peer_uid（历史）、GET /messages/unread（未读）、POST /messages/unread/:peer_uid/clear |

#### 数据库

| 文件 | 说明 |
|------|------|
| `scripts/sql/001_init_users.sql` | users 表 DDL，含 CHECK 约束（gender/role/status/age），4 个索引（username/status/openid/created_at），已在 chatroom 库执行 ✅ |
| `scripts/sql/002_friends.sql` | friends + friend_requests 表 DDL，好友关系与申请记录 ✅ |
| `scripts/sql/003_messages.sql` | messages 表 DDL，私聊消息存储 ✅ |

#### 路由注册（`internal/router/router.go`）

已注册路由（✅ 可用）：

```
GET  /health                              健康检查
GET  /websocket_test                      WS 测试（无需认证，Unicode rune 倒序回显，支持中文）
GET  /websocket_stream                    WS 流式测试（无需认证，模拟 AI 流式输出，20 字/秒）
GET  /ws?session_id=<sid>&device=<type>  WS 正式连接（需 Session）

POST   /api/v1/auth/register
POST   /api/v1/auth/login
POST   /api/v1/auth/logout               需登录

GET    /api/v1/user/me                   需登录
PUT    /api/v1/user/password             需登录

GET    /api/v1/users/search?keyword=     需登录，按用户名搜索

GET    /api/v1/friends                   需登录，获取好友列表
DELETE /api/v1/friends/:id               需登录，删除好友
POST   /api/v1/friends/requests          需登录，发送好友申请
GET    /api/v1/friends/requests/received 需登录，获取收到的申请
GET    /api/v1/friends/requests/sent     需登录，获取发出的申请
POST   /api/v1/friends/requests/:id/cancel   需登录，撤回申请
POST   /api/v1/friends/requests/:id/accept   需登录，接受申请
POST   /api/v1/friends/requests/:id/reject   需登录，拒绝申请

GET    /api/v1/messages/unread           需登录，获取所有未读数
POST   /api/v1/messages/unread/:peer_uid/clear  需登录，清零未读
GET    /api/v1/messages/:peer_uid        需登录，获取聊天历史

GET    /api/v1/admin/users               需管理员
POST   /api/v1/admin/users/:id/ban       需管理员
POST   /api/v1/admin/users/:id/unban     需管理员
DELETE /api/v1/admin/users/:id           需管理员
POST   /api/v1/admin/users/:id/restore   需管理员
POST   /api/v1/admin/users/:id/reset-password  需管理员
POST   /api/v1/admin/users/:id/kick      需管理员
```

#### 可运行状态

```bash
go build -o /tmp/chatroom ./cmd/server/main.go && /tmp/chatroom
```

全部已实现接口均通过手动 curl/wscat 测试验证：注册、登录（返回 Cookie + session_id）、鉴权中间件、管理员封禁（含原因+截止时间）、踢出在线用户（WS 连接即时关闭）、/websocket_test 字节倒序回显。

---

## 4. 待开发内容

### Phase 2 — 认证与用户（已完成）

> Phase 2 已完成认证、Session、用户管理、好友模块、消息基础功能。

- [x] `GET /api/v1/users/search?keyword=` — 按用户名搜索 ✅
- [x] 好友模块 `internal/friend/` — 好友申请/同意/拒绝、列表、删除好友 ✅
- [x] 消息模块基础 `internal/message/` — 私聊消息发送、历史查询、未读管理 ✅
- [ ] `PUT /api/v1/user/profile` — 更新个人信息（nickname / avatar_url / bio / gender / age）
- [ ] `GET /api/v1/users/:id` — 获取指定用户公开信息
- [ ] 会话模块基础 `internal/conversation/` — 创建单聊会话（配合好友模块）

---

### Phase 3 — 消息核心

> **目标**：用户可以通过 WebSocket 进行实时单聊与群聊，支持离线消息、可靠 ACK、历史分页。

#### 3.1 WebSocket 消息分发（复用已有 ws 框架）
- [ ] `internal/ws/dispatcher.go` — cmd 分发路由表（`msg.send`、`msg.ack`、`msg.revoke`、`ping`）
- [ ] Hub 消息处理回调（MessageHandler interface）实现

#### 3.2 RocketMQ 接入
- [ ] `go get github.com/apache/rocketmq-client-go/v2`
- [ ] `pkg/mq/producer.go` — 封装同步发送，支持 Topic 路由
- [ ] `pkg/mq/consumer.go` — 封装 Push Consumer，支持注册消息处理函数
- [ ] `internal/bootstrap/bootstrap.go` — 追加 MQ Producer 初始化

#### 3.3 消息模块 `internal/chat/`（增强）
- [ ] `model.go` — 会话 DTO、群聊消息 DTO
- [ ] `repository.go` — Redis ZSet 缓存读写，seq_id INCR
- [ ] `service.go` — 消息发送增强（投 MQ）、历史拉取（Redis 优先 → PG 兜底）、撤回
- [ ] `handler.go` — `GET /api/v1/conversations`、`GET /api/v1/conversations/:conv_id/messages`、`POST /api/v1/conversations`
- [ ] `persist_consumer.go` — 消费 `chatroom_msg_persist`：写库 + 缓存 + 投 deliver
- [ ] `deliver_consumer.go` — 消费 `chatroom_msg_deliver`：在线推送 / 离线计未读
- [ ] `ack_consumer.go` — 消费 `chatroom_msg_ack`：更新已读 seq_id + 清未读数

---

### Phase 4 — 群组

> **目标**：支持群组的全生命周期管理，含角色权限、禁言、公告。

#### 4.1 群组模块 `internal/group/`
- [ ] `model.go` — 群组/成员 DTO
- [ ] `repository.go` — 群组 CRUD，成员关系增删查改
- [ ] `service.go`
  - 创建群（自动将创建者设为群主，初始成员批量插入）
  - 邀请成员（检查群容量上限 `max_members`）
  - 踢出/退群（群主不可被踢，踢出后发 notify）
  - 修改角色（仅群主可提升/降级管理员）
  - 禁言（写入 `muted_until`，消息发送时检测）
  - 解散群（软删除 + 清理 Redis 相关 Key + 通知所有成员）
- [ ] `handler.go` — 覆盖 `docs/api.md` 第 7 节所有接口

#### 4.2 通知模块 `internal/notify/`
- [ ] `consumer.go` — 消费 `chatroom_notify`，在线推送 / 离线落库

---

### Phase 5 — 工程质量

> **目标**：覆盖测试、完善部署方案、建立 CI 基础。

#### 5.1 单元测试
- [ ] `pkg/logger` — 日志轮替逻辑测试（模拟跨天、多次启动）
- [ ] `internal/user/service_test.go` — 注册/登录业务规则测试（mock repository）
- [ ] `internal/session/manager_test.go` — Session 生成与 TTL 测试（mock Redis）
- [ ] `internal/chat/repository_test.go` — seq_id INCR 幂等性测试

#### 5.2 集成测试
- [ ] `tests/integration/` — 依赖真实 PG/Redis 的端到端接口测试
- [ ] 测试数据库隔离方案（每个 test case 使用独立 schema 或事务回滚）

#### 5.3 数据库迁移工具
- [ ] 引入 `golang-migrate/migrate` 或自研版本追踪
- [ ] 现有 `scripts/sql/001_init_users.sql` 纳入迁移版本管理

#### 5.4 容器化与部署
- [ ] `Dockerfile` — 多阶段构建（builder + distroless），最终镜像 < 30 MB
- [ ] `docker-compose.yaml` — 包含 PostgreSQL / Redis / RocketMQ / RocketMQ-Dashboard
- [ ] `docker-compose.override.yaml` — 本地开发专用，挂载代码目录支持热重载

#### 5.5 配置模板
- [ ] `config.example.json` — 去除真实密码的配置模板，纳入版本管理

---

### Phase 6 — 扩展能力

> **目标**：接入三方能力，完善生态。

#### 6.1 微信扫码登录
- [ ] `internal/user/wechat.go` — 获取 QR Code ticket，处理 OAuth 回调
- [ ] 用户表 `openid` 字段绑定逻辑（新用户自动注册，已有账号则合并）

#### 6.2 文件上传（OSS）
- [ ] `pkg/oss/oss.go` — 对象存储客户端封装（阿里云 OSS / 腾讯 COS 可插拔）
- [ ] `internal/upload/handler.go` — `POST /api/v1/upload`，支持图片/文件/音频
- [ ] 文件类型校验（MIME 白名单）、大小限制（可配置）

#### 6.3 AI 联系人推荐（预留接入点）
- [ ] `pkg/mq` — 在消息发送/好友查看路径上埋点，投递 `chatroom_behavior` 事件
- [ ] `internal/user/recommend.go` — `GET /api/v1/users/recommend`，初期读 `user_behavior_logs` 返回简单频率排序结果
- [ ] 预留外部 AI 服务调用接口（HTTP Client 封装，地址写入 `config.json`）

---

## 5. 技术债与已知风险

| 编号 | 类型 | 描述 | 优先级 | 计划解决阶段 |
|------|------|------|--------|-------------|
| TD-01 | 技术债 | 主键目前使用 PostgreSQL `BIGSERIAL`，分布式场景下需替换为雪花 ID，否则后期多写分片困难 | 中 | Phase 3 前 |
| TD-02 | 技术债 | `logs/` 目录已纳入 git 追踪（`.gitignore` 未排除），需补充排除规则 | 低 | ✅ 已解决 |
| TD-03 | 风险 | RocketMQ 尚未在本机安装，Phase 3 接入 MQ 时需提前准备环境或通过 Docker Compose 启动 | 高 | Phase 3 前 |
| TD-04 | 技术债 | 当前无数据库迁移版本管理，裸 SQL 直接执行，多人协作时存在数据库状态不一致风险 | 高 | Phase 5 |
| TD-05 | 风险 | Redis 密码仅通过 `CONFIG SET` 设置，重启后若 `redis.conf` 写入失败会导致无密码访问；需验证重启后仍生效 | 中 | 立即 |
| TD-06 | 技术债 | WebSocket 当前无消息协议分发（dispatcher），收到的消息由 Hub.Inbound 接收但无处理逻辑；Phase 3 接入聊天时需实现 MessageHandler | 中 | Phase 3 |
| TD-07 | 风险 | WebSocket 跨节点路由依赖 Redis `user:{uid}:node` Key，节点异常退出时该 Key 不会自动清理，导致消息投递到已离线节点 | 高 | Phase 3 |
| TD-08 | 技术债 | 当前无 API 版本协商机制，客户端升级时若接口破坏性变更无法平滑过渡 | 低 | Phase 5 |
| TD-09 | 风险 | 配置文件 `config.json` 含明文密码，生产环境应改用环境变量注入或密钥管理服务（如 Vault） | 高 | Phase 5 |
| TD-10 | 技术债 | Session DeleteAll() 用 Redis SCAN 全量遍历所有 `session:*` key 来找指定 uid 的 Session，用户量大时性能较差；后续可用 `uid:{uid}:sessions` Set 维护正向索引 | 低 | Phase 3 |
| TD-11 | 技术债 | 代码尚未提交到 GitHub（git push 受阻）；SSH key 已存在（`~/.ssh/id_ed25519.pub`），需完成 SSH 鉴权配置或改用 HTTPS credential helper | 中 | **待处理** |

---

## 6. 依赖组件状态

| 组件 | 版本 | 安装方式 | 运行状态 | 备注 |
|------|------|---------|---------|------|
| Go | 1.24.11 | 系统预装 | ✅ 正常 | — |
| PostgreSQL | 15.16 | dnf 安装 | ✅ running | systemd 开机自启，密码认证已配置，users 表已创建 |
| Redis | 7.2.7 | 系统预装 | ✅ running | requirepass 已配置（运行态 + 配置文件） |
| RocketMQ | — | **未安装** | ❌ 未运行 | Phase 3 前需安装，建议 Docker Compose 方式 |
| Nginx / LB | — | **未安装** | ❌ 未运行 | Phase 5 部署时引入 |

### Go 直接依赖

| 包 | 版本 | 用途 | 状态 |
|----|------|------|------|
| `github.com/gin-gonic/gin` | v1.10.0 | HTTP 框架 | ✅ 已引入 |
| `go.uber.org/zap` | v1.27.1 | 结构化日志 | ✅ 已引入 |
| `github.com/lib/pq` | v1.12.3 | PostgreSQL 驱动 | ✅ 已引入 |
| `github.com/redis/go-redis/v9` | v9.18.0 | Redis 客户端 | ✅ 已引入 |
| `github.com/gorilla/websocket` | v1.5.3 | WebSocket | ✅ 已引入（Phase 2）|
| `github.com/google/uuid` | v1.6.0 | Session ID 生成 | ✅ 已引入（Phase 2）|
| `golang.org/x/crypto` | v0.37.0 | bcrypt 密码哈希 | ✅ 已引入（Phase 2）|
| `apache/rocketmq-client-go/v2` | **待引入** | RocketMQ 客户端 | 🔲 Phase 3 |
| `golang-migrate/migrate` | **待引入** | 数据库迁移 | 🔲 Phase 5 |

---

## 7. 开发约定

### 分支策略
```
main          生产就绪代码，只接受 PR 合入
dev           日常开发分支，各 Phase 功能在此集成
feat/<name>   功能分支，从 dev 切出，完成后 PR 到 dev
fix/<name>    缺陷修复分支
```

### Commit 规范（Conventional Commits）
```
feat:     新功能
fix:      缺陷修复
refactor: 重构（不改变外部行为）
test:     测试相关
docs:     文档变更
chore:    构建/依赖/配置等杂项
```

### 模块内代码分层规范
```
handler.go    → 仅做参数绑定、校验、调用 service、返回响应；禁止直接操作 DB
service.go    → 业务逻辑；跨模块调用只能通过接口（interface）
repository.go → 仅做数据存取；禁止包含业务判断
model.go      → DTO、请求/响应结构体；禁止包含业务逻辑
consumer.go   → MQ 消费者；调用 service 处理消息
```

### 错误处理规范
- 底层错误使用 `fmt.Errorf("context: %w", err)` 包装后向上传递
- Handler 层统一通过 `response.Fail` / `response.FailWithStatus` 返回，不直接暴露内部错误信息
- 所有 `error` 在 service 层记录日志（`logger.L().Error(..., zap.Error(err))`），repository 层只包装不记录

### 配置管理
- 所有账密、密钥类信息只写入 `config.json`（已 gitignore）
- 新增配置项同步更新 `config.example.json` 和 `pkg/config/config.go`
- 生产环境账密通过环境变量覆盖（Phase 5 实现）
