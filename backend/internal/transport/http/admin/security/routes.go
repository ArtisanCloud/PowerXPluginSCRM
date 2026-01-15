package security

import (
	authx "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/middleware"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
)

// RegisterRoutes wires the admin security namespace.
func RegisterRoutes(rg *gin.RouterGroup, deps *app.Deps) {
	if rg == nil || deps == nil {
		return
	}
	auditWriter := CreateAuditWriter(deps.Config)
	consent := NewConsentHandler(deps, auditWriter)
	toolgrant := NewToolGrantHandler(deps)
	audit := NewAuditReportHandler(deps, auditWriter)
	advisory := NewAdvisoryHandler(deps)
	sec := rg.Group("/security")
	{
		sec.GET("/consent-tokens", consent.ListConsentTokens)
		sec.POST("/consent-tokens/:tokenId/revoke", consent.RevokeConsentToken)
		sec.GET("/lifecycle-events", consent.ListLifecycleEvents)
		sec.GET("/audit-reports", audit.ListReports)
		sec.POST("/advisories", advisory.Create)
		sec.GET("/advisories", advisory.List)
		sec.POST("/advisories/:advisoryId/publish", advisory.Publish)
		sec.POST("/toolgrants/revoke", toolgrant.Revoke)
		sec.GET("/toolgrants/revocations", toolgrant.ListRevocations)
		sec.GET("/toolgrants/usage", toolgrant.ListUsageEvents)
	}
}

// RBACEntries returns RBAC metadata for admin security endpoints.
func RBACEntries(prefix string) map[string]authx.Permission {
	if prefix == "" {
		prefix = "/api/v1"
	}
	return map[string]authx.Permission{
		prefix + "/admin/security/consent-tokens":                 {Resource: "admin.security.consent", Action: "read"},
		prefix + "/admin/security/consent-tokens/:tokenId":        {Resource: "admin.security.consent", Action: "write"},
		prefix + "/admin/security/lifecycle-events":               {Resource: "admin.security.lifecycle", Action: "read"},
		prefix + "/admin/security/audit-reports":                  {Resource: "admin.security.audit", Action: "read"},
		prefix + "/admin/security/advisories":                     {Resource: "admin.security.advisory", Action: "read"},
		prefix + "/admin/security/advisories#create":              {Resource: "admin.security.advisory", Action: "write"},
		prefix + "/admin/security/advisories/:advisoryId/publish": {Resource: "admin.security.advisory", Action: "write"},
		prefix + "/admin/security/toolgrants/revoke":              {Resource: "admin.security.toolgrant", Action: "write"},
		prefix + "/admin/security/toolgrants/revocations":         {Resource: "admin.security.toolgrant", Action: "read"},
		prefix + "/admin/security/toolgrants/usage":               {Resource: "admin.security.toolgrant", Action: "read"},
	}
}
