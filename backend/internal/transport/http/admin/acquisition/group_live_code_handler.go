package acquisition

import (
	"strconv"
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/contracts"
	acqrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/repository/acquisition"
	dto "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/dto/acquisition"
	acqsvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/acquisition"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/middleware"
	"github.com/gin-gonic/gin"
)

type GroupLiveCodeHandler struct {
	svc *acqsvc.GroupLiveCodeService
}

func NewGroupLiveCodeHandler(svc *acqsvc.GroupLiveCodeService) *GroupLiveCodeHandler {
	return &GroupLiveCodeHandler{svc: svc}
}

func (h *GroupLiveCodeHandler) Create(c *gin.Context) {
	if h == nil || h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "group live code service unavailable", nil)
		return
	}
	tenantUUID, ok := middleware.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	var req dto.GroupLiveCodeCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid body: "+err.Error())
		return
	}
	item, err := h.svc.Create(c.Request.Context(), acqsvc.GroupLiveCodeCreateRequest{
		TenantUUID:         tenantUUID,
		Channel:            req.Channel,
		AppType:            req.AppType,
		ChannelAccountUUID: req.ChannelAccountUUID,
		ActivityName:       req.ActivityName,
		JoinScene:          req.JoinScene,
		SkipVerify:         req.SkipVerify,
		AutoCreateRoom:     req.AutoCreateRoom,
	})
	if err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, item)
}

func (h *GroupLiveCodeHandler) Get(c *gin.Context) {
	if h == nil || h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "group live code service unavailable", nil)
		return
	}
	tenantUUID, ok := middleware.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	item, err := h.svc.Get(c.Request.Context(), tenantUUID, c.Param("group_code_uuid"))
	if err != nil {
		if err == acqrepo.ErrRecordNotFound {
			contracts.ResponseNotFound(c, "group code not found")
			return
		}
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, item)
}

func (h *GroupLiveCodeHandler) Update(c *gin.Context) {
	if h == nil || h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "group live code service unavailable", nil)
		return
	}
	tenantUUID, ok := middleware.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	var req dto.GroupLiveCodeUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid body: "+err.Error())
		return
	}
	item, err := h.svc.Update(c.Request.Context(), acqsvc.GroupLiveCodeUpdateRequest{
		TenantUUID:     tenantUUID,
		GroupCodeUUID:  c.Param("group_code_uuid"),
		ActivityName:   req.ActivityName,
		SkipVerify:     req.SkipVerify,
		AutoCreateRoom: req.AutoCreateRoom,
		Status:         req.Status,
	})
	if err != nil {
		if err == acqrepo.ErrRecordNotFound {
			contracts.ResponseNotFound(c, "group code not found")
			return
		}
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, item)
}

func (h *GroupLiveCodeHandler) Delete(c *gin.Context) {
	if h == nil || h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "group live code service unavailable", nil)
		return
	}
	tenantUUID, ok := middleware.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	err := h.svc.Delete(c.Request.Context(), tenantUUID, c.Param("group_code_uuid"))
	if err != nil {
		if err == acqrepo.ErrRecordNotFound {
			contracts.ResponseNotFound(c, "group code not found")
			return
		}
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, gin.H{"deleted": true})
}

func (h *GroupLiveCodeHandler) Sync(c *gin.Context) {
	if h == nil || h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "group live code service unavailable", nil)
		return
	}
	tenantUUID, ok := middleware.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	item, err := h.svc.Sync(c.Request.Context(), acqsvc.GroupLiveCodeSyncRequest{
		TenantUUID:    tenantUUID,
		GroupCodeUUID: c.Param("group_code_uuid"),
	})
	if err != nil {
		if err == acqrepo.ErrRecordNotFound {
			contracts.ResponseNotFound(c, "group code not found")
			return
		}
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, item)
}

func (h *GroupLiveCodeHandler) List(c *gin.Context) {
	if h == nil || h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "group live code service unavailable", nil)
		return
	}
	tenantUUID, ok := middleware.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	limit, _ := strconv.Atoi(strings.TrimSpace(c.Query("limit")))
	items, err := h.svc.List(c.Request.Context(), tenantUUID, limit)
	if err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, gin.H{"items": items})
}
