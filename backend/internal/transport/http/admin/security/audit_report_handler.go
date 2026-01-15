package security

import (
	"net/http"
	"strconv"

	secobs "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/observability/security"
	adminsec "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/security"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
)

// AuditReportHandler exposes read endpoints for audit reports.
type AuditReportHandler struct {
	service *adminsec.BaselineService
}

func NewAuditReportHandler(deps *app.Deps, audit *secobs.AuditWriter) *AuditReportHandler {
	logger := deps.RuntimeLogger(deps.Ctx, "admin_security_audit_reports", nil)
	service := adminsec.NewBaselineService(deps.DB, deps.Config, logger)
	service.WithRunner(nil) // runner provided when executing via API if needed
	return &AuditReportHandler{service: service}
}

func (h *AuditReportHandler) ListReports(c *gin.Context) {
	limit := 0
	if raw := c.Query("limit"); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil {
			limit = v
		}
	}
	reports, err := h.service.ListAuditReports(c.Request.Context(), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": reports})
}
