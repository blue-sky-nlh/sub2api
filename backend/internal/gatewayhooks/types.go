package gatewayhooks

import "context"

type AccountRef struct {
	Platform  string
	AccountID int64
}

type BeforeAccountSelectInput struct {
	Platform   string
	GroupID    int64
	UserID     int64
	Model      string
	Candidates []AccountRef
	Error      error
}

type BeforeAccountSelectResult struct {
	Candidates []AccountRef
	Error      error
}

type AfterUsageRecordedInput struct {
	Platform      string
	AccountID     int64
	GroupID       int64
	UserID        int64
	Model         string
	UpstreamModel string
	Cost          float64
	Applied       bool
}

type PolicyHook interface {
	BeforeAccountSelect(ctx context.Context, input BeforeAccountSelectInput) (BeforeAccountSelectResult, error)
}

type UsageHook interface {
	AfterUsageRecorded(ctx context.Context, input AfterUsageRecordedInput) error
}

type Hook interface {
	PolicyHook
	UsageHook
}
