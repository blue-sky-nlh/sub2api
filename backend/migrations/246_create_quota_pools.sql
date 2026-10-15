SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '10min';

CREATE TABLE IF NOT EXISTS quota_pools (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    enabled BOOLEAN NOT NULL DEFAULT true,
    period_type VARCHAR(20) NOT NULL DEFAULT 'monthly',
    default_action VARCHAR(20) NOT NULL DEFAULT 'allow',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS quota_pools_name_active_idx
    ON quota_pools (name)
    WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS quota_pools_enabled_idx ON quota_pools (enabled);
CREATE INDEX IF NOT EXISTS quota_pools_period_type_idx ON quota_pools (period_type);

CREATE TABLE IF NOT EXISTS quota_pool_accounts (
    id BIGSERIAL PRIMARY KEY,
    quota_pool_id BIGINT NOT NULL REFERENCES quota_pools(id) ON DELETE CASCADE,
    platform VARCHAR(50) NOT NULL,
    account_id BIGINT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    enabled BOOLEAN NOT NULL DEFAULT true,
    last_usage_percent DOUBLE PRECISION NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS quota_pool_accounts_pool_platform_account_active_idx
    ON quota_pool_accounts (quota_pool_id, platform, account_id)
    WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS quota_pool_accounts_platform_account_idx
    ON quota_pool_accounts (platform, account_id);
CREATE INDEX IF NOT EXISTS quota_pool_accounts_enabled_idx ON quota_pool_accounts (enabled);
ALTER TABLE quota_pool_accounts ADD COLUMN IF NOT EXISTS last_usage_percent DOUBLE PRECISION NOT NULL DEFAULT 0;

CREATE TABLE IF NOT EXISTS quota_pool_group_limits (
    id BIGSERIAL PRIMARY KEY,
    quota_pool_id BIGINT NOT NULL REFERENCES quota_pools(id) ON DELETE CASCADE,
    group_id BIGINT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    percent_basis_points INTEGER NOT NULL DEFAULT 0,
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT quota_pool_group_limits_percent_range CHECK (percent_basis_points >= 0 AND percent_basis_points <= 10000)
);

CREATE UNIQUE INDEX IF NOT EXISTS quota_pool_group_limits_pool_group_active_idx
    ON quota_pool_group_limits (quota_pool_id, group_id)
    WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS quota_pool_group_limits_group_id_idx ON quota_pool_group_limits (group_id);
CREATE INDEX IF NOT EXISTS quota_pool_group_limits_enabled_idx ON quota_pool_group_limits (enabled);

CREATE TABLE IF NOT EXISTS quota_pool_group_usage (
    id BIGSERIAL PRIMARY KEY,
    quota_pool_id BIGINT NOT NULL REFERENCES quota_pools(id) ON DELETE CASCADE,
    group_id BIGINT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    period_type VARCHAR(20) NOT NULL DEFAULT 'monthly',
    period_key VARCHAR(20) NOT NULL,
    used_cost DECIMAL(20,10) NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS quota_pool_group_usage_pool_group_period_idx
    ON quota_pool_group_usage (quota_pool_id, group_id, period_type, period_key);
CREATE INDEX IF NOT EXISTS quota_pool_group_usage_group_id_idx ON quota_pool_group_usage (group_id);
CREATE INDEX IF NOT EXISTS quota_pool_group_usage_period_idx ON quota_pool_group_usage (period_type, period_key);
