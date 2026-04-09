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

## 快速开始

```bash
# 1. 克隆项目
git clone <repo-url>
cd chatroom

# 2. 启动所有基础设施（PostgreSQL / Redis / RocketMQ）
docker-compose up -d

# 3. 初始化数据库
go run scripts/migrate.go

# 4. 启动服务
go run cmd/server/main.go
```

## 功能概览

- **账号体系**：手机号/邮箱注册登录，预留微信扫码登录（openid 字段）
- **单聊**：实时消息收发，离线消息，未读数统计，历史记录分页拉取
- **群聊**：群角色权限（群主/管理员/成员），群公告，禁言，踢人，拉人入群
- **消息类型**：文字、图片、文件、音频
- **消息可靠性**：客户端 ACK 确认机制，未 ACK 消息重投
- **高可用**：多节点部署，Redis 集中管理 WebSocket 连接路由，RocketMQ 解耦各模块
- **AI 扩展**：数据库与消息流设计预留 AI 联系人推荐接入点
