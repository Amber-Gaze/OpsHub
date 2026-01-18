-- 用户表
CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    username VARCHAR(64) UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    role VARCHAR(32) DEFAULT 'viewer',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Casbin 权限表
CREATE TABLE casbin_rule (
    p_type VARCHAR(8),
    v0 VARCHAR(128),
    v1 VARCHAR(128),
    v2 VARCHAR(128),
    v3 VARCHAR(128),
    v4 VARCHAR(128),
    v5 VARCHAR(128)
);

-- 配置分组
CREATE TABLE config_groups (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(128) NOT NULL,
    description TEXT,
    parent_id BIGINT REFERENCES config_groups(id) ON DELETE CASCADE,
    created_by BIGINT REFERENCES users(id),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- 配置项
CREATE TABLE config_items (
    id BIGSERIAL PRIMARY KEY,
    group_id BIGINT REFERENCES config_groups(id) ON DELETE CASCADE,
    key_name VARCHAR(256) NOT NULL,
    value JSONB NOT NULL,
    version BIGINT DEFAULT 1,
    created_by BIGINT REFERENCES users(id),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- 配置历史
CREATE TABLE config_history (
    id BIGSERIAL PRIMARY KEY,
    item_id BIGINT REFERENCES config_items(id) ON DELETE CASCADE,
    version BIGINT NOT NULL,
    value JSONB NOT NULL,
    changed_by BIGINT REFERENCES users(id),
    changed_at TIMESTAMPTZ DEFAULT NOW(),
    comment TEXT
);

-- 服务实例
CREATE TABLE service_instances (
    id BIGSERIAL PRIMARY KEY,
    service_name VARCHAR(128) NOT NULL,
    instance_id VARCHAR(128) NOT NULL,
    ip_address INET,
    port INT,
    status VARCHAR(32) DEFAULT 'unknown',
    last_heartbeat TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- 审计日志
CREATE TABLE audit_logs (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT REFERENCES users(id),
    action VARCHAR(64),
    target_type VARCHAR(64),
    target_id BIGINT,
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);