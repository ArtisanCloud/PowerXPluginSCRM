package security

import (
	secobs "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/observability/security"
	agentsec "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/agent/security"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
)

// RegisterRoutes wires the agent security namespace (privacy consent endpoints).
func RegisterRoutes(rg *gin.RouterGroup, deps *app.Deps) {
	if rg == nil || deps == nil {
		return
	}
	logger := deps.RuntimeLogger(deps.Ctx, "agent_security_privacy", nil)
	guard := agentsec.NewPrivacyGuard(deps.DB, deps.Config, logger)
	auditPath := deps.Config.SecurityBaselineConfig().ConsentDefaults.AuditChannel
	auditWriter, _ := secobs.NewFileAuditWriter(auditPath)
	h := NewPrivacyHandler(deps, guard, auditWriter)
	toolgrant := NewToolGrantHandler(deps)
	sec := rg.Group("/security")
	{
		sec.GET("/privacy/consent", h.GetActiveConsent)
		sec.POST("/privacy/lifecycle", h.AcknowledgeLifecycleEvent)
		sec.POST("/toolgrants/verify", toolgrant.Verify)
	}
}
