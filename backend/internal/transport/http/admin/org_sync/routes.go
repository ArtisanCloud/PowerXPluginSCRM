package org_sync

import (
	"os"
	"strings"

	fwwsbus "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/wsbus"
	orgrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/org_sync"
	socialrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/social_channel_governance"
	orgsvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/org_sync"
	orgdriver "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/org_sync/driver"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
	httpmw "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/middleware"
	"github.com/gin-gonic/gin"
)

// RegisterRoutes wires org sync admin endpoints.
func RegisterRoutes(rg *gin.RouterGroup, deps *app.Deps) {
	if rg == nil || deps == nil {
		return
	}
	orgdriver.ConfigureWeComCache(deps.Config)
	var (
		syncSvc    *orgsvc.SyncService
		unitSvc    *orgsvc.SourceUnitService
		memberSvc  *orgsvc.SourceMemberService
		syncLogSvc *orgsvc.SyncLogService
		defaultSvc *orgsvc.DefaultSourceAccountService
	)
	if deps.DB != nil {
		publisher := fwwsbus.NewAdapter(
			fwwsbus.NewLocalPublisher(deps.WSBusHub, nil),
			"",
			nil,
		)
		if deps != nil && deps.Config != nil && deps.Config.Gateway != nil && strings.TrimSpace(os.Getenv("POWERX_PROXY")) == "1" {
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

		sourceRepo := orgrepo.NewSourceAccountRepository(deps.DB)
		unitRepo := orgrepo.NewSourceUnitRepository(deps.DB)
		memberRepo := orgrepo.NewSourceMemberRepository(deps.DB)
		memberProfileRepo := orgrepo.NewSourceMemberProfileRepository(deps.DB)
		unitMappingRepo := orgrepo.NewUnitMappingRepository(deps.DB)
		memberMappingRepo := orgrepo.NewMemberMappingRepository(deps.DB)
		syncLogRepo := orgrepo.NewSyncLogRepository(deps.DB)
		accountRepo := socialrepo.NewAccountRepository(deps.DB)
		syncSvc = orgsvc.NewSyncService(sourceRepo, unitRepo, memberRepo, unitMappingRepo, memberMappingRepo, syncLogRepo, publisher)
		unitSvc = orgsvc.NewSourceUnitService(unitRepo)
		memberSvc = orgsvc.NewSourceMemberService(memberRepo, memberProfileRepo)
		syncLogSvc = orgsvc.NewSyncLogService(syncLogRepo, sourceRepo)
		defaultSvc = orgsvc.NewDefaultSourceAccountService(accountRepo)
	}
	handler := NewOrgSyncHandler(syncSvc, unitSvc, memberSvc, syncLogSvc, defaultSvc)
	var mappingHandler *MappingHandler
	var mainViewHandler *MainViewHandler
	if deps.DB != nil {
		memberRepo := orgrepo.NewSourceMemberRepository(deps.DB)
		memberMappingRepo := orgrepo.NewMemberMappingRepository(deps.DB)
		unitRepo := orgrepo.NewSourceUnitRepository(deps.DB)
		unitMappingRepo := orgrepo.NewUnitMappingRepository(deps.DB)
		matchSvc := orgsvc.NewMatchService(deps.DB, memberRepo, memberMappingRepo)
		mappingSvc := orgsvc.NewMappingService(unitRepo, memberRepo, unitMappingRepo, memberMappingRepo)
		mappingHandler = NewMappingHandler(matchSvc, mappingSvc)
		mainViewHandler = NewMainViewHandler(orgsvc.NewMainViewService(deps.DB, memberMappingRepo))
	}
	if mappingHandler == nil {
		mappingHandler = NewMappingHandler(nil, nil)
	}
	if mainViewHandler == nil {
		mainViewHandler = NewMainViewHandler(nil)
	}
	group := rg.Group("/org-sync", httpmw.EnsureTenant())
	{
		group.POST("/source-accounts/:source_account_uuid/sync", handler.TriggerSync)
		group.POST("/source-accounts/default/:account_uuid", handler.SetDefaultSourceAccount)
		group.GET("/source-units", handler.ListSourceUnits)
		group.GET("/source-members", handler.ListSourceMembers)
		group.GET("/sync-logs", handler.ListSyncLogs)
		group.GET("/mappings/suggestions", mappingHandler.Suggestions)
		group.POST("/mappings/confirm", mappingHandler.Confirm)
		group.GET("/main-org-view", mainViewHandler.List)
	}
}
