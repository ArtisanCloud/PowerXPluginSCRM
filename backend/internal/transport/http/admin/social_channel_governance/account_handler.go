package social_channel_governance

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/contracts"
	socialdto "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/dto/social_channel_governance"
	model "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/social_channel_governance"
	repository "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository"
	SocialRepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/social_channel_governance"
	orgdriver "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/org_sync/driver"
	SocialService "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/social_channel_governance"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/middleware"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
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
		Channel:         req.Channel,
		AppType:         req.AppType,
		AccountID:       req.AccountID,
		DisplayName:     req.DisplayName,
		OwnerMemberUUID: req.OwnerMemberUUID,
		Credentials:     req.Credentials,
		CallbackBaseURL: resolveRequestBaseURL(c),
	})
	if err != nil {
		switch {
		case errors.Is(err, SocialRepo.ErrAccountExists):
			contracts.ResponseError(c, http.StatusConflict, contracts.ErrCodeConflict, "account already exists")
		case errors.Is(err, repository.ErrTenantUuidRequired):
			contracts.ResponseBadRequest(c, "tenant_uuid is required")
		case isValidationError(err):
			contracts.ResponseBadRequest(c, err.Error())
		default:
			contracts.ResponseInternalError(c, err)
		}
		return
	}
	basePath := strings.TrimSuffix(c.Request.URL.Path, "/")
	c.Header("Location", fmt.Sprintf("%s/%s", basePath, account.AccountUUID))
	maskAccountCredentials(account)
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
	maskAccountListCredentials(accounts)
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
	maskAccountListCredentials(accounts)
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
		case errors.Is(err, SocialRepo.ErrAccountExists):
			contracts.ResponseError(c, http.StatusConflict, contracts.ErrCodeConflict, "account already exists")
		case errors.Is(err, repository.ErrTenantUuidRequired):
			contracts.ResponseBadRequest(c, "tenant_uuid is required")
		case isValidationError(err):
			contracts.ResponseBadRequest(c, err.Error())
		default:
			logrus.WithError(err).WithFields(logrus.Fields{
				"account_uuid": accountUUID,
				"tenant_uuid":  tenantUUID,
				"request_id":   c.GetString("request_id"),
			}).Error("update channel account failed")
			contracts.ResponseInternalError(c, err)
		}
		return
	}
	maskAccountCredentials(account)
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
		AccountID:       req.AccountID,
		DisplayName:     req.DisplayName,
		OwnerMemberUUID: req.OwnerMemberUUID,
		Status:          req.Status,
		Credentials:     req.Credentials,
		CallbackBaseURL: resolveRequestBaseURL(c),
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
	maskAccountCredentials(account)
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
	maskAccountCredentials(account)
	contracts.ResponseSuccess(c, account)
}

func (h *AccountHandler) TestConnection(c *gin.Context) {
	account, ok := h.loadWeComAccountForTest(c)
	if !ok {
		return
	}
	ipList, err := orgdriver.TestWeComConnection(c.Request.Context(), orgdriver.AccountContext{
		TenantUUID:         account.TenantUuid,
		ChannelAccountUUID: "",
		ChannelCode:        account.ChannelCode,
		AppType:            account.AppType,
		AccountID:          account.AccountID,
		DisplayName:        account.DisplayName,
		Credentials:        credentialsToMap(account.Credentials),
	})
	if err != nil {
		contracts.ResponseBadRequest(c, err.Error())
		return
	}
	contracts.ResponseSuccess(c, gin.H{
		"ip_list": ipList,
	})
}

func (h *AccountHandler) TestAppSecret(c *gin.Context) {
	account, ok := h.loadWeComAccountForTest(c)
	if !ok {
		return
	}
	credentials := credentialsToMap(account.Credentials)
	appSecret := strings.TrimSpace(credentials["app_secret"])
	if appSecret == "" {
		contracts.ResponseBadRequest(c, "应用 Secret 未填写")
		return
	}
	ipList, err := orgdriver.TestWeComConnection(c.Request.Context(), orgdriver.AccountContext{
		TenantUUID:         account.TenantUuid,
		ChannelAccountUUID: account.AccountUUID,
		ChannelCode:        account.ChannelCode,
		AppType:            account.AppType,
		AccountID:          account.AccountID,
		DisplayName:        account.DisplayName,
		Credentials:        credentials,
	})
	if err != nil {
		contracts.ResponseBadRequest(c, err.Error())
		return
	}
	contracts.ResponseSuccess(c, gin.H{
		"ip_list": ipList,
	})
}

func (h *AccountHandler) TestContactSecret(c *gin.Context) {
	account, ok := h.loadWeComAccountForTest(c)
	if !ok {
		return
	}
	var payload struct {
		HttpDebug   bool              `json:"http_debug"`
		Mode        string            `json:"mode"`
		Credentials map[string]string `json:"credentials"`
	}
	_ = c.ShouldBindJSON(&payload)
	credentials := credentialsToMap(account.Credentials)
	for key, value := range payload.Credentials {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			continue
		}
		credentials[trimmedKey] = strings.TrimSpace(value)
	}
	appSecret := strings.TrimSpace(credentials["app_secret"])
	if appSecret == "" {
		contracts.ResponseBadRequest(c, "应用 Secret 未填写")
		return
	}
	if payload.HttpDebug {
		credentials["http_debug"] = "true"
	}
	testFn := orgdriver.TestWeComContactsDetail
	if strings.EqualFold(strings.TrimSpace(payload.Mode), "quick") {
		testFn = orgdriver.TestWeComDepartments
	}
	result, err := testFn(c.Request.Context(), orgdriver.AccountContext{
		TenantUUID:         account.TenantUuid,
		ChannelAccountUUID: account.AccountUUID,
		ChannelCode:        account.ChannelCode,
		AppType:            account.AppType,
		AccountID:          account.AccountID,
		DisplayName:        account.DisplayName,
		Credentials:        credentials,
	})
	if err != nil {
		logrus.WithError(err).WithFields(logrus.Fields{
			"account_uuid": account.AccountUUID,
			"tenant_uuid":  account.TenantUuid,
			"channel":      account.ChannelCode,
			"app_type":     account.AppType,
		}).Error("wecom contact test failed")
		contracts.ResponseBadRequest(c, err.Error())
		return
	}
	contracts.ResponseSuccess(c, gin.H{
		"members_total": result.MembersTotal,
		"units_total":   result.UnitsTotal,
		"sync_mode":     "app_detail",
		"sync_hint":     "应用 Secret 模式：已使用 user/list + department/list 获取部门与成员详情。",
	})
}

func (h *AccountHandler) loadWeComAccountForTest(c *gin.Context) (*model.ChannelAccount, bool) {
	if h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "account service unavailable", nil)
		return nil, false
	}
	accountUUID := strings.TrimSpace(c.Param("account_uuid"))
	if accountUUID == "" {
		contracts.ResponseBadRequest(c, "account_uuid is required")
		return nil, false
	}
	tenantUUID, ok := middleware.TenantUUIDFromContext(c)
	if !ok || tenantUUID == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return nil, false
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
		return nil, false
	}
	if account == nil || !strings.EqualFold(account.ChannelCode, "wechat") || !strings.EqualFold(account.AppType, "wecom") {
		contracts.ResponseBadRequest(c, "only wecom accounts are supported")
		return nil, false
	}
	return account, true
}

func credentialsToMap(input map[string]interface{}) map[string]string {
	out := make(map[string]string, len(input))
	for key, value := range input {
		if value == nil {
			continue
		}
		out[key] = fmt.Sprintf("%v", value)
	}
	return out
}

func maskAccountListCredentials(accounts []*model.ChannelAccount) {
	for _, account := range accounts {
		maskAccountCredentials(account)
	}
}

func maskAccountCredentials(account *model.ChannelAccount) {
	if account == nil || len(account.Credentials) == 0 {
		return
	}
	maskCredentialMapInPlace(account.Credentials)
}

func maskCredentialMapInPlace(input map[string]interface{}) {
	if len(input) == 0 {
		return
	}
	for key := range input {
		if isSensitiveCredentialKey(key) {
			input[key] = "****************"
		}
	}
}

func isSensitiveCredentialKey(key string) bool {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "secret", "app_secret", "token", "refresh_token", "access_token":
		return true
	default:
		return false
	}
}

func resolveRequestBaseURL(c *gin.Context) string {
	if c == nil || c.Request == nil {
		return ""
	}
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	if forwarded := strings.TrimSpace(c.GetHeader("X-Forwarded-Proto")); forwarded != "" {
		scheme = strings.Split(forwarded, ",")[0]
	}
	host := strings.TrimSpace(c.Request.Host)
	if host == "" {
		host = strings.TrimSpace(c.GetHeader("X-Forwarded-Host"))
	}
	if host == "" {
		host = strings.TrimSpace(c.Request.URL.Host)
	}
	if host == "" {
		return ""
	}
	return fmt.Sprintf("%s://%s", scheme, host)
}

func isValidationError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "required") ||
		strings.Contains(msg, "invalid") ||
		strings.Contains(msg, "unsupported")
}
