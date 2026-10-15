package handler

import (
	"context"
	"encoding/json"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/gatewayhooks"
	coderws "github.com/coder/websocket"
)

func (h *OpenAIGatewayHandler) writeWeeklyQuotaSlotError(err error, write openAISlotErrorWriter) {
	q, ok := gatewayhooks.AsQuotaError(err)
	if !ok {
		q = &gatewayhooks.QuotaError{Status: 503, Code: "quota_pool_statistics_unavailable", Message: "Weekly quota verification failed"}
	}
	kind := "api_error"
	if q.Status == 429 {
		kind = "rate_limit_error"
	}
	if q.Status == 403 {
		kind = "permission_error"
	}
	write(q.Status, kind, q.Code, q.Message)
}
func writeWeeklyQuotaWSError(ctx context.Context, conn *coderws.Conn, err error) bool {
	q, ok := gatewayhooks.AsQuotaError(err)
	if !ok {
		return false
	}
	payload, _ := json.Marshal(map[string]any{"type": "error", "error": map[string]any{"type": "quota_error", "code": q.Code, "message": q.Message}})
	wctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
	defer cancel()
	_ = conn.Write(wctx, coderws.MessageText, payload)
	closeOpenAIClientWS(conn, coderws.StatusPolicyViolation, q.Code)
	return true
}
