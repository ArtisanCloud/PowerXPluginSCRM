package security

import (
	"net/http"
	"strconv"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/config"
	secobs "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/observability/security"
	adminsec "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/security"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
)

// ConsentHandler exposes admin endpoints for managing consent tokens and lifecycle records.
type ConsentHandler struct {
	service *adminsec.PrivacyService
}

func NewConsentHandler(deps *app.Deps, audit *secobs.AuditWriter) *ConsentHandler {
	logger := deps.RuntimeLogger(deps.Ctx, "admin_security_consent", nil)
	service := adminsec.NewPrivacyService(deps.DB, deps.Config, logger, audit)
	return &ConsentHandler{service: service}
}

func (h *ConsentHandler) ListConsentTokens(c *gin.Context) {
	tenantID := c.Query("tenant_uuid")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_uuid is required"})
		return
	}
	statuses := c.QueryArray("status")
	tokens, err := h.service.ListConsentTokens(c.Request.Context(), tenantID, statuses...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, NewConsentTokenListResponse(tokens))
}

func (h *ConsentHandler) RevokeConsentToken(c *gin.Context) {
	tenantID := c.Query("tenant_uuid")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_uuid is required"})
		return
	}
	tokenID := c.Param("tokenId")
	if tokenID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tokenId is required"})
		return
	}
	var req RevokeConsentRequest
	if err := c.ShouldBindJSON(&req); err != nil && err.Error() != "EOF" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	actor := req.RequestedBy
	if actor == "" {
		actor = "admin"
	}
	if err := h.service.RevokeConsentToken(c.Request.Context(), tenantID, tokenID, req.Reason, actor); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *ConsentHandler) ListLifecycleEvents(c *gin.Context) {
	tenantID := c.Query("tenant_uuid")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_uuid is required"})
		return
	}
	eventTypes := c.QueryArray("event_type")
	limit := 0
	if v := c.Query("limit"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	events, err := h.service.ListLifecycleEvents(c.Request.Context(), tenantID, eventTypes, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, NewLifecycleEventListResponse(events))
}

// Helper to build audit writer path using config defaults.
func CreateAuditWriter(cfg *config.Config) *secobs.AuditWriter {
	path := cfg.SecurityBaselineConfig().ConsentDefaults.AuditChannel
	if path == "" {
		path = secobs.DefaultAuditLogPath
	}
	writer, err := secobs.NewFileAuditWriter(path)
	if err != nil {
		return nil
	}
	return writer
}
