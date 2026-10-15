package service

import (
	"context"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

const (
	QuotaPoolPeriodCurrent = "current"
	QuotaPoolDefaultAllow  = "allow"
)

type QuotaPoolService struct {
	repo QuotaPoolRepository
}

func NewQuotaPoolService(repo QuotaPoolRepository) *QuotaPoolService {
	return &QuotaPoolService{repo: repo}
}

func (s *QuotaPoolService) Create(ctx context.Context, req CreateQuotaPoolRequest) (*QuotaPool, error) {
	pool := &QuotaPool{
		Name:          strings.TrimSpace(req.Name),
		Description:   req.Description,
		Enabled:       req.Enabled,
		PeriodType:    QuotaPoolPeriodWeekly,
		DefaultAction: normalizeQuotaPoolDefaultAction(req.DefaultAction),
	}
	if err := s.repo.Create(ctx, pool); err != nil {
		return nil, err
	}
	return pool, nil
}

func (s *QuotaPoolService) GetByID(ctx context.Context, id int64) (*QuotaPool, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *QuotaPoolService) Update(ctx context.Context, id int64, req UpdateQuotaPoolRequest) (*QuotaPool, error) {
	pool, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if req.Name != nil {
		pool.Name = strings.TrimSpace(*req.Name)
	}
	if req.Description != nil {
		pool.Description = req.Description
	}
	if req.Enabled != nil {
		pool.Enabled = *req.Enabled
	}
	pool.PeriodType = QuotaPoolPeriodWeekly
	pool.DefaultAction = QuotaPoolDefaultDeny
	if req.DefaultAction != nil {
		pool.DefaultAction = normalizeQuotaPoolDefaultAction(*req.DefaultAction)
	}
	if err := s.repo.Update(ctx, pool); err != nil {
		return nil, err
	}
	return pool, nil
}

func (s *QuotaPoolService) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}

func (s *QuotaPoolService) List(ctx context.Context, params pagination.PaginationParams) ([]QuotaPool, *pagination.PaginationResult, error) {
	return s.repo.List(ctx, params)
}

func (s *QuotaPoolService) ListAll(ctx context.Context) ([]QuotaPool, error) {
	return s.repo.ListAll(ctx)
}

func (s *QuotaPoolService) ListAccounts(ctx context.Context, poolID int64) ([]QuotaPoolAccount, error) {
	return s.repo.ListAccounts(ctx, poolID)
}

func (s *QuotaPoolService) AddAccount(ctx context.Context, poolID int64, req AddQuotaPoolAccountRequest) (*QuotaPoolAccount, error) {
	account := &QuotaPoolAccount{
		QuotaPoolID: poolID,
		Platform:    strings.TrimSpace(req.Platform),
		AccountID:   req.AccountID,
		Enabled:     req.Enabled,
	}
	if err := s.repo.AddAccount(ctx, account); err != nil {
		return nil, err
	}
	return account, nil
}

func (s *QuotaPoolService) DeleteAccountBinding(ctx context.Context, bindingID int64) error {
	return s.repo.DeleteAccountBinding(ctx, bindingID)
}

func (s *QuotaPoolService) ListGroupLimits(ctx context.Context, poolID int64) ([]QuotaPoolGroupLimit, error) {
	return s.repo.ListGroupLimits(ctx, poolID)
}

func (s *QuotaPoolService) UpsertGroupLimit(ctx context.Context, poolID, groupID int64, req UpsertQuotaPoolGroupLimitRequest) (*QuotaPoolGroupLimit, error) {
	limit := &QuotaPoolGroupLimit{
		QuotaPoolID:        poolID,
		GroupID:            groupID,
		PercentBasisPoints: req.PercentBasisPoints,
		Enabled:            req.Enabled,
	}
	if err := s.repo.UpsertGroupLimit(ctx, limit); err != nil {
		return nil, err
	}
	return limit, nil
}

func (s *QuotaPoolService) DeleteGroupLimit(ctx context.Context, poolID, groupID int64) error {
	return s.repo.DeleteGroupLimit(ctx, poolID, groupID)
}

func (s *QuotaPoolService) ListGroupUsage(ctx context.Context, poolID int64, periodType, periodKey string) ([]QuotaPoolGroupUsage, error) {
	return s.repo.ListGroupUsage(ctx, poolID, QuotaPoolPeriodWeekly, QuotaPoolPeriodCurrent)
}

func (s *QuotaPoolService) ResetGroupUsage(ctx context.Context, poolID int64, req ResetQuotaPoolUsageRequest) error {
	r, ok := s.repo.(QuotaPoolWeeklyRepository)
	if !ok {
		return ErrQuotaPoolNotFound
	}
	return r.ResetWeeklyGroup(ctx, poolID, req.GroupID, quotaPoolAdjustmentActor(ctx))
}

func normalizeQuotaPoolPeriod(value string) string {
	return QuotaPoolPeriodWeekly
}

func normalizeQuotaPoolDefaultAction(value string) string {
	return QuotaPoolDefaultDeny
}
