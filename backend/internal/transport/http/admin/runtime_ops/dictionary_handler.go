package runtime_ops

import (
	"errors"
	"net/http"
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/contracts"
	leadrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/lead_capture"
	runtimeops "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/runtime_ops"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/middleware"
	"github.com/gin-gonic/gin"
)

type DictionaryHandler struct {
	svc *runtimeops.DictionaryService
}

type dictionaryCreateRequest struct {
	Namespace string `json:"namespace" binding:"required"`
	Code      string `json:"code" binding:"required"`
	Label     string `json:"label" binding:"required"`
	Sort      int    `json:"sort"`
	Enabled   *bool  `json:"enabled"`
}

type dictionaryUpdateRequest struct {
	Namespace string `json:"namespace"`
	Code      string `json:"code"`
	Label     string `json:"label"`
	Sort      *int   `json:"sort"`
	Enabled   *bool  `json:"enabled"`
}

func NewDictionaryHandler(svc *runtimeops.DictionaryService) *DictionaryHandler {
	return &DictionaryHandler{svc: svc}
}

func (h *DictionaryHandler) List(c *gin.Context) {
	if h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "dictionary service unavailable", nil)
		return
	}
	tenantUUID, ok := middleware.TenantUUIDFromContext(c)
	if !ok || strings.TrimSpace(tenantUUID) == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	namespace := strings.TrimSpace(c.Query("namespace"))
	enabledOnly := strings.EqualFold(strings.TrimSpace(c.Query("enabled")), "true")
	items, err := h.svc.List(c.Request.Context(), tenantUUID, namespace, enabledOnly)
	if err != nil {
		switch {
		case errors.Is(err, runtimeops.ErrInvalidDictionaryPayload):
			contracts.ResponseError(c, http.StatusBadRequest, contracts.ErrCodeValidationFailed, "invalid dictionary payload")
		default:
			contracts.ResponseInternalError(c, err)
		}
		return
	}
	contracts.ResponseSuccess(c, gin.H{"items": items})
}

func (h *DictionaryHandler) Create(c *gin.Context) {
	if h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "dictionary service unavailable", nil)
		return
	}
	tenantUUID, ok := middleware.TenantUUIDFromContext(c)
	if !ok || strings.TrimSpace(tenantUUID) == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	var req dictionaryCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid body: "+err.Error())
		return
	}
	created, err := h.svc.Create(c.Request.Context(), tenantUUID, runtimeops.DictionaryCreateRequest{
		Namespace: req.Namespace,
		Code:      req.Code,
		Label:     req.Label,
		Sort:      req.Sort,
		Enabled:   req.Enabled,
	})
	if err != nil {
		switch {
		case errors.Is(err, runtimeops.ErrInvalidDictionaryPayload):
			contracts.ResponseError(c, http.StatusBadRequest, contracts.ErrCodeValidationFailed, "invalid dictionary payload")
		default:
			contracts.ResponseInternalError(c, err)
		}
		return
	}
	contracts.ResponseCreated(c, created)
}

func (h *DictionaryHandler) Update(c *gin.Context) {
	if h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "dictionary service unavailable", nil)
		return
	}
	tenantUUID, ok := middleware.TenantUUIDFromContext(c)
	if !ok || strings.TrimSpace(tenantUUID) == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	itemID := strings.TrimSpace(c.Param("item_id"))
	if itemID == "" {
		contracts.ResponseBadRequest(c, "item_id is required")
		return
	}
	var req dictionaryUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid body: "+err.Error())
		return
	}
	updated, err := h.svc.Update(c.Request.Context(), tenantUUID, itemID, req.Namespace, runtimeops.DictionaryUpdateRequest{
		Code:    req.Code,
		Label:   req.Label,
		Sort:    req.Sort,
		Enabled: req.Enabled,
	})
	if err != nil {
		switch {
		case errors.Is(err, runtimeops.ErrInvalidDictionaryPayload):
			contracts.ResponseError(c, http.StatusBadRequest, contracts.ErrCodeValidationFailed, "invalid dictionary payload")
		case errors.Is(err, leadrepo.ErrSourceCatalogNotFound):
			contracts.ResponseNotFound(c, "dictionary item not found")
		default:
			contracts.ResponseInternalError(c, err)
		}
		return
	}
	contracts.ResponseSuccess(c, updated)
}

func (h *DictionaryHandler) Delete(c *gin.Context) {
	if h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "dictionary service unavailable", nil)
		return
	}
	tenantUUID, ok := middleware.TenantUUIDFromContext(c)
	if !ok || strings.TrimSpace(tenantUUID) == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	itemID := strings.TrimSpace(c.Param("item_id"))
	if itemID == "" {
		contracts.ResponseBadRequest(c, "item_id is required")
		return
	}
	namespace := strings.TrimSpace(c.Query("namespace"))
	if err := h.svc.Delete(c.Request.Context(), tenantUUID, itemID, namespace); err != nil {
		switch {
		case errors.Is(err, runtimeops.ErrInvalidDictionaryPayload):
			contracts.ResponseError(c, http.StatusBadRequest, contracts.ErrCodeValidationFailed, "invalid dictionary delete payload")
		case errors.Is(err, leadrepo.ErrSourceCatalogNotFound):
			contracts.ResponseNotFound(c, "dictionary item not found")
		default:
			contracts.ResponseInternalError(c, err)
		}
		return
	}
	contracts.ResponseSuccess(c, gin.H{"deleted": true})
}
