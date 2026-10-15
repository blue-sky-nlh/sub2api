SET LOCAL lock_timeout = '5s';
SET LOCAL statement_timeout = '10min';

ALTER TABLE quota_pools
    ALTER COLUMN period_type SET DEFAULT 'current';

ALTER TABLE quota_pool_group_usage
    ALTER COLUMN period_type SET DEFAULT 'current',
    ALTER COLUMN period_key SET DEFAULT 'current';
