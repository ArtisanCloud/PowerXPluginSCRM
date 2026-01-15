package templates

import (
	"errors"
	"io"
	"strconv"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/contracts"
	srvtemplates "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/templates"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type TemplateHandler struct{ TemplateService *srvtemplates.TemplateService }

func NewTemplateHandler(deps *app.Deps) *TemplateHandler {
	return &TemplateHandler{TemplateService: srvtemplates.NewTemplateService(deps.DB)}
}

func (h *TemplateHandler) GetTemplates(c *gin.Context) {
	// capability: com.powerx.plugins.base.template.list
	var q TemplateListRequest
	if err := c.ShouldBindQuery(&q); err != nil {
		contracts.ResponseBadRequest(c, "invalid query: "+err.Error())
		return
	}

	res, err := h.TemplateService.List(c.Request.Context(), q.Q, q.Page, q.PageSize)
	if err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, res)
}

func (h *TemplateHandler) GetTemplate(c *gin.Context) {
	// capability: com.powerx.plugins.base.template.read
	id, err := parseUint64(c.Param("id"))
	if err != nil {
		contracts.ResponseBadRequest(c, "invalid id")
		return
	}
	tpl, err := h.TemplateService.GetByID(c.Request.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			contracts.ResponseNotFound(c, "not found: "+err.Error())
		default:
			contracts.ResponseInternalError(c, err)
		}
		return
	}
	contracts.ResponseSuccess(c, tpl)
}

func (h *TemplateHandler) CreateTemplate(c *gin.Context) {
	// capability: com.powerx.plugins.base.template.create
	var req CreateTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid body: "+err.Error())
		return
	}
	tpl, err := h.TemplateService.Create(c.Request.Context(), req.Name, req.Description, req.Content)
	if err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, tpl)
}

func (h *TemplateHandler) UpdateTemplate(c *gin.Context) {
	// capability: com.powerx.plugins.base.template.update
	id, err := parseUint64(c.Param("id"))
	if err != nil {
		contracts.ResponseBadRequest(c, "invalid id")
		return
	}
	var req UpdateTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid body: "+err.Error())
		return
	}
	tpl, err := h.TemplateService.Update(c.Request.Context(), id, req.Name, req.Description, req.Content)
	if err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, tpl)
}

func (h *TemplateHandler) DeleteTemplate(c *gin.Context) {
	// capability: com.powerx.plugins.base.template.delete
	id, err := parseUint64(c.Param("id"))
	if err != nil {
		contracts.ResponseBadRequest(c, "invalid id")
		return
	}
	if err := h.TemplateService.Delete(c.Request.Context(), id); err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, gin.H{"ok": true})
}

func (h *TemplateHandler) BatchCloneTemplates(c *gin.Context) {
	// capability: com.powerx.plugins.base.template.batch_clone
	if h == nil || h.TemplateService == nil {
		contracts.ResponseServiceUnavailable(c, "template service not available", nil)
		return
	}
	var req BatchCloneRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid body: "+err.Error())
		return
	}
	if len(req.SourceIDs) == 0 {
		contracts.ResponseBadRequest(c, "source_ids is required")
		return
	}
	result, err := h.TemplateService.BatchClone(
		c.Request.Context(),
		req.SourceIDs,
		req.Copies,
		srvtemplates.BatchCloneOptions{
			NamePrefix:        req.NamePrefix,
			DescriptionPrefix: req.DescriptionPrefix,
		},
	)
	if err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, result)
}

func (h *TemplateHandler) ValidateTemplateCapability(c *gin.Context) {
	// capability: com.powerx.plugins.base.template.validate
	if h == nil || h.TemplateService == nil {
		contracts.ResponseServiceUnavailable(c, "template service not available", nil)
		return
	}
	id, err := parseUint64(c.Param("id"))
	if err != nil {
		contracts.ResponseBadRequest(c, "invalid id")
		return
	}
	var req ValidateTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		contracts.ResponseBadRequest(c, "invalid body: "+err.Error())
		return
	}
	res, err := h.TemplateService.Validate(c.Request.Context(), id, req.Rules, req.Strict)
	if err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, res)
}

func parseUint64(s string) (uint64, error) {
	u, err := strconv.ParseUint(s, 10, 64)
	return uint64(u), err
}
