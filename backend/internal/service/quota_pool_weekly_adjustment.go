package service

import (
	"context"
	"strconv"
)

type quotaPoolAdjustmentActorKey struct{}

func WithQuotaPoolAdjustmentActor(ctx context.Context, userID int64) context.Context {
	return context.WithValue(ctx, quotaPoolAdjustmentActorKey{}, "admin:"+strconv.FormatInt(userID, 10))
}
func quotaPoolAdjustmentActor(ctx context.Context) string {
	actor, _ := ctx.Value(quotaPoolAdjustmentActorKey{}).(string)
	return actor
}
