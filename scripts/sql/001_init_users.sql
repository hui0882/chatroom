-- ============================================================
-- 001_init_users.sql
-- 用户表初始化迁移脚本
-- ============================================================

-- 扩展：uuid 生成（可选，若需要数据库层生成 UUID 主键时使用）
-- CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ── users 表 ──────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS users (
    id            BIGSERIAL       PRIMARY KEY,
    username      VARCHAR(30)     NOT NULL UNIQUE,
    password_hash TEXT            NOT NULL,
    nickname      VARCHAR(30)     NOT NULL,
    avatar_url    TEXT            NOT NULL DEFAULT '',
    bio           TEXT            NOT NULL DEFAULT '',
    gender        VARCHAR(10)     NOT NULL DEFAULT 'unknown'
                                  CHECK (gender IN ('unknown','male','female')),
    age           SMALLINT        CHECK (age IS NULL OR (age >= 0 AND age <= 150)),
    role          VARCHAR(10)     NOT NULL DEFAULT 'user'
                                  CHECK (role IN ('user','admin')),
    status        VARCHAR(10)     NOT NULL DEFAULT 'active'
                                  CHECK (status IN ('active','banned','deleted')),
    ban_reason    TEXT            NOT NULL DEFAULT '',
    ban_until     TIMESTAMPTZ,            -- NULL = 永久封禁
    openid        VARCHAR(64)     UNIQUE,  -- 微信 openid，预留
    created_at    TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    deleted_at    TIMESTAMPTZ             -- 逻辑删除时间，NULL = 未删除
);

-- 常用查询索引
CREATE INDEX IF NOT EXISTS idx_users_username   ON users(username);
CREATE INDEX IF NOT EXISTS idx_users_status     ON users(status);
CREATE INDEX IF NOT EXISTS idx_users_openid     ON users(openid) WHERE openid IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_users_created_at ON users(created_at DESC);

COMMENT ON TABLE  users               IS '用户账号表';
COMMENT ON COLUMN users.role          IS 'user=普通用户, admin=管理员';
COMMENT ON COLUMN users.status        IS 'active=正常, banned=封禁, deleted=逻辑删除';
COMMENT ON COLUMN users.ban_until     IS '封禁截止时间，NULL 表示永久封禁';
COMMENT ON COLUMN users.openid        IS '微信 openid，接入微信登录时使用';
COMMENT ON COLUMN users.deleted_at    IS '逻辑删除时间，非 NULL 表示已删除';
