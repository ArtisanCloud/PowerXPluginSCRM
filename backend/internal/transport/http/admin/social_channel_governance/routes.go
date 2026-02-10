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
	schemaLoader := SocialService.NewChannelSchemaLoader(SocialService.ChannelSchemaLoaderOptions{
		Logger: logrus.WithField("module", "social_channel_governance"),
	})
	if deps.DB != nil {
		repo := SocialRepo.NewAccountRepository(deps.DB)
		accountStatus := orgsync.NewAccountStatusService(
			orgrepo.NewMemberMappingRepository(deps.DB),
			orgrepo.NewUnitMappingRepository(deps.DB),
		)
		accountSvc = SocialService.NewChannelAccountService(repo, nil, schemaLoader, accountStatus, deps.Config)
		memberSvc = SocialService.NewChannelAccountMemberService(repo)
		capabilitySvc = SocialService.NewChannelAccountCapabilityService(repo)
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
	}
}
