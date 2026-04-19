package webhooks

import (
	"os"
	"strings"

	fwwsbus "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/wsbus"
	acqrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/repository/acquisition"
	leadrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/lead_capture"
	orgrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/org_sync"
	socialrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/social_channel_governance"
	leadobs "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/observability/lead_capture"
	acqsvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/acquisition"
	leadsvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/lead_capture"
	socialsvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/social_channel_governance"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
)

// RegisterRoutes registers public webhook endpoints.
func RegisterRoutes(rg *gin.RouterGroup, deps *app.Deps) {
	if rg == nil || deps == nil || deps.DB == nil {
		return
	}
	repo := socialrepo.NewAccountRepository(deps.DB)
	sourceRepo := orgrepo.NewSourceAccountRepository(deps.DB)
	memberRepo := orgrepo.NewSourceMemberRepository(deps.DB)
	profileRepo := orgrepo.NewSourceMemberProfileRepository(deps.DB)
	wecomHandler := NewWeComWebhookHandler(repo, sourceRepo, memberRepo, profileRepo, deps.Config)

	metrics := deps.LeadCaptureMetrics
	if metrics == nil {
		metrics = leadobs.NewMetrics()
	}
	domainRepos := deps.EnsureLeadCaptureRepos()
	conversationSvc := leadsvc.NewConversationService(
		leadrepo.NewConversationEventRepository(deps.DB),
		leadrepo.NewLeadConversationBindingRepository(deps.DB),
		leadrepo.NewLeadConversationPendingRepository(deps.DB),
		leadrepo.NewLeadRealtimeProjectionRepository(deps.DB),
		nil,
		metrics,
	)
	leadRepository := leadrepo.NewLeadRepository(deps.DB)
	conversationSvc = conversationSvc.
		WithLeadRepository(leadRepository).
		WithLeadService(leadsvc.NewLeadService(leadRepository)).
		WithChannelRuleRepository(leadrepo.NewChannelRuleRepository(deps.DB))
	conversationHandler := NewWeComConversationWebhookHandler(conversationSvc, repo)
	botCommandSvc := leadsvc.NewBotCommandService(
		leadrepo.NewConversationEventRepository(deps.DB),
		leadrepo.NewLeadConversationBindingRepository(deps.DB),
		leadRepository,
		leadsvc.NewLeadService(leadRepository),
	)
	botCommandHandler := NewWeComBotCommandHandler(botCommandSvc, repo)
	attributionSvc := leadsvc.NewAttributionService(domainRepos.Attributions, leadRepository, leadsvc.NewLeadService(leadRepository))
	channelCodeEventSvc := leadsvc.NewChannelCodeEventService(
		domainRepos.ChannelCodeEvents,
		domainRepos.ChannelCodes,
		repo,
		attributionSvc,
		metrics,
	)
	channelCodeEventHandler := NewChannelCodeEventsWebhookHandler(channelCodeEventSvc)
	acquisitionCodeEventHandler := NewAcquisitionCodeEventsWebhookHandler()
	groupChatWebhookHandler := NewAcquisitionGroupChatWebhookHandler(
		acqsvc.NewGroupChatSyncService(acqrepo.NewGroupChatSnapshotRepository(deps.DB)),
	)

	group := rg.Group("/webhooks")
	{
		group.GET("/wechat/wecom/:account_uuid", wecomHandler.Handle)
		group.POST("/wechat/wecom/:account_uuid", wecomHandler.Handle)
		group.GET("/wechat/wecom/:account_uuid/oauth", wecomHandler.HandleOAuth)
		group.GET("/wechat/wecom/:account_uuid/oauth/callback", wecomHandler.HandleOAuthCallback)
		group.POST("/wecom/conversations", conversationHandler.Ingest)
		group.POST("/wecom/bot/commands", botCommandHandler.Ingest)
		group.POST("/channels/:channel/code-events", channelCodeEventHandler.Ingest)
		group.POST("/channels/:channel/staff-code-events", acquisitionCodeEventHandler.IngestStaff)
		group.POST("/channels/:channel/group-code-events", acquisitionCodeEventHandler.IngestGroup)
		group.POST("/channels/:channel/group-chat-events", groupChatWebhookHandler.Ingest)
	}
}

// RegisterPublicRoutes registers webhook endpoints that must be reachable without admin JWT.
func RegisterPublicRoutes(rg *gin.RouterGroup, deps *app.Deps) {
	if rg == nil || deps == nil || deps.DB == nil {
		return
	}
	platformRepo := socialrepo.NewChannelPlatformSettingRepository(deps.DB)
	openWorkRepo := socialrepo.NewOpenWorkFoundationRepository(deps.DB)
	accountRepo := socialrepo.NewAccountRepository(deps.DB)
	openWorkFoundationSvc := socialsvc.NewOpenWorkFoundationService(openWorkRepo, accountRepo)
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
	openWorkHandler := NewOpenWorkCallbackHandler(platformRepo, openWorkRepo, openWorkFoundationSvc, deps, publisher)

	group := rg.Group("/webhooks")
	{
		group.GET("/wecom/openwork", openWorkHandler.Handle)
		group.POST("/wecom/openwork", openWorkHandler.Handle)
	}
}
