package iam

import (
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/contracts"
	srviam "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/iam"
	"github.com/gin-gonic/gin"
)

type UserDirectoryHandler struct {
	service *srviam.UserService
}

func NewUserDirectoryHandler(svc *srviam.UserService) *UserDirectoryHandler {
	return &UserDirectoryHandler{service: svc}
}

func (h *UserDirectoryHandler) List(c *gin.Context) {
	var query UserDirectoryListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		contracts.ResponseBadRequest(c, "invalid query: "+err.Error())
		return
	}
	items, err := h.service.ListDirectory(c.Request.Context(), srviam.UserDirectoryFilter{
		Status: query.Status,
		Query:  query.Query,
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
