package webhooks

import (
	orgrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/org_sync"
	SocialRepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/social_channel_governance"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
)

// RegisterRoutes registers public webhook endpoints.
func RegisterRoutes(rg *gin.RouterGroup, deps *app.Deps) {
	if rg == nil || deps == nil || deps.DB == nil {
		return
	}
	repo := SocialRepo.NewAccountRepository(deps.DB)
	sourceRepo := orgrepo.NewSourceAccountRepository(deps.DB)
	memberRepo := orgrepo.NewSourceMemberRepository(deps.DB)
	profileRepo := orgrepo.NewSourceMemberProfileRepository(deps.DB)
	handler := NewWeComWebhookHandler(repo, sourceRepo, memberRepo, profileRepo, deps.Config)
	group := rg.Group("/webhooks")
	{
		group.GET("/wechat/wecom/:account_uuid", handler.Handle)
		group.POST("/wechat/wecom/:account_uuid", handler.Handle)
		group.GET("/wechat/wecom/:account_uuid/oauth", handler.HandleOAuth)
		group.GET("/wechat/wecom/:account_uuid/oauth/callback", handler.HandleOAuthCallback)
	}
}
