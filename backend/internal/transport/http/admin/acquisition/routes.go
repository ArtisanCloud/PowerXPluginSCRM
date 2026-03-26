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

	staffHandler := NewStaffLiveCodeHandler(staffSvc)
	staffWelcomeHandler := NewStaffWelcomeHandler(staffWelcomeSvc)
	groupHandler := NewGroupLiveCodeHandler(groupSvc)

	group := rg.Group("/leads/acquisition", httpmw.EnsureTenant())
	{
		group.POST("/staff-codes", staffHandler.Create)
		group.GET("/staff-codes", staffHandler.List)
		group.GET("/staff-codes/code-key-available", staffHandler.CheckCodeKeyAvailable)
		group.PATCH("/staff-codes/:staff_code_uuid/status", staffHandler.UpdateStatus)
		group.PUT("/staff-codes/:staff_code_uuid/welcome-config", staffWelcomeHandler.Save)
		group.POST("/staff-codes/:staff_code_uuid/welcome-config/sync", staffWelcomeHandler.TriggerSync)
		group.GET("/staff-codes/:staff_code_uuid/welcome-config/sync-status", staffWelcomeHandler.GetSyncStatus)
		group.GET("/group-codes", groupHandler.List)
	}
}
