package org_sync

import (
	"errors"
	"strconv"
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/contracts"
	repository "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository"
	orgrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/org_sync"
	socialrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/social_channel_governance"
	orgsvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/org_sync"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/middleware"
	"github.com/gin-gonic/gin"
)

type OrgSyncHandler struct {
	syncSvc    *orgsvc.SyncService
	unitSvc    *orgsvc.SourceUnitService
	memberSvc  *orgsvc.SourceMemberService
	syncLogSvc *orgsvc.SyncLogService
	defaultSvc *orgsvc.DefaultSourceAccountService
}

func NewOrgSyncHandler(syncSvc *orgsvc.SyncService, unitSvc *orgsvc.SourceUnitService, memberSvc *orgsvc.SourceMemberService, syncLogSvc *orgsvc.SyncLogService, defaultSvc *orgsvc.DefaultSourceAccountService) *OrgSyncHandler {
	return &OrgSyncHandler{
		syncSvc:    syncSvc,
		unitSvc:    unitSvc,
		memberSvc:  memberSvc,
		syncLogSvc: syncLogSvc,
		defaultSvc: defaultSvc,
	}
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
	channelAccountUUID := strings.TrimSpace(c.Query("channel_account_uuid"))
	if sourceAccountUUID == "" && channelAccountUUID == "" {
		contracts.ResponseBadRequest(c, "source_account_uuid or channel_account_uuid is required")
		return
	}
	if sourceAccountUUID != "" {
		channelAccountUUID = ""
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
	items, err := h.unitSvc.List(c.Request.Context(), tenantUUID, sourceAccountUUID, channelAccountUUID, statusPtr)
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
	channelAccountUUID := strings.TrimSpace(c.Query("channel_account_uuid"))
	sourceUnitUUID := strings.TrimSpace(c.Query("source_unit_uuid"))
	sourceUnitUUIDsParam := strings.TrimSpace(c.Query("source_unit_uuids"))
	if sourceAccountUUID == "" && channelAccountUUID == "" {
		contracts.ResponseBadRequest(c, "source_account_uuid or channel_account_uuid is required")
		return
	}
	if (sourceUnitUUID != "" || sourceUnitUUIDsParam != "") && sourceAccountUUID == "" {
		contracts.ResponseBadRequest(c, "source_account_uuid is required when filtering by source_unit_uuid")
		return
	}
	if sourceAccountUUID != "" {
		channelAccountUUID = ""
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
	var unitPtr *string
	if sourceUnitUUID != "" {
		unitPtr = &sourceUnitUUID
	}
	sourceUnitUUIDs := []string{}
	if sourceUnitUUIDsParam != "" {
		for _, part := range strings.Split(sourceUnitUUIDsParam, ",") {
			clean := strings.TrimSpace(part)
			if clean == "" {
				continue
			}
			sourceUnitUUIDs = append(sourceUnitUUIDs, clean)
		}
	}
	items, err := h.memberSvc.List(c.Request.Context(), tenantUUID, sourceAccountUUID, channelAccountUUID, statusPtr, qPtr, unitPtr, sourceUnitUUIDs)
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

func (h *OrgSyncHandler) ListSyncLogs(c *gin.Context) {
	if h.syncLogSvc == nil {
		contracts.ResponseServiceUnavailable(c, "org sync log service unavailable", nil)
		return
	}
	sourceAccountUUID := strings.TrimSpace(c.Query("source_account_uuid"))
	channelAccountUUID := strings.TrimSpace(c.Query("channel_account_uuid"))
	if sourceAccountUUID == "" && channelAccountUUID == "" {
		contracts.ResponseBadRequest(c, "source_account_uuid or channel_account_uuid is required")
		return
	}
	if sourceAccountUUID != "" {
		channelAccountUUID = ""
	}
	limit, err := strconv.Atoi(strings.TrimSpace(c.Query("limit")))
	if err != nil || limit <= 0 {
		limit = 10
	}
	tenantUUID, ok := middleware.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	items, syncMode, syncHint, err := h.syncLogSvc.ListByAccount(c.Request.Context(), tenantUUID, sourceAccountUUID, channelAccountUUID, limit)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrTenantUuidRequired):
			contracts.ResponseBadRequest(c, "tenant_uuid is required")
		default:
			contracts.ResponseInternalError(c, err)
		}
		return
	}
	contracts.ResponseSuccess(c, gin.H{
		"items":     items,
		"sync_mode": syncMode,
		"sync_hint": syncHint,
	})
}

func (h *OrgSyncHandler) SetDefaultSourceAccount(c *gin.Context) {
	if h.defaultSvc == nil {
		contracts.ResponseServiceUnavailable(c, "org sync service unavailable", nil)
		return
	}
	accountUUID := strings.TrimSpace(c.Param("account_uuid"))
	if accountUUID == "" {
		contracts.ResponseBadRequest(c, "account_uuid is required")
		return
	}
	tenantUUID, ok := middleware.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	account, err := h.defaultSvc.SetDefault(c.Request.Context(), tenantUUID, accountUUID)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrTenantUuidRequired):
			contracts.ResponseBadRequest(c, "tenant_uuid is required")
		case errors.Is(err, socialrepo.ErrAccountNotFound):
			contracts.ResponseNotFound(c, "account not found")
		default:
			contracts.ResponseInternalError(c, err)
		}
		return
	}
	contracts.ResponseSuccess(c, account)
}
