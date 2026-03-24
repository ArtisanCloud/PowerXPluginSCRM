package lead_capture

import (
	"os"
	"strings"

	fwwsbus "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/wsbus"
	leadrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/lead_capture"
	socialrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/social_channel_governance"
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
		leadSvc          *leadsvc.LeadService
		wecomSyncSvc     *leadsvc.WeComSyncService
		conversationSvc  *leadsvc.ConversationService
		sourceCatalogSvc *leadsvc.LeadSourceCatalogService
		channelRuleSvc   *leadsvc.ChannelRuleService
		channelCodeSvc   *leadsvc.ChannelCodeService
		welcomeConfigSvc *leadsvc.WelcomeConfigService
	)
	if deps.DB != nil {
		leadRepository := leadrepo.NewLeadRepository(deps.DB)
		leadSvc = leadsvc.NewLeadService(leadRepository)
		sourceCatalogSvc = leadsvc.NewLeadSourceCatalogService(leadrepo.NewLeadSourceCatalogRepository(deps.DB))
		channelRuleRepo := leadrepo.NewChannelRuleRepository(deps.DB)
		channelRuleSvc = leadsvc.NewChannelRuleService(channelRuleRepo)
		domainRepos := deps.EnsureLeadCaptureRepos()

		metrics := deps.LeadCaptureMetrics
		if metrics == nil {
			metrics = leadobs.NewMetrics()
		}
		if domainRepos != nil {
			channelCodeSvc = leadsvc.NewChannelCodeService(domainRepos.ChannelCodes, metrics)
			welcomeConfigSvc = leadsvc.NewWelcomeConfigService(domainRepos.WelcomeConfigs, domainRepos.ConfigChangeLogs, metrics)
		}
		taskRepo := leadrepo.NewLeadSyncTaskRepository(deps.DB)
		channelAccountRepo := socialrepo.NewAccountRepository(deps.DB)
		providerAdapter := leadsvc.NewDefaultSyncTaskProviderAdapter(deps.Config, deps.EventEmitter)
		channelFactory := leadsvc.NewChannelSyncFactory()
		defaultIdentity := deps.DefaultLeadSyncChannelIdentity()
		_ = channelFactory.Register(
			defaultIdentity.Channel,
			defaultIdentity.AppType,
			leadsvc.NewDefaultWeComLeadAdapterWithAccountRepo(channelAccountRepo),
			providerAdapter,
		)
		wecomSyncSvc = leadsvc.NewWeComSyncService(taskRepo, metrics, nil).
			WithChannelFactory(channelFactory).
			WithLeadIngestion(leadRepository, nil).
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
			WithLeadRepository(leadRepository).
			WithLeadService(leadSvc).
			WithChannelRuleRepository(channelRuleRepo)
	}

	handler := NewLeadHandler(leadSvc)
	wecomSyncHandler := NewWeComSyncHandler(wecomSyncSvc)
	conversationHandler := NewConversationHandler(conversationSvc)
	sourceCatalogHandler := NewSourceCatalogHandler(sourceCatalogSvc)
	channelRuleHandler := NewChannelRuleHandler(channelRuleSvc)
	channelCodeHandler := NewChannelCodeHandler(channelCodeSvc)
	welcomeConfigHandler := NewWelcomeConfigHandler(welcomeConfigSvc)
	group := rg.Group("/leads", httpmw.EnsureTenant())
	{
		group.GET("", handler.List)
		group.POST("", handler.Create)
		group.GET("/source-catalogs", sourceCatalogHandler.List)
		group.POST("/source-catalogs", sourceCatalogHandler.Create)
		group.PATCH("/source-catalogs/:catalog_id", sourceCatalogHandler.Update)
		group.DELETE("/source-catalogs/:catalog_id", sourceCatalogHandler.Delete)
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
		group.GET("/channel-rules/wecom/customer-dm", channelRuleHandler.GetWeComCustomerDMRule)
		group.PUT("/channel-rules/wecom/customer-dm", channelRuleHandler.UpdateWeComCustomerDMRule)
		group.POST("/channel-codes", channelCodeHandler.Create)
		group.GET("/channel-codes", channelCodeHandler.List)
		group.PATCH("/channel-codes/:code_uuid/status", channelCodeHandler.UpdateStatus)
		group.PUT("/channel-codes/:code_uuid/welcome-config", welcomeConfigHandler.Save)
		group.GET("/channel-codes/:code_uuid/welcome-config/history", welcomeConfigHandler.ListHistory)
		group.GET("/:lead_id/conversations", conversationHandler.ListLeadConversations)
		group.POST("/:lead_id/conversations/bind", conversationHandler.BindConversation)
	}

	conversationGroup := rg.Group("/conversations", httpmw.EnsureTenant())
	{
		conversationGroup.GET("/:conversation_id/events", conversationHandler.ListConversationEvents)
	}
}
