package social_channel_governance

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/contracts"
	repository "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository"
	SocialRepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/social_channel_governance"
	SocialService "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/social_channel_governance"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/middleware"
	"github.com/gin-gonic/gin"
)

type AccountHandler struct {
	svc *SocialService.AccountService
}

func NewAccountHandler(svc *SocialService.AccountService) *AccountHandler {
	return &AccountHandler{svc: svc}
}

type accountCreateRequest struct {
	Channel       string `json:"channel" binding:"required"`
	AppType       string `json:"app_type" binding:"required"`
	AccountID     string `json:"account_id" binding:"required"`
	DisplayName   string `json:"display_name" binding:"required"`
	OwnerUserUUID string `json:"owner_user_uuid" binding:"required"`
}

func (h *AccountHandler) CreateAccount(c *gin.Context) {
	if h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "account service unavailable", nil)
		return
	}
	var req accountCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid body: "+err.Error())
		return
	}
	tenantUUID, ok := middleware.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}

	account, err := h.svc.CreateAccount(c.Request.Context(), tenantUUID, SocialService.AccountCreateRequest{
		Channel:       req.Channel,
		AppType:       req.AppType,
		AccountID:     req.AccountID,
		DisplayName:   req.DisplayName,
		OwnerUserUUID: req.OwnerUserUUID,
	})
	if err != nil {
		switch {
		case errors.Is(err, SocialService.ErrCredentialExpired):
			contracts.ResponseErrorWithDetails(c, http.StatusBadRequest, contracts.ErrCodeValidationFailed, "credentials expired", account)
		case errors.Is(err, SocialRepo.ErrAccountExists):
			contracts.ResponseError(c, http.StatusConflict, contracts.ErrCodeConflict, "account already exists")
		case errors.Is(err, repository.ErrTenantUuidRequired):
			contracts.ResponseBadRequest(c, "tenant_uuid is required")
		default:
			contracts.ResponseInternalError(c, err)
		}
		return
	}
	basePath := strings.TrimSuffix(c.Request.URL.Path, "/")
	c.Header("Location", fmt.Sprintf("%s/%s", basePath, account.AccountUUID))
	contracts.ResponseCreated(c, account)
}

func (h *AccountHandler) ListAccounts(c *gin.Context) {
	if h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "account service unavailable", nil)
		return
	}
	tenantUUID, ok := middleware.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	accounts, err := h.svc.ListAccounts(c.Request.Context(), tenantUUID)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrTenantUuidRequired):
			contracts.ResponseBadRequest(c, "tenant_uuid is required")
		default:
			contracts.ResponseInternalError(c, err)
		}
		return
	}
	contracts.ResponseSuccess(c, map[string]any{"items": accounts})
}

func (h *AccountHandler) GetAccount(c *gin.Context) {
	if h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "account service unavailable", nil)
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
	account, err := h.svc.GetAccount(c.Request.Context(), tenantUUID, accountUUID)
	if err != nil {
		switch {
		case errors.Is(err, SocialRepo.ErrAccountNotFound):
			contracts.ResponseNotFound(c, "account not found")
		case errors.Is(err, repository.ErrTenantUuidRequired):
			contracts.ResponseBadRequest(c, "tenant_uuid is required")
		default:
			contracts.ResponseInternalError(c, err)
		}
		return
	}
	contracts.ResponseSuccess(c, account)
}
