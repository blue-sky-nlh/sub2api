package main

import "github.com/Wei-Shaw/sub2api/internal/service"

// Preserve the existing cleanup arguments while stopping quota refresh before infrastructure.
type applicationCleanup func()

func provideWeeklyCleanup(cleanup applicationCleanup, quota *service.OpenAIQuotaService) func() {
	return func() {
		if quota != nil {
			quota.StopWeeklyRefresh()
		}
		cleanup()
	}
}
