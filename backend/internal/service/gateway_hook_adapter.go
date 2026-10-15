package service

import (
	"context"

	"github.com/Wei-Shaw/sub2api/internal/gatewayhooks"
)

func filterAccountsByGatewayHooks(ctx context.Context, manager *gatewayhooks.Manager, accounts []Account, platform string, groupID *int64, userID int64, model string) ([]Account, error) {
	if WeeklyQuotaCatalogRead(ctx) {
		return accounts, nil
	}
	if manager == nil || (WeeklyQuotaControlled(ctx) && !manager.WeeklyCheckerReady()) {
		if platform == PlatformOpenAI && WeeklyQuotaControlled(ctx) {
			return nil, &gatewayhooks.QuotaError{Status: 503, Code: "quota_pool_checker_unavailable", Message: "Weekly quota checker is unavailable"}
		}
		return accounts, nil
	}
	gid := int64(0)
	if groupID != nil {
		gid = *groupID
	}
	refs := make([]gatewayhooks.AccountRef, 0, len(accounts))
	for _, account := range accounts {
		refs = append(refs, gatewayhooks.AccountRef{Platform: account.Platform, AccountID: account.ID})
	}
	result := manager.BeforeAccountSelect(ctx, gatewayhooks.BeforeAccountSelectInput{
		Platform:   platform,
		GroupID:    gid,
		UserID:     userID,
		Model:      model,
		Candidates: refs,
	})
	if result.Error != nil {
		return nil, result.Error
	}
	if len(result.Candidates) == len(accounts) {
		return accounts, nil
	}
	allowed := make(map[int64]struct{}, len(result.Candidates))
	for _, candidate := range result.Candidates {
		allowed[candidate.AccountID] = struct{}{}
	}
	filtered := make([]Account, 0, len(result.Candidates))
	for _, account := range accounts {
		if _, ok := allowed[account.ID]; ok {
			filtered = append(filtered, account)
		}
	}
	return filtered, nil
}

func afterUsageRecordedGatewayHooks(ctx context.Context, manager *gatewayhooks.Manager, platform string, account *Account, apiKey *APIKey, user *User, model string, upstreamModel string, cost float64, applied bool) {
	if manager == nil || account == nil || apiKey == nil || apiKey.GroupID == nil {
		return
	}
	var userID int64
	if user != nil {
		userID = user.ID
	}
	manager.AfterUsageRecorded(ctx, gatewayhooks.AfterUsageRecordedInput{
		Platform:      platform,
		AccountID:     account.ID,
		GroupID:       *apiKey.GroupID,
		UserID:        userID,
		Model:         model,
		UpstreamModel: upstreamModel,
		Cost:          cost,
		Applied:       applied,
	})
}
