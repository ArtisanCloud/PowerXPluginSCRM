package console

import (
	"strings"

	authx "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/middleware"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
)

// RegisterRoutes wires the admin console API group.
func RegisterRoutes(router *gin.RouterGroup, deps *app.Deps) *gin.RouterGroup {
	if router == nil {
		return nil
	}
	group := router.Group("/dev-console")
	configHandler := NewConfigHandler(deps)
	if configHandler != nil {
		group.GET("/config/sections", configHandler.ListSections)
		group.PUT("/config/sections/:sectionKey", configHandler.UpdateSection)
	}
	auditHandler := NewAuditHandler(deps)
	if auditHandler != nil {
		group.GET("/audit/events", auditHandler.ListEvents)
		group.GET("/audit/export", auditHandler.ExportEvents)
	}
	jobHandler := NewJobHandler(deps)
	if jobHandler != nil {
		group.GET("/jobs/runs", jobHandler.ListRuns)
		group.POST("/jobs/runs/:runId/retry", jobHandler.RetryRun)
		group.POST("/safe-ops/actions", jobHandler.ExecuteSafeOp)
	}
	troubleshootHandler := NewTroubleshootHandler(deps)
	if troubleshootHandler != nil {
		group.GET("/troubleshooting/summary", troubleshootHandler.Summary)
		group.GET("/webhooks/attempts", troubleshootHandler.ListWebhookAttempts)
		group.GET("/webhooks/attempts/:attemptId", troubleshootHandler.GetWebhookAttempt)
	}
	return group
}

// RBACEntries exposes RBAC mappings for Dev Console endpoints.
func RBACEntries(prefix string) map[string]authx.Permission {
	base := strings.TrimRight(prefix, "/") + "/admin/dev-console"
	return map[string]authx.Permission{
		"GET:" + base + "/config/sections":         {Resource: "operations.plugin.admin", Action: "read"},
		"PUT:" + base + "/config/sections/*":       {Resource: "operations.plugin.admin", Action: "manage"},
		"GET:" + base + "/audit/events":            {Resource: "operations.plugin.audit", Action: "read"},
		"GET:" + base + "/audit/export":            {Resource: "operations.plugin.audit", Action: "export"},
		"GET:" + base + "/jobs/runs":               {Resource: "operations.plugin.ops", Action: "read"},
		"POST:" + base + "/jobs/runs/*/retry":      {Resource: "operations.plugin.ops", Action: "execute"},
		"POST:" + base + "/safe-ops/actions":       {Resource: "operations.plugin.ops", Action: "execute"},
		"GET:" + base + "/troubleshooting/summary": {Resource: "operations.plugin.ops", Action: "read"},
		"GET:" + base + "/webhooks/attempts":       {Resource: "operations.plugin.ops", Action: "read"},
		"GET:" + base + "/webhooks/attempts/*":     {Resource: "operations.plugin.ops", Action: "read"},
	}
}
