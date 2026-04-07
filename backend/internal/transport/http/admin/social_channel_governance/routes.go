package social_channel_governance

import (
	orgrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/org_sync"
	SocialRepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/social_channel_governance"
	orgsync "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/org_sync"
	orgdriver "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/org_sync/driver"
	SocialService "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/social_channel_governance"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
	httpmw "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/middleware"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// RegisterRoutes wires social channel governance admin endpoints.
func RegisterRoutes(rg *gin.RouterGroup, deps *app.Deps) {
	if rg == nil || deps == nil {
		return
	}
	orgdriver.ConfigureWeComCache(deps.Config)

	var accountSvc *SocialService.ChannelAccountService
	var memberSvc *SocialService.ChannelAccountMemberService
	var capabilitySvc *SocialService.ChannelAccountCapabilityService
	var openworkHandler *OpenWorkFoundationHandler
	var platformSettingHandler *ChannelPlatformSettingHandler
	var syncJobHandler *SyncJobHandler
	var conflictHandler *ConflictHandler
	schemaLoader := SocialService.NewChannelSchemaLoader(SocialService.ChannelSchemaLoaderOptions{
		Logger: logrus.WithField("module", "social_channel_governance"),
	})
	if deps.DB != nil {
		repo := SocialRepo.NewAccountRepository(deps.DB)
		openworkRepo := SocialRepo.NewOpenWorkFoundationRepository(deps.DB)
		syncRepo := SocialRepo.NewSyncFoundationRepository(deps.DB)
		platformSettingRepo := SocialRepo.NewChannelPlatformSettingRepository(deps.DB)
		accountStatus := orgsync.NewAccountStatusService(
			orgrepo.NewMemberMappingRepository(deps.DB),
			orgrepo.NewUnitMappingRepository(deps.DB),
		)
		accountSvc = SocialService.NewChannelAccountService(repo, openworkRepo, nil, schemaLoader, accountStatus, deps.Config)
		memberSvc = SocialService.NewChannelAccountMemberService(repo)
		capabilitySvc = SocialService.NewChannelAccountCapabilityService(repo)
		openworkHandler = NewOpenWorkFoundationHandler(SocialService.NewOpenWorkFoundationService(openworkRepo, repo))
		platformSettingHandler = NewChannelPlatformSettingHandler(SocialService.NewChannelPlatformSettingService(platformSettingRepo))
		factory := SocialService.NewChannelFactory()
		capabilityMatrixSvc := SocialService.NewCapabilityService(factory)
		idempotencySvc := SocialService.NewIdempotencyService()
		scheduler := SocialService.NewSyncScheduler()
		syncJobSvc := SocialService.NewSyncJobService(syncRepo, idempotencySvc, scheduler, capabilityMatrixSvc)
		orchestrator := SocialService.NewSyncOrchestrator(factory, scheduler, syncJobSvc)
		syncJobHandler = NewSyncJobHandler(syncJobSvc, orchestrator, capabilityMatrixSvc)
		conflictHandler = NewConflictHandler()
	}
	accountHandler := NewAccountHandler(accountSvc)
	schemaHandler := NewChannelSchemaHandler(schemaLoader, deps.Config)
	membersHandler := NewChannelAccountMembersHandler(memberSvc)
	capabilityHandler := NewChannelAccountCapabilityHandler(capabilitySvc)

	group := rg.Group("/social", httpmw.EnsureTenant())
	{
		group.GET("/channel-schema", schemaHandler.GetSchema)
		group.GET("/channel-accounts", accountHandler.ListAccounts)
		group.GET("/channel-accounts/deleted", accountHandler.ListDeletedAccounts)
		group.POST("/channel-accounts", accountHandler.CreateAccount)
		group.GET("/channel-accounts/:account_uuid", accountHandler.GetAccount)
		group.PUT("/channel-accounts/:account_uuid", accountHandler.UpdateAccount)
		group.POST("/channel-accounts/:account_uuid/test-connection", accountHandler.TestConnection)
		group.POST("/channel-accounts/:account_uuid/test-app-secret", accountHandler.TestAppSecret)
		group.POST("/channel-accounts/:account_uuid/test-contact-secret", accountHandler.TestContactSecret)
		group.DELETE("/channel-accounts/:account_uuid", accountHandler.DeleteAccount)
		group.POST("/channel-accounts/:account_uuid/restore", accountHandler.RestoreAccount)
		group.POST("/channel-accounts/:account_uuid/channel-members", membersHandler.UpdateChannelAccountMembers)
		group.PATCH("/channel-accounts/:account_uuid/capabilities", capabilityHandler.UpdateChannelAccountCapabilities)

		if openworkHandler != nil {
			group.POST("/openwork/wecom/events", openworkHandler.IngestEvent)
			group.POST("/openwork/wecom/authorize/start", openworkHandler.StartAuthorization)
			group.POST("/openwork/wecom/authorize/complete", openworkHandler.CompleteAuthorization)
			group.GET("/openwork/wecom/authorize/status", openworkHandler.GetAuthorizationStatus)
			group.GET("/openwork/wecom/bindings", openworkHandler.ListBindings)
			group.POST("/openwork/wecom/bindings/:binding_uuid/default", openworkHandler.SetDefaultBinding)
			group.POST("/openwork/wecom/sync/jobs", openworkHandler.CreateSyncJob)
			group.GET("/openwork/wecom/sync/jobs", openworkHandler.ListSyncJobs)
			group.GET("/openwork/wecom/sync/conflicts", openworkHandler.ListSyncConflicts)
			group.POST("/openwork/wecom/sync/conflicts/:conflict_uuid/replay", openworkHandler.ReplaySyncConflict)
			group.GET("/openwork/wecom/sync/dashboard", openworkHandler.GetDashboard)
			group.GET("/openwork/wecom/go-live-gates", openworkHandler.GetGoLiveGates)
		}
		if syncJobHandler != nil && conflictHandler != nil {
			group.POST("/openwork/foundation/sync/jobs", syncJobHandler.Create)
			group.GET("/openwork/foundation/sync/jobs", syncJobHandler.List)
			group.GET("/openwork/foundation/capabilities", syncJobHandler.Capabilities)
			group.GET("/openwork/foundation/sync/conflicts", conflictHandler.List)
			group.POST("/openwork/foundation/sync/conflicts/:conflict_uuid/replay", conflictHandler.Replay)
			group.GET("/openwork/foundation/sync/dead-letters", conflictHandler.ListDeadLetters)
			group.POST("/openwork/foundation/sync/dead-letters/:dead_letter_uuid/replay", conflictHandler.ReplayDeadLetter)
		}
		if platformSettingHandler != nil {
			platformGroup := group.Group("/channel-platform/wecom/openwork", httpmw.EnsureRootRole())
			platformGroup.GET("", platformSettingHandler.GetWeComOpenWork)
			platformGroup.PUT("", platformSettingHandler.SaveWeComOpenWork)
			platformGroup.GET("/templates", platformSettingHandler.ListWeComOpenWorkTemplates)
			platformGroup.POST("/templates", platformSettingHandler.CreateWeComOpenWorkTemplate)
			platformGroup.PUT("/templates/:template_id", platformSettingHandler.UpdateWeComOpenWorkTemplate)
			platformGroup.DELETE("/templates/:template_id", platformSettingHandler.DeleteWeComOpenWorkTemplate)
			platformGroup.POST("/templates/:template_id/default", platformSettingHandler.SetDefaultWeComOpenWorkTemplate)
			platformGroup.POST("/suite-ticket/refresh", platformSettingHandler.RefreshWeComSuiteTicket)
			platformGroup.POST("/suite-ticket/verify", platformSettingHandler.VerifyWeComSuiteTicket)
		}
	}
}
