package iam

import (
	srviam "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/iam"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(admin *gin.RouterGroup, deps *app.Deps) {
	if admin == nil || deps == nil || deps.DB == nil {
		return
	}
	group := admin.Group("/iam")

	audit := srviam.NewAuditService(deps.DB)
	stsSvc := srviam.NewSTSService(deps.Config, audit, app.PluginID, "")
	roleSvc := srviam.NewRoleService(deps.DB, audit, app.PluginID)

	tenantHandler := NewTenantHandler(srviam.NewTenantService(deps.DB, audit))
	departmentHandler := NewDepartmentHandler(srviam.NewDepartmentService(deps.DB, audit))
	memberHandler := NewMemberHandler(srviam.NewUserService(deps.DB, audit))
	userDirectoryHandler := NewUserDirectoryHandler(srviam.NewUserService(deps.DB, audit))
	roleHandler := NewRoleHandler(roleSvc)
	rolePermissionsHandler := NewRolePermissionsHandler(roleSvc)
	roleMembersHandler := NewRoleMembersHandler(roleSvc)
	permissionHandler := NewPermissionHandler(roleSvc)
	auditHandler := NewAuditHandler(audit)
	stsHandler := NewSTSHandler(deps.IAMMode, stsSvc)

	group.GET("/tenants", tenantHandler.List)
	group.POST("/tenants", tenantHandler.Create)
	group.PATCH("/tenants/:id", tenantHandler.Update)

	group.GET("/departments", departmentHandler.List)
	group.GET("/departments/tree", departmentHandler.Tree)
	group.POST("/departments", departmentHandler.Create)
	group.PATCH("/departments/:id", departmentHandler.Update)
	group.DELETE("/departments/:id", departmentHandler.Delete)

	group.GET("/members", memberHandler.List)
	group.POST("/members", memberHandler.Create)
	group.PATCH("/members/:id", memberHandler.Update)
	group.POST("/members/import", memberHandler.BulkImport)
	group.GET("/user-directory", userDirectoryHandler.List)

	group.GET("/roles", roleHandler.List)
	group.POST("/roles", roleHandler.Create)
	group.GET("/roles/:id", roleHandler.Get)
	group.PATCH("/roles/:id", roleHandler.Update)
	group.DELETE("/roles/:id", roleHandler.Delete)
	group.PUT("/roles/:id/permissions", rolePermissionsHandler.Replace)
	group.POST("/roles/:id/members", roleMembersHandler.Add)
	group.DELETE("/roles/:id/members", roleMembersHandler.Remove)

	group.GET("/permissions", permissionHandler.List)
	group.GET("/audit/logs", auditHandler.List)
	group.POST("/auth/local/sts", stsHandler.Mint)

	// Legacy aliases for compatibility (deprecated)
	group.GET("/users", memberHandler.List)
	group.POST("/users", memberHandler.Create)
	group.PATCH("/users/:id", memberHandler.Update)
	group.POST("/users/import", memberHandler.BulkImport)
}
