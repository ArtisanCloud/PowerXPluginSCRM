package opportunity

import (
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
)

// RegisterRoutes wires Opportunity admin routes.
// Phase 1 only creates the route group skeleton; handlers are added in later phases.
func RegisterRoutes(rg *gin.RouterGroup, deps *app.Deps) {
	if rg == nil || deps == nil {
		return
	}

	op := rg.Group("/opportunity")
	_ = op
}
