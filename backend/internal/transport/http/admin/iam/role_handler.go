package iam

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/contracts"
	authmw "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/middleware"
	srviam "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/iam"
	"github.com/gin-gonic/gin"
)

type RoleHandler struct {
	service *srviam.RoleService
}

func NewRoleHandler(svc *srviam.RoleService) *RoleHandler {
	return &RoleHandler{service: svc}
}

func (h *RoleHandler) Get(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		contracts.ResponseBadRequest(c, err.Error())
		return
	}
	role, err := h.service.Get(c.Request.Context(), id)
	if handleRoleError(c, err) {
		return
	}
	contracts.ResponseSuccess(c, role)
}

func (h *RoleHandler) List(c *gin.Context) {
	var query RoleListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		contracts.ResponseBadRequest(c, "invalid query: "+err.Error())
		return
	}
	roles, err := h.service.List(c.Request.Context(), srviam.RoleFilter{
		TenantUUID: query.TenantUUID,
		Query:      query.Query,
		ScopeType:  query.ScopeType,
	})
	if handleRoleError(c, err) {
		return
	}
	contracts.ResponseSuccess(c, gin.H{"items": roles})
}

func (h *RoleHandler) Create(c *gin.Context) {
	var req CreateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid body: "+err.Error())
		return
	}
	actorID := currentActorID(c)
	role, err := h.service.Create(c.Request.Context(), srviam.CreateRoleInput{
		TenantUUID:    req.TenantUUID,
		Code:          req.Code,
		Name:          req.Name,
		Description:   req.Description,
		ScopeType:     req.ScopeType,
		CloneRoleID:   req.CloneRoleID,
		PermissionIDs: req.PermissionIDs,
		MemberIDs:     req.MemberIDs,
		ActorID:       actorID,
	})
	if handleRoleError(c, err) {
		return
	}
	contracts.ResponseCreated(c, role)
}

func (h *RoleHandler) Update(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		contracts.ResponseBadRequest(c, err.Error())
		return
	}
	var req UpdateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid body: "+err.Error())
		return
	}
	role, err := h.service.Update(c.Request.Context(), id, srviam.UpdateRoleInput{
		Name:        req.Name,
		Description: req.Description,
		ScopeType:   req.ScopeType,
		ActorID:     currentActorID(c),
	})
	if handleRoleError(c, err) {
		return
	}
	contracts.ResponseSuccess(c, role)
}

func (h *RoleHandler) Delete(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		contracts.ResponseBadRequest(c, err.Error())
		return
	}
	if handleRoleError(c, h.service.Delete(c.Request.Context(), id, currentActorID(c))) {
		return
	}
	contracts.ResponseSuccessWithMessage(c, gin.H{"role_id": id}, "deleted")
}

type RolePermissionsHandler struct {
	service *srviam.RoleService
}

func NewRolePermissionsHandler(svc *srviam.RoleService) *RolePermissionsHandler {
	return &RolePermissionsHandler{service: svc}
}

func (h *RolePermissionsHandler) Replace(c *gin.Context) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		contracts.ResponseBadRequest(c, err.Error())
		return
	}
	var req ReplaceRolePermissionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid body: "+err.Error())
		return
	}
	role, err := h.service.ReplacePermissions(c.Request.Context(), srviam.ReplaceRolePermissionsInput{
		RoleID:        id,
		TenantUUID:    req.TenantUUID,
		PermissionIDs: req.PermissionIDs,
		ActorID:       currentActorID(c),
	})
	if handleRoleError(c, err) {
		return
	}
	contracts.ResponseSuccess(c, role)
}

type RoleMembersHandler struct {
	service *srviam.RoleService
}

func NewRoleMembersHandler(svc *srviam.RoleService) *RoleMembersHandler {
	return &RoleMembersHandler{service: svc}
}

func (h *RoleMembersHandler) Add(c *gin.Context) {
	h.mutateMembers(c, true)
}

func (h *RoleMembersHandler) Remove(c *gin.Context) {
	h.mutateMembers(c, false)
}

func (h *RoleMembersHandler) mutateMembers(c *gin.Context, add bool) {
	id, err := parseUintParam(c, "id")
	if err != nil {
		contracts.ResponseBadRequest(c, err.Error())
		return
	}
	var req RoleMembersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid body: "+err.Error())
		return
	}
	input := srviam.RoleMembersInput{
		RoleID:     id,
		TenantUUID: req.TenantUUID,
		MemberIDs:  req.MemberIDs,
		ActorID:    currentActorID(c),
	}
	if add {
		err = h.service.AddMembers(c.Request.Context(), input)
	} else {
		err = h.service.RemoveMembers(c.Request.Context(), input)
	}
	if handleRoleError(c, err) {
		return
	}
	action := "removed"
	if add {
		action = "added"
	}
	contracts.ResponseSuccessWithMessage(c, gin.H{"role_id": id, "member_ids": req.MemberIDs}, action)
}

type PermissionHandler struct {
	service *srviam.RoleService
}

func NewPermissionHandler(svc *srviam.RoleService) *PermissionHandler {
	return &PermissionHandler{service: svc}
}

func (h *PermissionHandler) List(c *gin.Context) {
	perms, err := h.service.ListPermissions(c.Request.Context())
	if handleRoleError(c, err) {
		return
	}
	contracts.ResponseSuccess(c, gin.H{"items": perms})
}

func parseUintParam(c *gin.Context, name string) (uint64, error) {
	value := c.Param(name)
	if value == "" {
		value = c.Query(name)
	}
	id, err := strconv.ParseUint(value, 10, 64)
	if err != nil || id == 0 {
		return 0, errors.New("invalid " + name)
	}
	return id, nil
}

func currentActorID(c *gin.Context) *uint64 {
	tc, _ := authmw.GetTenantContext(c)
	if tc.UserID <= 0 {
		return nil
	}
	uid := uint64(tc.UserID)
	return &uid
}

func handleRoleError(c *gin.Context, err error) bool {
	if err == nil {
		return false
	}
	switch {
	case errors.Is(err, srviam.ErrTenantRequired):
		contracts.ResponseBadRequest(c, err.Error())
	case errors.Is(err, srviam.ErrPermissionDenied):
		contracts.ResponseError(c, http.StatusForbidden, contracts.ErrCodePermissionDenied, err.Error())
	case errors.Is(err, srviam.ErrRoleNotFound):
		contracts.ResponseNotFound(c, err.Error())
	default:
		contracts.ResponseInternalError(c, err)
	}
	return true
}
