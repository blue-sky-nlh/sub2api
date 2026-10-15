-- Keep legacy current/monthly records as history. Empty state is deliberately
-- not admission-ready: the first verified upstream observation establishes baseline.
ALTER TABLE groups ADD COLUMN IF NOT EXISTS quota_pool_required BOOLEAN NOT NULL DEFAULT false;
UPDATE groups SET quota_pool_required=true WHERE id IN (
 SELECT group_id FROM quota_pool_group_limits
 UNION SELECT ag.group_id FROM account_groups ag JOIN quota_pool_accounts b ON b.account_id=ag.account_id
);
-- Parents routed into commercial groups retain the original access-group scope.
WITH RECURSIVE commercial(id) AS (
 SELECT id FROM groups WHERE quota_pool_required
 UNION
 SELECT parent.id FROM groups parent JOIN commercial child
 ON parent.fallback_group_id=child.id OR parent.fallback_group_id_on_invalid_request=child.id
)
UPDATE groups SET quota_pool_required=true WHERE id IN (SELECT id FROM commercial);
ALTER TABLE quota_pool_accounts ADD COLUMN IF NOT EXISTS weekly_state JSONB NOT NULL DEFAULT '{}';
ALTER TABLE quota_pool_accounts ADD COLUMN IF NOT EXISTS weekly_history JSONB NOT NULL DEFAULT '[]';
ALTER TABLE quota_pools ALTER COLUMN default_action SET DEFAULT 'deny';
UPDATE quota_pools SET default_action='deny', period_type='weekly';
