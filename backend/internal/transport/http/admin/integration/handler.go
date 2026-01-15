package integration

import (
	"errors"
	"strings"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/contracts"
	model "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/integration"
	repo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/integration"
	service "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/integration"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
	httpmw "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/middleware"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Handler aggregates integration admin endpoints.
type Handler struct {
	deps           *app.Deps
	webhookService *service.WebhookService
}

// NewHandler constructs a handler with required dependencies.
func NewHandler(deps *app.Deps) *Handler {
	h := &Handler{deps: deps}
	if deps != nil && deps.DB != nil {
		subRepo := repo.NewWebhookSubscriptionRepository(deps.DB)
		attemptRepo := repo.NewDeliveryAttemptRepository(deps.DB)
		approvalRepo := repo.NewApprovalRepository(deps.DB)
		h.webhookService = service.NewWebhookService(deps.Config, subRepo, attemptRepo, approvalRepo)
	}
	return h
}

type webhookCreateRequest struct {
	EventType   string         `json:"event_type" binding:"required"`
	TargetURL   string         `json:"target_url" binding:"required"`
	Secret      string         `json:"secret"`
	RetryPolicy []int          `json:"retry_policy"`
	Metadata    map[string]any `json:"metadata"`
	Status      string         `json:"status"`
}

type webhookUpdateRequest struct {
	TargetURL   *string        `json:"target_url,omitempty"`
	Status      *string        `json:"status,omitempty"`
	Secret      *string        `json:"secret,omitempty"`
	RetryPolicy []int          `json:"retry_policy"`
	Metadata    map[string]any `json:"metadata"`
}

type webhookListQuery struct {
	Status string `form:"status"`
}

type attemptListQuery struct {
	Limit int `form:"limit,default=20"`
}

// ListApprovals 列出待审批项（占位）。
func (h *Handler) ListApprovals(c *gin.Context) {
	contracts.ResponseSuccess(c, gin.H{"message": "approval workflow not implemented yet"})
}

// Approve 通过审批（占位）。
func (h *Handler) Approve(c *gin.Context) {
	contracts.ResponseSuccess(c, gin.H{"ok": true})
}

// Reject 拒绝审批（占位）。
func (h *Handler) Reject(c *gin.Context) {
	contracts.ResponseSuccess(c, gin.H{"ok": true})
}

// ListGrantMatrix 返回当前策略（占位）。
func (h *Handler) ListGrantMatrix(c *gin.Context) {
	contracts.ResponseSuccess(c, gin.H{"message": "grant matrix admin view not implemented"})
}

// ListWebhooks 返回租户的订阅列表。
func (h *Handler) ListWebhooks(c *gin.Context) {
	if h.webhookService == nil {
		contracts.ResponseServiceUnavailable(c, "webhook service not available", nil)
		return
	}
	var query webhookListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		contracts.ResponseBadRequest(c, "invalid query: "+err.Error())
		return
	}
	statuses := []string{}
	if strings.TrimSpace(query.Status) != "" {
		for _, status := range strings.Split(query.Status, ",") {
			statuses = append(statuses, strings.ToUpper(strings.TrimSpace(status)))
		}
	}
	tenantID, ok := httpmw.TenantUuidString(c)
	if !ok {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	subs, err := h.webhookService.ListSubscriptions(c.Request.Context(), tenantID, statuses)
	if err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, subs)
}

// CreateWebhook 创建 webhook 订阅。
func (h *Handler) CreateWebhook(c *gin.Context) {
	if h.webhookService == nil {
		contracts.ResponseServiceUnavailable(c, "webhook service not available", nil)
		return
	}
	var req webhookCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid request body: "+err.Error())
		return
	}
	tenantID, ok := httpmw.TenantUuidString(c)
	if !ok {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	sub, err := h.webhookService.CreateSubscription(c.Request.Context(), service.CreateSubscriptionParams{
		TenantUuid:      tenantID,
		EventType:       req.EventType,
		TargetURL:       req.TargetURL,
		SecretPlaintext: req.Secret,
		RetryPolicy:     req.RetryPolicy,
		Metadata:        req.Metadata,
		Status:          req.Status,
	})
	if err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, sub)
}

// UpdateWebhook 更新订阅。
func (h *Handler) UpdateWebhook(c *gin.Context) {
	if h.webhookService == nil {
		contracts.ResponseServiceUnavailable(c, "webhook service not available", nil)
		return
	}
	var req webhookUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid request body: "+err.Error())
		return
	}
	tenantID, ok := httpmw.TenantUuidString(c)
	if !ok {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	sub, err := h.webhookService.UpdateSubscription(c.Request.Context(), service.UpdateSubscriptionParams{
		TenantUuid:     tenantID,
		SubscriptionID: c.Param("id"),
		TargetURL:      req.TargetURL,
		Status:         req.Status,
		NewSecretPlain: req.Secret,
		RetryPolicy:    req.RetryPolicy,
		Metadata:       req.Metadata,
	})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			contracts.ResponseNotFound(c, "subscription not found")
			return
		}
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, sub)
}

// DeleteWebhook 删除订阅。
func (h *Handler) DeleteWebhook(c *gin.Context) {
	if h.webhookService == nil {
		contracts.ResponseServiceUnavailable(c, "webhook service not available", nil)
		return
	}
	tenantID, ok := httpmw.TenantUuidString(c)
	if !ok {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	if err := h.webhookService.DeleteSubscription(c.Request.Context(), tenantID, c.Param("id")); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			contracts.ResponseNotFound(c, "subscription not found")
			return
		}
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, gin.H{"ok": true})
}

// ListWebhookAttempts 返回订阅的最近投递记录。
func (h *Handler) ListWebhookAttempts(c *gin.Context) {
	if h.webhookService == nil {
		contracts.ResponseServiceUnavailable(c, "webhook service not available", nil)
		return
	}
	var query attemptListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		contracts.ResponseBadRequest(c, "invalid query: "+err.Error())
		return
	}
	attempts, err := h.webhookService.ListAttemptsBySubscription(c.Request.Context(), c.Param("id"), query.Limit)
	if err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, attempts)
}

// ReplayAttempt 从 DLQ 或失败状态重新入队。
func (h *Handler) ReplayAttempt(c *gin.Context) {
	if h.webhookService == nil {
		contracts.ResponseServiceUnavailable(c, "webhook service not available", nil)
		return
	}

	attempt, err := h.webhookService.GetAttempt(c.Request.Context(), c.Param("attemptId"))
	if err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	if attempt == nil {
		contracts.ResponseNotFound(c, "attempt not found")
		return
	}
	if attempt.Status != model.AttemptStatusDLQ && attempt.Status != model.AttemptStatusFailed {
		contracts.ResponseBadRequest(c, "attempt is not eligible for replay")
		return
	}

	sub, err := h.webhookService.GetSubscriptionByID(c.Request.Context(), attempt.SubscriptionID)
	if err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	if sub == nil {
		contracts.ResponseNotFound(c, "subscription not found")
		return
	}

	now := time.Now().UTC()
	if err := h.webhookService.UpdateAttemptStatus(c.Request.Context(), attempt.ID, model.AttemptStatusPending, attempt.RetryCount, &now, "", sub.TenantUuid); err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, gin.H{"ok": true})
}
