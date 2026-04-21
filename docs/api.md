# RESTful HTTP API 设计

> **文档版本**：v0.3.1  
> **最后更新**：2026-04-21  
> **当前实现状态**：✅ 已实现 | 🔲 待实现

---

## 基础约定

- Base URL：`/api/v1`
- 请求/响应格式：`application/json`
- 认证方式：Session Cookie（`sessionId`）或 Header `X-Session-Id`，优先取 Cookie
- 统一响应结构：

```json
{
  "code": 0,     // 0 成功，非 0 业务错误码
  "msg": "ok",   // 错误描述
  "data": {}     // 业务数据（成功时）
}
```

- 分页参数统一：`?page=1&page_size=20`
- 时间格式：RFC3339（`2026-04-16T10:00:00+08:00`）

---

## 错误码定义

| code | 含义 |
|------|------|
| 0 | 成功 |
| 1001 | 参数校验失败 |
| 1002 | 未登录 / Session 已过期 |
| 1003 | 权限不足（如非管理员） |
| 1004 | 资源不存在 |
| 1005 | 账号已存在 |
| 1006 | 用户名或密码错误 |
| 1007 | 账号已被封禁（含封禁原因） |
| 5000 | 服务内部错误 |

---

## 其他接口

### 健康检查 ✅

```
GET /health
```

**Response:**
```json
{
  "code": 0,
  "msg": "ok",
  "data": {
    "status": "ok",
    "node": "node1",
    "timestamp": "2026-04-16T17:00:00+08:00",
    "uptime": "1h2m3s",
    "go_version": "go1.24.11",
    "services": {
      "postgresql": "healthy",
      "redis": "healthy"
    }
  }
}
```

---

## 1. 认证模块 `/api/v1/auth`

### 1.1 注册 ✅

```
POST /api/v1/auth/register
```

**Request:**
```json
{
  "username": "alice",        // 必填，3-30位，只允许字母/数字/下划线
  "password": "Abc12345",     // 必填，至少8位，必须同时包含字母和数字
  "nickname": "Alice",        // 必填，1-30位
  "gender": "female",         // 选填：unknown / male / female，默认 unknown
  "age": 25                   // 选填，0-150
}
```

**Response:**
```json
{
  "code": 0,
  "msg": "ok",
  "data": {
    "id": 1,
    "username": "alice",
    "nickname": "Alice",
    "avatar_url": "",
    "bio": "",
    "gender": "female",
    "age": 25,
    "status": "active",
    "created_at": "2026-04-16T17:08:17+08:00"
  }
}
```

**错误示例：**
```json
{ "code": 1005, "msg": "username already exists" }
{ "code": 1001, "msg": "password must be at least 8 characters" }
{ "code": 1001, "msg": "password must contain both letters and digits" }
```

---

### 1.2 登录 ✅

```
POST /api/v1/auth/login
```

**Request:**
```json
{
  "username": "alice",
  "password": "Abc12345"
}
```

**Response:**（同时在 Cookie 中写入 `sessionId`，HttpOnly，7天有效期）
```json
{
  "code": 0,
  "msg": "ok",
  "data": {
    "session_id": "1ea75389-1baa-40a3-8a13-b6612c383717",
    "user": {
      "id": 1,
      "username": "alice",
      "nickname": "Alice",
      "avatar_url": "",
      "bio": "",
      "gender": "female",
      "age": 25,
      "status": "active",
      "created_at": "2026-04-16T17:08:17+08:00"
    }
  }
}
```

**错误示例：**
```json
{ "code": 1006, "msg": "wrong username or password" }
{ "code": 1007, "msg": "account is banned: 违规发言 (until 2026-04-30 00:00:00)" }
```

---

### 1.3 登出 ✅

```
POST /api/v1/auth/logout
Authorization: 需要登录（Cookie 或 X-Session-Id）
```

删除服务端 Session，清除 Cookie。

**Response:**
```json
{ "code": 0, "msg": "ok" }
```

---

### 1.4 微信扫码登录（预留）🔲

```
GET  /api/v1/auth/wechat/qrcode     // 获取微信二维码 ticket
GET  /api/v1/auth/wechat/callback   // 微信回调，写入 openid，返回 Session
```

---

## 2. 用户模块 `/api/v1/user`

### 2.1 获取当前登录用户信息 ✅

```
GET /api/v1/user/me
Authorization: 需要登录
```

**Response:**
```json
{
  "code": 0,
  "msg": "ok",
  "data": {
    "id": 1,
    "username": "alice",
    "nickname": "Alice",
    "avatar_url": "",
    "bio": "",
    "gender": "female",
    "age": 25,
    "status": "active",
    "created_at": "2026-04-16T17:08:17+08:00"
  }
}
```

---

### 2.2 修改密码 ✅

```
PUT /api/v1/user/password
Authorization: 需要登录
```

**Request:**
```json
{
  "old_password": "Abc12345",
  "new_password": "NewPass99"
}
```

**Response:**
```json
{ "code": 0, "msg": "ok" }
```

---

### 2.3 更新个人信息 🔲

```
PUT /api/v1/user/profile
Authorization: 需要登录
```

**Request:**
```json
{
  "nickname": "Alice2",
  "avatar_url": "https://...",
  "bio": "Hello World",
  "gender": "female",
  "age": 26
}
```

---

### 2.4 搜索用户 🔲

```
GET /api/v1/users/search?keyword=alice
Authorization: 需要登录
```

---

### 2.5 获取指定用户公开信息 🔲

```
GET /api/v1/users/:id
Authorization: 需要登录
```

---

### 2.6 AI 推荐联系人（预留）🔲

```
GET /api/v1/users/recommend
```

---

## 3. 管理员模块 `/api/v1/admin`

> 所有接口需要登录且角色为 `admin`，否则返回 `1003`。

### 3.1 获取用户列表 ✅

```
GET /api/v1/admin/users?page=1&page_size=20&status=active&keyword=alice
Authorization: 需要管理员
```

**Query 参数：**
| 参数 | 类型 | 说明 |
|------|------|------|
| page | int | 页码，默认 1 |
| page_size | int | 每页数量，默认 20 |
| status | string | 过滤状态：active / banned / deleted |
| keyword | string | 按用户名或昵称模糊搜索 |

**Response:**
```json
{
  "code": 0,
  "msg": "ok",
  "data": {
    "total": 100,
    "list": [
      {
        "id": 1,
        "username": "alice",
        "nickname": "Alice",
        "avatar_url": "",
        "gender": "female",
        "age": 25,
        "role": "user",
        "status": "active",
        "created_at": "2026-04-16T17:08:17+08:00"
      }
    ]
  }
}
```

---

### 3.2 封禁用户 ✅

```
POST /api/v1/admin/users/:id/ban
Authorization: 需要管理员
```

**Request:**
```json
{
  "reason": "发布违规内容",         // 必填
  "ban_until": "2026-05-01T00:00:00+08:00"  // 选填，null 或不传 = 永久封禁
}
```

封禁后自动：清除该用户所有 Session + 踢出 WebSocket 连接。

**Response:**
```json
{ "code": 0, "msg": "ok" }
```

---

### 3.3 解封用户 ✅

```
POST /api/v1/admin/users/:id/unban
Authorization: 需要管理员
```

**Response:**
```json
{ "code": 0, "msg": "ok" }
```

---

### 3.4 逻辑删除用户 ✅

```
DELETE /api/v1/admin/users/:id
Authorization: 需要管理员
```

删除后：账号 status 置为 `deleted`，前端展示为「已注销用户」并将头像置灰。同时清除 Session 和 WebSocket 连接。历史消息、群组关系等原始数据保留。

**Response:**
```json
{ "code": 0, "msg": "ok" }
```

---

### 3.5 恢复逻辑删除账号 ✅

```
POST /api/v1/admin/users/:id/restore
Authorization: 需要管理员
```

**Response:**
```json
{ "code": 0, "msg": "ok" }
```

---

### 3.6 重置用户密码 ✅

```
POST /api/v1/admin/users/:id/reset-password
Authorization: 需要管理员
```

**Request:**
```json
{
  "new_password": "Reset1234"  // 同样需满足密码强度规则
}
```

**Response:**
```json
{ "code": 0, "msg": "ok" }
```

---

### 3.7 强制踢出在线用户 ✅

```
POST /api/v1/admin/users/:id/kick
Authorization: 需要管理员
```

清除该用户所有 Session 并关闭其 WebSocket 连接，用户需要重新登录。

**Response:**
```json
{ "code": 0, "msg": "ok" }
```

---

## 4. WebSocket 接口

### 4.1 测试接口（无需认证）✅

```
GET /websocket_test
```

建立连接后，服务器将每条收到的文本消息按 **Unicode 码点（rune）倒序**后原路返回，用于验证 WebSocket 连通性。支持中文及任意 Unicode 字符。

**示例：**
```
Client → "Hello World"
Server → "dlroW olleH"

Client → "chatroom"
Server → "moortahc"

Client → "你好世界"
Server → "界世好你"
```

---

### 4.2 流式输出测试（无需认证）✅

```
GET /websocket_stream
```

模拟 AI 流式输出场景。建立连接后：
1. 客户端发送一条文本消息
2. 服务器将文本按 Unicode 码点倒序
3. 以 **20 字/秒**（每 50ms 一帧）的速度逐字符推送 `chunk` 帧
4. 全部推送完毕后发送 `done` 帧，然后等待下一条消息

**服务器推送帧格式：**
```json
// 内容帧（每个字符一帧）
{ "type": "chunk", "content": "界" }

// 结束帧
{ "type": "done" }
```

**示例流程：**
```
Client → "你好世界"
Server → {"type":"chunk","content":"界"}   // 第 0ms
Server → {"type":"chunk","content":"世"}   // 第 50ms
Server → {"type":"chunk","content":"好"}   // 第 100ms
Server → {"type":"chunk","content":"你"}   // 第 150ms
Server → {"type":"done"}                   // 第 200ms
```

---

### 4.3 正式连接（需登录）✅

```
GET /ws?session_id=<session_id>&device=pc
```

或使用 Header：`X-Session-Id: <session_id>`

| 参数 | 位置 | 说明 |
|------|------|------|
| session_id | URL query / Header / Cookie | 登录后获取的 Session ID |
| device | URL query | 设备类型：web / pc / mobile，默认 web |

连接帧格式详见 `docs/websocket-protocol.md`。

---

## 5. 好友模块 `/api/v1/friends` 🔲

### 5.1 获取好友列表

```
GET /api/v1/friends
```

### 5.2 发送好友申请

```
POST /api/v1/friends/requests
```

**Request:**
```json
{
  "target_id": 10002,
  "message": "我是 Alice，加个好友吧"
}
```

### 5.3 处理好友申请

```
PUT /api/v1/friends/requests/:request_id
```

**Request:**
```json
{
  "action": "accept"   // "accept" | "reject"
}
```

### 5.4 删除好友

```
DELETE /api/v1/friends/:id
```

### 5.5 拉黑 / 取消拉黑

```
PUT /api/v1/friends/:id/block
PUT /api/v1/friends/:id/unblock
```

---

## 6. 会话模块 `/api/v1/conversations` 🔲

### 6.1 获取会话列表

```
GET /api/v1/conversations
```

**Response data:**
```json
[
  {
    "conversation_id": 1,
    "type": 1,
    "target": { "id": 10002, "nickname": "Bob", "avatar_url": "..." },
    "last_msg": "你好！",
    "last_msg_at": "2026-04-16T10:00:00+08:00",
    "unread_count": 3
  }
]
```

### 6.2 获取消息历史（分页）

```
GET /api/v1/conversations/:conv_id/messages?before_seq=100&limit=20
```

优先读 Redis ZSet 缓存，缓存未命中则查 PostgreSQL。

### 6.3 创建单聊会话

```
POST /api/v1/conversations
```

**Request:**
```json
{
  "type": 1,
  "target_id": 10002
}
```

---

## 7. 群组模块 `/api/v1/groups` 🔲

### 7.1 创建群组

```
POST /api/v1/groups
```

**Request:**
```json
{
  "name": "开发交流群",
  "avatar_url": "",
  "member_ids": [10002, 10003]
}
```

### 7.2 获取群信息

```
GET /api/v1/groups/:group_id
```

### 7.3 更新群信息（群主/管理员）

```
PUT /api/v1/groups/:group_id
```

**Request:**
```json
{
  "name": "新名字",
  "avatar_url": "...",
  "announcement": "欢迎加入！"
}
```

### 7.4 解散群（群主）

```
DELETE /api/v1/groups/:group_id
```

### 7.5 获取群成员列表

```
GET /api/v1/groups/:group_id/members
```

### 7.6 邀请成员入群

```
POST /api/v1/groups/:group_id/members
```

**Request:**
```json
{
  "user_ids": [10004, 10005]
}
```

### 7.7 踢出成员（群主/管理员）

```
DELETE /api/v1/groups/:group_id/members/:uid
```

### 7.8 修改成员角色（群主操作）

```
PUT /api/v1/groups/:group_id/members/:uid/role
```

**Request:**
```json
{
  "role": "admin"   // "admin" | "member"
}
```

### 7.9 禁言 / 解禁成员（群主/管理员）

```
PUT /api/v1/groups/:group_id/members/:uid/mute
```

**Request:**
```json
{
  "duration": 3600   // 禁言秒数，0 表示解禁
}
```

### 7.10 主动退群

```
DELETE /api/v1/groups/:group_id/members/me
```

---

## 8. 文件上传 `/api/v1/upload` 🔲

```
POST /api/v1/upload
Content-Type: multipart/form-data
Authorization: 需要登录

file: <binary>
type: image | file | audio
```

**Response:**
```json
{
  "code": 0,
  "data": {
    "url": "https://oss.example.com/...",
    "size": 102400,
    "mime_type": "image/jpeg"
  }
}
```
