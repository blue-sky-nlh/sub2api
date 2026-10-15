package service

import (
	"context"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

var (
	ErrQuotaPoolNotFound = infraerrors.NotFound("QUOTA_POOL_NOT_FOUND", "quota pool not found")
	ErrQuotaPoolExists   = infraerrors.Conflict("QUOTA_POOL_EXISTS", "quota pool name already exists")
)

type QuotaPoolRepository interface {
	Create(ctx context.Context, pool *QuotaPool) error
	GetByID(ctx context.Context, id int64) (*QuotaPool, error)
	Update(ctx context.Context, pool *QuotaPool) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, params pagination.PaginationParams) ([]QuotaPool, *pagination.PaginationResult, error)
	ListAll(ctx context.Context) ([]QuotaPool, error)

	ListAccounts(ctx context.Context, poolID int64) ([]QuotaPoolAccount, error)
	AddAccount(ctx context.Context, account *QuotaPoolAccount) error
	DeleteAccountBinding(ctx context.Context, bindingID int64) error
	UpdateAccountUsageSnapshot(ctx context.Context, bindingID int64, percent float64) error
	CountEnabledPoolAccounts(ctx context.Context, poolID int64) (int, error)
	FindEnabledPoolsByAccounts(ctx context.Context, accounts []QuotaPoolAccountRef) ([]QuotaPoolAccountPolicyRow, error)

	ListGroupLimits(ctx context.Context, poolID int64) ([]QuotaPoolGroupLimit, error)
	UpsertGroupLimit(ctx context.Context, limit *QuotaPoolGroupLimit) error
	DeleteGroupLimit(ctx context.Context, poolID, groupID int64) error
	GetEnabledGroupLimits(ctx context.Context, poolIDs []int64, groupID int64) ([]QuotaPoolGroupLimit, error)

	ListGroupUsage(ctx context.Context, poolID int64, periodType, periodKey string) ([]QuotaPoolGroupUsage, error)
	GetGroupUsage(ctx context.Context, poolID, groupID int64, periodType, periodKey string) (*QuotaPoolGroupUsage, error)
	AddGroupAccountUsage(ctx context.Context, poolID, bindingID, groupID int64, periodType, periodKey string, usedPercent float64) error
	ResetAccountUsageWindow(ctx context.Context, poolID, bindingID int64, periodType, periodKey string) error
	ResetGroupUsage(ctx context.Context, poolID, groupID int64, periodType, periodKey string) error
}

type QuotaPool struct {
	ID            int64      `json:"id"`
	Name          string     `json:"name"`
	Description   *string    `json:"description,omitempty"`
	Enabled       bool       `json:"enabled"`
	PeriodType    string     `json:"period_type"`
	DefaultAction string     `json:"default_action"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	DeletedAt     *time.Time `json:"deleted_at,omitempty"`
}

type QuotaPoolAccount struct {
	ID               int64      `json:"id"`
	QuotaPoolID      int64      `json:"quota_pool_id"`
	Platform         string     `json:"platform"`
	AccountID        int64      `json:"account_id"`
	Enabled          bool       `json:"enabled"`
	LastUsagePercent float64    `json:"last_usage_percent"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	DeletedAt        *time.Time `json:"deleted_at,omitempty"`
}

type QuotaPoolGroupLimit struct {
	ID                 int64      `json:"id"`
	QuotaPoolID        int64      `json:"quota_pool_id"`
	GroupID            int64      `json:"group_id"`
	PercentBasisPoints int        `json:"percent_basis_points"`
	Enabled            bool       `json:"enabled"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
	DeletedAt          *time.Time `json:"deleted_at,omitempty"`
}

type QuotaPoolGroupUsage struct {
	ID                 int64     `json:"id"`
	QuotaPoolID        int64     `json:"quota_pool_id"`
	QuotaPoolAccountID int64     `json:"quota_pool_account_id,omitempty"`
	GroupID            int64     `json:"group_id"`
	PeriodType         string    `json:"period_type"`
	PeriodKey          string    `json:"period_key"`
	UsedCost           float64   `json:"used_cost"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type QuotaPoolAccountRef struct {
	Platform  string
	AccountID int64
}

type QuotaPoolAccountPolicyRow struct {
	Pool       QuotaPool
	Binding    QuotaPoolAccount
	Account    *Account
	GroupLimit *QuotaPoolGroupLimit
	Usage      *QuotaPoolGroupUsage
}

type CreateQuotaPoolRequest struct {
	Name          string  `json:"name"`
	Description   *string `json:"description"`
	Enabled       bool    `json:"enabled"`
	PeriodType    string  `json:"period_type"`
	DefaultAction string  `json:"default_action"`
}

type UpdateQuotaPoolRequest struct {
	Name          *string `json:"name"`
	Description   *string `json:"description"`
	Enabled       *bool   `json:"enabled"`
	PeriodType    *string `json:"period_type"`
	DefaultAction *string `json:"default_action"`
}

type AddQuotaPoolAccountRequest struct {
	Platform  string `json:"platform"`
	AccountID int64  `json:"account_id"`
	Enabled   bool   `json:"enabled"`
}

type UpsertQuotaPoolGroupLimitRequest struct {
	PercentBasisPoints int  `json:"percent_basis_points"`
	Enabled            bool `json:"enabled"`
}

type ResetQuotaPoolUsageRequest struct {
	GroupID int64 `json:"group_id"`
}
