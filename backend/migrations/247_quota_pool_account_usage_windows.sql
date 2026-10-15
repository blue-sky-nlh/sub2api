SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '10min';

ALTER TABLE quota_pool_accounts
    ADD COLUMN IF NOT EXISTS last_usage_percent DOUBLE PRECISION NOT NULL DEFAULT 0;

ALTER TABLE quota_pool_group_usage
    ADD COLUMN IF NOT EXISTS quota_pool_account_id BIGINT REFERENCES quota_pool_accounts(id) ON DELETE CASCADE;

DROP INDEX IF EXISTS quota_pool_group_usage_pool_group_period_idx;

CREATE UNIQUE INDEX IF NOT EXISTS quota_pool_group_usage_pool_account_group_period_idx
    ON quota_pool_group_usage (quota_pool_id, quota_pool_account_id, group_id, period_type, period_key);
CREATE INDEX IF NOT EXISTS quota_pool_group_usage_pool_group_period_idx
    ON quota_pool_group_usage (quota_pool_id, group_id, period_type, period_key);
CREATE INDEX IF NOT EXISTS quota_pool_group_usage_account_id_idx ON quota_pool_group_usage (quota_pool_account_id);
