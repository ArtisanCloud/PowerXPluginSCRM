package social_channel_governance

import (
	"errors"
	"net/http"
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/contracts"
	dto "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/dto/social_channel_governance"
	svc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/social_channel_governance"
	"github.com/gin-gonic/gin"
)

type ChannelPlatformSettingHandler struct {
	svc *svc.ChannelPlatformSettingService
}

func NewChannelPlatformSettingHandler(svc *svc.ChannelPlatformSettingService) *ChannelPlatformSettingHandler {
	return &ChannelPlatformSettingHandler{svc: svc}
}

func (h *ChannelPlatformSettingHandler) GetWeComOpenWork(c *gin.Context) {
	if h == nil || h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "channel platform setting service unavailable", nil)
		return
	}
	data, err := h.svc.GetWeComOpenWorkConfig(c.Request.Context())
	if err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, h.configToMap(data))
}

func (h *ChannelPlatformSettingHandler) SaveWeComOpenWork(c *gin.Context) {
	if h == nil || h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "channel platform setting service unavailable", nil)
		return
	}
	var req dto.WeComOpenWorkPlatformConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid body: "+err.Error())
		return
	}
	templates := make([]svc.WeComOpenWorkTemplate, 0, len(req.Templates))
	for _, item := range req.Templates {
		templates = append(templates, svc.WeComOpenWorkTemplate{
			TemplateID:              item.TemplateID,
			TemplateSecret:          item.TemplateSecret,
			TemplateTicket:          item.TemplateTicket,
			TemplateTicketUpdatedAt: item.TemplateTicketUpdatedAt,
			TemplateTicketSource:    item.TemplateTicketSource,
			ProviderCorpID:          item.ProviderCorpID,
			ProviderSecret:          item.ProviderSecret,
		})
	}
	data, err := h.svc.SaveWeComOpenWorkConfig(c.Request.Context(), svc.WeComOpenWorkPlatformConfig{
		Enabled:                 req.Enabled,
		TemplateID:              req.TemplateID,
		TemplateSecret:          req.TemplateSecret,
		TemplateTicket:          req.TemplateTicket,
		TemplateTicketUpdatedAt: req.TemplateTicketUpdatedAt,
		TemplateTicketSource:    req.TemplateTicketSource,
		ProviderCorpID:          req.ProviderCorpID,
		ProviderSecret:          req.ProviderSecret,
		Token:                   req.Token,
		AESKey:                  req.AESKey,
		HTTPDebug:               req.HTTPDebug,
		CallbackHost:            req.CallbackHost,
		RedirectURI:             req.RedirectURI,
		DefaultTemplateID:       req.DefaultTemplateID,
		Templates:               templates,
	})
	if err != nil {
		contracts.ResponseBadRequest(c, err.Error())
		return
	}
	contracts.ResponseSuccess(c, h.configToMap(data))
}

func (h *ChannelPlatformSettingHandler) ListWeComOpenWorkTemplates(c *gin.Context) {
	if h == nil || h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "channel platform setting service unavailable", nil)
		return
	}
	items, err := h.svc.ListWeComOpenWorkTemplates(c.Request.Context())
	if err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	list := make([]gin.H, 0, len(items))
	for _, item := range items {
		list = append(list, h.templateToMap(item))
	}
	contracts.ResponseSuccess(c, gin.H{"items": list})
}

func (h *ChannelPlatformSettingHandler) CreateWeComOpenWorkTemplate(c *gin.Context) {
	if h == nil || h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "channel platform setting service unavailable", nil)
		return
	}
	var req dto.WeComOpenWorkTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid body: "+err.Error())
		return
	}
	item, err := h.svc.CreateWeComOpenWorkTemplate(c.Request.Context(), svc.WeComOpenWorkTemplate{
		TemplateID:              req.TemplateID,
		TemplateSecret:          req.TemplateSecret,
		TemplateTicket:          req.TemplateTicket,
		TemplateTicketUpdatedAt: req.TemplateTicketUpdatedAt,
		TemplateTicketSource:    req.TemplateTicketSource,
		ProviderCorpID:          req.ProviderCorpID,
		ProviderSecret:          req.ProviderSecret,
	})
	if err != nil {
		h.handleTemplateError(c, err)
		return
	}
	contracts.ResponseSuccess(c, h.templateToMap(*item))
}

func (h *ChannelPlatformSettingHandler) UpdateWeComOpenWorkTemplate(c *gin.Context) {
	if h == nil || h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "channel platform setting service unavailable", nil)
		return
	}
	templateID := strings.TrimSpace(c.Param("template_id"))
	if templateID == "" {
		contracts.ResponseBadRequest(c, "template_id is required")
		return
	}
	var req dto.WeComOpenWorkTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid body: "+err.Error())
		return
	}
	item, err := h.svc.UpdateWeComOpenWorkTemplate(c.Request.Context(), templateID, svc.WeComOpenWorkTemplate{
		TemplateID:              req.TemplateID,
		TemplateSecret:          req.TemplateSecret,
		TemplateTicket:          req.TemplateTicket,
		TemplateTicketUpdatedAt: req.TemplateTicketUpdatedAt,
		TemplateTicketSource:    req.TemplateTicketSource,
		ProviderCorpID:          req.ProviderCorpID,
		ProviderSecret:          req.ProviderSecret,
	})
	if err != nil {
		h.handleTemplateError(c, err)
		return
	}
	contracts.ResponseSuccess(c, h.templateToMap(*item))
}

func (h *ChannelPlatformSettingHandler) DeleteWeComOpenWorkTemplate(c *gin.Context) {
	if h == nil || h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "channel platform setting service unavailable", nil)
		return
	}
	templateID := strings.TrimSpace(c.Param("template_id"))
	if templateID == "" {
		contracts.ResponseBadRequest(c, "template_id is required")
		return
	}
	if err := h.svc.DeleteWeComOpenWorkTemplate(c.Request.Context(), templateID); err != nil {
		h.handleTemplateError(c, err)
		return
	}
	contracts.ResponseSuccess(c, gin.H{"deleted": true, "template_id": templateID})
}

func (h *ChannelPlatformSettingHandler) SetDefaultWeComOpenWorkTemplate(c *gin.Context) {
	if h == nil || h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "channel platform setting service unavailable", nil)
		return
	}
	templateID := strings.TrimSpace(c.Param("template_id"))
	if templateID == "" {
		contracts.ResponseBadRequest(c, "template_id is required")
		return
	}
	item, err := h.svc.SetDefaultWeComOpenWorkTemplate(c.Request.Context(), templateID)
	if err != nil {
		h.handleTemplateError(c, err)
		return
	}
	contracts.ResponseSuccess(c, h.templateToMap(*item))
}

func (h *ChannelPlatformSettingHandler) RefreshWeComSuiteTicket(c *gin.Context) {
	if h == nil || h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "channel platform setting service unavailable", nil)
		return
	}
	status, err := h.svc.RefreshWeComSuiteTicketStatus(c.Request.Context())
	if err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, gin.H{
		"ready":                  status.Ready,
		"template_id":            status.TemplateID,
		"template_ticket":        status.TemplateTicket,
		"template_ticket_masked": status.TemplateTicketMasked,
		"template_ticket_source": status.TemplateTicketSource,
		"updated_at":             status.UpdatedAt,
		"checked_at":             status.CheckedAt,
		"message":                status.Message,
	})
}

func (h *ChannelPlatformSettingHandler) VerifyWeComSuiteTicket(c *gin.Context) {
	if h == nil || h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "channel platform setting service unavailable", nil)
		return
	}
	result, err := h.svc.VerifyWeComSuiteTicket(c.Request.Context())
	if err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, gin.H{
		"valid":                  result.Valid,
		"template_id":            result.TemplateID,
		"checked_at":             result.CheckedAt,
		"errcode":                result.ErrCode,
		"errmsg":                 result.ErrMsg,
		"expires_in":             result.ExpiresIn,
		"has_suite_access_token": result.HasSuiteAccessToken,
		"message":                result.Message,
	})
}

func (h *ChannelPlatformSettingHandler) templateToMap(item svc.WeComOpenWorkTemplate) gin.H {
	return gin.H{
		"template_id":                item.TemplateID,
		"template_secret":            item.TemplateSecret,
		"template_ticket":            item.TemplateTicket,
		"template_ticket_updated_at": item.TemplateTicketUpdatedAt,
		"template_ticket_source":     item.TemplateTicketSource,
		"provider_corpid":            item.ProviderCorpID,
		"provider_secret":            item.ProviderSecret,
		"is_default":                 item.IsDefault,
	}
}

func (h *ChannelPlatformSettingHandler) configToMap(data *svc.WeComOpenWorkPlatformConfig) gin.H {
	if data == nil {
		return gin.H{}
	}
	templates := make([]gin.H, 0, len(data.Templates))
	for _, item := range data.Templates {
		templates = append(templates, h.templateToMap(item))
	}
	return gin.H{
		"enabled":                    data.Enabled,
		"template_id":                data.TemplateID,
		"template_secret":            data.TemplateSecret,
		"template_ticket":            data.TemplateTicket,
		"template_ticket_updated_at": data.TemplateTicketUpdatedAt,
		"template_ticket_source":     data.TemplateTicketSource,
		"provider_corpid":            data.ProviderCorpID,
		"provider_secret":            data.ProviderSecret,
		"token":                      data.Token,
		"aes_key":                    data.AESKey,
		"http_debug":                 data.HTTPDebug,
		"callback_host":              data.CallbackHost,
		"redirect_uri":               data.RedirectURI,
		"default_template_id":        data.DefaultTemplateID,
		"templates":                  templates,
	}
}

func (h *ChannelPlatformSettingHandler) handleTemplateError(c *gin.Context, err error) {
	if err == nil {
		return
	}
	switch {
	case errors.Is(err, svc.ErrWeComTemplateNotFound):
		contracts.ResponseNotFound(c, err.Error())
	case errors.Is(err, svc.ErrWeComTemplateAlreadyExists):
		contracts.ResponseError(c, http.StatusConflict, contracts.ErrCodeInvalidRequest, err.Error())
	default:
		contracts.ResponseBadRequest(c, err.Error())
	}
}
