# 数据库设计（PostgreSQL）

## 建表规范

- 所有表主键使用 `BIGSERIAL`（雪花 ID 方案后续替换为 `BIGINT`）
- `created_at` / `updated_at` 由应用层写入，不使用数据库触发器
- 逻辑删除字段统一使用 `deleted_at TIMESTAMPTZ`，为 NULL 时表示未删除
- 枚举类型使用 `SMALLINT` + 常量注释，避免 PostgreSQL ENUM 迁移困难

---

## 1. 用户表 `users`

```sql
CREATE TABLE users (
    id           BIGSERIAL      PRIMARY KEY,
    username     VARCHAR(64)    NOT NULL UNIQUE,          -- 用户名（登录用）
    nickname     VARCHAR(64)    NOT NULL DEFAULT '',      -- 显示昵称
    phone        VARCHAR(20)    UNIQUE,                   -- 手机号（可为空）
    email        VARCHAR(128)   UNIQUE,                   -- 邮箱（可为空）
    password     VARCHAR(128)   NOT NULL DEFAULT '',      -- bcrypt 哈希，微信登录时为空
    avatar_url   VARCHAR(512)   NOT NULL DEFAULT '',      -- 头像 URL
    openid       VARCHAR(128)   UNIQUE,                   -- 微信 openid（预留）
    gender       SMALLINT       NOT NULL DEFAULT 0,       -- 0:未知 1:男 2:女
    region       VARCHAR(64)    NOT NULL DEFAULT '',      -- 地区（AI 推荐用）
    bio          VARCHAR(256)   NOT NULL DEFAULT '',      -- 个人简介
    status       SMALLINT       NOT NULL DEFAULT 1,       -- 1:正常 2:封禁
    last_online  TIMESTAMPTZ,                             -- 最后在线时间
    created_at   TIMESTAMPTZ    NOT NULL,
    updated_at   TIMESTAMPTZ    NOT NULL,
    deleted_at   TIMESTAMPTZ                              -- 注销软删除
);

CREATE INDEX idx_users_phone   ON users(phone)   WHERE phone IS NOT NULL;
CREATE INDEX idx_users_email   ON users(email)   WHERE email IS NOT NULL;
CREATE INDEX idx_users_openid  ON users(openid)  WHERE openid IS NOT NULL;
```

---

## 2. 好友关系表 `friendships`

```sql
-- 双向好友：user_id < friend_id 保证唯一性
CREATE TABLE friendships (
    id           BIGSERIAL    PRIMARY KEY,
    user_id      BIGINT       NOT NULL REFERENCES users(id),
    friend_id    BIGINT       NOT NULL REFERENCES users(id),
    remark       VARCHAR(64)  NOT NULL DEFAULT '',    -- 备注名
    status       SMALLINT     NOT NULL DEFAULT 1,     -- 1:正常 2:拉黑
    created_at   TIMESTAMPTZ  NOT NULL,
    updated_at   TIMESTAMPTZ  NOT NULL,
    CONSTRAINT uq_friendship UNIQUE (user_id, friend_id),
    CONSTRAINT chk_order CHECK (user_id < friend_id)
);

CREATE INDEX idx_friendships_user   ON friendships(user_id);
CREATE INDEX idx_friendships_friend ON friendships(friend_id);
```

---

## 3. 群组表 `groups`

```sql
CREATE TABLE groups (
    id           BIGSERIAL     PRIMARY KEY,
    name         VARCHAR(64)   NOT NULL,
    avatar_url   VARCHAR(512)  NOT NULL DEFAULT '',
    owner_id     BIGINT        NOT NULL REFERENCES users(id),
    announcement TEXT          NOT NULL DEFAULT '',   -- 群公告
    max_members  INT           NOT NULL DEFAULT 500,
    status       SMALLINT      NOT NULL DEFAULT 1,    -- 1:正常 2:解散
    created_at   TIMESTAMPTZ   NOT NULL,
    updated_at   TIMESTAMPTZ   NOT NULL,
    deleted_at   TIMESTAMPTZ
);

CREATE INDEX idx_groups_owner ON groups(owner_id);
```

---

## 4. 群成员表 `group_members`

```sql
CREATE TABLE group_members (
    id          BIGSERIAL    PRIMARY KEY,
    group_id    BIGINT       NOT NULL REFERENCES groups(id),
    user_id     BIGINT       NOT NULL REFERENCES users(id),
    role        SMALLINT     NOT NULL DEFAULT 3,    -- 1:群主 2:管理员 3:普通成员
    nickname    VARCHAR(64)  NOT NULL DEFAULT '',   -- 群内昵称
    muted_until TIMESTAMPTZ,                        -- NULL 表示未被禁言
    joined_at   TIMESTAMPTZ  NOT NULL,
    CONSTRAINT uq_group_member UNIQUE (group_id, user_id)
);

CREATE INDEX idx_gm_group  ON group_members(group_id);
CREATE INDEX idx_gm_user   ON group_members(user_id);
```

---

## 5. 会话表 `conversations`

> 统一管理单聊和群聊的会话，方便会话列表查询。

```sql
CREATE TABLE conversations (
    id           BIGSERIAL    PRIMARY KEY,
    type         SMALLINT     NOT NULL,             -- 1:单聊 2:群聊
    -- 单聊时 target_id = 对方 user_id；群聊时 target_id = group_id
    target_id    BIGINT       NOT NULL,
    -- 当前最新消息的 seq_id，用于客户端增量同步
    last_seq_id  BIGINT       NOT NULL DEFAULT 0,
    last_msg     TEXT         NOT NULL DEFAULT '',  -- 最新消息摘要（冗余，加速列表渲染）
    last_msg_at  TIMESTAMPTZ,
    created_at   TIMESTAMPTZ  NOT NULL,
    updated_at   TIMESTAMPTZ  NOT NULL
);

-- 用户会话关系表（每个用户的会话列表）
CREATE TABLE user_conversations (
    id              BIGSERIAL    PRIMARY KEY,
    user_id         BIGINT       NOT NULL REFERENCES users(id),
    conversation_id BIGINT       NOT NULL REFERENCES conversations(id),
    read_seq_id     BIGINT       NOT NULL DEFAULT 0,  -- 该用户已读到的 seq_id
    is_pinned       BOOLEAN      NOT NULL DEFAULT FALSE,
    is_muted        BOOLEAN      NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ  NOT NULL,
    updated_at      TIMESTAMPTZ  NOT NULL,
    CONSTRAINT uq_user_conv UNIQUE (user_id, conversation_id)
);

CREATE INDEX idx_uc_user ON user_conversations(user_id);
```

---

## 6. 消息表 `messages`

```sql
CREATE TABLE messages (
    id              BIGSERIAL    PRIMARY KEY,
    conversation_id BIGINT       NOT NULL REFERENCES conversations(id),
    seq_id          BIGINT       NOT NULL,            -- 会话内单调递增序号
    sender_id       BIGINT       NOT NULL REFERENCES users(id),
    msg_type        SMALLINT     NOT NULL,
    -- 1:文字 2:图片 3:文件 4:音频 5:系统通知
    content         TEXT         NOT NULL DEFAULT '', -- 文字内容 / JSON（图片/文件/音频用）
    is_revoked      BOOLEAN      NOT NULL DEFAULT FALSE,
    ref_msg_id      BIGINT,                           -- 引用回复的消息 ID（预留）
    extra           JSONB,                            -- 扩展字段（预留）
    created_at      TIMESTAMPTZ  NOT NULL,
    deleted_at      TIMESTAMPTZ,
    CONSTRAINT uq_conv_seq UNIQUE (conversation_id, seq_id)
);

CREATE INDEX idx_msg_conv_seq ON messages(conversation_id, seq_id DESC);
CREATE INDEX idx_msg_sender   ON messages(sender_id);
```

> **分区预留**：消息量增大后可对 `messages` 按 `conversation_id % N` 进行哈希分区，或按 `created_at` 做时间分区。

---

## 7. 消息 ACK 记录表 `message_acks`

```sql
CREATE TABLE message_acks (
    id         BIGSERIAL    PRIMARY KEY,
    msg_id     BIGINT       NOT NULL REFERENCES messages(id),
    user_id    BIGINT       NOT NULL REFERENCES users(id),
    acked_at   TIMESTAMPTZ  NOT NULL,
    CONSTRAINT uq_ack UNIQUE (msg_id, user_id)
);
```

---

## 8. 用户行为日志表 `user_behavior_logs`（AI 推荐预留）

```sql
CREATE TABLE user_behavior_logs (
    id           BIGSERIAL    PRIMARY KEY,
    user_id      BIGINT       NOT NULL REFERENCES users(id),
    action       VARCHAR(32)  NOT NULL,   -- 'send_msg' | 'view_profile' | 'add_friend'
    target_type  VARCHAR(16)  NOT NULL,   -- 'user' | 'group'
    target_id    BIGINT       NOT NULL,
    extra        JSONB,
    created_at   TIMESTAMPTZ  NOT NULL
);

CREATE INDEX idx_behavior_user ON user_behavior_logs(user_id, created_at DESC);
```

---

## Redis 数据结构汇总

| Key 格式 | 类型 | TTL | 说明 |
|---------|------|-----|------|
| `session:{session_id}` | Hash | 7d | 用户 Session 信息 |
| `user:{uid}:node` | String | 动态 | 用户所在节点 ID |
| `user:{uid}:unread:{conv_id}` | String | 永久 | 未读消息数 |
| `conv:{conv_id}:msgs` | ZSet(score=seq_id) | 72h | 近期消息缓存 |
| `conv:{conv_id}:seq` | String | 永久 | 会话 seq_id 计数器 |
| `node:{node_id}:conns` | Set | 动态 | 节点上的在线用户集合 |
