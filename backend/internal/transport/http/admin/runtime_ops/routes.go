package runtime_ops

import (
	runtimeops "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/runtime_ops"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
)

// RegisterRoutes wires runtime ops endpoints behind the admin router.
func RegisterRoutes(router *gin.RouterGroup, deps *app.Deps) {
	bootstrap := NewBootstrapHandler(runtimeops.NewService())
	router.POST("/bootstrap", bootstrap.Bootstrap)

	sessions := NewSessionsHandler(deps)
	router.POST("/sessions/register", sessions.Register)
	router.POST("/sessions/:sessionID/ack", sessions.Ack)
	router.POST("/sessions/:sessionID/heartbeat", sessions.Heartbeat)
	router.POST("/sessions/:sessionID/close", sessions.Close)
	router.POST("/sessions/:sessionID/invoke", sessions.Invoke)

	quotaHandler := NewQuotaHandler(deps, deps.Config.RuntimeOps)
	quota := router.Group("/quota")
	quota.GET("/status", quotaHandler.GetStatus)
	quota.POST("/overrides", quotaHandler.SetOverride)

	router.GET("/metrics", MetricsHandler)

	if deps != nil && deps.Config != nil && deps.Config.Server.DevMode {
		router.POST("/event-bridge/emit", EventBridgeEmitHandler(deps))
	}
}
