package iam

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/contracts"
	authmw "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/middleware"
	srviam "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/iam"
	"github.com/gin-gonic/gin"
)

type MemberHandler struct {
	service *srviam.UserService
}

func NewMemberHandler(svc *srviam.UserService) *MemberHandler {
	return &MemberHandler{service: svc}
}

func (h *MemberHandler) List(c *gin.Context) {
	var query MemberListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		contracts.ResponseBadRequest(c, "invalid query: "+err.Error())
		return
	}
	var orgSyncBound *bool
	switch strings.ToLower(strings.TrimSpace(query.OrgSyncBound)) {
	case "1", "true", "yes", "y", "on":
		value := true
		orgSyncBound = &value
	case "0", "false", "no", "n", "off":
		value := false
		orgSyncBound = &value
	}
	items, err := h.service.List(c.Request.Context(), srviam.UserFilter{
		TenantUUID:   query.TenantUUID,
		Status:       query.Status,
		Query:        query.Query,
		OrgSyncBound: orgSyncBound,
	})
	if err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, gin.H{
		"items":     items,
		"page":      resultPage(query.Page),
		"page_size": resultPageSize(query.PageSize),
	})
}

func (h *MemberHandler) Create(c *gin.Context) {
	var req CreateMemberRequest
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
	member, err := h.service.Create(c.Request.Context(), srviam.CreateUserInput{
		TenantUUID:   req.TenantUUID,
		Email:        req.Email,
		DisplayName:  req.DisplayName,
		Username:     req.Username,
		Phone:        req.Phone,
		DepartmentID: req.DepartmentID,
		Status:       req.Status,
		Roles:        req.Roles,
		ActorID:      actorID,
	})
	if err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, member)
}

func (h *MemberHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		contracts.ResponseBadRequest(c, "invalid user id")
		return
	}
	var req UpdateMemberRequest
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
	member, err := h.service.Update(c.Request.Context(), id, srviam.UpdateUserInput{
		DisplayName:  req.DisplayName,
		Status:       req.Status,
		DepartmentID: req.DepartmentID,
		Roles:        req.Roles,
		ReplaceRoles: req.ReplaceRoles,
		ActorID:      actorID,
	})
	if err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, member)
}

func (h *MemberHandler) BulkImport(c *gin.Context) {
	var req BulkImportMembersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid body: "+err.Error())
		return
	}
	if len(req.Users) == 0 {
		contracts.ResponseBadRequest(c, "users required")
		return
	}
	payloads := make([]srviam.CreateUserInput, 0, len(req.Users))
	for _, entry := range req.Users {
		payloads = append(payloads, srviam.CreateUserInput{
			TenantUUID:   req.TenantUUID,
			Email:        entry.Email,
			DisplayName:  entry.DisplayName,
			Username:     entry.Username,
			Phone:        entry.Phone,
			DepartmentID: entry.DepartmentID,
			Status:       entry.Status,
			Roles:        entry.Roles,
		})
	}
	result, err := h.service.BulkImport(c.Request.Context(), payloads)
	if result == nil {
		result = &srviam.UserBulkImportResult{}
	}
	if err != nil && len(result.Failed) == len(payloads) {
		contracts.ResponseError(c, http.StatusBadRequest, contracts.ErrCodeInvalidRequest, err.Error())
		return
	}
	contracts.ResponseSuccess(c, result)
}
