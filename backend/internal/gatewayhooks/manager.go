package gatewayhooks

import (
	"context"
	"log/slog"
)

type Manager struct {
	policyHooks []PolicyHook
	usageHooks  []UsageHook
}

func NewManager(hooks ...Hook) *Manager {
	m := &Manager{}
	for _, hook := range hooks {
		m.RegisterHook(hook)
	}
	return m
}

func (m *Manager) RegisterHook(hook Hook) {
	if m == nil || hook == nil {
		return
	}
	m.policyHooks = append(m.policyHooks, hook)
	m.usageHooks = append(m.usageHooks, hook)
}

func (m *Manager) BeforeAccountSelect(ctx context.Context, input BeforeAccountSelectInput) BeforeAccountSelectResult {
	result := BeforeAccountSelectResult{Candidates: input.Candidates}
	if m == nil || len(m.policyHooks) == 0 {
		return result
	}
	current := input
	for _, hook := range m.policyHooks {
		current.Candidates = result.Candidates
		next, err := hook.BeforeAccountSelect(ctx, current)
		if err != nil {
			if _, quota := AsQuotaError(err); quota {
				result.Candidates = []AccountRef{}
				result.Error = err
				return result
			}
			slog.Warn("gateway hook policy failed", "err", err)
			continue
		}
		if next.Candidates != nil {
			result.Candidates = next.Candidates
		}
	}
	return result
}

func (m *Manager) AfterUsageRecorded(ctx context.Context, input AfterUsageRecordedInput) {
	if m == nil || len(m.usageHooks) == 0 {
		return
	}
	for _, hook := range m.usageHooks {
		if err := hook.AfterUsageRecorded(ctx, input); err != nil {
			slog.Error("gateway hook usage persistence failed", "err", err)
		}
	}
}
