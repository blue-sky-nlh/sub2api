package repository

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/shopspring/decimal"
)

func (r *quotaPoolRepository) listWeeklyGroupUsage(ctx context.Context, poolID int64) ([]service.QuotaPoolGroupUsage, error) {
	rows, err := r.sql.QueryContext(ctx, `WITH capacity AS (
 SELECT CASE WHEN BOOL_AND(COALESCE(
 (weekly_state->>'cycle')::bigint > 0 AND
 (weekly_state->>'baseline')::numeric BETWEEN 0 AND 100, false))
 THEN SUM(100 - (weekly_state->>'baseline')::numeric) END AS points
 FROM quota_pool_accounts WHERE quota_pool_id=$1 AND enabled AND deleted_at IS NULL
 )
 SELECT MIN(u.id),u.quota_pool_id,u.group_id,SUM(u.used_cost)::text,
 (SELECT points::text FROM capacity),
 MIN(u.created_at),MAX(u.updated_at)
 FROM quota_pool_group_usage u JOIN quota_pool_accounts b ON b.id=u.quota_pool_account_id
 WHERE u.quota_pool_id=$1 AND u.period_type='weekly' AND b.enabled AND b.deleted_at IS NULL
 AND u.period_key=(b.weekly_state->>'cycle')
 GROUP BY u.quota_pool_id,u.group_id ORDER BY u.group_id`, poolID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []service.QuotaPoolGroupUsage{}
	for rows.Next() {
		item := service.QuotaPoolGroupUsage{PeriodType: service.QuotaPoolPeriodWeekly, PeriodKey: service.QuotaPoolPeriodCurrent}
		var rawPoints string
		var rawCapacity sql.NullString
		if err = rows.Scan(&item.ID, &item.QuotaPoolID, &item.GroupID, &rawPoints, &rawCapacity, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		if !rawCapacity.Valid {
			return nil, fmt.Errorf("weekly cycle capacity is unavailable")
		}
		points, err := decimal.NewFromString(rawPoints)
		if err != nil {
			return nil, err
		}
		capacity, err := decimal.NewFromString(rawCapacity.String)
		if err != nil || capacity.IsNegative() || points.IsNegative() {
			return nil, fmt.Errorf("invalid weekly capacity or usage")
		}
		if capacity.IsZero() {
			if !points.IsZero() {
				return nil, fmt.Errorf("weekly usage has no cycle capacity")
			}
		} else {
			item.UsedCost, _ = points.Mul(decimal.NewFromInt(100)).Div(capacity).Float64()
		}
		out = append(out, item)
	}
	return out, rows.Err()
}
