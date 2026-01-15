package miniapp

import (
	"errors"
	"strconv"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/contracts"
	miniappsvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/mini-app"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type TemplateListRequest struct {
	Page     int    `form:"page,default=1"`
	PageSize int    `form:"page_size,default=20"`
	Q        string `form:"q"`
}

type MiniAppTemplateHandler struct {
	service *miniappsvc.TemplateService
}

func NewMiniAppTemplateHandler(deps *app.Deps) *MiniAppTemplateHandler {
	if deps == nil || deps.DB == nil {
		return &MiniAppTemplateHandler{service: nil}
	}
	return &MiniAppTemplateHandler{service: miniappsvc.NewTemplateService(deps.DB)}
}

func (h *MiniAppTemplateHandler) ListPublished(c *gin.Context) {
	if h == nil || h.service == nil {
		contracts.ResponseServiceUnavailable(c, "template service not available", nil)
		return
	}
	var q TemplateListRequest
	if err := c.ShouldBindQuery(&q); err != nil {
		contracts.ResponseBadRequest(c, "invalid query: "+err.Error())
		return
	}
	res, err := h.service.ListPublished(c.Request.Context(), q.Q, q.Page, q.PageSize)
	if err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, res)
}

func (h *MiniAppTemplateHandler) GetPublished(c *gin.Context) {
	if h == nil || h.service == nil {
		contracts.ResponseServiceUnavailable(c, "template service not available", nil)
		return
	}
	id, err := parseUint64(c.Param("id"))
	if err != nil {
		contracts.ResponseBadRequest(c, "invalid id")
		return
	}
	tpl, err := h.service.GetPublishedByID(c.Request.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			contracts.ResponseNotFound(c, "not found")
		default:
			contracts.ResponseInternalError(c, err)
		}
		return
	}
	contracts.ResponseSuccess(c, tpl)
}

func parseUint64(s string) (uint64, error) {
	u, err := strconv.ParseUint(s, 10, 64)
	return uint64(u), err
}
