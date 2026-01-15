package iam

import (
	"strconv"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/contracts"
	authmw "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/middleware"
	srviam "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/iam"
	"github.com/gin-gonic/gin"
)

type TenantHandler struct {
	service *srviam.TenantService
}

func NewTenantHandler(svc *srviam.TenantService) *TenantHandler {
	return &TenantHandler{service: svc}
}

func (h *TenantHandler) List(c *gin.Context) {
	var query TenantListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		contracts.ResponseBadRequest(c, "invalid query: "+err.Error())
		return
	}
	result, err := h.service.ListWithFilter(c.Request.Context(), srviam.TenantListFilter{
		Status:   query.Status,
		Query:    query.Query,
		Page:     query.Page,
		PageSize: query.PageSize,
	})
	if err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, gin.H{
		"items":     result.Items,
		"total":     result.Total,
		"page":      resultPage(query.Page),
		"page_size": resultPageSize(query.PageSize),
	})
}

func (h *TenantHandler) Create(c *gin.Context) {
	if h == nil || h.service == nil {
		contracts.ResponseServiceUnavailable(c, "tenant service unavailable", nil)
		return
	}
	var req CreateTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid body: "+err.Error())
		return
	}
	tc, _ := authmw.GetTenantContext(c)
	var actorID *uint64
	if tc.UserID > 0 {
		id := uint64(tc.UserID)
		actorID = &id
	}
	tenant, err := h.service.Create(c.Request.Context(), srviam.CreateTenantInput{
		Key:     req.Key,
		Name:    req.Name,
		Status:  req.Status,
		Plan:    req.Plan,
		ActorID: actorID,
	})
	if err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseCreated(c, tenant)
}

func (h *TenantHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		contracts.ResponseBadRequest(c, "invalid tenant id")
		return
	}
	var req UpdateTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid body: "+err.Error())
		return
	}
	tc, _ := authmw.GetTenantContext(c)
	var actorID *uint64
	if tc.UserID > 0 {
		uid := uint64(tc.UserID)
		actorID = &uid
	}
	tenant, err := h.service.Update(c.Request.Context(), id, srviam.UpdateTenantInput{
		Name:    req.Name,
		Status:  req.Status,
		Plan:    req.Plan,
		ActorID: actorID,
	})
	if err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, tenant)
}

func resultPage(page int) int {
	if page <= 0 {
		return 1
	}
	return page
}

func resultPageSize(size int) int {
	if size <= 0 {
		return 20
	}
	if size > 100 {
		return 100
	}
	return size
}
