package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/gatewayhooks"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

// Pool then binding row order is shared by admission, observation, group
// allocation changes, and adjustments. No in-memory correctness lock is needed.
func (r *quotaPoolRepository) WithWeeklyAccount(ctx context.Context, accountID, expectedPool int64, fn func(*service.QuotaPoolWeeklyView, *service.QuotaPoolWeeklyMember) error) error {
	if r.db == nil {
		return fmt.Errorf("weekly quota transaction dependency is missing")
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var poolID int64
	var count int
	err = tx.QueryRowContext(ctx, `SELECT COALESCE(MIN(quota_pool_id),0),COUNT(*) FROM quota_pool_accounts
 WHERE account_id=$1 AND platform='openai' AND deleted_at IS NULL`, accountID).Scan(&poolID, &count)
	if err != nil {
		return err
	}
	if count != 1 || poolID == 0 || (expectedPool != 0 && expectedPool != poolID) {
		return &gatewayhooks.QuotaError{Status: 403, Code: "quota_pool_binding_invalid", Message: "A single authorized weekly pool binding is required"}
	}
	v, err := loadWeeklyView(ctx, tx, poolID)
	if err != nil {
		return err
	}
	var selected *service.QuotaPoolWeeklyMember
	for i := range v.Members {
		if v.Members[i].Binding.AccountID == accountID {
			selected = &v.Members[i]
			break
		}
	}
	if selected == nil {
		return fmt.Errorf("weekly binding disappeared")
	}
	oldState, err := json.Marshal(selected.State)
	if err != nil {
		return err
	}
	oldCycle := selected.State.Cycle
	if err = fn(v, selected); err != nil {
		return err
	}
	newState, err := json.Marshal(selected.State)
	if err != nil {
		return err
	}
	if string(newState) != string(oldState) {
		history := []byte("null")
		if oldCycle != selected.State.Cycle && oldCycle > 0 {
			history = oldState
		}
		_, err = tx.ExecContext(ctx, `UPDATE quota_pool_accounts SET weekly_state=$2::jsonb,
 last_usage_percent=COALESCE(($2::jsonb->'snapshot'->>'percent')::double precision,last_usage_percent),
 weekly_history=CASE WHEN $3::jsonb='null'::jsonb THEN weekly_history ELSE weekly_history || jsonb_build_array($3::jsonb) END,
 updated_at=NOW() WHERE id=$1`, selected.Binding.ID, string(newState), string(history))
		if err != nil {
			return err
		}
		// Upsert absolute totals, never additive retry updates. Old-cycle rows remain.
		for group, points := range selected.Contributions {
			_, err = tx.ExecContext(ctx, `INSERT INTO quota_pool_group_usage
 (quota_pool_id,quota_pool_account_id,group_id,period_type,period_key,used_cost,created_at,updated_at)
 VALUES($1,$2,$3,'weekly',$4,$5::numeric,NOW(),NOW())
 ON CONFLICT(quota_pool_id,quota_pool_account_id,group_id,period_type,period_key)
 DO UPDATE SET used_cost=EXCLUDED.used_cost,updated_at=NOW()`,
				poolID, selected.Binding.ID, group, service.WeeklyCycleKey(selected.State.Cycle), points)
			if err != nil {
				return err
			}
		}
	}
	return tx.Commit()
}

func loadWeeklyView(ctx context.Context, tx *sql.Tx, poolID int64) (*service.QuotaPoolWeeklyView, error) {
	v := &service.QuotaPoolWeeklyView{}
	err := tx.QueryRowContext(ctx, `SELECT id,name,enabled,period_type,default_action,deleted_at FROM quota_pools WHERE id=$1 FOR UPDATE`, poolID).
		Scan(&v.Pool.ID, &v.Pool.Name, &v.Pool.Enabled, &v.Pool.PeriodType, &v.Pool.DefaultAction, &v.Pool.DeletedAt)
	if err != nil {
		return nil, err
	}
	rows, err := tx.QueryContext(ctx, `SELECT b.id,b.quota_pool_id,b.platform,b.account_id,b.enabled,b.deleted_at,b.weekly_state,
 COALESCE(a.type='oauth' AND a.platform='openai' AND a.parent_account_id IS NULL AND a.deleted_at IS NULL,false),
 COALESCE(NULLIF(a.credentials->>'chatgpt_account_id',''),NULLIF(a.credentials->>'organization_id',''),'')
 FROM quota_pool_accounts b LEFT JOIN accounts a ON a.id=b.account_id
 WHERE b.quota_pool_id=$1 AND b.deleted_at IS NULL ORDER BY b.id FOR UPDATE OF b`, poolID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var m service.QuotaPoolWeeklyMember
		var raw []byte
		if err = rows.Scan(&m.Binding.ID, &m.Binding.QuotaPoolID, &m.Binding.Platform, &m.Binding.AccountID, &m.Binding.Enabled, &m.Binding.DeletedAt, &raw, &m.AccountValid, &m.AccountIdentity); err != nil {
			rows.Close()
			return nil, err
		}
		if err = json.Unmarshal(raw, &m.State); err != nil {
			rows.Close()
			return nil, err
		}
		m.Contributions = map[int64]string{}
		v.Members = append(v.Members, m)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	rows, err = tx.QueryContext(ctx, `SELECT group_id,percent_basis_points,enabled,deleted_at FROM quota_pool_group_limits WHERE quota_pool_id=$1 AND deleted_at IS NULL`, poolID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		l := service.QuotaPoolGroupLimit{QuotaPoolID: poolID}
		if err = rows.Scan(&l.GroupID, &l.PercentBasisPoints, &l.Enabled, &l.DeletedAt); err != nil {
			rows.Close()
			return nil, err
		}
		v.Limits = append(v.Limits, l)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	rows, err = tx.QueryContext(ctx, `SELECT u.quota_pool_account_id,u.group_id,u.used_cost::text
 FROM quota_pool_group_usage u JOIN quota_pool_accounts b ON b.id=u.quota_pool_account_id
 WHERE b.quota_pool_id=$1 AND b.deleted_at IS NULL AND u.period_type='weekly'
 AND u.period_key=(b.weekly_state->>'cycle')`, poolID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var binding, group int64
		var points string
		if err = rows.Scan(&binding, &group, &points); err != nil {
			rows.Close()
			return nil, err
		}
		for i := range v.Members {
			if v.Members[i].Binding.ID == binding {
				v.Members[i].Contributions[group] = points
			}
		}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	for i := range v.Members {
		if err := service.ValidateWeeklyMember(&v.Members[i]); err != nil {
			return nil, err
		}
	}
	return v, nil
}
func (r *quotaPoolRepository) ListWeeklyAccountIDs(ctx context.Context) ([]int64, error) {
	rows, err := r.sql.QueryContext(ctx, `SELECT DISTINCT b.account_id FROM quota_pool_accounts b
 JOIN quota_pools p ON p.id=b.quota_pool_id WHERE b.platform='openai' AND b.enabled
 AND b.deleted_at IS NULL AND p.enabled AND p.deleted_at IS NULL ORDER BY b.account_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := []int64{}
	for rows.Next() {
		var id int64
		if err = rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
func (r *quotaPoolRepository) ResetWeeklyGroup(ctx context.Context, poolID, groupID int64, actor string) error {
	if r.db == nil {
		return fmt.Errorf("weekly quota transaction dependency is missing")
	}
	if actor == "" {
		return &gatewayhooks.QuotaError{Status: 403, Code: "quota_pool_adjustment_identity_required", Message: "Authenticated admin identity is required"}
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	v, err := loadWeeklyView(ctx, tx, poolID)
	if err != nil {
		return err
	}
	for _, m := range v.Members {
		for g, points := range m.Contributions {
			if groupID != 0 && groupID != g {
				continue
			}
			m.State.Adjustments = append(m.State.Adjustments, service.QuotaPoolWeeklyAdjustment{Actor: actor, At: time.Now().UTC(), GroupID: g, Previous: points})
			raw, e := json.Marshal(m.State)
			if e != nil {
				return e
			}
			_, err = tx.ExecContext(ctx, `UPDATE quota_pool_accounts SET weekly_state=$2::jsonb,updated_at=NOW() WHERE id=$1`, m.Binding.ID, string(raw))
			if err != nil {
				return err
			}
			_, err = tx.ExecContext(ctx, `UPDATE quota_pool_group_usage SET used_cost=0,updated_at=NOW()
 WHERE quota_pool_account_id=$1 AND group_id=$2 AND period_type='weekly' AND period_key=$3`, m.Binding.ID, g, service.WeeklyCycleKey(m.State.Cycle))
			if err != nil {
				return err
			}
		}
	}
	return tx.Commit()
}
func (r *quotaPoolRepository) upsertWeeklyGroupLimit(ctx context.Context, l *service.QuotaPoolGroupLimit) error {
	if r.db == nil {
		return fmt.Errorf("weekly quota transaction dependency is missing")
	}
	if l.PercentBasisPoints < 0 || l.PercentBasisPoints > 10000 {
		return &gatewayhooks.QuotaError{Status: 400, Code: "quota_pool_invalid_share", Message: "Group allocation must be between 0 and 100%"}
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var id int64
	if err = tx.QueryRowContext(ctx, `SELECT id FROM quota_pools WHERE id=$1 AND deleted_at IS NULL FOR UPDATE`, l.QuotaPoolID).Scan(&id); err != nil {
		return err
	}
	var sum int
	if err = tx.QueryRowContext(ctx, `SELECT COALESCE(SUM(percent_basis_points),0) FROM quota_pool_group_limits
 WHERE quota_pool_id=$1 AND group_id<>$2 AND enabled AND deleted_at IS NULL`, l.QuotaPoolID, l.GroupID).Scan(&sum); err != nil {
		return err
	}
	if l.Enabled && sum+l.PercentBasisPoints > 10000 {
		return &gatewayhooks.QuotaError{Status: 400, Code: "quota_pool_allocation_exceeded", Message: "Enabled group allocations exceed 100%"}
	}
	if _, err = tx.ExecContext(ctx, "UPDATE groups SET quota_pool_required=true WHERE id=$1", l.GroupID); err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO quota_pool_group_limits(quota_pool_id,group_id,percent_basis_points,enabled,created_at,updated_at)
 VALUES($1,$2,$3,$4,NOW(),NOW()) ON CONFLICT(quota_pool_id,group_id) WHERE deleted_at IS NULL
 DO UPDATE SET percent_basis_points=EXCLUDED.percent_basis_points,enabled=EXCLUDED.enabled,updated_at=NOW()`, l.QuotaPoolID, l.GroupID, l.PercentBasisPoints, l.Enabled)
	if err != nil {
		return err
	}
	return tx.Commit()
}

var _ service.QuotaPoolWeeklyRepository = (*quotaPoolRepository)(nil)
