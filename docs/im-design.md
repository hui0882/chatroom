# IM 消息系统设计文档

> 版本：v1.0  
> 日期：2026-04-27  
> 作者：claude

---

## 一、整体架构

```
前端 (React)                         后端 (Go/Gin)
─────────────────────────────────────────────────────────────────
                                                            
  AppLayout                                                
    └─ useEffect: connect(sessionId)                        
         │                                                  
         ▼                                                  
    wsStore ─── WebSocket ──────────────────►  ws.Hub      
      │         (单例连接)                        │          
      │                                           │ Register/Unregister Hook
      │                                           ▼          
      │                                    message.Dispatcher
      │                                      │   │          
      │  chat/ack/online/offline/unread_init │   │ SendMessage
      │◄──────────────────────────────────────   │          
      │                                          ▼          
   ContactPage                           message.Service   
     └─ ChatWindow                              │           
          ├─ subscribeChatMessages()            ├─► CacheRepository
          └─ subscribeChatAck()                 │     ├─► Redis ZSet (热数据)
                                                │     └─► PostgreSQL (持久化)
                                                └─► UnreadStore (Redis Hash)
```

---

## 二、数据库设计

### messages 表（`scripts/sql/003_messages.sql`）

```sql
CREATE TABLE messages (
    id          BIGSERIAL PRIMARY KEY,
    from_uid    BIGINT      NOT NULL,
    to_uid      BIGINT      NOT NULL,
    content     TEXT        NOT NULL,
    msg_type    VARCHAR(16) NOT NULL DEFAULT 'text',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ          -- 软删除（预留）
);

-- 主查询索引：LEAST/GREATEST 对两端 uid 排序，使 uid1-uid2 和 uid2-uid1 共享同一索引
CREATE INDEX idx_messages_conv ON messages (
    LEAST(from_uid, to_uid),
    GREATEST(from_uid, to_uid),
    id DESC
) WHERE deleted_at IS NULL;
```

---

## 三、Redis 数据结构

### 3.1 消息缓存（ZSet）

```
key:    msg:conv:{min_uid}:{max_uid}
member: "{id}|{from_uid}|{to_uid}|{content}|{msg_type}|{unix_nano}"
score:  msg.ID
TTL:    7 天
容量上限: 每个对话最多保留 100 条（超出时删除最旧的）
```

**读写策略：**
- 写：DB 落库成功后，异步 `ZADD` + `ZREMRANGEBYRANK`
- 读：先查 Redis；cache miss 时 fallback 到 DB，并异步预热缓存（仅首页）

### 3.2 未读计数（Hash）

```
key:    unread:{to_uid}
field:  {from_uid}
value:  count（整数字符串）
```

**操作时机：**
- 收到新消息：`HINCRBY unread:{to_uid} {from_uid} 1`
- 打开会话窗口：`HSET unread:{to_uid} {from_uid} 0`（HTTP 接口 + WS 订阅时清零）
- WS 连接建立：`HGETALL unread:{uid}` → `cmd=unread_init` 帧推送全量未读

---

## 四、WebSocket 帧协议

所有帧为 JSON，格式：`{"cmd": "<command>", "data": {...}}`

### 客户端 → 服务端

| cmd | data 结构 | 说明 |
|-----|-----------|------|
| `chat` | `{"to_uid": 123, "content": "hello"}` | 发送私聊消息 |
| `__ping__` | （文本帧，非 JSON） | 客户端心跳（每 25s） |

### 服务端 → 客户端

| cmd | data 结构 | 触发时机 |
|-----|-----------|---------|
| `chat` | `{id, from_uid, to_uid, content, msg_type, created_at}` | 接收方收到新消息 |
| `chat_ack` | `{id, to_uid, content, created_at}` | 发送方确认（含服务端 ID） |
| `unread_init` | `{"peer_uid": count, ...}` | WS 连接建立时推送全量未读 |
| `online` | `{"uid": 123}` | 好友上线 |
| `offline` | `{"uid": 123}` | 好友下线（所有设备均断开） |
| `error` | `{"code": 1001, "msg": "..."}` | 操作失败通知 |

---

## 五、HTTP 接口

| Method | Path | 说明 |
|--------|------|------|
| GET | `/api/v1/messages/:peer_uid` | 查询聊天历史（游标翻页） |
| GET | `/api/v1/messages/unread` | 获取全部对话未读数 |
| POST | `/api/v1/messages/unread/:peer_uid/clear` | 清零指定对话未读数 |

### 游标翻页说明

```
GET /api/v1/messages/123?cursor=0&limit=50

Response:
{
  "code": 0,
  "data": {
    "list": [                    // 按 id DESC 排列（最新在前）
      { "id": 50, ... },
      { "id": 49, ... }
    ],
    "next_cursor": 40            // 下一页传 cursor=40；0 表示没有更多
  }
}
```

---

## 六、前端状态管理

### wsStore（`src/store/wsStore.ts`）

| 状态 | 类型 | 说明 |
|------|------|------|
| `ws` | `WebSocket \| null` | 当前 WS 连接实例 |
| `connected` | `boolean` | 是否已连接 |
| `unread` | `Record<string, number>` | 各对话未读数 |
| `onlineUids` | `Set<number>` | 当前在线好友 UID |

| 方法 | 说明 |
|------|------|
| `connect(sessionId)` | 建立 WS 连接（幂等，已连接则跳过） |
| `disconnect()` | 断开连接 |
| `sendChat(toUid, content)` | 发送消息帧 |
| `clearUnread(peerUid)` | 本地清零未读数 |

### 发布订阅（`src/store/wsStore.ts`）

```typescript
// 订阅来自某个好友的新消息（ChatWindow 内使用）
const unsub = subscribeChatMessages(peerUid, (push) => { ... });

// 订阅发送 ack
const unsub = subscribeChatAck(peerUid, (ack) => { ... });
```

---

## 七、消息发送完整流程

```
用户按下 Enter
    ↓
ChatWindow.handleSend()
    ├─ 1. 本地插入 pending 消息（临时负数 ID）
    ├─ 2. wsStore.sendChat(toUid, content)
    │         └─ ws.send({"cmd":"chat","data":{...}})
    │
    │   [服务端]
    │   ├─ Dispatcher.handleChat()
    │   ├─ message.Service.SendMessage()  —— 好友校验 → DB 落库 → Redis 未读 +1
    │   ├─ hub.SendToUser(toUID, chat push)
    │   └─ client.Send ← chat_ack
    │
    ├─ 3. subscribeChatAck 回调：pending 消息替换为真实 ID
    └─ 4. 若 WS 未连接：标记 failed
```

---

## 八、在线状态推送流程

```
用户 A 建立 WS 连接
    ↓
Hub.Register → onConnect(client)
    ↓
Dispatcher.OnConnect()
    ├─ 1. PushUnreadInit → cmd=unread_init 帧推给 A
    └─ 2. 查询 A 的好友列表 → hub.BroadcastToUsers(cmd=online, targets=好友UIDs)

用户 A 断开连接
    ↓
Hub.Unregister → onDisconnect(client)
    ↓
Dispatcher.OnDisconnect()
    ├─ 检查 A 是否还有其他设备在线
    └─ 若全部离线：广播 cmd=offline 给好友
```

---

## 九、已实现功能清单

- [x] messages 表创建及索引
- [x] PostgreSQL 消息存储（`message.pgRepository`）
- [x] Redis ZSet 缓存包装层（`message.CacheRepository`），异步写、预热缓存
- [x] Redis Hash 未读计数（`message.redisUnreadStore`）
- [x] 消息 Service（好友校验 → 落库 → 未读 +1）
- [x] HTTP 接口（历史查询游标翻页 / 未读查询 / 未读清零）
- [x] WebSocket 帧协议定义（InboundFrame / OutboundFrame）
- [x] Dispatcher：chat 帧路由、chat_ack 回推、error 帧
- [x] Hub 扩展：SetHandler / SetConnectHooks / BroadcastToUsers / OnlineUIDs
- [x] 上线/下线好友广播（online / offline 帧）
- [x] WS 连接时推送 unread_init 全量未读
- [x] 前端 wsStore：单例 WS 连接 + 帧分发 + 未读 + 在线状态
- [x] 前端发布订阅：subscribeChatMessages / subscribeChatAck
- [x] ChatWindow：历史消息加载（游标翻页"加载更多"）
- [x] ChatWindow：乐观更新（pending 消息 → ack 后替换真实 ID）
- [x] ChatWindow：发送失败标记
- [x] 好友列表在线状态绿点 + 未读角标
- [x] AppLayout 侧边栏联系人菜单项总未读角标

---

## 十、待完成 / 遗留事项（TODO）

| 优先级 | 事项 | 说明 |
|--------|------|------|
| P1 | **多窗口同时聊天** | 当前只支持打开一个 ChatWindow，多个好友需排列或 Tab 管理 |
| P1 | **离线消息投递** | 接收方不在线时，unread +1 已做；但消息通知（如 App 推送）未实现 |
| P1 | **消息去重** | 网络抖动重连后可能收到重复帧，前端需按 msg ID 去重 |
| P2 | **消息队列解耦** | 目前同步写 DB；计划引入 RocketMQ（config.json 已有 rocketmq 配置）异步消费落库 |
| P2 | **Redis ZSet 缓存一致性** | 历史翻页时若 Redis 中该页范围内条数不足（部分命中），目前直接 fallback 到 DB；可优化为 merge 策略 |
| P2 | **消息已读回执** | 接收方读取消息后，给发送方发送"已读"状态 |
| P3 | **消息撤回** | 发送方 2 分钟内可撤回消息（DB 软删除 + WS 广播 `msg_recall` 帧） |
| P3 | **图片/文件消息** | `msg_type` 字段已预留，需要添加 OSS 上传接口（config.json 已有 oss 配置） |
| P3 | **消息搜索** | 按关键词全文搜索聊天记录（PostgreSQL full-text search 或 Elasticsearch） |
| P3 | **分布式扩展** | 当前 Hub 是单进程内存结构；多节点部署需用 Redis Pub/Sub 或消息队列做跨节点路由 |
