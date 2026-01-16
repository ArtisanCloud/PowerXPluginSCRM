package social_channel_governance

import (
	SocialRepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/social_channel_governance"
	SocialService "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/social_channel_governance"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
	httpmw "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/middleware"
	"github.com/gin-gonic/gin"
)

// RegisterRoutes wires social channel governance admin endpoints.
func RegisterRoutes(rg *gin.RouterGroup, deps *app.Deps) {
	if rg == nil || deps == nil {
		return
	}

	var accountSvc *SocialService.ChannelAccountService
	var memberSvc *SocialService.ChannelAccountMemberService
	if deps.DB != nil {
		repo := SocialRepo.NewAccountRepository(deps.DB)
		accountSvc = SocialService.NewChannelAccountService(repo, nil)
		memberSvc = SocialService.NewChannelAccountMemberService(repo)
	}
	accountHandler := NewAccountHandler(accountSvc)
	membersHandler := NewChannelAccountMembersHandler(memberSvc)

	group := rg.Group("/social", httpmw.EnsureTenant())
	{
		group.GET("/channel-accounts", accountHandler.ListAccounts)
		group.POST("/channel-accounts", accountHandler.CreateAccount)
		group.GET("/channel-accounts/:account_uuid", accountHandler.GetAccount)
		group.POST("/channel-accounts/:account_uuid/channel-members", membersHandler.UpdateChannelAccountMembers)
	}
}
