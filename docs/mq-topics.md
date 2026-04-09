# RocketMQ Topic 与消息流转设计

## Topic 总览

| Topic | 生产者 | 消费者 | 说明 |
|-------|--------|--------|------|
| `chatroom_msg_persist` | Gateway 模块 | MessagePersistConsumer | 消息持久化到 PostgreSQL + Redis |
| `chatroom_msg_deliver` | MessagePersistConsumer | MessageDeliverConsumer（各节点） | 消息投递到目标 WebSocket 连接 |
| `chatroom_msg_ack` | Gateway 模块 | MessageAckConsumer | 处理客户端 ACK，更新已读状态 |
| `chatroom_notify` | 各业务模块 | NotifyConsumer | 系统通知推送（好友申请/群变更等） |
| `chatroom_behavior` | Gateway / Chat 模块 | BehaviorLogConsumer | 用户行为日志写入（AI 推荐预留） |

---

## 消息格式规范

所有 Topic 消息体为 JSON，顶层结构：

```json
{
  "event":      "msg.new",           // 事件类型
  "trace_id":   "uuid",              // 链路追踪 ID
  "timestamp":  1712649600000,       // 毫秒时间戳
  "payload":    {}                   // 事件载荷，见各 Topic 定义
}
```

---

## 1. chatroom_msg_persist

**生产时机**：Gateway 收到客户端 `msg.send` 命令后，校验通过立即投递。

**消息体：**
```json
{
  "event": "msg.persist",
  "trace_id": "xxx",
  "timestamp": 1712649600000,
  "payload": {
    "client_seq": 1001,
    "sender_uid": 10001,
    "conversation_id": 42,
    "conv_type": 1,
    "msg_type": 1,
    "content": "你好！",
    "ref_msg_id": null
  }
}
```

**消费逻辑（MessagePersistConsumer）：**
1. 使用 Redis INCR 获取 `conv:{conv_id}:seq` 作为 `seq_id`
2. 写入 PostgreSQL `messages` 表
3. 写入 Redis ZSet `conv:{conv_id}:msgs`（score=seq_id，value=消息JSON）
4. 更新 `conversations.last_seq_id` 和 `last_msg`
5. 投递到 `chatroom_msg_deliver` Topic

---

## 2. chatroom_msg_deliver

**消息体：**
```json
{
  "event": "msg.deliver",
  "trace_id": "xxx",
  "timestamp": 1712649600000,
  "payload": {
    "msg_id": 88888,
    "seq_id": 56,
    "client_seq": 1001,
    "sender_uid": 10001,
    "conversation_id": 42,
    "conv_type": 1,
    "msg_type": 1,
    "content": "你好！",
    "ref_msg_id": null,
    "created_at": "2025-04-09T10:00:00Z",
    "target_uids": [10002]        // 单聊=[对方uid]；群聊=所有在线成员uid列表
  }
}
```

**消费逻辑（MessageDeliverConsumer，每个节点各自消费）：**

```
for uid in target_uids:
    node = Redis.GET("user:{uid}:node")
    if node == current_node_id:
        conn = localConnPool.Get(uid)
        if conn != nil:
            conn.Send(msg.new)         // 在线 → 直接推送
            // 等待 ACK，超时未 ACK 则重试（最多3次）
        else:
            Redis.INCR("user:{uid}:unread:{conv_id}")  // 离线 → 计未读数
    // 不属于本节点 → 忽略，由对应节点处理
```

**发送方 ACK（`msg.ack`）**：
- 消费完成后，向发送方推送 `msg.ack`（携带 `msg_id` 和 `seq_id`）
- 查找发送方的节点，通过相同机制推送

---

## 3. chatroom_msg_ack

**生产时机**：Gateway 收到客户端 `msg.ack` 命令。

**消息体：**
```json
{
  "event": "msg.ack",
  "trace_id": "xxx",
  "timestamp": 1712649600000,
  "payload": {
    "user_id": 10002,
    "msg_id": 88888,
    "conversation_id": 42
  }
}
```

**消费逻辑（MessageAckConsumer）：**
1. 写入 `message_acks` 表
2. 更新 `user_conversations.read_seq_id`
3. 清零或减少 Redis `user:{uid}:unread:{conv_id}`

---

## 4. chatroom_notify

**生产者**：auth 模块（好友申请）、group 模块（群成员变化）等。

**消息体：**
```json
{
  "event": "notify.friend_request",
  "trace_id": "xxx",
  "timestamp": 1712649600000,
  "payload": {
    "target_uid": 10002,
    "request_id": 5,
    "from_uid": 10001,
    "from_nickname": "Alice",
    "from_avatar": "...",
    "message": "加个好友吧"
  }
}
```

**消费逻辑（NotifyConsumer）：**
- 查 Redis 目标用户是否在线，在线则推送对应 WebSocket 通知帧
- 离线则写入通知表（预留），用户下次登录时通过 REST 接口拉取

---

## 5. chatroom_behavior

**消息体：**
```json
{
  "event": "behavior.log",
  "trace_id": "xxx",
  "timestamp": 1712649600000,
  "payload": {
    "user_id": 10001,
    "action": "send_msg",
    "target_type": "user",
    "target_id": 10002,
    "extra": {}
  }
}
```

**消费逻辑（BehaviorLogConsumer）：**
- 批量写入 `user_behavior_logs` 表（预留 AI 推荐数据源）

---

## 消费者组配置建议

| Consumer Group | 并发度 | 消费模式 |
|----------------|--------|---------|
| `cg_msg_persist` | 8 | 集群消费，顺序消费（按 conversation_id 分区） |
| `cg_msg_deliver` | 每节点独立，16 | 集群消费，广播到各节点 |
| `cg_msg_ack` | 4 | 集群消费 |
| `cg_notify` | 4 | 集群消费 |
| `cg_behavior` | 2 | 集群消费，批量消费（每次最多 100 条） |
