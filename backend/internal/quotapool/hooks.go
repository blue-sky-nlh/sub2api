package quotapool

import (
	"context"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/gatewayhooks"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type Hook struct {
	repo service.QuotaPoolRepository
	now  func() time.Time
}

func NewHook(repo service.QuotaPoolRepository) *Hook { return &Hook{repo: repo, now: time.Now} }
func (h *Hook) weeklyRepo() (service.QuotaPoolWeeklyRepository, error) {
	if h != nil && h.repo != nil {
		if r, ok := h.repo.(service.QuotaPoolWeeklyRepository); ok {
			return r, nil
		}
	}
	return nil, &gatewayhooks.QuotaError{Status: 503, Code: "quota_pool_checker_unavailable", Message: "Weekly quota repository is unavailable"}
}
func quotaFailure(err error) error {
	if err == nil {
		return nil
	}
	if _, ok := gatewayhooks.AsQuotaError(err); ok {
		return err
	}
	return &gatewayhooks.QuotaError{Status: 503, Code: "quota_pool_statistics_unavailable", Message: "Weekly quota statistics could not be verified"}
}
func (h *Hook) CheckWeekly(ctx context.Context, account, group, pool int64, short bool) (int64, error) {
	r, err := h.weeklyRepo()
	if err != nil {
		return 0, err
	}
	resolved := int64(0)
	err = r.WithWeeklyAccount(ctx, account, pool, func(v *service.QuotaPoolWeeklyView, m *service.QuotaPoolWeeklyMember) error {
		if err := service.WeeklyAdmission(v, m, group, h.now(), short); err != nil {
			return err
		}
		resolved = v.Pool.ID
		return nil
	})
	return resolved, quotaFailure(err)
}
func (h *Hook) BeforeAccountSelect(ctx context.Context, in gatewayhooks.BeforeAccountSelectInput) (gatewayhooks.BeforeAccountSelectResult, error) {
	out := gatewayhooks.BeforeAccountSelectResult{Candidates: in.Candidates}
	originalGroup, expectedPool, controlled := service.WeeklyQuotaScopeValues(ctx)
	if controlled {
		in.GroupID = originalGroup
		in.Platform = service.PlatformOpenAI
	}
	if !controlled {
		return out, nil
	}
	if _, err := h.weeklyRepo(); err != nil {
		return out, err
	}
	// A commercial OpenAI request is controlled before binding lookup, including
	// zero candidates. Missing/deleted bindings never turn it into pool-free traffic.
	if in.GroupID <= 0 {
		return out, &gatewayhooks.QuotaError{Status: 403, Code: "quota_pool_forbidden", Message: "An authenticated group is required"}
	}
	if len(in.Candidates) == 0 {
		repo, err := h.weeklyRepo()
		if err != nil {
			return out, err
		}
		ids, err := repo.ListWeeklyGroupAccountIDs(ctx, in.GroupID)
		if err != nil {
			return out, quotaFailure(err)
		}
		if len(ids) == 0 {
			return out, &gatewayhooks.QuotaError{Status: 403, Code: "quota_pool_forbidden", Message: "No authorized weekly pool members"}
		}
		for _, id := range ids {
			if _, err := h.CheckWeekly(ctx, id, in.GroupID, expectedPool, false); err != nil {
				return out, err
			}
		}
		return out, nil
	}
	filtered := make([]gatewayhooks.AccountRef, 0, len(in.Candidates))
	var shortErr error
	for _, a := range in.Candidates {
		if a.Platform != service.PlatformOpenAI {
			return out, &gatewayhooks.QuotaError{Status: 403, Code: "quota_pool_scope_mismatch", Message: "Cross-platform quota fallback is forbidden"}
		}
		_, err := h.CheckWeekly(ctx, a.AccountID, in.GroupID, expectedPool, true)
		if e, ok := gatewayhooks.AsQuotaError(err); ok && (e.Code == "quota_pool_account_short_term_limited" || e.Code == "quota_pool_account_weekly_limited") {
			shortErr = err
			continue
		}
		if err != nil {
			return out, err
		}
		filtered = append(filtered, a)
	}
	if len(filtered) == 0 && shortErr != nil {
		return out, shortErr
	}
	out.Candidates = filtered
	return out, nil
}

// Billing result is deliberately not weekly consumption evidence. Dispatch
// lifecycle below retains failed/cancelled attempts independently of billing.
func (h *Hook) AfterUsageRecorded(context.Context, gatewayhooks.AfterUsageRecordedInput) error {
	return nil
}
func (h *Hook) BeginWeekly(ctx context.Context, a gatewayhooks.WeeklyAttempt) (int64, error) {
	r, err := h.weeklyRepo()
	if err != nil {
		return 0, err
	}
	pool := int64(0)
	err = r.WithWeeklyAccount(ctx, a.AccountID, a.PoolID, func(v *service.QuotaPoolWeeklyView, m *service.QuotaPoolWeeklyMember) error {
		if err := service.WeeklyAdmission(v, m, a.GroupID, h.now(), true); err != nil {
			return err
		}
		pool = v.Pool.ID
		return service.ApplyWeeklyMutation(m, service.QuotaPoolWeeklyMutation{Attempt: &service.QuotaPoolWeeklyAttempt{
			ID: a.ID, GroupID: a.GroupID, StartedAt: a.StartedAt,
		}})
	})
	return pool, quotaFailure(err)
}
func (h *Hook) FinishWeekly(ctx context.Context, a gatewayhooks.WeeklyAttempt) error {
	r, err := h.weeklyRepo()
	if err != nil {
		return err
	}
	return r.WithWeeklyAccount(ctx, a.AccountID, a.PoolID, func(v *service.QuotaPoolWeeklyView, m *service.QuotaPoolWeeklyMember) error {
		return service.ApplyWeeklyMutation(m, service.QuotaPoolWeeklyMutation{Finish: &service.QuotaPoolWeeklyAttempt{
			ID: a.ID, FinishedAt: a.FinishedAt, Weight: a.Weight, Known: a.Known,
		}})
	})
}
func (h *Hook) ObserveWeekly(ctx context.Context, id int64, s *gatewayhooks.WeeklySnapshot) error {
	r, err := h.weeklyRepo()
	if err != nil {
		return err
	}
	return r.WithWeeklyAccount(ctx, id, 0, func(v *service.QuotaPoolWeeklyView, m *service.QuotaPoolWeeklyMember) error {
		in := service.QuotaPoolWeeklyMutation{RefreshError: s == nil}
		if s != nil {
			observation := *s
			s = &observation
			if s.Plan == "" || s.Identity == "" {
				if m.State.Snapshot == nil {
					return service.ApplyWeeklyMutation(m, service.QuotaPoolWeeklyMutation{RefreshError: true})
				}
				s.Plan = m.State.Snapshot.Plan
				s.Identity = m.State.Snapshot.Identity
			}
			in.Snapshot = &service.QuotaPoolWeeklySnapshot{
				Percent: s.Percent, ResetAt: s.ResetAt, ObservedAt: s.ObservedAt,
				FiveHourPercent: s.FiveHourPercent, FiveHourResetAt: s.FiveHourResetAt, Plan: s.Plan, Identity: s.Identity,
			}
		}
		return service.ApplyWeeklyMutation(m, in)
	})
}
func (h *Hook) MarkWeeklyReset(ctx context.Context, a gatewayhooks.WeeklyReset) error {
	r, err := h.weeklyRepo()
	if err != nil {
		return err
	}
	return r.WithWeeklyAccount(ctx, a.AccountID, 0, func(v *service.QuotaPoolWeeklyView, m *service.QuotaPoolWeeklyMember) error {
		return service.ApplyWeeklyMutation(m, service.QuotaPoolWeeklyMutation{Reset: &service.QuotaPoolWeeklyReset{ID: a.ID, At: a.At}})
	})
}
func (h *Hook) WeeklyAccountIDs(ctx context.Context) ([]int64, error) {
	r, err := h.weeklyRepo()
	if err != nil {
		return nil, err
	}
	return r.ListWeeklyAccountIDs(ctx)
}

var _ gatewayhooks.Hook = (*Hook)(nil)
var _ gatewayhooks.WeeklyHook = (*Hook)(nil)
