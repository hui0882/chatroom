-- ============================================================
-- 002_friends.sql
-- 好友申请 & 好友关系表
-- ============================================================

-- ── friend_requests 好友申请表 ────────────────────────────────
CREATE TABLE IF NOT EXISTS friend_requests (
    id          BIGSERIAL       PRIMARY KEY,
    from_uid    BIGINT          NOT NULL REFERENCES users(id),
    to_uid      BIGINT          NOT NULL REFERENCES users(id),
    message     TEXT            NOT NULL DEFAULT '', -- 附言
    status      VARCHAR(10)     NOT NULL DEFAULT 'pending'
                                CHECK (status IN ('pending','accepted','rejected','cancelled')),
    created_at  TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ     NOT NULL DEFAULT NOW(),

    -- 同一对用户只允许存在一条 pending 申请
    CONSTRAINT uq_friend_request UNIQUE (from_uid, to_uid)
);

CREATE INDEX IF NOT EXISTS idx_freq_from   ON friend_requests(from_uid, status);
CREATE INDEX IF NOT EXISTS idx_freq_to     ON friend_requests(to_uid, status);

COMMENT ON TABLE  friend_requests           IS '好友申请记录';
COMMENT ON COLUMN friend_requests.status   IS 'pending=待处理, accepted=已接受, rejected=已拒绝, cancelled=已撤回';

-- ── friends 好友关系表（双向存储，查询更高效）──────────────────
CREATE TABLE IF NOT EXISTS friends (
    uid         BIGINT          NOT NULL REFERENCES users(id),
    friend_uid  BIGINT          NOT NULL REFERENCES users(id),
    remark      TEXT            NOT NULL DEFAULT '', -- 备注名
    created_at  TIMESTAMPTZ     NOT NULL DEFAULT NOW(),

    PRIMARY KEY (uid, friend_uid),
    CONSTRAINT no_self_friend CHECK (uid != friend_uid)
);

CREATE INDEX IF NOT EXISTS idx_friends_uid ON friends(uid);

COMMENT ON TABLE  friends            IS '好友关系表（双向存储）';
COMMENT ON COLUMN friends.remark     IS '好友备注名，为空则展示对方昵称';
