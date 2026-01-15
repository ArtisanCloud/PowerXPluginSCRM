package capability

import (
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/contracts"
	srvcap "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/capability"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
)

// ExposureHandler manages capability exposure endpoints.
type ExposureHandler struct {
	service *srvcap.ExposureService
}

// NewExposureHandler creates a handler instance.
func NewExposureHandler(deps *app.Deps) *ExposureHandler {
	if deps == nil {
		return nil
	}
	return &ExposureHandler{
		service: srvcap.NewExposureService(deps),
	}
}

// GetTemplate returns default exposure options.
func (h *ExposureHandler) GetTemplate(c *gin.Context) {
	if h == nil || h.service == nil {
		contracts.ResponseServiceUnavailable(c, "capability exposure service not available", nil)
		return
	}
	contracts.ResponseSuccess(c, h.service.Template())
}

// Get returns the exposure package for a capability.
func (h *ExposureHandler) Get(c *gin.Context) {
	if h == nil || h.service == nil {
		contracts.ResponseServiceUnavailable(c, "capability exposure service not available", nil)
		return
	}
	capabilityID := strings.TrimSpace(c.Param("capabilityID"))
	if capabilityID == "" {
		contracts.ResponseBadRequest(c, "capability id required")
		return
	}
	record, err := h.service.Get(c.Request.Context(), capabilityID)
	if err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, gin.H{
		"capability_id": capabilityID,
		"package":       record,
	})
}

// Upsert saves the exposure package.
func (h *ExposureHandler) Upsert(c *gin.Context) {
	if h == nil || h.service == nil {
		contracts.ResponseServiceUnavailable(c, "capability exposure service not available", nil)
		return
	}
	var payload srvcap.ExposureInput
	if err := c.ShouldBindJSON(&payload); err != nil {
		contracts.ResponseBadRequest(c, "invalid payload: "+err.Error())
		return
	}
	if payload.CapabilityID == "" {
		payload.CapabilityID = strings.TrimSpace(c.Param("capabilityID"))
	}
	payload.Actor = actorFromContext(c)
	record, err := h.service.Upsert(c.Request.Context(), &payload)
	if err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, record)
}

func actorFromContext(c *gin.Context) string {
	if c == nil {
		return ""
	}
	if v := c.GetString("user.email"); v != "" {
		return v
	}
	if claims, ok := c.Get("user.claims"); ok {
		if email, ok := claims.(map[string]any)["email"].(string); ok {
			return email
		}
	}
	return ""
}
