package console

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/contracts"
	consolesvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/console"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
)

// TroubleshootHandler exposes troubleshooting summary and webhook diagnostics.
type TroubleshootHandler struct {
	svc *consolesvc.TroubleshootService
}

// NewTroubleshootHandler constructs the handler when dependencies exist.
func NewTroubleshootHandler(deps *app.Deps) *TroubleshootHandler {
	if deps == nil || deps.DB == nil {
		return &TroubleshootHandler{}
	}
	svc := consolesvc.NewService(deps)
	return &TroubleshootHandler{svc: svc.Troubleshoot()}
}

type summaryQuery struct {
	TenantUuid string `form:"tenant_uuid"`
}

// Summary returns troubleshooting dashboard data.
func (h *TroubleshootHandler) Summary(c *gin.Context) {
	if h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "troubleshoot service unavailable", nil)
		return
	}
	var query summaryQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		contracts.ResponseBadRequest(c, "invalid query parameters: "+err.Error())
		return
	}
	var tenantPtr *string
	if strings.TrimSpace(query.TenantUuid) != "" {
		clean := strings.TrimSpace(query.TenantUuid)
		tenantPtr = &clean
	}
	summary, err := h.svc.Summary(c.Request.Context(), consolesvc.TroubleshootSummaryInput{TenantUuid: tenantPtr})
	if err != nil {
		if errors.Is(err, consolesvc.ErrJobServiceUnavailable) {
			contracts.ResponseServiceUnavailable(c, err.Error(), nil)
			return
		}
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, summary)
}

type attemptsQuery struct {
	TenantUuid     string `form:"tenant_uuid"`
	Status         string `form:"status"`
	SubscriptionID string `form:"subscription_id"`
	Since          string `form:"since"`
	Cursor         string `form:"cursor"`
	Limit          int    `form:"limit"`
}

// ListWebhookAttempts returns webhook delivery attempts for a tenant scope.
func (h *TroubleshootHandler) ListWebhookAttempts(c *gin.Context) {
	if h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "troubleshoot service unavailable", nil)
		return
	}
	var query attemptsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		contracts.ResponseBadRequest(c, "invalid query parameters: "+err.Error())
		return
	}
	tenant := strings.TrimSpace(query.TenantUuid)
	if tenant == "" {
		contracts.ResponseError(c, http.StatusBadRequest, contracts.ErrCodeInvalidRequest, "tenant_uuid is required")
		return
	}
	var sincePtr *time.Time
	if strings.TrimSpace(query.Since) != "" {
		parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(query.Since))
		if err != nil {
			contracts.ResponseBadRequest(c, "invalid since timestamp")
			return
		}
		sincePtr = &parsed
	}
	list, err := h.svc.ListWebhookAttempts(c.Request.Context(), consolesvc.WebhookAttemptListInput{
		TenantUuid:     tenant,
		Status:         strings.TrimSpace(query.Status),
		SubscriptionID: strings.TrimSpace(query.SubscriptionID),
		Since:          sincePtr,
		Cursor:         strings.TrimSpace(query.Cursor),
		Limit:          query.Limit,
	})
	if err != nil {
		if errors.Is(err, consolesvc.ErrJobServiceUnavailable) {
			contracts.ResponseServiceUnavailable(c, err.Error(), nil)
			return
		}
		if field, msg, ok := consolesvc.IsValidationError(err); ok {
			contracts.ResponseErrorWithDetails(c, http.StatusBadRequest, contracts.ErrCodeValidationFailed, msg, gin.H{"field": field})
			return
		}
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, list)
}

type attemptQuery struct {
	TenantUuid string `form:"tenant_uuid"`
}

// GetWebhookAttempt returns attempt details when found.
func (h *TroubleshootHandler) GetWebhookAttempt(c *gin.Context) {
	if h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "troubleshoot service unavailable", nil)
		return
	}
	attemptID := strings.TrimSpace(c.Param("attemptId"))
	if attemptID == "" {
		contracts.ResponseBadRequest(c, "attempt id is required")
		return
	}
	var query attemptQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		contracts.ResponseBadRequest(c, "invalid query parameters: "+err.Error())
		return
	}
	tenant := strings.TrimSpace(query.TenantUuid)
	if tenant == "" {
		contracts.ResponseError(c, http.StatusBadRequest, contracts.ErrCodeInvalidRequest, "tenant_uuid is required")
		return
	}
	attempt, err := h.svc.GetWebhookAttempt(c.Request.Context(), attemptID, tenant)
	if err != nil {
		if errors.Is(err, consolesvc.ErrJobServiceUnavailable) {
			contracts.ResponseServiceUnavailable(c, err.Error(), nil)
			return
		}
		contracts.ResponseInternalError(c, err)
		return
	}
	if attempt == nil {
		contracts.ResponseNotFound(c, "webhook attempt not found")
		return
	}
	contracts.ResponseSuccess(c, attempt)
}
