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
)

type GroupLiveCodeHandler struct {
	svc *acqsvc.GroupLiveCodeService
}

func NewGroupLiveCodeHandler(svc *acqsvc.GroupLiveCodeService) *GroupLiveCodeHandler {
	return &GroupLiveCodeHandler{svc: svc}
}

func (h *GroupLiveCodeHandler) Create(c *gin.Context) {
	if h == nil || h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "group live code service unavailable", nil)
		return
	}
	tenantUUID, ok := middleware.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	var req dto.GroupLiveCodeCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid body: "+err.Error())
		return
	}
	item, err := h.svc.Create(c.Request.Context(), acqsvc.GroupLiveCodeCreateRequest{
		TenantUUID:         tenantUUID,
		Channel:            req.Channel,
		AppType:            req.AppType,
		ChannelAccountUUID: req.ChannelAccountUUID,
		ActivityName:       req.ActivityName,
		CorpTagIDs:         req.CorpTagIDs,
		RemarkEnabled:      req.NewCustomerRemarkEnabled,
		JoinScene:          req.JoinScene,
		SkipVerify:         req.SkipVerify,
		AutoCreateRoom:     req.AutoCreateRoom,
	})
	if err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, item)
}

func (h *GroupLiveCodeHandler) Get(c *gin.Context) {
	if h == nil || h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "group live code service unavailable", nil)
		return
	}
	tenantUUID, ok := middleware.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	item, err := h.svc.Get(c.Request.Context(), tenantUUID, c.Param("group_code_uuid"))
	if err != nil {
		if err == acqrepo.ErrRecordNotFound {
			contracts.ResponseNotFound(c, "group code not found")
			return
		}
		if err == acqsvc.ErrGroupLiveCodeNoTargetChats {
			contracts.ResponseBadRequest(c, "请先选择至少一个群聊后再启用")
			return
		}
		if err == acqsvc.ErrDefaultChannelAccountNotFound {
			contracts.ResponseBadRequest(c, "未找到可用默认渠道账号，请先在渠道治理中配置")
			return
		}
		if errors.Is(err, acqsvc.ErrUnsupportedChannelAccountType) {
			contracts.ResponseBadRequest(c, "当前渠道账号类型暂不支持群活码，请切换为企业微信账号")
			return
		}
		if msg, ok := humanizeGroupLiveCodeError(err); ok {
			contracts.ResponseBadRequest(c, msg)
			return
		}
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, item)
}

func (h *GroupLiveCodeHandler) Update(c *gin.Context) {
	if h == nil || h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "group live code service unavailable", nil)
		return
	}
	tenantUUID, ok := middleware.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	var req dto.GroupLiveCodeUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid body: "+err.Error())
		return
	}
	item, err := h.svc.Update(c.Request.Context(), acqsvc.GroupLiveCodeUpdateRequest{
		TenantUUID:     tenantUUID,
		GroupCodeUUID:  c.Param("group_code_uuid"),
		ActivityName:   req.ActivityName,
		CorpTagIDs:     req.CorpTagIDs,
		RemarkEnabled:  req.NewCustomerRemarkEnabled,
		SkipVerify:     req.SkipVerify,
		AutoCreateRoom: req.AutoCreateRoom,
		Status:         req.Status,
	})
	if err != nil {
		if err == acqrepo.ErrRecordNotFound {
			contracts.ResponseNotFound(c, "group code not found")
			return
		}
		if err == acqsvc.ErrGroupLiveCodeNoTargetChats {
			contracts.ResponseBadRequest(c, "请先选择至少一个群聊后再发布")
			return
		}
		if err == acqsvc.ErrDefaultChannelAccountNotFound {
			contracts.ResponseBadRequest(c, "未找到可用默认渠道账号，请先在渠道治理中配置")
			return
		}
		if errors.Is(err, acqsvc.ErrUnsupportedChannelAccountType) {
			contracts.ResponseBadRequest(c, "当前渠道账号类型暂不支持群活码，请切换为企业微信账号")
			return
		}
		if msg, ok := humanizeGroupLiveCodeError(err); ok {
			contracts.ResponseBadRequest(c, msg)
			return
		}
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, item)
}

func (h *GroupLiveCodeHandler) Delete(c *gin.Context) {
	if h == nil || h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "group live code service unavailable", nil)
		return
	}
	tenantUUID, ok := middleware.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	err := h.svc.Delete(c.Request.Context(), tenantUUID, c.Param("group_code_uuid"))
	if err != nil {
		if err == acqrepo.ErrRecordNotFound {
			contracts.ResponseNotFound(c, "group code not found")
			return
		}
		if err == acqsvc.ErrGroupLiveCodeNoTargetChats {
			contracts.ResponseBadRequest(c, "请先选择至少一个群聊后再同步")
			return
		}
		if err == acqsvc.ErrDefaultChannelAccountNotFound {
			contracts.ResponseBadRequest(c, "未找到可用默认渠道账号，请先在渠道治理中配置")
			return
		}
		if errors.Is(err, acqsvc.ErrUnsupportedChannelAccountType) {
			contracts.ResponseBadRequest(c, "当前渠道账号类型暂不支持群活码，请切换为企业微信账号")
			return
		}
		if msg, ok := humanizeGroupLiveCodeError(err); ok {
			contracts.ResponseBadRequest(c, msg)
			return
		}
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, gin.H{"deleted": true})
}

func (h *GroupLiveCodeHandler) Sync(c *gin.Context) {
	if h == nil || h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "group live code service unavailable", nil)
		return
	}
	tenantUUID, ok := middleware.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	var req dto.GroupLiveCodeSyncRequest
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			contracts.ResponseBadRequest(c, "invalid body: "+err.Error())
			return
		}
	}
	item, err := h.svc.Sync(c.Request.Context(), acqsvc.GroupLiveCodeSyncRequest{
		TenantUUID:    tenantUUID,
		GroupCodeUUID: c.Param("group_code_uuid"),
		ChatIDs:       req.ChatIDs,
	})
	if err != nil {
		if err == acqrepo.ErrRecordNotFound {
			contracts.ResponseNotFound(c, "group code not found")
			return
		}
		if err == acqsvc.ErrGroupLiveCodeNoTargetChats {
			contracts.ResponseBadRequest(c, "请先选择至少一个群聊后再发布")
			return
		}
		if err == acqsvc.ErrDefaultChannelAccountNotFound {
			contracts.ResponseBadRequest(c, "未找到可用默认渠道账号，请先在渠道治理中配置")
			return
		}
		if errors.Is(err, acqsvc.ErrUnsupportedChannelAccountType) {
			contracts.ResponseBadRequest(c, "当前渠道账号类型暂不支持群活码，请切换为企业微信账号")
			return
		}
		if msg, ok := humanizeGroupLiveCodeError(err); ok {
			contracts.ResponseBadRequest(c, msg)
			return
		}
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, item)
}

func (h *GroupLiveCodeHandler) List(c *gin.Context) {
	if h == nil || h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "group live code service unavailable", nil)
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
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, gin.H{"items": items})
}

func humanizeGroupLiveCodeError(err error) (string, bool) {
	if err == nil {
		return "", false
	}
	msg := strings.ToLower(strings.TrimSpace(err.Error()))
	switch {
	case strings.Contains(msg, "group live code skip_verify is not supported"):
		return "群活码当前不支持“免验证入群”（企微群活码接口无此参数），请关闭该开关", true
	case strings.Contains(msg, "invalid group live code corp_tag_ids"):
		return "标签不合法：请仅选择“企业客户标签”（tag_id 形如 etsdn...），不要使用员工标签或纯数字ID", true
	case strings.Contains(msg, "701170"):
		return "企业微信“群活码/进群方式”试用已到期（701170），请在企微后台续期或开通正式能力后再同步", true
	case strings.Contains(msg, "81011"):
		return "当前账号缺少客户群活码权限，请在企业微信应用权限中开通“客户群/客户联系”后重试", true
	case strings.Contains(msg, "60011"):
		return "当前账号缺少客户群访问权限，请检查应用可见范围与客户联系权限", true
	case strings.Contains(msg, "self-built account requires corp_id and app_secret"):
		return "当前账号缺少自建应用凭证（corp_id/app_secret），请在渠道账号中补全后重试", true
	case strings.Contains(msg, "openwork account requires delegated_template credentials"):
		return "当前代开发账号凭证不完整，请检查模板ID、模板Secret、Provider凭证与永久授权码", true
	default:
		return "", false
	}
}
