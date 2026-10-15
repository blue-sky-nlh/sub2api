package service

import (
	"context"
	"github.com/Wei-Shaw/sub2api/internal/gatewayhooks"
)

func RejectNonWeeklyGateway(ctx context.Context) error {
	if !WeeklyQuotaControlled(ctx) {
		return nil
	}
	return &gatewayhooks.QuotaError{Status: 403, Code: "quota_pool_scope_mismatch", Message: "Commercial weekly quota traffic requires the GPT gateway"}
}
