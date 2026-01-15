package templates

import (
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
	httpmw "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/middleware"
	"github.com/gin-gonic/gin"
)

func RegisterAPIRoutes(rg *gin.RouterGroup, deps *app.Deps) {
	h := NewTemplateHandler(deps)

	g := rg.Group("/templates", httpmw.EnsureTenant())
	{
		g.GET("", h.GetTemplates)
		g.GET("/:id", h.GetTemplate)
		g.POST("", h.CreateTemplate)
		g.PUT("/:id", h.UpdateTemplate)
		g.DELETE("/:id", h.DeleteTemplate)
		g.POST("/batch-clone", h.BatchCloneTemplates)
		g.POST("/:id/validate", h.ValidateTemplateCapability)
	}

	adminGroup := rg.Group("/admin/templates", httpmw.EnsureTenant())
	{
		adminGroup.POST("/batch-clone", h.BatchCloneTemplates)
		adminGroup.POST("/:id/validate", h.ValidateTemplateCapability)
	}
}
