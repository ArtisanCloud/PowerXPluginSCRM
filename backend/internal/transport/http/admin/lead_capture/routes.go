package lead_capture

import (
	leadrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/lead_capture"
	leadsvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/lead_capture"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
	httpmw "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/middleware"
	"github.com/gin-gonic/gin"
)

// RegisterRoutes wires lead capture admin endpoints.
func RegisterRoutes(rg *gin.RouterGroup, deps *app.Deps) {
	if rg == nil || deps == nil {
		return
	}
	var leadSvc *leadsvc.LeadService
	if deps.DB != nil {
		repo := leadrepo.NewLeadRepository(deps.DB)
		leadSvc = leadsvc.NewLeadService(repo)
	}
	handler := NewLeadHandler(leadSvc)
	group := rg.Group("/leads", httpmw.EnsureTenant())
	{
		group.GET("", handler.List)
		group.POST("", handler.Create)
		group.POST("/import", handler.Import)
		group.POST("/import/preview", handler.ImportPreview)
		group.POST("/import/confirm", handler.ImportConfirm)
		group.GET("/:lead_id", handler.Get)
		group.POST("/:lead_id/assign", handler.Assign)
		group.POST("/:lead_id/status", handler.UpdateStatus)
		group.GET("/:lead_id/assignments", handler.ListAssignments)
		group.GET("/:lead_id/status-history", handler.ListStatusHistory)
	}
}
