package marketplace

import (
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
	httpmw "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/middleware"
	"github.com/gin-gonic/gin"
)

// RegisterRoutes wires tenant-facing marketplace license endpoints.
func RegisterRoutes(group *gin.RouterGroup, deps *app.Deps) {
	if group == nil || deps == nil {
		return
	}

	handler := NewLicenseHandler(deps)
	if handler == nil {
		return
	}

	licenses := group.Group("/licenses", httpmw.EnsureTenant())
	{
		licenses.POST("", handler.Create)
		licenses.GET("/:id", handler.Get)
		licenses.POST("/:id", handler.Renew)
		licenses.POST("/:id/offline-extend", handler.ExtendOffline)
	}
}
