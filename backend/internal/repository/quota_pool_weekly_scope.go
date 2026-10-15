package repository

import "context"

func (r *quotaPoolRepository) ListWeeklyGroupAccountIDs(ctx context.Context, groupID int64) ([]int64, error) {
	rows, err := r.sql.QueryContext(ctx, `SELECT DISTINCT b.account_id FROM quota_pool_accounts b
 JOIN quota_pool_group_limits l ON l.quota_pool_id=b.quota_pool_id
 WHERE l.group_id=$1 AND l.deleted_at IS NULL AND b.platform='openai' AND b.deleted_at IS NULL ORDER BY b.account_id`, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []int64{}
	for rows.Next() {
		var id int64
		if err = rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}
