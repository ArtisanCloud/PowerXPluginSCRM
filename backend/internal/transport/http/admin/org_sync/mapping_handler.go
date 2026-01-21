package org_sync

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/contracts"
	authmw "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/middleware"
	repository "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository"
	orgrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/org_sync"
	orgsvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/org_sync"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/middleware"
	"github.com/gin-gonic/gin"
)

type MappingHandler struct {
	matchSvc   *orgsvc.MatchService
	mappingSvc *orgsvc.MappingService
}

func NewMappingHandler(matchSvc *orgsvc.MatchService, mappingSvc *orgsvc.MappingService) *MappingHandler {
	return &MappingHandler{matchSvc: matchSvc, mappingSvc: mappingSvc}
}

type MappingConfirmRequest struct {
	UnitMappings   []MappingUnitInput   `json:"unit_mappings"`
	MemberMappings []MappingMemberInput `json:"member_mappings"`
}

type MappingUnitInput struct {
	SourceUnitID string `json:"source_unit_id"`
	MainUnitID   string `json:"main_unit_id"`
}

type MappingMemberInput struct {
	SourceMemberID string `json:"source_member_id"`
	MainMemberID   string `json:"main_member_id"`
}

func (h *MappingHandler) Suggestions(c *gin.Context) {
	if h == nil || h.matchSvc == nil {
		contracts.ResponseServiceUnavailable(c, "org sync service unavailable", nil)
		return
	}
	sourceAccountUUID := strings.TrimSpace(c.Query("source_account_uuid"))
	if sourceAccountUUID == "" {
		contracts.ResponseBadRequest(c, "source_account_uuid is required")
		return
	}
	tenantUUID, ok := middleware.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	suggestions, err := h.matchSvc.SuggestMappings(c.Request.Context(), tenantUUID, sourceAccountUUID)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrTenantUuidRequired):
			contracts.ResponseBadRequest(c, "tenant_uuid is required")
		default:
			contracts.ResponseInternalError(c, err)
		}
		return
	}
	contracts.ResponseSuccess(c, suggestions)
}

func (h *MappingHandler) Confirm(c *gin.Context) {
	if h == nil || h.mappingSvc == nil {
		contracts.ResponseServiceUnavailable(c, "org sync service unavailable", nil)
		return
	}
	tc, ok := authmw.GetTenantContext(c)
	if !ok || strings.TrimSpace(tc.TenantUUID) == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	if !isOrgAdmin(tc) {
		contracts.ResponseError(c, http.StatusForbidden, contracts.ErrCodeForbidden, "permission denied")
		return
	}
	var req MappingConfirmRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid body: "+err.Error())
		return
	}
	if len(req.UnitMappings) == 0 && len(req.MemberMappings) == 0 {
		contracts.ResponseBadRequest(c, "no mappings provided")
		return
	}
	confirmedBy := strconv.FormatInt(tc.UserID, 10)
	result, err := h.mappingSvc.ConfirmMappings(c.Request.Context(), tc.TenantUUID, orgsvc.ConfirmMappingsRequest{
		UnitMappings:   toUnitInputs(req.UnitMappings),
		MemberMappings: toMemberInputs(req.MemberMappings),
		ConfirmedBy:    confirmedBy,
	})
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrTenantUuidRequired):
			contracts.ResponseBadRequest(c, "tenant_uuid is required")
		case errors.Is(err, orgrepo.ErrSourceUnitNotFound):
			contracts.ResponseNotFound(c, "source unit not found")
		case errors.Is(err, orgrepo.ErrSourceMemberNotFound):
			contracts.ResponseNotFound(c, "source member not found")
		default:
			contracts.ResponseInternalError(c, err)
		}
		return
	}
	contracts.ResponseSuccess(c, result)
}

func toUnitInputs(items []MappingUnitInput) []orgsvc.UnitMappingInput {
	out := make([]orgsvc.UnitMappingInput, 0, len(items))
	for _, item := range items {
		out = append(out, orgsvc.UnitMappingInput{
			SourceUnitUUID: strings.TrimSpace(item.SourceUnitID),
			MainUnitID:     strings.TrimSpace(item.MainUnitID),
		})
	}
	return out
}

func toMemberInputs(items []MappingMemberInput) []orgsvc.MemberMappingInput {
	out := make([]orgsvc.MemberMappingInput, 0, len(items))
	for _, item := range items {
		out = append(out, orgsvc.MemberMappingInput{
			SourceMemberUUID: strings.TrimSpace(item.SourceMemberID),
			MainMemberID:     strings.TrimSpace(item.MainMemberID),
		})
	}
	return out
}

func isOrgAdmin(tc authmw.TenantContext) bool {
	for _, role := range tc.Roles {
		switch strings.ToLower(strings.TrimSpace(role)) {
		case "superadmin", "system.admin", "org.admin":
			return true
		}
	}
	for _, perm := range tc.Permissions {
		candidate := strings.ToLower(strings.TrimSpace(perm))
		if candidate == "*" || candidate == "*:*" || candidate == "org:admin" {
			return true
		}
	}
	return false
}
