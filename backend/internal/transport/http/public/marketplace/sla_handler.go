package marketplace

import (
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/contracts"
	oprepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/operations"
	opservice "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/operations"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
)

// SLAHandler exposes public SLA transparency endpoint.
type SLAHandler struct {
	svc *opservice.SLAService
}

// NewSLAHandler constructs a public SLA handler.
func NewSLAHandler(repo *oprepo.SLARepository, svc *opservice.SLAService) *SLAHandler {
	if svc == nil && repo != nil {
		svc = opservice.NewSLAService(repo, nil, nil)
	}
	return &SLAHandler{svc: svc}
}

// Register attaches routes to given router group.
func Register(router *gin.RouterGroup, handler *SLAHandler) {
	if router == nil || handler == nil {
		return
	}
	router.GET("/sla/:pluginId", handler.GetPublicSLA)
}

// GetPublicSLA handles GET /api/v1/marketplace/sla/{plugin_id}.
func (h *SLAHandler) GetPublicSLA(c *gin.Context) {
	if h == nil || h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "sla service unavailable", nil)
		return
	}
	pluginID := c.Param("pluginId")
	if pluginID == "" {
		pluginID = app.PluginID
	}
	records, err := h.svc.GetPublicSLA(c.Request.Context(), pluginID)
	if err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, records)
}
