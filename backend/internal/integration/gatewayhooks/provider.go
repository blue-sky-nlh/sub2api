package gatewayhooksintegration

import (
	"github.com/Wei-Shaw/sub2api/internal/gatewayhooks"
	"github.com/Wei-Shaw/sub2api/internal/quotapool"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

func ProvideManager(quotaPoolRepo service.QuotaPoolRepository) *gatewayhooks.Manager {
	return gatewayhooks.NewManager(quotapool.NewHook(quotaPoolRepo))
}
