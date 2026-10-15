package repository

import (
	"context"
	"fmt"
)

// Both explicit allocations and access to pool members establish commercial
// scope. Remember it so removal cannot turn that scope into unrestricted traffic.
func (r *quotaPoolRepository) WeeklyGroupRequired(ctx context.Context, id int64) (bool, error) {
	if r == nil || r.db == nil {
		return false, fmt.Errorf("weekly business scope database is unavailable")
	}
	if id <= 0 {
		return false, nil
	}
	var remembered, memberAccess bool
	err := r.db.QueryRowContext(ctx, `WITH RECURSIVE reachable(id) AS (
 SELECT id FROM groups WHERE id=$1
 UNION
 SELECT child.id FROM reachable r JOIN groups parent ON parent.id=r.id
 JOIN groups child ON child.id=parent.fallback_group_id OR child.id=parent.fallback_group_id_on_invalid_request
 )
 SELECT g.quota_pool_required,EXISTS(
 SELECT 1 FROM reachable r JOIN groups target ON target.id=r.id
 WHERE target.quota_pool_required OR EXISTS(
 SELECT 1 FROM account_groups ag JOIN quota_pool_accounts b ON b.account_id=ag.account_id
 WHERE ag.group_id=target.id AND b.deleted_at IS NULL))
 FROM groups g WHERE g.id=$1`, id).Scan(&remembered, &memberAccess)
	if err != nil {
		return false, err
	}
	if memberAccess && !remembered {
		_, err = r.db.ExecContext(ctx, "UPDATE groups SET quota_pool_required=true WHERE id=$1", id)
		if err != nil {
			return false, err
		}
	}
	return remembered || memberAccess, nil
}
