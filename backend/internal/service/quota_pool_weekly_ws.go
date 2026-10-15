package service

import (
	"context"
	"sync"

	"github.com/Wei-Shaw/sub2api/internal/gatewayhooks"
)

// Both ingress transports share turn evidence. The first frame gets an explicit
// admission because passthrough does not invoke its first BeforeTurn callback.
func (s *OpenAIGatewayService) prepareWeeklyWS(ctx context.Context, account *Account, hooks *OpenAIWSIngressHooks) (
	context.Context, *OpenAIWSIngressHooks, func(), error,
) {
	if err := s.CheckWeeklyQuota(ctx, account, nil); err != nil {
		return ctx, hooks, func() {}, err
	}
	if !WeeklyQuotaControlled(ctx) {
		return ctx, hooks, func() {}, nil
	}
	a, err := s.BeginWeeklyQuotaAttempt(ctx, account)
	if err != nil {
		return ctx, hooks, func() {}, err
	}
	copyHooks := OpenAIWSIngressHooks{}
	if hooks != nil {
		copyHooks = *hooks
	}
	oldBefore, oldAfter := copyHooks.BeforeTurn, copyHooks.AfterTurn
	var mu sync.Mutex
	attempts := map[int]*gatewayhooks.WeeklyAttempt{1: a}
	copyHooks.BeforeTurn = func(turn int) error {
		if oldBefore != nil {
			if err := oldBefore(turn); err != nil {
				return err
			}
		}
		if err := s.CheckWeeklyQuota(ctx, account, nil); err != nil {
			return err
		}
		if turn == 1 {
			return nil
		}
		next, err := s.BeginWeeklyQuotaAttempt(ctx, account)
		if err != nil {
			return err
		}
		mu.Lock()
		attempts[turn] = next
		mu.Unlock()
		return nil
	}
	copyHooks.AfterTurn = func(turn int, result *OpenAIForwardResult, err error) {
		mu.Lock()
		attempt := attempts[turn]
		delete(attempts, turn)
		mu.Unlock()
		s.FinishWeeklyQuotaAttempt(attempt, result)
		if oldAfter != nil {
			oldAfter(turn, result, err)
		}
	}
	cleanup := func() {
		mu.Lock()
		pending := attempts
		attempts = map[int]*gatewayhooks.WeeklyAttempt{}
		mu.Unlock()
		for _, attempt := range pending {
			s.FinishWeeklyQuotaAttempt(attempt, nil)
		}
	}
	ctx = context.WithValue(ctx, weeklyForwardKey{}, &weeklyForwardTrace{websocket: true})
	return ctx, &copyHooks, cleanup, nil
}
