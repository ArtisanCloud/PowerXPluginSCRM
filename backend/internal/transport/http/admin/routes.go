package admin

import (
	iamservice "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/iam"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
	admincapability "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/admin/capability"
	adminconsole "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/admin/console"
	adminiam "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/admin/iam"
	adminintegration "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/admin/integration"
	adminmarketplace "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/admin/marketplace"
	adminoperations "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/admin/operations"
	adminruntime "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/admin/runtime_ops"
	adminsecurity "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/admin/security"
	httpmw "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/middleware"
	"github.com/gin-gonic/gin"
)

// Register 注册 Admin 路由
func RegisterAPIRoutes(rg *gin.RouterGroup, deps *app.Deps) {
	adminHandler := NewAdminHandler(deps)
	admin := rg.Group("/admin")
	{
		// 基础管理功能
		admin.GET("/manifest", adminHandler.GetManifest) // 获取插件清单
		admin.GET("/rbac", adminHandler.GetRBACInfo)     // 获取权限信息

		runtimeOps := admin.Group("/runtime", httpmw.EnsureTenant())
		adminruntime.RegisterRoutes(runtimeOps, deps)

		adminmarketplace.RegisterRoutes(adminTenantGroup(admin, deps), deps)
		adminoperations.RegisterRoutes(admin, deps)
		adminconsole.RegisterRoutes(admin, deps)
		admincapability.RegisterRoutes(adminTenantGroup(admin, deps), deps)
		adminintegration.RegisterRoutes(admin, deps)
		adminsecurity.RegisterRoutes(adminTenantGroup(admin, deps), deps)
		adminiam.RegisterRoutes(admin, deps)
	}
}

func adminTenantGroup(parent *gin.RouterGroup, deps *app.Deps) *gin.RouterGroup {
	if parent == nil {
		return nil
	}
	needTenant := false
	if deps != nil && deps.IAMMode == iamservice.IAMModeDelegated {
		needTenant = true
	}
	if !needTenant {
		return parent.Group("")
	}
	return parent.Group("", httpmw.EnsureTenant())
}
