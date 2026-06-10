package lead_capture

import (
	"errors"
	"net/http"
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/contracts"
	dto "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/dto/lead_capture"
	leadrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/lead_capture"
	leadsvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/lead_capture"
	"github.com/gin-gonic/gin"
)

type SourceCatalogHandler struct {
	svc *leadsvc.LeadSourceCatalogService
}

func NewSourceCatalogHandler(svc *leadsvc.LeadSourceCatalogService) *SourceCatalogHandler {
	return &SourceCatalogHandler{svc: svc}
}

func (h *SourceCatalogHandler) List(c *gin.Context) {
	if h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "source catalog service unavailable", nil)
		return
	}
	category := strings.TrimSpace(c.Query("category"))
	enabledOnly := strings.EqualFold(strings.TrimSpace(c.Query("enabled")), "true")
	items, err := h.svc.List(c.Request.Context(), category, enabledOnly)
	if err != nil {
		switch {
		case errors.Is(err, leadsvc.ErrInvalidSourceCatalogCategory):
			contracts.ResponseError(c, http.StatusBadRequest, contracts.ErrCodeValidationFailed, "invalid category")
		default:
			contracts.ResponseInternalError(c, err)
		}
		return
	}
	contracts.ResponseSuccess(c, gin.H{"items": items})
}

func (h *SourceCatalogHandler) Create(c *gin.Context) {
	if h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "source catalog service unavailable", nil)
		return
	}
	var req dto.LeadSourceCatalogCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid body: "+err.Error())
		return
	}
	created, err := h.svc.Create(c.Request.Context(), leadsvc.LeadSourceCatalogCreateRequest{
		Category: req.Category,
		Code:     req.Code,
		Label:    req.Label,
		Sort:     req.Sort,
		Enabled:  req.Enabled,
	})
	if err != nil {
		switch {
		case errors.Is(err, leadsvc.ErrInvalidSourceCatalogPayload):
			contracts.ResponseError(c, http.StatusBadRequest, contracts.ErrCodeValidationFailed, "invalid source catalog payload")
		default:
			contracts.ResponseInternalError(c, err)
		}
		return
	}
	contracts.ResponseCreated(c, created)
}

func (h *SourceCatalogHandler) Update(c *gin.Context) {
	if h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "source catalog service unavailable", nil)
		return
	}
	catalogUUID := strings.TrimSpace(c.Param("catalog_id"))
	if catalogUUID == "" {
		contracts.ResponseBadRequest(c, "catalog_id is required")
		return
	}
	var req dto.LeadSourceCatalogUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid body: "+err.Error())
		return
	}
	updated, err := h.svc.Update(c.Request.Context(), catalogUUID, leadsvc.LeadSourceCatalogUpdateRequest{
		Code:    req.Code,
		Label:   req.Label,
		Sort:    req.Sort,
		Enabled: req.Enabled,
	})
	if err != nil {
		switch {
		case errors.Is(err, leadsvc.ErrInvalidSourceCatalogPayload):
			contracts.ResponseError(c, http.StatusBadRequest, contracts.ErrCodeValidationFailed, "invalid source catalog payload")
		case errors.Is(err, leadrepo.ErrSourceCatalogNotFound):
			contracts.ResponseNotFound(c, "source catalog not found")
		default:
			contracts.ResponseInternalError(c, err)
		}
		return
	}
	contracts.ResponseSuccess(c, updated)
}

func (h *SourceCatalogHandler) Delete(c *gin.Context) {
	if h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "source catalog service unavailable", nil)
		return
	}
	catalogUUID := strings.TrimSpace(c.Param("catalog_id"))
	if catalogUUID == "" {
		contracts.ResponseBadRequest(c, "catalog_id is required")
		return
	}
	if err := h.svc.Delete(c.Request.Context(), catalogUUID); err != nil {
		switch {
		case errors.Is(err, leadrepo.ErrSourceCatalogNotFound):
			contracts.ResponseNotFound(c, "source catalog not found")
		default:
			contracts.ResponseInternalError(c, err)
		}
		return
	}
	contracts.ResponseSuccess(c, gin.H{"deleted": true})
}
