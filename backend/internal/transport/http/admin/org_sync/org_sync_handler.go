package org_sync

import (
	"errors"
	"net/http"
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/contracts"
	repository "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository"
	orgrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/org_sync"
	orgsvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/org_sync"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/middleware"
	"github.com/gin-gonic/gin"
)

type OrgSyncHandler struct {
	syncSvc   *orgsvc.SyncService
	unitSvc   *orgsvc.SourceUnitService
	memberSvc *orgsvc.SourceMemberService
}

func NewOrgSyncHandler(syncSvc *orgsvc.SyncService, unitSvc *orgsvc.SourceUnitService, memberSvc *orgsvc.SourceMemberService) *OrgSyncHandler {
	return &OrgSyncHandler{syncSvc: syncSvc, unitSvc: unitSvc, memberSvc: memberSvc}
}

func (h *OrgSyncHandler) TriggerSync(c *gin.Context) {
	if h.syncSvc == nil {
		contracts.ResponseServiceUnavailable(c, "org sync service unavailable", nil)
		return
	}
	sourceAccountUUID := strings.TrimSpace(c.Param("source_account_uuid"))
	if sourceAccountUUID == "" {
		contracts.ResponseBadRequest(c, "source_account_uuid is required")
		return
	}
	tenantUUID, ok := middleware.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	account, err := h.syncSvc.TriggerSync(c.Request.Context(), tenantUUID, sourceAccountUUID)
	if err != nil {
		switch {
		case errors.Is(err, orgrepo.ErrSourceAccountNotFound):
			contracts.ResponseNotFound(c, "source account not found")
		case errors.Is(err, repository.ErrTenantUuidRequired):
			contracts.ResponseBadRequest(c, "tenant_uuid is required")
		default:
			contracts.ResponseInternalError(c, err)
		}
		return
	}
	contracts.ResponseSuccess(c, account)
}

func (h *OrgSyncHandler) ListSourceUnits(c *gin.Context) {
	if h.unitSvc == nil {
		contracts.ResponseServiceUnavailable(c, "org sync service unavailable", nil)
		return
	}
	sourceAccountUUID := strings.TrimSpace(c.Query("source_account_uuid"))
	if sourceAccountUUID == "" {
		contracts.ResponseBadRequest(c, "source_account_uuid is required")
		return
	}
	status := strings.TrimSpace(c.Query("status"))
	var statusPtr *string
	if status != "" {
		statusPtr = &status
	}
	tenantUUID, ok := middleware.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	items, err := h.unitSvc.List(c.Request.Context(), tenantUUID, sourceAccountUUID, statusPtr)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrTenantUuidRequired):
			contracts.ResponseBadRequest(c, "tenant_uuid is required")
		default:
			contracts.ResponseInternalError(c, err)
		}
		return
	}
	contracts.ResponseSuccess(c, gin.H{"items": items})
}

func (h *OrgSyncHandler) ListSourceMembers(c *gin.Context) {
	if h.memberSvc == nil {
		contracts.ResponseServiceUnavailable(c, "org sync service unavailable", nil)
		return
	}
	sourceAccountUUID := strings.TrimSpace(c.Query("source_account_uuid"))
	if sourceAccountUUID == "" {
		contracts.ResponseBadRequest(c, "source_account_uuid is required")
		return
	}
	status := strings.TrimSpace(c.Query("status"))
	var statusPtr *string
	if status != "" {
		statusPtr = &status
	}
	q := strings.TrimSpace(c.Query("q"))
	var qPtr *string
	if q != "" {
		qPtr = &q
	}
	tenantUUID, ok := middleware.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	items, err := h.memberSvc.List(c.Request.Context(), tenantUUID, sourceAccountUUID, statusPtr, qPtr)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrTenantUuidRequired):
			contracts.ResponseBadRequest(c, "tenant_uuid is required")
		default:
			contracts.ResponseInternalError(c, err)
		}
		return
	}
	contracts.ResponseSuccess(c, gin.H{"items": items})
}
