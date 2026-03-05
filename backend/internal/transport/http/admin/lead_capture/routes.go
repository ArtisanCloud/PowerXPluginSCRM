package lead_capture

import (
	"os"
	"strings"

	fwwsbus "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/wsbus"
	leadrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/lead_capture"
	leadobs "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/observability/lead_capture"
	leadsvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/lead_capture"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
	httpmw "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/middleware"
	"github.com/gin-gonic/gin"
)

// RegisterRoutes wires lead capture admin endpoints.
func RegisterRoutes(rg *gin.RouterGroup, deps *app.Deps) {
	if rg == nil || deps == nil {
		return
	}

	var (
		leadSvc         *leadsvc.LeadService
		wecomSyncSvc    *leadsvc.WeComSyncService
		conversationSvc *leadsvc.ConversationService
	)
	if deps.DB != nil {
		leadRepository := leadrepo.NewLeadRepository(deps.DB)
		leadSvc = leadsvc.NewLeadService(leadRepository)

		metrics := deps.LeadCaptureMetrics
		if metrics == nil {
			metrics = leadobs.NewMetrics()
		}
		taskRepo := leadrepo.NewLeadSyncTaskRepository(deps.DB)
		providerAdapter := leadsvc.NewDefaultSyncTaskProviderAdapter(deps.Config, deps.EventEmitter)
		wecomSyncSvc = leadsvc.NewWeComSyncService(taskRepo, metrics, providerAdapter).
			WithLeadIngestion(leadRepository, leadsvc.NewDefaultWeComLeadAdapter()).
			WithLeadService(leadSvc)

		eventRepo := leadrepo.NewConversationEventRepository(deps.DB)
		bindingRepo := leadrepo.NewLeadConversationBindingRepository(deps.DB)
		pendingRepo := leadrepo.NewLeadConversationPendingRepository(deps.DB)
		projectionRepo := leadrepo.NewLeadRealtimeProjectionRepository(deps.DB)

		publisher := fwwsbus.NewAdapter(
			fwwsbus.NewLocalPublisher(deps.WSBusHub, nil),
			"",
			nil,
		)
		if deps.Config != nil && deps.Config.Gateway != nil && strings.TrimSpace(os.Getenv("POWERX_PROXY")) == "1" {
			hostTenantUUID := strings.TrimSpace(deps.Config.Gateway.TenantUUID)
			if strings.TrimSpace(os.Getenv("POWERX_PROXY")) == "1" {
				hostTenantUUID = ""
			}
			if hostClient, err := fwwsbus.NewHostClient(fwwsbus.HostClientConfig{
				BaseURL:    strings.TrimSpace(deps.Config.Gateway.BaseURL),
				APIPrefix:  strings.TrimSpace(deps.Config.Gateway.APIPrefix),
				AuthScheme: strings.TrimSpace(deps.Config.Gateway.AuthScheme),
				Token:      strings.TrimSpace(deps.Config.Gateway.ToolToken),
				APIKey:     strings.TrimSpace(deps.Config.Gateway.APIKey),
				TenantUUID: hostTenantUUID,
				UserAgent:  strings.TrimSpace(deps.Config.Gateway.UserAgent),
				Timeout:    deps.Config.Gateway.Timeout,
			}); err == nil {
				publisher = fwwsbus.NewAdapter(hostClient, "", nil)
			}
		}
		realtime := leadsvc.NewConversationRealtimePublisher(publisher, metrics)
		conversationSvc = leadsvc.NewConversationService(eventRepo, bindingRepo, pendingRepo, projectionRepo, realtime, metrics).
			WithLeadRepository(leadRepository)
	}

	handler := NewLeadHandler(leadSvc)
	wecomSyncHandler := NewWeComSyncHandler(wecomSyncSvc)
	conversationHandler := NewConversationHandler(conversationSvc)
	group := rg.Group("/leads", httpmw.EnsureTenant())
	{
		group.GET("", handler.List)
		group.POST("", handler.Create)
		group.POST("/import", handler.Import)
		group.POST("/import/preview", handler.ImportPreview)
		group.POST("/import/confirm", handler.ImportConfirm)
		group.GET("/:lead_id", handler.Get)
		group.POST("/:lead_id/assign", handler.Assign)
		group.POST("/:lead_id/status", handler.UpdateStatus)
		group.GET("/:lead_id/assignments", handler.ListAssignments)
		group.GET("/:lead_id/status-history", handler.ListStatusHistory)
		group.GET("/:lead_id/activities", handler.ListActivities)
		group.GET("/:lead_id/sources", handler.ListSourceEvents)

		group.POST("/wecom/sync", wecomSyncHandler.TriggerSync)
		group.GET("/wecom/sync-tasks", wecomSyncHandler.ListSyncTasks)
		group.GET("/:lead_id/conversations", conversationHandler.ListLeadConversations)
		group.POST("/:lead_id/conversations/bind", conversationHandler.BindConversation)
	}

	conversationGroup := rg.Group("/conversations", httpmw.EnsureTenant())
	{
		conversationGroup.GET("/:conversation_id/events", conversationHandler.ListConversationEvents)
	}
}
