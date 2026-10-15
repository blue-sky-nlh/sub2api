package admin

import (
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type QuotaPoolHandler struct {
	quotaPoolService *service.QuotaPoolService
}

func NewQuotaPoolHandler(quotaPoolService *service.QuotaPoolService) *QuotaPoolHandler {
	return &QuotaPoolHandler{quotaPoolService: quotaPoolService}
}

func (h *QuotaPoolHandler) List(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	items, pageResult, err := h.quotaPoolService.List(c.Request.Context(), pagination.PaginationParams{Page: page, PageSize: pageSize})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, items, pageResult.Total, page, pageSize)
}

func (h *QuotaPoolHandler) ListAll(c *gin.Context) {
	items, err := h.quotaPoolService.ListAll(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, items)
}

func (h *QuotaPoolHandler) GetByID(c *gin.Context) {
	poolID, ok := parsePositiveID(c, "id", "Invalid quota pool ID")
	if !ok {
		return
	}
	pool, err := h.quotaPoolService.GetByID(c.Request.Context(), poolID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, pool)
}

func (h *QuotaPoolHandler) Create(c *gin.Context) {
	var req service.CreateQuotaPoolRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	pool, err := h.quotaPoolService.Create(c.Request.Context(), req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, pool)
}

func (h *QuotaPoolHandler) Update(c *gin.Context) {
	poolID, ok := parsePositiveID(c, "id", "Invalid quota pool ID")
	if !ok {
		return
	}
	var req service.UpdateQuotaPoolRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	pool, err := h.quotaPoolService.Update(c.Request.Context(), poolID, req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, pool)
}

func (h *QuotaPoolHandler) Delete(c *gin.Context) {
	poolID, ok := parsePositiveID(c, "id", "Invalid quota pool ID")
	if !ok {
		return
	}
	if err := h.quotaPoolService.Delete(c.Request.Context(), poolID); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

func (h *QuotaPoolHandler) ListAccounts(c *gin.Context) {
	poolID, ok := parsePositiveID(c, "id", "Invalid quota pool ID")
	if !ok {
		return
	}
	items, err := h.quotaPoolService.ListAccounts(c.Request.Context(), poolID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, items)
}

func (h *QuotaPoolHandler) AddAccount(c *gin.Context) {
	poolID, ok := parsePositiveID(c, "id", "Invalid quota pool ID")
	if !ok {
		return
	}
	var req service.AddQuotaPoolAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	account, err := h.quotaPoolService.AddAccount(c.Request.Context(), poolID, req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, account)
}

func (h *QuotaPoolHandler) DeleteAccountBinding(c *gin.Context) {
	bindingID, ok := parsePositiveID(c, "binding_id", "Invalid binding ID")
	if !ok {
		return
	}
	if err := h.quotaPoolService.DeleteAccountBinding(c.Request.Context(), bindingID); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

func (h *QuotaPoolHandler) ListGroupLimits(c *gin.Context) {
	poolID, ok := parsePositiveID(c, "id", "Invalid quota pool ID")
	if !ok {
		return
	}
	items, err := h.quotaPoolService.ListGroupLimits(c.Request.Context(), poolID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, items)
}

func (h *QuotaPoolHandler) UpsertGroupLimit(c *gin.Context) {
	poolID, ok := parsePositiveID(c, "id", "Invalid quota pool ID")
	if !ok {
		return
	}
	groupID, ok := parsePositiveID(c, "group_id", "Invalid group ID")
	if !ok {
		return
	}
	var req service.UpsertQuotaPoolGroupLimitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	limit, err := h.quotaPoolService.UpsertGroupLimit(c.Request.Context(), poolID, groupID, req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, limit)
}

func (h *QuotaPoolHandler) DeleteGroupLimit(c *gin.Context) {
	poolID, ok := parsePositiveID(c, "id", "Invalid quota pool ID")
	if !ok {
		return
	}
	groupID, ok := parsePositiveID(c, "group_id", "Invalid group ID")
	if !ok {
		return
	}
	if err := h.quotaPoolService.DeleteGroupLimit(c.Request.Context(), poolID, groupID); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

func (h *QuotaPoolHandler) ListGroupUsage(c *gin.Context) {
	poolID, ok := parsePositiveID(c, "id", "Invalid quota pool ID")
	if !ok {
		return
	}
	items, err := h.quotaPoolService.ListGroupUsage(c.Request.Context(), poolID, "", "")
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, items)
}

func (h *QuotaPoolHandler) ResetGroupUsage(c *gin.Context) {
	poolID, ok := parsePositiveID(c, "id", "Invalid quota pool ID")
	if !ok {
		return
	}
	var req service.ResetQuotaPoolUsageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		response.Unauthorized(c, "Admin identity is required")
		return
	}
	ctx := service.WithQuotaPoolAdjustmentActor(c.Request.Context(), subject.UserID)
	if err := h.quotaPoolService.ResetGroupUsage(ctx, poolID, req); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"reset": true})
}

func parsePositiveID(c *gin.Context, name string, message string) (int64, bool) {
	id, err := strconv.ParseInt(c.Param(name), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, message)
		return 0, false
	}
	return id, true
}
