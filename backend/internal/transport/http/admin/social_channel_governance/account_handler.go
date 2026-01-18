package social_channel_governance

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/contracts"
	socialdto "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/dto/social_channel_governance"
	repository "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository"
	SocialRepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/social_channel_governance"
	SocialService "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/social_channel_governance"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/middleware"
	"github.com/gin-gonic/gin"
)

type AccountHandler struct {
	svc *SocialService.ChannelAccountService
}

func NewAccountHandler(svc *SocialService.ChannelAccountService) *AccountHandler {
	return &AccountHandler{svc: svc}
}

func (h *AccountHandler) CreateAccount(c *gin.Context) {
	if h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "account service unavailable", nil)
		return
	}
	var req socialdto.ChannelAccountCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid body: "+err.Error())
		return
	}
	tenantUUID, ok := middleware.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}

	account, err := h.svc.CreateAccount(c.Request.Context(), tenantUUID, SocialService.ChannelAccountCreateRequest{
		Channel:       req.Channel,
		AppType:       req.AppType,
		AccountID:     req.AccountID,
		DisplayName:   req.DisplayName,
		OwnerUserUUID: req.OwnerUserUUID,
		Credentials:   req.Credentials,
	})
	if err != nil {
		switch {
		case errors.Is(err, SocialService.ErrChannelAccountCredentialExpired):
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

func (h *AccountHandler) ListDeletedAccounts(c *gin.Context) {
	if h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "account service unavailable", nil)
		return
	}
	tenantUUID, ok := middleware.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	accounts, err := h.svc.ListDeletedAccounts(c.Request.Context(), tenantUUID)
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

func (h *AccountHandler) UpdateAccount(c *gin.Context) {
	if h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "account service unavailable", nil)
		return
	}
	accountUUID := strings.TrimSpace(c.Param("account_uuid"))
	if accountUUID == "" {
		contracts.ResponseBadRequest(c, "account_uuid is required")
		return
	}
	var req socialdto.ChannelAccountUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid body: "+err.Error())
		return
	}
	tenantUUID, ok := middleware.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}

	account, err := h.svc.UpdateAccount(c.Request.Context(), tenantUUID, accountUUID, SocialService.ChannelAccountUpdateRequest{
		DisplayName:   req.DisplayName,
		OwnerUserUUID: req.OwnerUserUUID,
		Status:        req.Status,
		Credentials:   req.Credentials,
	})
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

func (h *AccountHandler) DeleteAccount(c *gin.Context) {
	if h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "account service unavailable", nil)
		return
	}
	accountUUID := strings.TrimSpace(c.Param("account_uuid"))
	if accountUUID == "" {
		contracts.ResponseBadRequest(c, "account_uuid is required")
		return
	}
	tenantUUID, ok := middleware.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	err := h.svc.DeleteAccount(c.Request.Context(), tenantUUID, accountUUID)
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
	contracts.ResponseSuccess(c, gin.H{"account_uuid": accountUUID})
}

func (h *AccountHandler) RestoreAccount(c *gin.Context) {
	if h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "account service unavailable", nil)
		return
	}
	accountUUID := strings.TrimSpace(c.Param("account_uuid"))
	if accountUUID == "" {
		contracts.ResponseBadRequest(c, "account_uuid is required")
		return
	}
	tenantUUID, ok := middleware.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	account, err := h.svc.RestoreAccount(c.Request.Context(), tenantUUID, accountUUID)
	if err != nil {
		switch {
		case errors.Is(err, SocialRepo.ErrAccountNotFound):
			contracts.ResponseNotFound(c, "account not found")
		case errors.Is(err, SocialRepo.ErrAccountNotDeleted):
			contracts.ResponseError(c, http.StatusConflict, contracts.ErrCodeConflict, "account is not deleted")
		case errors.Is(err, repository.ErrTenantUuidRequired):
			contracts.ResponseBadRequest(c, "tenant_uuid is required")
		default:
			contracts.ResponseInternalError(c, err)
		}
		return
	}
	contracts.ResponseSuccess(c, account)
}
