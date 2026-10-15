package service

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/gatewayhooks"
)

// Snapshot parsing never guesses window roles or treats missing percentages as 0.
func weeklySnapshotFromUsage(u *OpenAIQuotaUsage) *gatewayhooks.WeeklySnapshot {
	if u == nil || u.RateLimit == nil {
		return nil
	}
	var week, short *OpenAIRateLimitWindow
	for _, w := range []*OpenAIRateLimitWindow{u.RateLimit.PrimaryWindow, u.RateLimit.SecondaryWindow} {
		if w == nil {
			continue
		}
		switch w.LimitWindowSeconds {
		case 604800:
			if week != nil {
				return nil
			}
			week = w
		case 18000:
			if short != nil {
				return nil
			}
			short = w
		}
	}
	if week == nil || short == nil || !week.weeklyPercentPresent || !short.weeklyPercentPresent || week.ResetAt <= 0 || short.ResetAt <= 0 {
		return nil
	}
	at := u.weeklyObservedAt
	if at.IsZero() {
		at = time.Unix(u.FetchedAt, 0)
	}
	identity := u.weeklyIdentity
	if identity == "" {
		identity = u.AccountID
	}
	s := &gatewayhooks.WeeklySnapshot{Percent: week.UsedPercent, ResetAt: time.Unix(week.ResetAt, 0),
		ObservedAt: at, FiveHourPercent: short.UsedPercent, FiveHourResetAt: time.Unix(short.ResetAt, 0),
		Plan: u.PlanType, Identity: identity}
	if !validWeeklySnapshot(&QuotaPoolWeeklySnapshot{
		Percent: s.Percent, ResetAt: s.ResetAt, ObservedAt: s.ObservedAt,
		FiveHourPercent: s.FiveHourPercent, FiveHourResetAt: s.FiveHourResetAt, Plan: s.Plan, Identity: s.Identity,
	}) {
		return nil
	}
	return s
}
func (s *OpenAIQuotaService) observeWeeklyUsage(ctx context.Context, id int64, u *OpenAIQuotaUsage) error {
	if s.weeklyHooks == nil {
		return nil
	} // standalone non-commercial quota-card calls
	err := s.weeklyHooks.ObserveWeekly(ctx, id, weeklySnapshotFromUsage(u))
	if e, ok := gatewayhooks.AsQuotaError(err); ok && e.Code == "quota_pool_binding_invalid" {
		return nil
	}
	return err
}

// Bounded read-only refresh survives denied traffic and restart. Each process
// may query, but persistent timestamps and row locks prevent duplicate settlement.
func (s *OpenAIQuotaService) StartWeeklyRefresh(hooks *gatewayhooks.Manager) {
	s.weeklyHooks = hooks
	ctx, cancel := context.WithCancel(context.Background())
	s.weeklyCancel = cancel
	s.weeklyWG.Add(1)
	go func() {
		defer s.weeklyWG.Done()
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		next := map[int64]time.Time{}
		failures := map[int64]int{}
		var refreshMu sync.Mutex
		for {
			ids, err := hooks.WeeklyAccountIDs(ctx)
			if err != nil {
				slog.Error("weekly_quota_refresh_members_failed", "err", err)
			}
			var wg sync.WaitGroup
			sem := make(chan struct{}, 2)
			for _, id := range ids {
				refreshMu.Lock()
				skip := time.Now().Before(next[id])
				refreshMu.Unlock()
				if skip {
					continue
				}
				select {
				case sem <- struct{}{}:
				case <-ctx.Done():
					wg.Wait()
					return
				}
				wg.Add(1)
				// Result collection is mutex protected; selection only reads before launches.
				go func(id int64) {
					defer wg.Done()
					defer func() { <-sem }()
					qctx, qcancel := context.WithTimeout(ctx, 45*time.Second)
					defer qcancel()
					usage, e := s.QueryUsage(qctx, id)
					if e == nil {
						e = s.accountRepo.UpdateExtra(qctx, id, buildOpenAIAutoResetUsageUpdates(usage, time.Now()))
					}
					refreshMu.Lock()
					if e != nil {
						failures[id]++
						backoff := time.Duration(1<<min(failures[id], 3)) * 15 * time.Second
						next[id] = time.Now().Add(backoff)
						slog.Warn("weekly_quota_refresh_failed", "account_id", id, "err", e)
					} else {
						failures[id] = 0
						next[id] = time.Now().Add(30 * time.Second)
					}
					refreshMu.Unlock()
				}(id)
			}
			wg.Wait()
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}
func (s *OpenAIQuotaService) StopWeeklyRefresh() {
	if s.weeklyCancel != nil {
		s.weeklyCancel()
		s.weeklyWG.Wait()
	}
}
