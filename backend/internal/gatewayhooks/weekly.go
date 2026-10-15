package gatewayhooks

import (
	"context"
	"errors"
	"fmt"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"time"
)

type QuotaError struct {
	Status        int
	Code, Message string
}

func (e *QuotaError) Unwrap() error { return infraerrors.New(e.Status, e.Code, e.Message) }
func (e *QuotaError) Error() string { return fmt.Sprintf("%s: %s", e.Code, e.Message) }
func AsQuotaError(err error) (*QuotaError, bool) {
	var e *QuotaError
	ok := errors.As(err, &e)
	return e, ok
}

type WeeklySnapshot struct {
	Percent             float64
	ResetAt, ObservedAt time.Time
	FiveHourPercent     float64
	FiveHourResetAt     time.Time
	Plan, Identity      string
}
type WeeklyAttempt struct {
	ID                         string
	AccountID, GroupID, PoolID int64
	StartedAt, FinishedAt      time.Time
	Weight                     string
	Known                      bool
}
type WeeklyReset struct {
	ID        string
	AccountID int64
	At        time.Time
}
type WeeklyHook interface {
	CheckWeekly(context.Context, int64, int64, int64, bool) (int64, error)
	BeginWeekly(context.Context, WeeklyAttempt) (int64, error)
	FinishWeekly(context.Context, WeeklyAttempt) error
	ObserveWeekly(context.Context, int64, *WeeklySnapshot) error
	MarkWeeklyReset(context.Context, WeeklyReset) error
	WeeklyAccountIDs(context.Context) ([]int64, error)
}

func (m *Manager) WeeklyCheckerReady() bool { _, err := m.weeklyHook(); return err == nil }
func (m *Manager) weeklyHook() (WeeklyHook, error) {
	if m != nil {
		for _, h := range m.policyHooks {
			if w, ok := h.(WeeklyHook); ok {
				return w, nil
			}
		}
	}
	return nil, &QuotaError{503, "quota_pool_checker_unavailable", "Weekly quota checker is unavailable"}
}
func (m *Manager) CheckWeekly(ctx context.Context, a, g, p int64, short bool) (int64, error) {
	h, e := m.weeklyHook()
	if e != nil {
		return 0, e
	}
	return h.CheckWeekly(ctx, a, g, p, short)
}
func (m *Manager) BeginWeekly(ctx context.Context, a WeeklyAttempt) (int64, error) {
	h, e := m.weeklyHook()
	if e != nil {
		return 0, e
	}
	return h.BeginWeekly(ctx, a)
}
func (m *Manager) FinishWeekly(ctx context.Context, a WeeklyAttempt) error {
	h, e := m.weeklyHook()
	if e != nil {
		return e
	}
	return h.FinishWeekly(ctx, a)
}
func (m *Manager) ObserveWeekly(ctx context.Context, id int64, s *WeeklySnapshot) error {
	h, e := m.weeklyHook()
	if e != nil {
		return e
	}
	return h.ObserveWeekly(ctx, id, s)
}
func (m *Manager) MarkWeeklyReset(ctx context.Context, r WeeklyReset) error {
	h, e := m.weeklyHook()
	if e != nil {
		return e
	}
	return h.MarkWeeklyReset(ctx, r)
}
func (m *Manager) WeeklyAccountIDs(ctx context.Context) ([]int64, error) {
	h, e := m.weeklyHook()
	if e != nil {
		return nil, e
	}
	return h.WeeklyAccountIDs(ctx)
}
