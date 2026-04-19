package acquisition

import (
	"context"

	acqrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/repository/acquisition"
	socialrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/social_channel_governance"
	acqsvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/acquisition"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
	httpmw "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/middleware"
	"github.com/gin-gonic/gin"
)

type defaultAccountResolver struct {
	repo *socialrepo.AccountRepository
}

func (r *defaultAccountResolver) ResolveDefaultChannelAccount(ctx context.Context, tenantUUID, channel, appType string) (string, error) {
	if r == nil || r.repo == nil {
		return "", socialrepo.ErrAccountNotFound
	}
	return r.repo.ResolveDefaultAccountUUID(ctx, tenantUUID, channel, appType)
}

func RegisterRoutes(rg *gin.RouterGroup, deps *app.Deps) {
	if rg == nil || deps == nil || deps.DB == nil {
		return
	}
	repos := acqrepo.NewBundle(deps.DB)
	accountResolver := &defaultAccountResolver{repo: socialrepo.NewAccountRepository(deps.DB)}
	staffSvc := acqsvc.NewStaffLiveCodeService(repos.StaffLiveCodes, accountResolver)
	staffWelcomeSvc := acqsvc.NewStaffWelcomeService(repos.StaffLiveCodes, repos.StaffWelcomeConfigs, repos.StaffWelcomeAttempt)
	groupSvc := acqsvc.NewGroupLiveCodeService(repos.GroupLiveCodes)
	groupChatSvc := acqsvc.NewGroupChatSyncService(repos.GroupChatSnapshots)
	groupTagSvc := acqsvc.NewGroupTagService(repos.GroupTags, repos.GroupChatSnapshots, acqsvc.NewGroupTagRuleService())

	staffHandler := NewStaffLiveCodeHandler(staffSvc)
	staffWelcomeHandler := NewStaffWelcomeHandler(staffWelcomeSvc)
	groupHandler := NewGroupLiveCodeHandler(groupSvc)
	groupChatHandler := NewGroupChatSyncHandler(groupChatSvc)
	groupTagHandler := NewGroupTagHandler(groupTagSvc)

	group := rg.Group("/leads/acquisition", httpmw.EnsureTenant())
	{
		group.POST("/staff-codes", staffHandler.Create)
		group.GET("/staff-codes", staffHandler.List)
		group.GET("/staff-codes/code-key-available", staffHandler.CheckCodeKeyAvailable)
		group.PATCH("/staff-codes/:staff_code_uuid/status", staffHandler.UpdateStatus)
		group.PUT("/staff-codes/:staff_code_uuid/welcome-config", staffWelcomeHandler.Save)
		group.POST("/staff-codes/:staff_code_uuid/welcome-config/sync", staffWelcomeHandler.TriggerSync)
		group.GET("/staff-codes/:staff_code_uuid/welcome-config/sync-status", staffWelcomeHandler.GetSyncStatus)

		group.POST("/group-codes", groupHandler.Create)
		group.GET("/group-codes", groupHandler.List)
		group.GET("/group-codes/:group_code_uuid", groupHandler.Get)
		group.PUT("/group-codes/:group_code_uuid", groupHandler.Update)
		group.DELETE("/group-codes/:group_code_uuid", groupHandler.Delete)
		group.POST("/group-codes/:group_code_uuid/sync", groupHandler.Sync)

		group.POST("/group-chats/sync", groupChatHandler.Sync)
		group.GET("/group-chats", groupChatHandler.List)
		group.GET("/group-chats/:chat_id", groupChatHandler.Get)

		group.POST("/group-tags", groupTagHandler.Create)
		group.GET("/group-tags", groupTagHandler.List)
		group.POST("/group-tags/:group_tag_uuid/bindings", groupTagHandler.Bind)
		group.GET("/group-tags/:group_tag_uuid/bindings", groupTagHandler.ListBindings)
		group.POST("/group-tags/:group_tag_uuid/rules/replay", groupTagHandler.ReplayRule)
	}
}
