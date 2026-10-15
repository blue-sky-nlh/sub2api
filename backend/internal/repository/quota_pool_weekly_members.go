package repository

import (
	"context"
	"github.com/Wei-Shaw/sub2api/internal/gatewayhooks"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

func (r *quotaPoolRepository) addWeeklyAccount(ctx context.Context, a *service.QuotaPoolAccount) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	// Across pools, one local account must have only one active weekly capacity.
	if _, err = tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock($1)", -a.AccountID); err != nil {
		return err
	}
	var pool int64
	if err = tx.QueryRowContext(ctx, "SELECT id FROM quota_pools WHERE id=$1 AND deleted_at IS NULL FOR UPDATE", a.QuotaPoolID).Scan(&pool); err != nil {
		return err
	}
	var valid bool
	err = tx.QueryRowContext(ctx, `SELECT type='oauth' AND platform='openai' AND parent_account_id IS NULL
 FROM accounts WHERE id=$1 AND deleted_at IS NULL`, a.AccountID).Scan(&valid)
	if err != nil {
		return err
	}
	if a.Platform != service.PlatformOpenAI || !valid {
		return &gatewayhooks.QuotaError{Status: 400, Code: "quota_pool_invalid_account", Message: "Weekly pools require independent GPT OAuth accounts"}
	}
	var exists bool
	if err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM quota_pool_accounts WHERE account_id=$1 AND deleted_at IS NULL)`, a.AccountID).Scan(&exists); err != nil {
		return err
	}
	if exists {
		return service.ErrQuotaPoolExists
	}
	err = tx.QueryRowContext(ctx, `INSERT INTO quota_pool_accounts(quota_pool_id,platform,account_id,enabled,last_usage_percent,created_at,updated_at)
 VALUES($1,$2,$3,$4,0,NOW(),NOW()) RETURNING id,created_at,updated_at`, a.QuotaPoolID, a.Platform, a.AccountID, a.Enabled).Scan(&a.ID, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, "UPDATE groups SET quota_pool_required=true WHERE id IN (SELECT group_id FROM account_groups WHERE account_id=$1)", a.AccountID); err != nil {
		return err
	}
	return tx.Commit()
}
func (r *quotaPoolRepository) deleteWeeklyAccount(ctx context.Context, binding int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var pool int64
	if err = tx.QueryRowContext(ctx, "SELECT quota_pool_id FROM quota_pool_accounts WHERE id=$1", binding).Scan(&pool); err != nil {
		return err
	}
	var id int64
	if err = tx.QueryRowContext(ctx, "SELECT id FROM quota_pools WHERE id=$1 FOR UPDATE", pool).Scan(&id); err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, "UPDATE quota_pool_accounts SET deleted_at=NOW(),updated_at=NOW() WHERE id=$1 AND deleted_at IS NULL", binding)
	if err != nil {
		return err
	}
	return tx.Commit()
}
func (r *quotaPoolRepository) deleteWeeklyGroupLimit(ctx context.Context, pool, group int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var id int64
	if err = tx.QueryRowContext(ctx, "SELECT id FROM quota_pools WHERE id=$1 FOR UPDATE", pool).Scan(&id); err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `UPDATE quota_pool_group_limits SET deleted_at=NOW(),updated_at=NOW() WHERE quota_pool_id=$1 AND group_id=$2 AND deleted_at IS NULL`, pool, group)
	if err != nil {
		return err
	}
	return tx.Commit()
}
