-- 私聊消息表
CREATE TABLE IF NOT EXISTS messages (
    id          BIGSERIAL PRIMARY KEY,
    from_uid    BIGINT      NOT NULL,
    to_uid      BIGINT      NOT NULL,
    content     TEXT        NOT NULL,
    msg_type    VARCHAR(16) NOT NULL DEFAULT 'text', -- 消息类型，当前只支持 text
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ          -- 软删除（预留）
);

-- 查询两人会话历史的主要索引（游标翻页用 id < cursor 倒序）
CREATE INDEX IF NOT EXISTS idx_messages_conv ON messages (
    LEAST(from_uid, to_uid),
    GREATEST(from_uid, to_uid),
    id DESC
) WHERE deleted_at IS NULL;

-- 单用户收件箱扫描（未读统计 fallback 用）
CREATE INDEX IF NOT EXISTS idx_messages_to_uid ON messages (to_uid, id DESC)
    WHERE deleted_at IS NULL;
