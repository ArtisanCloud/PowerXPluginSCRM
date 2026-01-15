package miniapp

import (
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/contracts"
	customermw "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/middleware/customer"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
	httpmw "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/middleware"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/mini-app/customerhttp"
	"github.com/gin-gonic/gin"
)

// RegisterAPIRoutes registers /mini-app routes.
// 注意：本阶段仅完成路由组与上下文注入，具体 token 校验逻辑会在后续任务中完善。
func RegisterAPIRoutes(rg *gin.RouterGroup, deps *app.Deps) {
	if rg == nil {
		return
	}

	base := rg.Group("/mini-app")

	// Auth endpoints (Skeleton/local only) should not require existing customer token.
	auth := base.Group("/auth")
	handler := NewCustomerHandler(deps)
	auth.POST("/register", handler.Register)
	auth.POST("/login", handler.Login)

	// Protected mini-app endpoints require a validated customer token.
	protected := base.Group("", customerhttp.Authenticate(deps), httpmw.EnsureTenant())
	protected.GET("/ping", ping)

	templates := NewMiniAppTemplateHandler(deps)
	protected.GET("/templates", templates.ListPublished)
	protected.GET("/templates/:id", templates.GetPublished)
}

func ping(c *gin.Context) {
	tenantUUID, _ := httpmw.TenantUUIDString(c)
	cc, _ := customermw.GetCustomerContext(c)
	contracts.ResponseSuccess(c, gin.H{
		"tenant_uuid": tenantUUID,
		"customer":    cc,
	})
}
