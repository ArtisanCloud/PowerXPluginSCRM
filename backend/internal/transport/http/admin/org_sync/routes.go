package org_sync

import (
	orgrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/org_sync"
	orgsvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/org_sync"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
	httpmw "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/middleware"
	"github.com/gin-gonic/gin"
)

// RegisterRoutes wires org sync admin endpoints.
func RegisterRoutes(rg *gin.RouterGroup, deps *app.Deps) {
	if rg == nil || deps == nil {
		return
	}
	var (
		syncSvc   *orgsvc.SyncService
		unitSvc   *orgsvc.SourceUnitService
		memberSvc *orgsvc.SourceMemberService
	)
	if deps.DB != nil {
		syncSvc = orgsvc.NewSyncService(orgrepo.NewSourceAccountRepository(deps.DB))
		unitSvc = orgsvc.NewSourceUnitService(orgrepo.NewSourceUnitRepository(deps.DB))
		memberSvc = orgsvc.NewSourceMemberService(orgrepo.NewSourceMemberRepository(deps.DB))
	}
	handler := NewOrgSyncHandler(syncSvc, unitSvc, memberSvc)
	var mappingHandler *MappingHandler
	if deps.DB != nil {
		memberRepo := orgrepo.NewSourceMemberRepository(deps.DB)
		memberMappingRepo := orgrepo.NewMemberMappingRepository(deps.DB)
		unitRepo := orgrepo.NewSourceUnitRepository(deps.DB)
		unitMappingRepo := orgrepo.NewUnitMappingRepository(deps.DB)
		matchSvc := orgsvc.NewMatchService(deps.DB, memberRepo, memberMappingRepo)
		mappingSvc := orgsvc.NewMappingService(unitRepo, memberRepo, unitMappingRepo, memberMappingRepo)
		mappingHandler = NewMappingHandler(matchSvc, mappingSvc)
	}
	if mappingHandler == nil {
		mappingHandler = NewMappingHandler(nil, nil)
	}
	group := rg.Group("/org-sync", httpmw.EnsureTenant())
	{
		group.POST("/source-accounts/:source_account_uuid/sync", handler.TriggerSync)
		group.GET("/source-units", handler.ListSourceUnits)
		group.GET("/source-members", handler.ListSourceMembers)
		group.GET("/mappings/suggestions", mappingHandler.Suggestions)
		group.POST("/mappings/confirm", mappingHandler.Confirm)
	}
}
