package service

import (
	"context"
	"github.com/Wei-Shaw/sub2api/internal/gatewayhooks"
)

type QuotaPoolWeeklyBusinessRepository interface {
	WeeklyGroupRequired(context.Context, int64) (bool, error)
}

func (s *APIKeyService) WeeklyQuotaBusinessContext(ctx context.Context, key *APIKey) (context.Context, error) {
	if !s.weeklyScopeRequired {
		return WithWeeklyQuotaOrdinaryBusiness(ctx, key.Group), nil
	}
	r, ok := s.weeklyScopeRepo.(QuotaPoolWeeklyBusinessRepository)
	if !ok {
		return ctx, &gatewayhooks.QuotaError{Status: 503, Code: "quota_pool_scope_unavailable", Message: "Commercial quota scope repository is unavailable"}
	}
	id := int64(0)
	if key.GroupID != nil {
		id = *key.GroupID
	}
	required, err := r.WeeklyGroupRequired(ctx, id)
	if err != nil {
		return ctx, &gatewayhooks.QuotaError{Status: 503, Code: "quota_pool_scope_unavailable", Message: "Commercial quota scope could not be verified"}
	}
	ctx = context.WithValue(ctx, weeklyBusinessVerifiedKey{}, true)
	ctx = WithWeeklyQuotaOrdinaryBusiness(ctx, key.Group)
	if required {
		return context.WithValue(ctx, weeklyScopeKey{}, &weeklyQuotaScope{GroupID: id}), nil
	}
	return ctx, nil
}
func WithWeeklyQuotaOrdinaryBusiness(ctx context.Context, group *Group) context.Context {
	if WeeklyQuotaControlled(ctx) {
		return ctx
	}
	if _, ok := ctx.Value(weeklyBusinessKey{}).(string); ok {
		return ctx
	}
	platform := "ungrouped"
	if group != nil {
		platform = group.Platform
	}
	return context.WithValue(ctx, weeklyBusinessKey{}, platform)
}
