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
	schemaLoader := SocialService.NewChannelSchemaLoader(SocialService.ChannelSchemaLoaderOptions{
		Logger: logrus.WithField("module", "social_channel_governance"),
	})
	if deps.DB != nil {
		repo := SocialRepo.NewAccountRepository(deps.DB)
		openworkRepo := SocialRepo.NewOpenWorkFoundationRepository(deps.DB)
		platformSettingRepo := SocialRepo.NewChannelPlatformSettingRepository(deps.DB)
		accountStatus := orgsync.NewAccountStatusService(
			orgrepo.NewMemberMappingRepository(deps.DB),
			orgrepo.NewUnitMappingRepository(deps.DB),
		)
		accountSvc = SocialService.NewChannelAccountService(repo, nil, schemaLoader, accountStatus, deps.Config)
		memberSvc = SocialService.NewChannelAccountMemberService(repo)
		capabilitySvc = SocialService.NewChannelAccountCapabilityService(repo)
		openworkHandler = NewOpenWorkFoundationHandler(SocialService.NewOpenWorkFoundationService(openworkRepo, repo))
		platformSettingHandler = NewChannelPlatformSettingHandler(SocialService.NewChannelPlatformSettingService(platformSettingRepo))
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
		if platformSettingHandler != nil {
			group.GET("/channel-platform/wecom/openwork", platformSettingHandler.GetWeComOpenWork)
			group.PUT("/channel-platform/wecom/openwork", platformSettingHandler.SaveWeComOpenWork)
			group.GET("/channel-platform/wecom/openwork/templates", platformSettingHandler.ListWeComOpenWorkTemplates)
			group.POST("/channel-platform/wecom/openwork/templates", platformSettingHandler.CreateWeComOpenWorkTemplate)
			group.PUT("/channel-platform/wecom/openwork/templates/:template_id", platformSettingHandler.UpdateWeComOpenWorkTemplate)
			group.DELETE("/channel-platform/wecom/openwork/templates/:template_id", platformSettingHandler.DeleteWeComOpenWorkTemplate)
			group.POST("/channel-platform/wecom/openwork/templates/:template_id/default", platformSettingHandler.SetDefaultWeComOpenWorkTemplate)
			group.POST("/channel-platform/wecom/openwork/suite-ticket/refresh", platformSettingHandler.RefreshWeComSuiteTicket)
			group.POST("/channel-platform/wecom/openwork/suite-ticket/verify", platformSettingHandler.VerifyWeComSuiteTicket)
		}
	}
}
