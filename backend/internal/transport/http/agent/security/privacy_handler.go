package security

import (
	"encoding/json"
	"net/http"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/privacy"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/middleware"
	secobs "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/observability/security"
	agentsec "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/agent/security"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
)

type PrivacyHandler struct {
	guard *agentsec.PrivacyGuard
	audit *secobs.AuditWriter
	deps  *app.Deps
}

func NewPrivacyHandler(deps *app.Deps, guard *agentsec.PrivacyGuard, audit *secobs.AuditWriter) *PrivacyHandler {
	return &PrivacyHandler{guard: guard, audit: audit, deps: deps}
}

func (h *PrivacyHandler) GetActiveConsent(c *gin.Context) {
	tenantID, ok := middleware.TenantUUIDFromContext(c.Request.Context())
	if !ok || tenantID == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "tenant context missing"})
		return
	}
	tokens, err := h.guard.ActiveConsentTokens(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	scope := make(map[string]struct{})
	for _, token := range tokens {
		values, _ := token.ScopeValues()
		for _, asset := range values {
			scope[asset] = struct{}{}
		}
	}
	assets := make([]string, 0, len(scope))
	for asset := range scope {
		assets = append(assets, asset)
	}
	c.JSON(http.StatusOK, gin.H{"tenant_uuid": tenantID, "assets": assets})
}

func (h *PrivacyHandler) AcknowledgeLifecycleEvent(c *gin.Context) {
	var payload struct {
		EventType string                 `json:"event_type" binding:"required"`
		AssetKey  string                 `json:"asset_key" binding:"required"`
		Metadata  map[string]interface{} `json:"metadata"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}
	tenantID, ok := middleware.TenantUUIDFromContext(c.Request.Context())
	if !ok || tenantID == "" {
		c.JSON(http.StatusForbidden, gin.H{"error": "tenant context missing"})
		return
	}
	event := &privacy.LifecycleEvent{
		TenantUuid: tenantID,
		EventType:  payload.EventType,
		AssetKey:   payload.AssetKey,
		RecordedBy: "agent",
		Status:     privacy.LifecycleStatusSucceeded,
	}
	if payload.Metadata != nil {
		filtered := h.guard.FilterAIData(payload.Metadata)
		blob, _ := json.Marshal(filtered)
		event.Payload = datatypes.JSON(blob)
	}
	if _, err := h.guard.RecordLifecycleEvent(c.Request.Context(), event); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if h.audit != nil {
		if err := h.audit.EmitLifecycleSuccess(tenantID, payload.EventType, "agent", payload.Metadata); err != nil {
			if h.deps != nil {
				if logger := h.deps.RuntimeLogger(c.Request.Context(), "agent_privacy_handler", nil); logger != nil {
					logger.WithError(err).Warn("failed to emit lifecycle success audit event")
				}
			}
		}
	}
	c.Status(http.StatusAccepted)
}
