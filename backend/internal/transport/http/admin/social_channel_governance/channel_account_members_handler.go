package social_channel_governance

import (
	"errors"
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/contracts"
	repository "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository"
	SocialRepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/social_channel_governance"
	SocialService "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/social_channel_governance"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/middleware"
	"github.com/gin-gonic/gin"
)

type ChannelAccountMembersHandler struct {
	svc *SocialService.ChannelAccountMemberService
}

func NewChannelAccountMembersHandler(svc *SocialService.ChannelAccountMemberService) *ChannelAccountMembersHandler {
	return &ChannelAccountMembersHandler{svc: svc}
}

type channelAccountMemberUpdateRequest struct {
	OwnerMemberUUID *string  `json:"owner_member_uuid"`
	MemberUserUUIDs []string `json:"member_user_uuids" binding:"required"`
}

func (h *ChannelAccountMembersHandler) UpdateChannelAccountMembers(c *gin.Context) {
	if h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "channel account member service unavailable", nil)
		return
	}
	var req channelAccountMemberUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid body: "+err.Error())
		return
	}
	tenantUUID, ok := middleware.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	accountUUID := strings.TrimSpace(c.Param("account_uuid"))
	if accountUUID == "" {
		contracts.ResponseBadRequest(c, "account_uuid is required")
		return
	}

	account, err := h.svc.UpdateChannelAccountMembers(c.Request.Context(), tenantUUID, accountUUID, SocialService.ChannelAccountMemberUpdateRequest{
		OwnerMemberUUID: req.OwnerMemberUUID,
		MemberUserUUIDs: req.MemberUserUUIDs,
	})
	if err != nil {
		switch {
		case errors.Is(err, SocialRepo.ErrAccountNotFound):
			contracts.ResponseNotFound(c, "account not found")
		case errors.Is(err, repository.ErrTenantUuidRequired):
			contracts.ResponseBadRequest(c, "tenant_uuid is required")
		default:
			contracts.ResponseBadRequest(c, err.Error())
		}
		return
	}
	contracts.ResponseSuccess(c, account)
}
