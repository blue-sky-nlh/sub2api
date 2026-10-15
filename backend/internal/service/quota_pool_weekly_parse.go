package service

import (
	"context"
	"encoding/json"
	"github.com/Wei-Shaw/sub2api/internal/gatewayhooks"
	"log/slog"
	"time"
)

func (w *OpenAIRateLimitWindow) UnmarshalJSON(data []byte) error {
	type plain OpenAIRateLimitWindow
	var value plain
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	*w = OpenAIRateLimitWindow(value)
	raw, ok := fields["used_percent"]
	w.weeklyPercentPresent = ok && string(raw) != "null"
	return nil
}
func (s *OpenAIGatewayService) observeWeeklyHeaders(ctx context.Context, accountID int64, snapshot *OpenAICodexUsageSnapshot) {
	if s.gatewayHooks == nil {
		return
	}
	observed := weeklyHeaderSnapshot(snapshot)
	if observed == nil {
		return
	}
	if err := s.gatewayHooks.ObserveWeekly(context.WithoutCancel(ctx), accountID, observed); err != nil {
		if q, ok := gatewayhooks.AsQuotaError(err); !ok || q.Code != "quota_pool_binding_invalid" {
			slog.Error("weekly_quota_header_observation_failed", "account_id", accountID, "err", err)
		}
	}
}
func weeklyHeaderSnapshot(snapshot *OpenAICodexUsageSnapshot) *gatewayhooks.WeeklySnapshot {
	if snapshot == nil {
		return nil
	}
	at, err := time.Parse(time.RFC3339Nano, snapshot.UpdatedAt)
	if err != nil {
		return nil
	}
	result := &gatewayhooks.WeeklySnapshot{ObservedAt: at}
	hasWeek, hasShort := false, false
	windows := []struct {
		percent        *float64
		reset, minutes *int
	}{
		{snapshot.PrimaryUsedPercent, snapshot.PrimaryResetAfterSeconds, snapshot.PrimaryWindowMinutes},
		{snapshot.SecondaryUsedPercent, snapshot.SecondaryResetAfterSeconds, snapshot.SecondaryWindowMinutes},
	}
	for _, w := range windows {
		if w.percent == nil || w.reset == nil || w.minutes == nil || *w.reset <= 0 {
			continue
		}
		switch *w.minutes {
		case 10080:
			if hasWeek {
				return nil
			}
			hasWeek = true
			result.Percent = *w.percent
			result.ResetAt = at.Add(time.Duration(*w.reset) * time.Second)
		case 300:
			if hasShort {
				return nil
			}
			hasShort = true
			result.FiveHourPercent = *w.percent
			result.FiveHourResetAt = at.Add(time.Duration(*w.reset) * time.Second)
		}
	}
	if !hasWeek || !hasShort {
		return nil
	}
	return result
}
