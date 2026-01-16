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

	var svc *SocialService.AccountService
	if deps.DB != nil {
		repo := SocialRepo.NewAccountRepository(deps.DB)
		svc = SocialService.NewAccountService(repo, nil)
	}
	handler := NewAccountHandler(svc)

	group := rg.Group("/social", httpmw.EnsureTenant())
	{
		group.GET("/channel-accounts", handler.ListAccounts)
		group.POST("/channel-accounts", handler.CreateAccount)
		group.GET("/channel-accounts/:account_uuid", handler.GetAccount)
	}
}
