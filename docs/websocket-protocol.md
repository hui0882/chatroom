# WebSocket 消息协议设计

## 1. 连接建立

### 握手 URL

```
ws://host/ws?session_id=<session_id>
```

或通过 Cookie 携带 `session_id`，服务端在 HTTP Upgrade 阶段校验 Session 合法性，非法则返回 `401` 拒绝升级。

### 心跳机制

- 客户端每 **30 秒**发送一次 `ping` 消息
- 服务端收到 `ping` 后立即回复 `pong`
- 服务端若 **90 秒**内未收到任何消息，主动关闭连接
- 关闭时客户端应按指数退避策略重连（1s → 2s → 4s → ... → 60s 封顶）

---

## 2. 消息帧格式

所有 WebSocket 消息均为 **Text Frame**，内容为 JSON，顶层结构如下：

```json
{
  "cmd":     "<命令字符串>",
  "seq":     12345,
  "data":    {}
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `cmd` | string | 消息命令，见下方命令表 |
| `seq` | int64 | 客户端自增序号，服务端 ACK 时原样返回，用于请求/响应匹配 |
| `data` | object | 各命令的具体载荷，见下方定义 |

---

## 3. 命令字（cmd）总览

### 客户端 → 服务端

| cmd | 说明 |
|-----|------|
| `ping` | 心跳 |
| `msg.send` | 发送消息（单聊或群聊） |
| `msg.ack` | 客户端确认收到消息 |
| `msg.revoke` | 撤回消息 |

### 服务端 → 客户端

| cmd | 说明 |
|-----|------|
| `pong` | 心跳回复 |
| `msg.new` | 推送新消息 |
| `msg.ack` | 确认客户端发送成功（附服务端分配的 seq_id） |
| `msg.revoked` | 通知消息被撤回 |
| `notify.friend_request` | 好友申请通知 |
| `notify.friend_accepted` | 好友申请被接受通知 |
| `notify.group_invite` | 被邀请入群通知 |
| `notify.group_member_change` | 群成员变化通知（踢人/退群） |
| `notify.group_announcement` | 群公告更新通知 |
| `error` | 服务端推送的错误通知 |

---

## 4. 消息 data 字段定义

### 4.1 心跳

```json
// 客户端发
{ "cmd": "ping", "seq": 0, "data": {} }

// 服务端回
{ "cmd": "pong", "seq": 0, "data": {} }
```

---

### 4.2 发送消息 `msg.send`

```json
// 客户端发
{
  "cmd": "msg.send",
  "seq": 1001,
  "data": {
    "conversation_id": 42,          // 会话 ID（提前通过 REST 创建或获取）
    "msg_type": 1,                  // 1:文字 2:图片 3:文件 4:音频
    "content": "你好！",            // 文字时为纯文本
    "ref_msg_id": null              // 引用回复的消息 ID，可为 null
    // 图片/文件/音频时 content 为 JSON 字符串：
    // { "url": "...", "size": 102400, "mime_type": "image/jpeg", "width": 800, "height": 600 }
    // 音频额外字段：{ "url": "...", "duration": 5 }
  }
}
```

---

### 4.3 服务端确认发送成功 `msg.ack`（服务端 → 客户端）

```json
{
  "cmd": "msg.ack",
  "seq": 1001,        // 对应客户端发送时的 seq
  "data": {
    "msg_id": 88888,              // 服务端分配的消息数据库 ID
    "seq_id": 56,                 // 会话内递增序号
    "conversation_id": 42,
    "created_at": "2025-04-09T10:00:00Z"
  }
}
```

---

### 4.4 推送新消息 `msg.new`（服务端 → 客户端）

```json
{
  "cmd": "msg.new",
  "seq": 0,
  "data": {
    "msg_id": 88888,
    "seq_id": 56,
    "conversation_id": 42,
    "conv_type": 1,               // 1:单聊 2:群聊
    "sender": {
      "uid": 10001,
      "nickname": "Alice",
      "avatar_url": "..."
    },
    "msg_type": 1,
    "content": "你好！",
    "ref_msg_id": null,
    "created_at": "2025-04-09T10:00:00Z"
  }
}
```

---

### 4.5 客户端 ACK `msg.ack`（客户端 → 服务端）

> 客户端收到 `msg.new` 后，必须发送此帧告知服务端已收到，服务端写入 `message_acks` 表并更新 `user_conversations.read_seq_id`。

```json
{
  "cmd": "msg.ack",
  "seq": 0,
  "data": {
    "msg_id": 88888,
    "conversation_id": 42
  }
}
```

---

### 4.6 撤回消息 `msg.revoke`（客户端 → 服务端）

> 只能撤回自己发出的、2 分钟内的消息。

```json
{
  "cmd": "msg.revoke",
  "seq": 1002,
  "data": {
    "msg_id": 88888,
    "conversation_id": 42
  }
}
```

---

### 4.7 消息被撤回通知 `msg.revoked`（服务端 → 客户端）

```json
{
  "cmd": "msg.revoked",
  "seq": 0,
  "data": {
    "msg_id": 88888,
    "conversation_id": 42,
    "operator_uid": 10001
  }
}
```

---

### 4.8 系统通知示例（服务端 → 客户端）

```json
// 好友申请
{
  "cmd": "notify.friend_request",
  "seq": 0,
  "data": {
    "request_id": 5,
    "from": { "uid": 10002, "nickname": "Bob", "avatar_url": "..." },
    "message": "我是 Bob，加个好友吧"
  }
}

// 被邀请入群
{
  "cmd": "notify.group_invite",
  "seq": 0,
  "data": {
    "group_id": 100,
    "group_name": "开发交流群",
    "inviter": { "uid": 10001, "nickname": "Alice" }
  }
}
```

---

### 4.9 错误帧 `error`（服务端 → 客户端）

```json
{
  "cmd": "error",
  "seq": 1001,      // 触发错误的客户端 seq，若为主动推送则为 0
  "data": {
    "code": 1003,
    "msg": "权限不足，无法发送消息"
  }
}
```

---

## 5. 离线消息补偿策略

1. 客户端重连后，通过 REST 接口 `GET /api/v1/conversations` 获取各会话的最新 `last_seq_id`
2. 对比客户端本地存储的 `read_seq_id`，调用 `GET /api/v1/conversations/:conv_id/messages?after_seq=<read_seq_id>&limit=50` 拉取未读消息
3. 此方案无需额外 WebSocket 命令，由客户端主动拉取保证离线消息不丢失
