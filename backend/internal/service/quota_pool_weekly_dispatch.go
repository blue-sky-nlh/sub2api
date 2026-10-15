package service

import (
	"context"
	"github.com/gin-gonic/gin"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/gatewayhooks"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type weeklyScopeKey struct{}
type weeklyBusinessKey struct{}
type weeklyBusinessVerifiedKey struct{}
type weeklyCatalogReadKey struct{}

func WithWeeklyQuotaCatalogRead(ctx context.Context) context.Context {
	return context.WithValue(ctx, weeklyCatalogReadKey{}, true)
}
func WeeklyQuotaCatalogRead(ctx context.Context) bool {
	read, _ := ctx.Value(weeklyCatalogReadKey{}).(bool)
	return read
}

type weeklyQuotaScope struct {
	GroupID int64
	PoolID  int64
	mu      sync.Mutex
}

// Commercial scope derives from authenticated business routing, never from a
// successful pool lookup. It is shared across retries and composite subgroups.
func WithWeeklyQuotaScope(ctx context.Context, groupID *int64, group *Group) context.Context {
	if _, ok := ctx.Value(weeklyScopeKey{}).(*weeklyQuotaScope); ok {
		return ctx
	}
	if group == nil {
		return ctx
	}
	ctx = context.WithValue(ctx, weeklyBusinessKey{}, group.Platform)
	if group.Platform != PlatformOpenAI {
		return ctx
	}
	id := int64(0)
	if groupID != nil {
		id = *groupID
	}
	return context.WithValue(ctx, weeklyScopeKey{}, &weeklyQuotaScope{GroupID: id})
}

type weeklyForwardKey struct{}
type weeklyForwardTrace struct {
	mu        sync.Mutex
	attempts  []*gatewayhooks.WeeklyAttempt
	websocket bool
}

func (s *OpenAIGatewayService) prepareWeeklyForward(ctx context.Context, account *Account) (context.Context, *weeklyForwardTrace, error) {
	if err := s.CheckWeeklyQuota(ctx, account, nil); err != nil {
		return ctx, nil, err
	}
	if !WeeklyQuotaControlled(ctx) {
		return ctx, nil, nil
	}
	if _, ok := ctx.Value(weeklyForwardKey{}).(*weeklyForwardTrace); ok {
		return ctx, nil, nil
	}
	if err := s.CheckWeeklyQuota(ctx, account, nil); err != nil {
		return ctx, nil, err
	}
	trace := &weeklyForwardTrace{}
	return context.WithValue(ctx, weeklyForwardKey{}, trace), trace, nil
}
func (s *OpenAIGatewayService) recordWeeklyHTTPDispatch(request *http.Request, account *Account) error {
	return s.recordWeeklyDispatch(request.Context(), account)
}
func (s *OpenAIGatewayService) recordWeeklyDispatch(ctx context.Context, account *Account) error {
	if !WeeklyQuotaControlled(ctx) {
		return nil
	}
	trace, ok := ctx.Value(weeklyForwardKey{}).(*weeklyForwardTrace)
	if !ok {
		return &gatewayhooks.QuotaError{Status: 503, Code: "quota_pool_evidence_unavailable", Message: "Weekly dispatch evidence is not initialized"}
	}
	if trace.websocket {
		return s.CheckWeeklyQuota(ctx, account, nil)
	}
	a, err := s.BeginWeeklyQuotaAttempt(ctx, account)
	if err != nil {
		return err
	}
	trace.mu.Lock()
	trace.attempts = append(trace.attempts, a)
	trace.mu.Unlock()
	return nil
}
func (s *OpenAIGatewayService) finishWeeklyForward(trace *weeklyForwardTrace, result *OpenAIForwardResult) {
	if trace == nil {
		return
	}
	trace.mu.Lock()
	attempts := append([]*gatewayhooks.WeeklyAttempt(nil), trace.attempts...)
	trace.mu.Unlock()
	for i, a := range attempts {
		// Only the final response has token evidence. Earlier transport/recovery
		// attempts stay explicitly uncertain instead of borrowing its usage.
		if i == len(attempts)-1 {
			s.FinishWeeklyQuotaAttempt(a, result)
		} else {
			s.FinishWeeklyQuotaAttempt(a, nil)
		}
	}
}
func weeklyForwardContext(ctx context.Context, c *gin.Context) context.Context {
	if c != nil && c.Request != nil && !WeeklyQuotaControlled(ctx) {
		if scope, ok := c.Request.Context().Value(weeklyScopeKey{}).(*weeklyQuotaScope); ok {
			return context.WithValue(ctx, weeklyScopeKey{}, scope)
		}
	}
	return ctx
}
func WeeklyQuotaScopeValues(ctx context.Context) (int64, int64, bool) {
	scope, ok := ctx.Value(weeklyScopeKey{}).(*weeklyQuotaScope)
	if !ok {
		return 0, 0, false
	}
	scope.mu.Lock()
	defer scope.mu.Unlock()
	return scope.GroupID, scope.PoolID, true
}
func WeeklyQuotaControlled(ctx context.Context) bool {
	_, ok := ctx.Value(weeklyScopeKey{}).(*weeklyQuotaScope)
	return ok
}
func (s *OpenAIGatewayService) CheckWeeklyQuota(ctx context.Context, account *Account, groupID *int64) error {
	verified, _ := ctx.Value(weeklyBusinessVerifiedKey{}).(bool)
	if s != nil && s.weeklyQuotaRequired && !verified {
		return &gatewayhooks.QuotaError{Status: 503, Code: "quota_pool_scope_unavailable", Message: "Authenticated quota business scope is unavailable"}
	}
	scope, ok := ctx.Value(weeklyScopeKey{}).(*weeklyQuotaScope)
	if !ok {
		return nil
	}
	if s == nil || s.gatewayHooks == nil {
		return &gatewayhooks.QuotaError{Status: 503, Code: "quota_pool_checker_unavailable", Message: "Weekly quota checker is unavailable"}
	}
	if account == nil || account.Platform != PlatformOpenAI || account.Type != AccountTypeOAuth || account.IsShadow() {
		return &gatewayhooks.QuotaError{Status: 403, Code: "quota_pool_scope_mismatch", Message: "Weekly quota traffic requires an independent GPT OAuth pool account"}
	}
	scope.mu.Lock()
	defer scope.mu.Unlock()
	pool, err := s.gatewayHooks.CheckWeekly(ctx, account.ID, scope.GroupID, scope.PoolID, true)
	if err == nil {
		scope.PoolID = pool
	}
	return err
}
func (s *OpenAIGatewayService) BeginWeeklyQuotaAttempt(ctx context.Context, account *Account) (*gatewayhooks.WeeklyAttempt, error) {
	scope, ok := ctx.Value(weeklyScopeKey{}).(*weeklyQuotaScope)
	if !ok {
		return nil, nil
	}
	if err := s.CheckWeeklyQuota(ctx, account, nil); err != nil {
		return nil, err
	}
	scope.mu.Lock()
	defer scope.mu.Unlock()
	a := &gatewayhooks.WeeklyAttempt{ID: uuid.NewString(), AccountID: account.ID, GroupID: scope.GroupID, PoolID: scope.PoolID, StartedAt: time.Now().UTC()}
	pool, err := s.gatewayHooks.BeginWeekly(ctx, *a)
	if err != nil {
		return nil, err
	}
	scope.PoolID = pool
	a.PoolID = pool
	return a, nil
}

// Frozen raw token weights are only an interval attribution estimate. Cached
// tokens have a fixed 0.1 weight and output tokens 4, independent of group bills.
func (s *OpenAIGatewayService) FinishWeeklyQuotaAttempt(a *gatewayhooks.WeeklyAttempt, result *OpenAIForwardResult) {
	if a == nil {
		return
	}
	finished := *a
	finished.FinishedAt = time.Now().UTC()
	if result != nil {
		u := result.Usage
		if u.InputTokens >= 0 && u.OutputTokens >= 0 && u.CacheReadInputTokens >= 0 && u.CacheReadInputTokens <= u.InputTokens {
			w := decimal.NewFromInt(int64(u.InputTokens - u.CacheReadInputTokens)).
				Add(decimal.NewFromInt(int64(u.CacheReadInputTokens)).Mul(decimal.RequireFromString("0.1"))).
				Add(decimal.NewFromInt(int64(u.OutputTokens)).Mul(decimal.NewFromInt(4)))
			finished.Weight = w.String()
			finished.Known = w.IsPositive()
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.gatewayHooks.FinishWeekly(ctx, finished); err != nil {
		// Begin evidence remains persisted and makes new consumption pending. No
		// billing success or failed completion can silently erase this attempt.
		slog.Error("weekly_quota_attempt_finish_failed", "account_id", a.AccountID, "attempt_id", a.ID, "err", err)
	}
}
