package webhooks

import (
	leadrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/lead_capture"
	orgrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/org_sync"
	socialrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/social_channel_governance"
	leadobs "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/observability/lead_capture"
	leadsvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/lead_capture"
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
	}
}
