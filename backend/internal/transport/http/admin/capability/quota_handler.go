package capability

import (
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/contracts"
	srvcap "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/capability"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
)

// QuotaHandler exposes tenant quota endpoints.
type QuotaHandler struct {
	service *srvcap.ExposureService
}

// NewQuotaHandler builds a handler using shared dependencies.
func NewQuotaHandler(deps *app.Deps) *QuotaHandler {
	if deps == nil {
		return nil
	}
	return &QuotaHandler{
		service: srvcap.NewExposureService(deps),
	}
}

// List returns tenant quotas for a capability.
func (h *QuotaHandler) List(c *gin.Context) {
	if h == nil || h.service == nil {
		contracts.ResponseServiceUnavailable(c, "capability exposure service not available", nil)
		return
	}
	capabilityID := strings.TrimSpace(c.Param("capabilityID"))
	if capabilityID == "" {
		contracts.ResponseBadRequest(c, "capability id required")
		return
	}
	quotas := h.service.ListQuotas(capabilityID)
	contracts.ResponseSuccess(c, gin.H{
		"capability_id": capabilityID,
		"quotas":        quotas,
	})
}

// Upsert adjusts a tenant quota entry.
func (h *QuotaHandler) Upsert(c *gin.Context) {
	if h == nil || h.service == nil {
		contracts.ResponseServiceUnavailable(c, "capability exposure service not available", nil)
		return
	}
	capabilityID := strings.TrimSpace(c.Param("capabilityID"))
	if capabilityID == "" {
		contracts.ResponseBadRequest(c, "capability id required")
		return
	}
	var payload srvcap.TenantQuota
	if err := c.ShouldBindJSON(&payload); err != nil {
		contracts.ResponseBadRequest(c, "invalid payload: "+err.Error())
		return
	}
	record, err := h.service.UpdateQuota(c.Request.Context(), capabilityID, payload)
	if err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, record)
}
