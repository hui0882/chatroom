# RESTful HTTP API 设计

## 基础约定

- Base URL：`/api/v1`
- 请求/响应格式：`application/json`
- 认证方式：Session Cookie（`session_id`）或 Header `X-Session-Id`
- 统一响应结构：

```json
{
  "code": 0,          // 0 表示成功，非 0 表示业务错误码
  "msg": "ok",        // 错误描述
  "data": {}          // 业务数据
}
```

- 分页参数统一：`?page=1&page_size=20`
- 时间格式：RFC3339（`2025-04-09T10:00:00Z`）

---

## 错误码定义

| code | 含义 |
|------|------|
| 0 | 成功 |
| 1001 | 参数校验失败 |
| 1002 | 未登录 / Session 已过期 |
| 1003 | 权限不足 |
| 1004 | 资源不存在 |
| 1005 | 账号已存在 |
| 1006 | 用户名或密码错误 |
| 1007 | 账号已被封禁 |
| 5000 | 服务内部错误 |

---

## 1. 认证模块 `/api/v1/auth`

### 1.1 注册

```
POST /api/v1/auth/register
```

**Request:**
```json
{
  "username": "alice",
  "password": "Abc@12345",
  "nickname": "Alice",
  "phone": "13800138000",   // 与 email 至少提供一个
  "email": "alice@example.com"
}
```

**Response:**
```json
{
  "code": 0,
  "msg": "ok",
  "data": {
    "uid": 10001,
    "username": "alice",
    "nickname": "Alice"
  }
}
```

---

### 1.2 登录

```
POST /api/v1/auth/login
```

**Request:**
```json
{
  "username": "alice",    // 支持用户名 / 手机号 / 邮箱
  "password": "Abc@12345"
}
```

**Response:**（同时在 Cookie 中写入 `session_id`）
```json
{
  "code": 0,
  "msg": "ok",
  "data": {
    "session_id": "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",
    "uid": 10001,
    "nickname": "Alice",
    "avatar_url": ""
  }
}
```

---

### 1.3 登出

```
POST /api/v1/auth/logout
```

删除服务端 Session，清除 Cookie。

---

### 1.4 微信扫码登录（预留）

```
GET  /api/v1/auth/wechat/qrcode     // 获取微信二维码 ticket
GET  /api/v1/auth/wechat/callback   // 微信回调，写入 openid，返回 Session
```

---

## 2. 用户模块 `/api/v1/users`

### 2.1 获取当前登录用户信息

```
GET /api/v1/users/me
```

### 2.2 更新当前用户信息

```
PUT /api/v1/users/me
```

**Request:**
```json
{
  "nickname": "Alice2",
  "avatar_url": "https://...",
  "bio": "Hello World",
  "region": "广东",
  "gender": 1
}
```

### 2.3 搜索用户（加好友前）

```
GET /api/v1/users/search?keyword=alice
```

### 2.4 获取指定用户公开信息

```
GET /api/v1/users/:uid
```

### 2.5 AI 推荐联系人（预留）

```
GET /api/v1/users/recommend
```

---

## 3. 好友模块 `/api/v1/friends`

### 3.1 获取好友列表

```
GET /api/v1/friends
```

### 3.2 发送好友申请

```
POST /api/v1/friends/requests
```

**Request:**
```json
{
  "target_uid": 10002,
  "message": "我是 Alice，加个好友吧"
}
```

### 3.3 处理好友申请

```
PUT /api/v1/friends/requests/:request_id
```

**Request:**
```json
{
  "action": "accept"   // "accept" | "reject"
}
```

### 3.4 删除好友

```
DELETE /api/v1/friends/:uid
```

### 3.5 拉黑 / 取消拉黑

```
PUT /api/v1/friends/:uid/block
PUT /api/v1/friends/:uid/unblock
```

---

## 4. 会话模块 `/api/v1/conversations`

### 4.1 获取会话列表

```
GET /api/v1/conversations
```

**Response data:**
```json
[
  {
    "conversation_id": 1,
    "type": 1,
    "target": { "uid": 10002, "nickname": "Bob", "avatar_url": "..." },
    "last_msg": "你好！",
    "last_msg_at": "2025-04-09T10:00:00Z",
    "unread_count": 3
  }
]
```

### 4.2 获取消息历史（分页）

```
GET /api/v1/conversations/:conv_id/messages?before_seq=100&limit=20
```

优先读 Redis ZSet 缓存，缓存未命中则查 PostgreSQL。

### 4.3 创建单聊会话

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

## 5. 群组模块 `/api/v1/groups`

### 5.1 创建群组

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

### 5.2 获取群信息

```
GET /api/v1/groups/:group_id
```

### 5.3 更新群信息（群主/管理员）

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

### 5.4 解散群（群主）

```
DELETE /api/v1/groups/:group_id
```

### 5.5 获取群成员列表

```
GET /api/v1/groups/:group_id/members
```

### 5.6 邀请成员入群

```
POST /api/v1/groups/:group_id/members
```

**Request:**
```json
{
  "user_ids": [10004, 10005]
}
```

### 5.7 踢出成员（群主/管理员）

```
DELETE /api/v1/groups/:group_id/members/:uid
```

### 5.8 修改成员角色（群主操作）

```
PUT /api/v1/groups/:group_id/members/:uid/role
```

**Request:**
```json
{
  "role": 2   // 2:管理员 3:普通成员
}
```

### 5.9 禁言 / 解禁成员（群主/管理员）

```
PUT /api/v1/groups/:group_id/members/:uid/mute
```

**Request:**
```json
{
  "duration": 3600   // 禁言秒数，0 表示解禁
}
```

### 5.10 主动退群

```
DELETE /api/v1/groups/:group_id/members/me
```

---

## 6. 文件上传 `/api/v1/upload`

```
POST /api/v1/upload
Content-Type: multipart/form-data

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
