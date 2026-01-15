package iam

import (
	"strconv"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/contracts"
	iamm "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/iam"
	authmw "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/middleware"
	srviam "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/iam"
	"github.com/gin-gonic/gin"
)

type DepartmentHandler struct {
	service *srviam.DepartmentService
}

type departmentNode struct {
	iamm.Department
	Children []*departmentNode `json:"children,omitempty"`
}

func NewDepartmentHandler(svc *srviam.DepartmentService) *DepartmentHandler {
	return &DepartmentHandler{service: svc}
}

func (h *DepartmentHandler) List(c *gin.Context) {
	var query DepartmentListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		contracts.ResponseBadRequest(c, "invalid query: "+err.Error())
		return
	}
	deps, err := h.service.List(c.Request.Context(), srviam.DepartmentFilter{TenantUUID: query.TenantUUID})
	if err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, gin.H{"items": deps})
}

func (h *DepartmentHandler) Tree(c *gin.Context) {
	var query DepartmentListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		contracts.ResponseBadRequest(c, "invalid query: "+err.Error())
		return
	}
	deps, err := h.service.List(c.Request.Context(), srviam.DepartmentFilter{TenantUUID: query.TenantUUID})
	if err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	nodes := make(map[uint64]*departmentNode, len(deps))
	for i := range deps {
		copy := deps[i]
		nodes[copy.ID] = &departmentNode{Department: copy}
	}
	var roots []*departmentNode
	for _, node := range nodes {
		if node.ParentID != nil {
			if parent, ok := nodes[*node.ParentID]; ok {
				parent.Children = append(parent.Children, node)
				continue
			}
		}
		roots = append(roots, node)
	}
	contracts.ResponseSuccess(c, roots)
}

func (h *DepartmentHandler) Create(c *gin.Context) {
	var req CreateDepartmentRequest
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
	dept, err := h.service.Create(c.Request.Context(), srviam.CreateDepartmentInput{
		TenantUUID:  req.TenantUUID,
		Name:        req.Name,
		Code:        req.Code,
		ParentID:    req.ParentID,
		Description: req.Description,
		SortOrder:   req.SortOrder,
		ActorID:     actorID,
	})
	if err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, dept)
}

func (h *DepartmentHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		contracts.ResponseBadRequest(c, "invalid department id")
		return
	}
	var req UpdateDepartmentRequest
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
	dept, err := h.service.Update(c.Request.Context(), id, srviam.UpdateDepartmentInput{
		Name:        req.Name,
		Description: req.Description,
		SortOrder:   req.SortOrder,
		ParentID:    req.ParentID,
		ActorID:     actorID,
	})
	if err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, dept)
}

func (h *DepartmentHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		contracts.ResponseBadRequest(c, "invalid department id")
		return
	}
	tc, _ := authmw.GetTenantContext(c)
	var actorID *uint64
	if tc.UserID > 0 {
		uid := uint64(tc.UserID)
		actorID = &uid
	}
	if err := h.service.Delete(c.Request.Context(), id, actorID); err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, gin.H{"ok": true})
}
