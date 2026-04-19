package social_channel_governance

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/contracts"
	repository "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository"
	orgsvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/org_sync"
	socialsvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/social_channel_governance"
	httpmw "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/middleware"
	"github.com/gin-gonic/gin"
)

type StaffTagHandler struct {
	svc           *socialsvc.StaffTagService
	sourceMemberS *orgsvc.SourceMemberService
}

func NewStaffTagHandler(svc *socialsvc.StaffTagService, sourceMemberSvc *orgsvc.SourceMemberService) *StaffTagHandler {
	return &StaffTagHandler{svc: svc, sourceMemberS: sourceMemberSvc}
}

func (h *StaffTagHandler) ListTags(c *gin.Context) {
	if h == nil || h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "staff tag service unavailable", nil)
		return
	}
	tenantUUID, ok := httpmw.TenantUUIDFromContext(c)
	if !ok || strings.TrimSpace(tenantUUID) == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	channelAccountUUID := strings.TrimSpace(c.Query("channel_account_uuid"))
	if channelAccountUUID == "" {
		contracts.ResponseBadRequest(c, "channel_account_uuid is required")
		return
	}
	includeWritable := true
	if raw := strings.TrimSpace(c.Query("include_writable")); raw != "" {
		parsed, parseErr := strconv.ParseBool(raw)
		if parseErr == nil {
			includeWritable = parsed
		}
	}
	items, err := h.svc.ListByChannel(c.Request.Context(), tenantUUID, channelAccountUUID, includeWritable)
	if err != nil {
		contracts.ResponseError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}
	contracts.ResponseSuccess(c, gin.H{"items": items})
}

func (h *StaffTagHandler) GetTagMembers(c *gin.Context) {
	if h == nil || h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "staff tag service unavailable", nil)
		return
	}
	tenantUUID, ok := httpmw.TenantUUIDFromContext(c)
	if !ok || strings.TrimSpace(tenantUUID) == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	channelAccountUUID := strings.TrimSpace(c.Query("channel_account_uuid"))
	if channelAccountUUID == "" {
		contracts.ResponseBadRequest(c, "channel_account_uuid is required")
		return
	}
	tagID, err := strconv.ParseInt(strings.TrimSpace(c.Param("tag_id")), 10, 64)
	if err != nil || tagID <= 0 {
		contracts.ResponseBadRequest(c, "invalid tag_id")
		return
	}
	detail, detailErr := h.svc.GetDetailByChannel(c.Request.Context(), tenantUUID, channelAccountUUID, tagID)
	if detailErr != nil {
		contracts.ResponseError(c, http.StatusBadRequest, "INVALID_REQUEST", detailErr.Error())
		return
	}
	contracts.ResponseSuccess(c, detail)
}

func (h *StaffTagHandler) CreateTag(c *gin.Context) {
	if h == nil || h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "staff tag service unavailable", nil)
		return
	}
	tenantUUID, ok := httpmw.TenantUUIDFromContext(c)
	if !ok || strings.TrimSpace(tenantUUID) == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	var req struct {
		ChannelAccountUUID string `json:"channel_account_uuid"`
		TagName            string `json:"tag_name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid body: "+err.Error())
		return
	}
	item, err := h.svc.CreateByChannel(c.Request.Context(), tenantUUID, req.ChannelAccountUUID, req.TagName)
	if err != nil {
		contracts.ResponseError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}
	contracts.ResponseSuccess(c, item)
}

func (h *StaffTagHandler) UpdateTag(c *gin.Context) {
	if h == nil || h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "staff tag service unavailable", nil)
		return
	}
	tenantUUID, ok := httpmw.TenantUUIDFromContext(c)
	if !ok || strings.TrimSpace(tenantUUID) == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	tagID, err := strconv.ParseInt(strings.TrimSpace(c.Param("tag_id")), 10, 64)
	if err != nil || tagID <= 0 {
		contracts.ResponseBadRequest(c, "invalid tag_id")
		return
	}
	var req struct {
		ChannelAccountUUID string `json:"channel_account_uuid"`
		TagName            string `json:"tag_name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid body: "+err.Error())
		return
	}
	if err := h.svc.UpdateByChannel(c.Request.Context(), tenantUUID, req.ChannelAccountUUID, tagID, req.TagName); err != nil {
		contracts.ResponseError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}
	contracts.ResponseSuccess(c, gin.H{"tag_id": tagID, "tag_name": strings.TrimSpace(req.TagName)})
}

func (h *StaffTagHandler) DeleteTag(c *gin.Context) {
	if h == nil || h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "staff tag service unavailable", nil)
		return
	}
	tenantUUID, ok := httpmw.TenantUUIDFromContext(c)
	if !ok || strings.TrimSpace(tenantUUID) == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	tagID, err := strconv.ParseInt(strings.TrimSpace(c.Param("tag_id")), 10, 64)
	if err != nil || tagID <= 0 {
		contracts.ResponseBadRequest(c, "invalid tag_id")
		return
	}
	channelAccountUUID := strings.TrimSpace(c.Query("channel_account_uuid"))
	if channelAccountUUID == "" {
		contracts.ResponseBadRequest(c, "channel_account_uuid is required")
		return
	}
	if err := h.svc.DeleteByChannel(c.Request.Context(), tenantUUID, channelAccountUUID, tagID); err != nil {
		contracts.ResponseError(c, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
		return
	}
	contracts.ResponseSuccess(c, gin.H{"tag_id": tagID, "deleted": true})
}

func (h *StaffTagHandler) PatchTagMembers(c *gin.Context) {
	if h == nil || h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "staff tag service unavailable", nil)
		return
	}
	tenantUUID, ok := httpmw.TenantUUIDFromContext(c)
	if !ok || strings.TrimSpace(tenantUUID) == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	tagID, err := strconv.ParseInt(strings.TrimSpace(c.Param("tag_id")), 10, 64)
	if err != nil || tagID <= 0 {
		contracts.ResponseBadRequest(c, "invalid tag_id")
		return
	}
	var req struct {
		ChannelAccountUUID string   `json:"channel_account_uuid"`
		AddUserIDs         []string `json:"add_user_ids"`
		RemoveUserIDs      []string `json:"remove_user_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid body: "+err.Error())
		return
	}
	result, patchErr := h.svc.PatchTagUsersByChannel(c.Request.Context(), tenantUUID, req.ChannelAccountUUID, tagID, req.AddUserIDs, req.RemoveUserIDs)
	if patchErr != nil {
		contracts.ResponseError(c, http.StatusBadRequest, "INVALID_REQUEST", patchErr.Error())
		return
	}
	contracts.ResponseSuccess(c, result)
}

func (h *StaffTagHandler) ListSourceMembers(c *gin.Context) {
	if h == nil || h.sourceMemberS == nil {
		contracts.ResponseServiceUnavailable(c, "source member service unavailable", nil)
		return
	}
	tenantUUID, ok := httpmw.TenantUUIDFromContext(c)
	if !ok || strings.TrimSpace(tenantUUID) == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	channelAccountUUID := strings.TrimSpace(c.Query("channel_account_uuid"))
	if channelAccountUUID == "" {
		contracts.ResponseBadRequest(c, "channel_account_uuid is required")
		return
	}
	q := strings.TrimSpace(c.Query("q"))
	var qPtr *string
	if q != "" {
		qPtr = &q
	}
	items, err := h.sourceMemberS.List(c.Request.Context(), tenantUUID, "", channelAccountUUID, nil, qPtr, nil, nil)
	if err != nil {
		switch {
		case err == repository.ErrTenantUuidRequired:
			contracts.ResponseBadRequest(c, "tenant_uuid is required")
		default:
			contracts.ResponseInternalError(c, err)
		}
		return
	}
	contracts.ResponseSuccess(c, gin.H{"items": items})
}
