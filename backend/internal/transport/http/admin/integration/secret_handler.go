package integration

import (
	"errors"
	"fmt"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/contracts"
	repo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/integration"
	authmw "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/middleware"
	service "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/integration"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
	httpmw "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/middleware"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// SecretHandler exposes admin APIs for secret lifecycle management.
type SecretHandler struct {
	service *service.SecretService
}

// NewSecretHandler wires dependencies for secret handler.
func NewSecretHandler(deps *app.Deps) *SecretHandler {
	if deps == nil || deps.DB == nil {
		return &SecretHandler{}
	}
	secretRepo := repo.NewSecretRepository(deps.DB)
	approvalRepo := repo.NewApprovalRepository(deps.DB)
	secretSvc := service.NewSecretService(deps.Config, secretRepo, service.NewRandomSecretProvider(nil), approvalRepo)
	return &SecretHandler{service: secretSvc}
}

type secretCreateRequest struct {
	IntegrationType      string         `json:"integration_type" binding:"required"`
	RotationIntervalDays int            `json:"rotation_interval_days"`
	Metadata             map[string]any `json:"metadata"`
	Generate             bool           `json:"generate"`
	SecretRef            string         `json:"secret_ref"`
}

type secretRotateRequest struct {
	Generate *bool `json:"generate"`
}

// ListSecrets returns tenant secrets.
func (h *SecretHandler) ListSecrets(c *gin.Context) {
	if h.service == nil {
		contracts.ResponseServiceUnavailable(c, "secret service not available", nil)
		return
	}
	tenantID, ok := httpmw.TenantUuidString(c)
	if !ok {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	secrets, err := h.service.ListSecrets(c.Request.Context(), tenantID)
	if err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, secrets)
}

// CreateSecret registers a new secret metadata record.
func (h *SecretHandler) CreateSecret(c *gin.Context) {
	if h.service == nil {
		contracts.ResponseServiceUnavailable(c, "secret service not available", nil)
		return
	}
	var req secretCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid body: "+err.Error())
		return
	}
	tenantID, ok := httpmw.TenantUuidString(c)
	if !ok {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	result, err := h.service.CreateSecret(c.Request.Context(), service.CreateSecretParams{
		TenantUuid:           tenantID,
		IntegrationType:      req.IntegrationType,
		RotationIntervalDays: req.RotationIntervalDays,
		Metadata:             req.Metadata,
		Generate:             req.Generate,
		ExistingSecretRef:    req.SecretRef,
		Actor:                actorFromContext(c),
	})
	if err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, result)
}

// RotateSecret triggers rotation for a secret.
func (h *SecretHandler) RotateSecret(c *gin.Context) {
	if h.service == nil {
		contracts.ResponseServiceUnavailable(c, "secret service not available", nil)
		return
	}
	generate := true
	if c.Request.ContentLength > 0 {
		var req secretRotateRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			contracts.ResponseBadRequest(c, "invalid body: "+err.Error())
			return
		}
		if req.Generate != nil {
			generate = *req.Generate
		}
	}
	tenantID, ok := httpmw.TenantUuidString(c)
	if !ok {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	result, err := h.service.RotateSecret(c.Request.Context(), service.RotateSecretParams{
		TenantUuid: tenantID,
		SecretID:   c.Param("id"),
		Generate:   generate,
		Actor:      actorFromContext(c),
	})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			contracts.ResponseNotFound(c, "secret not found")
			return
		}
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, result)
}

// CompleteRotation finalizes pending secret.
func (h *SecretHandler) CompleteRotation(c *gin.Context) {
	if h.service == nil {
		contracts.ResponseServiceUnavailable(c, "secret service not available", nil)
		return
	}
	tenantID, ok := httpmw.TenantUuidString(c)
	if !ok {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	secret, err := h.service.CompleteRotation(c.Request.Context(), tenantID, c.Param("id"), actorFromContext(c))
	if err != nil {
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			contracts.ResponseNotFound(c, "secret not found")
		default:
			contracts.ResponseInternalError(c, err)
		}
		return
	}
	contracts.ResponseSuccess(c, secret)
}

// RevokeSecret revokes and disables the secret.
func (h *SecretHandler) RevokeSecret(c *gin.Context) {
	if h.service == nil {
		contracts.ResponseServiceUnavailable(c, "secret service not available", nil)
		return
	}
	tenantID, ok := httpmw.TenantUuidString(c)
	if !ok {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	secret, err := h.service.RevokeSecret(c.Request.Context(), tenantID, c.Param("id"), actorFromContext(c))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			contracts.ResponseNotFound(c, "secret not found")
			return
		}
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, secret)
}

// GetAuditLog returns the audit entries for a secret.
func (h *SecretHandler) GetAuditLog(c *gin.Context) {
	if h.service == nil {
		contracts.ResponseServiceUnavailable(c, "secret service not available", nil)
		return
	}
	tenantID, ok := httpmw.TenantUuidString(c)
	if !ok {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	entries, err := h.service.GetAuditLog(c.Request.Context(), tenantID, c.Param("id"))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			contracts.ResponseNotFound(c, "secret not found")
			return
		}
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, entries)
}

func actorFromContext(c *gin.Context) string {
	if tc, ok := authmw.GetTenantContext(c); ok {
		if tc.UserID > 0 {
			return fmt.Sprintf("user:%d", tc.UserID)
		}
	}
	return "admin"
}
