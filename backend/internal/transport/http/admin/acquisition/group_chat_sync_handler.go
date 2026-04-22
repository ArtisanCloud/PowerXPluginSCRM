package acquisition

import (
	"errors"
	"strconv"
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/contracts"
	acqrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/repository/acquisition"
	dto "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/dto/acquisition"
	acqsvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/acquisition"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/middleware"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgconn"
)

type GroupChatSyncHandler struct {
	svc *acqsvc.GroupChatSyncService
}

func NewGroupChatSyncHandler(svc *acqsvc.GroupChatSyncService) *GroupChatSyncHandler {
	return &GroupChatSyncHandler{svc: svc}
}

func (h *GroupChatSyncHandler) Sync(c *gin.Context) {
	if h == nil || h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "group chat sync service unavailable", nil)
		return
	}
	tenantUUID, ok := middleware.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	var req dto.GroupChatSyncRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid body: "+err.Error())
		return
	}
	task, err := h.svc.TriggerSyncAsync(c.Request.Context(), acqsvc.GroupChatSyncRequest{
		TenantUUID: tenantUUID,
		Mode:       req.Mode,
	})
	if err != nil {
		if errors.Is(err, acqsvc.ErrDefaultChannelAccountNotFound) {
			contracts.ResponseBadRequest(c, "未找到可用默认渠道账号，请先在渠道治理中配置")
			return
		}
		if errors.Is(err, acqsvc.ErrUnsupportedChannelAccountType) {
			contracts.ResponseBadRequest(c, "当前渠道账号类型暂不支持群聊同步，请切换为企业微信账号")
			return
		}
		if isInvalidUUIDSyntaxErr(err) {
			contracts.ResponseBadRequest(c, "默认渠道账号配置异常，请在渠道治理重新设置默认账号")
			return
		}
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, gin.H{
		"job_uuid":              task.JobUUID,
		"task_uuid":             task.TaskUUID,
		"job_status":            task.Status,
		"mode":                  task.Mode,
		"channel_account_uuid":  task.ChannelAccountUUID,
		"resolved_channel_code": task.ResolvedChannelCode,
		"resolved_app_type":     task.ResolvedAppType,
	})
}

func (h *GroupChatSyncHandler) ListSyncTasks(c *gin.Context) {
	if h == nil || h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "group chat sync service unavailable", nil)
		return
	}
	tenantUUID, ok := middleware.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	limit, _ := strconv.Atoi(strings.TrimSpace(c.Query("limit")))
	status := strings.TrimSpace(c.Query("status"))
	items, err := h.svc.ListTasks(c.Request.Context(), tenantUUID, status, limit)
	if err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, gin.H{"items": items})
}

func (h *GroupChatSyncHandler) ClearSyncTasks(c *gin.Context) {
	if h == nil || h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "group chat sync service unavailable", nil)
		return
	}
	tenantUUID, ok := middleware.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	var req struct {
		IncludeInFlight bool `json:"include_in_flight"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid body: "+err.Error())
		return
	}
	deleted, err := h.svc.ClearTasks(c.Request.Context(), tenantUUID, req.IncludeInFlight)
	if err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, gin.H{"deleted": deleted})
}

func (h *GroupChatSyncHandler) List(c *gin.Context) {
	if h == nil || h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "group chat sync service unavailable", nil)
		return
	}
	tenantUUID, ok := middleware.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	limit, _ := strconv.Atoi(strings.TrimSpace(c.Query("limit")))
	items, err := h.svc.List(c.Request.Context(), tenantUUID, limit)
	if err != nil {
		if isUndefinedRelationErr(err) {
			contracts.ResponseSuccess(c, gin.H{"items": []any{}})
			return
		}
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, gin.H{"items": items})
}

func (h *GroupChatSyncHandler) Get(c *gin.Context) {
	if h == nil || h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "group chat sync service unavailable", nil)
		return
	}
	tenantUUID, ok := middleware.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	item, err := h.svc.Get(c.Request.Context(), tenantUUID, c.Param("chat_id"))
	if err != nil {
		if err == acqrepo.ErrRecordNotFound {
			contracts.ResponseNotFound(c, "group chat not found")
			return
		}
		if isUndefinedRelationErr(err) {
			contracts.ResponseNotFound(c, "group chat snapshots not initialized")
			return
		}
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, item)
}

func (h *GroupChatSyncHandler) GetCustomerTimeline(c *gin.Context) {
	if h == nil || h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "group chat sync service unavailable", nil)
		return
	}
	tenantUUID, ok := middleware.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	chatID := strings.TrimSpace(c.Param("chat_id"))
	externalUserID := strings.TrimSpace(c.Param("external_userid"))
	limit, _ := strconv.Atoi(strings.TrimSpace(c.Query("limit")))
	if chatID == "" || externalUserID == "" {
		contracts.ResponseBadRequest(c, "chat_id and external_userid are required")
		return
	}
	items, err := h.svc.GetCustomerTimeline(c.Request.Context(), tenantUUID, chatID, externalUserID, limit)
	if err != nil {
		if err == acqrepo.ErrRecordNotFound {
			contracts.ResponseNotFound(c, "group chat not found")
			return
		}
		if isUndefinedRelationErr(err) {
			contracts.ResponseNotFound(c, "group chat snapshots not initialized")
			return
		}
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, gin.H{"items": items})
}

func (h *GroupChatSyncHandler) GetCustomerFollowups(c *gin.Context) {
	if h == nil || h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "group chat sync service unavailable", nil)
		return
	}
	tenantUUID, ok := middleware.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	chatID := strings.TrimSpace(c.Param("chat_id"))
	externalUserID := strings.TrimSpace(c.Param("external_userid"))
	limit, _ := strconv.Atoi(strings.TrimSpace(c.Query("limit")))
	if chatID == "" || externalUserID == "" {
		contracts.ResponseBadRequest(c, "chat_id and external_userid are required")
		return
	}
	out, err := h.svc.GetCustomerFollowups(c.Request.Context(), tenantUUID, chatID, externalUserID, limit)
	if err != nil {
		if err == acqrepo.ErrRecordNotFound {
			contracts.ResponseNotFound(c, "group chat not found")
			return
		}
		if isUndefinedRelationErr(err) {
			contracts.ResponseNotFound(c, "group chat snapshots not initialized")
			return
		}
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, out)
}

func (h *GroupChatSyncHandler) GetCustomerRelatedChats(c *gin.Context) {
	if h == nil || h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "group chat sync service unavailable", nil)
		return
	}
	tenantUUID, ok := middleware.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	externalUserID := strings.TrimSpace(c.Param("external_userid"))
	limit, _ := strconv.Atoi(strings.TrimSpace(c.Query("limit")))
	if externalUserID == "" {
		contracts.ResponseBadRequest(c, "external_userid is required")
		return
	}
	items, err := h.svc.GetCustomerRelatedChats(c.Request.Context(), tenantUUID, externalUserID, limit)
	if err != nil {
		if isUndefinedRelationErr(err) {
			contracts.ResponseSuccess(c, gin.H{"items": []any{}})
			return
		}
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, gin.H{"items": items})
}

func isUndefinedRelationErr(err error) bool {
	if err == nil {
		return false
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "42P01" {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "sqlstate 42p01") ||
		(strings.Contains(msg, "relation") && strings.Contains(msg, "does not exist"))
}

func isInvalidUUIDSyntaxErr(err error) bool {
	if err == nil {
		return false
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "22P02" {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "sqlstate 22p02") || strings.Contains(msg, "invalid input syntax for type uuid")
}
