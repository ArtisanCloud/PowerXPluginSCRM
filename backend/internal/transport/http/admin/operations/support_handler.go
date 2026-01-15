package operations

import (
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/contracts"
	oprepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/operations"
	operationsvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/operations"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
	httpmw "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/middleware"
	"github.com/gin-gonic/gin"
)

// SupportHandler exposes admin HTTP endpoints for support playbook management.
type SupportHandler struct {
	svc *operationsvc.SupportService
}

// NewSupportHandler wires dependencies for support handler.
func NewSupportHandler(deps *app.Deps) *SupportHandler {
	if deps == nil || deps.DB == nil {
		return &SupportHandler{}
	}
	repo := oprepo.NewSupportRepository(deps.DB)
	svc := operationsvc.NewSupportService(repo, deps.Config, deps.OperationsMetrics, nil)
	return &SupportHandler{svc: svc}
}

type supportPlaybookQuery struct {
	TenantUuid string `form:"tenant_uuid"`
}

// GetPlaybook returns the current support playbook configuration.
func (h *SupportHandler) GetPlaybook(c *gin.Context) {
	if h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "support service unavailable", nil)
		return
	}
	var query supportPlaybookQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		contracts.ResponseBadRequest(c, "invalid query parameters: "+err.Error())
		return
	}
	var tenantID *string
	if strings.TrimSpace(query.TenantUuid) != "" {
		clean := strings.TrimSpace(query.TenantUuid)
		tenantID = &clean
	}
	payload, err := h.svc.GetPlaybook(c.Request.Context(), tenantID)
	if err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, payload)
}

// UpdatePlaybook upserts channels and knowledge base references.
func (h *SupportHandler) UpdatePlaybook(c *gin.Context) {
	if h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "support service unavailable", nil)
		return
	}
	var req operationsvc.ConfigurePlaybookInput
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid request body: "+err.Error())
		return
	}
	payload, err := h.svc.ConfigurePlaybook(c.Request.Context(), req)
	if err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, payload)
}

// TestChannels triggers synthetic validation (placeholder).
func (h *SupportHandler) TestChannels(c *gin.Context) {
	if h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "support service unavailable", nil)
		return
	}
	contracts.ResponseSuccessWithMessage(c, gin.H{"status": "ok"}, "channel validation dispatched")
}

// GetMetrics returns aggregated support KPIs.
func (h *SupportHandler) GetMetrics(c *gin.Context) {
	if h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "support service unavailable", nil)
		return
	}
	tenantID, _ := httpmw.TenantUuidString(c) // optional for future scoping
	_ = tenantID
	metrics, err := h.svc.ComputeMetrics(c.Request.Context())
	if err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, metrics)
}
