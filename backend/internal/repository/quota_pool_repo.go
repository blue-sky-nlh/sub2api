package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/quotapool"
	"github.com/Wei-Shaw/sub2api/ent/quotapoolaccount"
	"github.com/Wei-Shaw/sub2api/ent/quotapoolgrouplimit"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type quotaPoolRepository struct {
	client *dbent.Client
	sql    sqlExecutor
	db     *sql.DB
}

func NewQuotaPoolRepository(client *dbent.Client, sqlDB *sql.DB) service.QuotaPoolRepository {
	return &quotaPoolRepository{client: client, sql: sqlDB, db: sqlDB}
}

func (r *quotaPoolRepository) Create(ctx context.Context, pool *service.QuotaPool) error {
	created, err := r.client.QuotaPool.Create().
		SetName(pool.Name).
		SetNillableDescription(pool.Description).
		SetEnabled(pool.Enabled).
		SetPeriodType(pool.PeriodType).
		SetDefaultAction(pool.DefaultAction).
		Save(ctx)
	if err != nil {
		return translatePersistenceError(err, nil, service.ErrQuotaPoolExists)
	}
	*pool = quotaPoolEntityToService(created)
	return nil
}

func (r *quotaPoolRepository) GetByID(ctx context.Context, id int64) (*service.QuotaPool, error) {
	row, err := r.client.QuotaPool.Query().Where(quotapool.IDEQ(id), quotapool.DeletedAtIsNil()).Only(ctx)
	if dbent.IsNotFound(err) {
		return nil, service.ErrQuotaPoolNotFound
	}
	if err != nil {
		return nil, err
	}
	out := quotaPoolEntityToService(row)
	return &out, nil
}

func (r *quotaPoolRepository) Update(ctx context.Context, pool *service.QuotaPool) error {
	builder := r.client.QuotaPool.UpdateOneID(pool.ID).
		SetName(pool.Name).
		SetNillableDescription(pool.Description).
		SetEnabled(pool.Enabled).
		SetPeriodType(pool.PeriodType).
		SetDefaultAction(pool.DefaultAction)
	row, err := builder.Save(ctx)
	if dbent.IsNotFound(err) {
		return service.ErrQuotaPoolNotFound
	}
	if err != nil {
		return translatePersistenceError(err, service.ErrQuotaPoolNotFound, service.ErrQuotaPoolExists)
	}
	*pool = quotaPoolEntityToService(row)
	return nil
}

func (r *quotaPoolRepository) Delete(ctx context.Context, id int64) error {
	now := time.Now()
	_, err := r.client.QuotaPool.Update().Where(quotapool.IDEQ(id), quotapool.DeletedAtIsNil()).SetDeletedAt(now).Save(ctx)
	return err
}

func (r *quotaPoolRepository) List(ctx context.Context, params pagination.PaginationParams) ([]service.QuotaPool, *pagination.PaginationResult, error) {
	q := r.client.QuotaPool.Query().Where(quotapool.DeletedAtIsNil())
	total, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, nil, err
	}
	rows, err := q.Order(dbent.Desc(quotapool.FieldID)).Offset(params.Offset()).Limit(params.Limit()).All(ctx)
	if err != nil {
		return nil, nil, err
	}
	out := make([]service.QuotaPool, 0, len(rows))
	for _, row := range rows {
		out = append(out, quotaPoolEntityToService(row))
	}
	return out, paginationResultFromTotal(int64(total), params), nil
}

func (r *quotaPoolRepository) ListAll(ctx context.Context) ([]service.QuotaPool, error) {
	rows, err := r.client.QuotaPool.Query().Where(quotapool.DeletedAtIsNil()).Order(dbent.Asc(quotapool.FieldID)).All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]service.QuotaPool, 0, len(rows))
	for _, row := range rows {
		out = append(out, quotaPoolEntityToService(row))
	}
	return out, nil
}

func (r *quotaPoolRepository) ListAccounts(ctx context.Context, poolID int64) ([]service.QuotaPoolAccount, error) {
	rows, err := r.client.QuotaPoolAccount.Query().Where(quotapoolaccount.QuotaPoolIDEQ(poolID), quotapoolaccount.DeletedAtIsNil()).Order(dbent.Asc(quotapoolaccount.FieldID)).All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]service.QuotaPoolAccount, 0, len(rows))
	for _, row := range rows {
		out = append(out, quotaPoolAccountEntityToService(row))
	}
	return out, nil
}

func (r *quotaPoolRepository) AddAccount(ctx context.Context, account *service.QuotaPoolAccount) error {
	return r.addWeeklyAccount(ctx, account)
}

func (r *quotaPoolRepository) DeleteAccountBinding(ctx context.Context, bindingID int64) error {
	return r.deleteWeeklyAccount(ctx, bindingID)
}

func (r *quotaPoolRepository) UpdateAccountUsageSnapshot(ctx context.Context, bindingID int64, percent float64) error {
	_, err := r.client.QuotaPoolAccount.Update().Where(quotapoolaccount.IDEQ(bindingID), quotapoolaccount.DeletedAtIsNil()).SetLastUsagePercent(percent).Save(ctx)
	return err
}

func (r *quotaPoolRepository) CountEnabledPoolAccounts(ctx context.Context, poolID int64) (int, error) {
	return r.client.QuotaPoolAccount.Query().Where(quotapoolaccount.QuotaPoolIDEQ(poolID), quotapoolaccount.EnabledEQ(true), quotapoolaccount.DeletedAtIsNil()).Count(ctx)
}

func (r *quotaPoolRepository) FindEnabledPoolsByAccounts(ctx context.Context, accounts []service.QuotaPoolAccountRef) ([]service.QuotaPoolAccountPolicyRow, error) {
	if len(accounts) == 0 {
		return nil, nil
	}
	ids := make([]int64, 0, len(accounts))
	for _, account := range accounts {
		ids = append(ids, account.AccountID)
	}
	rows, err := r.client.QuotaPoolAccount.Query().
		Where(quotapoolaccount.AccountIDIn(ids...), quotapoolaccount.EnabledEQ(true), quotapoolaccount.DeletedAtIsNil()).
		WithQuotaPool(func(q *dbent.QuotaPoolQuery) {
			q.Where(quotapool.EnabledEQ(true), quotapool.DeletedAtIsNil())
		}).
		WithAccount().
		All(ctx)
	if err != nil {
		return nil, err
	}
	wanted := make(map[string]struct{}, len(accounts))
	for _, account := range accounts {
		wanted[fmt.Sprintf("%s:%d", account.Platform, account.AccountID)] = struct{}{}
	}
	out := make([]service.QuotaPoolAccountPolicyRow, 0, len(rows))
	for _, row := range rows {
		if _, ok := wanted[fmt.Sprintf("%s:%d", row.Platform, row.AccountID)]; !ok || row.Edges.QuotaPool == nil {
			continue
		}
		var account *service.Account
		if row.Edges.Account != nil {
			account = accountEntityToService(row.Edges.Account)
		}
		out = append(out, service.QuotaPoolAccountPolicyRow{
			Pool:    quotaPoolEntityToService(row.Edges.QuotaPool),
			Binding: quotaPoolAccountEntityToService(row),
			Account: account,
		})
	}
	return out, nil
}

func (r *quotaPoolRepository) ListGroupLimits(ctx context.Context, poolID int64) ([]service.QuotaPoolGroupLimit, error) {
	rows, err := r.client.QuotaPoolGroupLimit.Query().Where(quotapoolgrouplimit.QuotaPoolIDEQ(poolID), quotapoolgrouplimit.DeletedAtIsNil()).Order(dbent.Asc(quotapoolgrouplimit.FieldGroupID)).All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]service.QuotaPoolGroupLimit, 0, len(rows))
	for _, row := range rows {
		out = append(out, quotaPoolGroupLimitEntityToService(row))
	}
	return out, nil
}

func (r *quotaPoolRepository) UpsertGroupLimit(ctx context.Context, limit *service.QuotaPoolGroupLimit) error {
	return r.upsertWeeklyGroupLimit(ctx, limit)
}

func (r *quotaPoolRepository) DeleteGroupLimit(ctx context.Context, poolID, groupID int64) error {
	return r.deleteWeeklyGroupLimit(ctx, poolID, groupID)
}

func (r *quotaPoolRepository) GetEnabledGroupLimits(ctx context.Context, poolIDs []int64, groupID int64) ([]service.QuotaPoolGroupLimit, error) {
	if len(poolIDs) == 0 || groupID == 0 {
		return nil, nil
	}
	rows, err := r.client.QuotaPoolGroupLimit.Query().Where(
		quotapoolgrouplimit.QuotaPoolIDIn(poolIDs...),
		quotapoolgrouplimit.GroupIDEQ(groupID),
		quotapoolgrouplimit.EnabledEQ(true),
		quotapoolgrouplimit.DeletedAtIsNil(),
	).All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]service.QuotaPoolGroupLimit, 0, len(rows))
	for _, row := range rows {
		out = append(out, quotaPoolGroupLimitEntityToService(row))
	}
	return out, nil
}

func (r *quotaPoolRepository) ListGroupUsage(ctx context.Context, poolID int64, periodType, periodKey string) ([]service.QuotaPoolGroupUsage, error) {
	if periodType == service.QuotaPoolPeriodWeekly {
		return r.listWeeklyGroupUsage(ctx, poolID)
	}
	rows, err := r.sql.QueryContext(ctx, `SELECT MIN(id), quota_pool_id, group_id, period_type, period_key, COALESCE(SUM(used_cost), 0), MIN(created_at), MAX(updated_at)
		FROM quota_pool_group_usage
		WHERE quota_pool_id = $1 AND period_type = $2 AND period_key = $3
		GROUP BY quota_pool_id, group_id, period_type, period_key
		ORDER BY group_id`, poolID, periodType, periodKey)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []service.QuotaPoolGroupUsage{}
	for rows.Next() {
		var item service.QuotaPoolGroupUsage
		if err := rows.Scan(&item.ID, &item.QuotaPoolID, &item.GroupID, &item.PeriodType, &item.PeriodKey, &item.UsedCost, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r *quotaPoolRepository) GetGroupUsage(ctx context.Context, poolID, groupID int64, periodType, periodKey string) (*service.QuotaPoolGroupUsage, error) {
	rows, err := r.sql.QueryContext(ctx, `SELECT COALESCE(MIN(id), 0), $1, $2, $3, $4, COALESCE(SUM(used_cost), 0), COALESCE(MIN(created_at), NOW()), COALESCE(MAX(updated_at), NOW())
		FROM quota_pool_group_usage
		WHERE quota_pool_id = $1 AND group_id = $2 AND period_type = $3 AND period_key = $4`, poolID, groupID, periodType, periodKey)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, rows.Err()
	}
	var out service.QuotaPoolGroupUsage
	if err := rows.Scan(&out.ID, &out.QuotaPoolID, &out.GroupID, &out.PeriodType, &out.PeriodKey, &out.UsedCost, &out.CreatedAt, &out.UpdatedAt); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if out.ID == 0 {
		return nil, nil
	}
	return &out, nil
}

func (r *quotaPoolRepository) AddGroupAccountUsage(ctx context.Context, poolID, bindingID, groupID int64, periodType, periodKey string, usedPercent float64) error {
	_, err := r.sql.ExecContext(ctx, `INSERT INTO quota_pool_group_usage (quota_pool_id, quota_pool_account_id, group_id, period_type, period_key, used_cost, created_at, updated_at)
	VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
	ON CONFLICT (quota_pool_id, quota_pool_account_id, group_id, period_type, period_key)
	DO UPDATE SET used_cost = quota_pool_group_usage.used_cost + EXCLUDED.used_cost, updated_at = NOW()`, poolID, bindingID, groupID, periodType, periodKey, usedPercent)
	return err
}

func (r *quotaPoolRepository) ResetAccountUsageWindow(ctx context.Context, poolID, bindingID int64, periodType, periodKey string) error {
	_, err := r.sql.ExecContext(ctx, `DELETE FROM quota_pool_group_usage
		WHERE quota_pool_id = $1 AND quota_pool_account_id = $2 AND period_type = $3 AND period_key = $4`, poolID, bindingID, periodType, periodKey)
	return err
}

func (r *quotaPoolRepository) ResetGroupUsage(ctx context.Context, poolID, groupID int64, periodType, periodKey string) error {
	_, err := r.sql.ExecContext(ctx, `DELETE FROM quota_pool_group_usage
		WHERE quota_pool_id = $1 AND group_id = $2 AND period_type = $3 AND period_key = $4`, poolID, groupID, periodType, periodKey)
	return err
}

func quotaPoolEntityToService(row *dbent.QuotaPool) service.QuotaPool {
	return service.QuotaPool{ID: row.ID, Name: row.Name, Description: row.Description, Enabled: row.Enabled, PeriodType: row.PeriodType, DefaultAction: row.DefaultAction, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt, DeletedAt: row.DeletedAt}
}

func quotaPoolAccountEntityToService(row *dbent.QuotaPoolAccount) service.QuotaPoolAccount {
	return service.QuotaPoolAccount{ID: row.ID, QuotaPoolID: row.QuotaPoolID, Platform: row.Platform, AccountID: row.AccountID, Enabled: row.Enabled, LastUsagePercent: row.LastUsagePercent, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt, DeletedAt: row.DeletedAt}
}

func quotaPoolGroupLimitEntityToService(row *dbent.QuotaPoolGroupLimit) service.QuotaPoolGroupLimit {
	return service.QuotaPoolGroupLimit{ID: row.ID, QuotaPoolID: row.QuotaPoolID, GroupID: row.GroupID, PercentBasisPoints: row.PercentBasisPoints, Enabled: row.Enabled, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt, DeletedAt: row.DeletedAt}
}

func quotaPoolGroupUsageEntityToService(row *dbent.QuotaPoolGroupUsage) service.QuotaPoolGroupUsage {
	return service.QuotaPoolGroupUsage{ID: row.ID, QuotaPoolID: row.QuotaPoolID, QuotaPoolAccountID: row.QuotaPoolAccountID, GroupID: row.GroupID, PeriodType: row.PeriodType, PeriodKey: row.PeriodKey, UsedCost: row.UsedCost, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
}
