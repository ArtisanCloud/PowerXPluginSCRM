package webhooks

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/ArtisanCloud/PowerWeChat/v3/src/kernel"
	contract "github.com/ArtisanCloud/PowerWeChat/v3/src/kernel/contract"
	"github.com/ArtisanCloud/PowerWeChat/v3/src/work"
	workuser "github.com/ArtisanCloud/PowerWeChat/v3/src/work/user/response"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/config"
	orgmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/org_sync"
	socialmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/social_channel_governance"
	orgrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/org_sync"
	SocialRepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/social_channel_governance"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type WeComWebhookHandler struct {
	repo        *SocialRepo.AccountRepository
	sourceRepo  *orgrepo.SourceAccountRepository
	memberRepo  *orgrepo.SourceMemberRepository
	profileRepo *orgrepo.SourceMemberProfileRepository
	cfg         *config.Config
}

func NewWeComWebhookHandler(repo *SocialRepo.AccountRepository, sourceRepo *orgrepo.SourceAccountRepository, memberRepo *orgrepo.SourceMemberRepository, profileRepo *orgrepo.SourceMemberProfileRepository, cfg *config.Config) *WeComWebhookHandler {
	return &WeComWebhookHandler{
		repo:        repo,
		sourceRepo:  sourceRepo,
		memberRepo:  memberRepo,
		profileRepo: profileRepo,
		cfg:         cfg,
	}
}

func (h *WeComWebhookHandler) Handle(c *gin.Context) {
	if h == nil || h.repo == nil {
		c.String(http.StatusServiceUnavailable, "service unavailable")
		return
	}
	account, err := h.loadAccount(c)
	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}
	credentials := credentialsToMap(account.Credentials)
	app, err := newWeComWebhookApp(credentials)
	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}
	if c.Request.Method == http.MethodGet {
		rs, err := app.Server.VerifyURL(c.Request)
		if err != nil {
			c.String(http.StatusBadRequest, err.Error())
			return
		}
		body, _ := io.ReadAll(rs.Body)
		c.String(rs.StatusCode, string(body))
		return
	}
	rs, err := app.Server.Notify(c.Request, func(event contract.EventInterface) interface{} {
		return kernel.SUCCESS_EMPTY_RESPONSE
	})
	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}
	if rs == nil {
		c.String(http.StatusOK, kernel.SUCCESS_EMPTY_RESPONSE)
		return
	}
	body, _ := io.ReadAll(rs.Body)
	c.String(rs.StatusCode, string(body))
}

func (h *WeComWebhookHandler) HandleOAuth(c *gin.Context) {
	account, err := h.loadAccount(c)
	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}
	requestBase := resolveRequestBaseURL(c)
	redirect := sanitizeRedirect(c.Query("redirect"))
	oauthURL, err := buildWeComOAuthURL(account, requestBase, resolveAPIPrefix(h.cfg), redirect, h.cfg)
	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}
	c.Redirect(http.StatusFound, oauthURL)
}

func (h *WeComWebhookHandler) HandleOAuthCallback(c *gin.Context) {
	if h == nil || h.repo == nil || h.sourceRepo == nil || h.memberRepo == nil {
		c.String(http.StatusServiceUnavailable, "service unavailable")
		return
	}
	account, err := h.loadAccount(c)
	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}
	code := strings.TrimSpace(c.Query("code"))
	if code == "" {
		c.String(http.StatusBadRequest, "missing oauth code")
		return
	}
	requestBase := resolveRequestBaseURL(c)
	callbackURL := buildWeComOAuthCallback(requestBase, resolveAPIPrefix(h.cfg), account.AccountUUID)
	app, err := newWeComOAuthApp(credentialsToMap(account.Credentials), callbackURL)
	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}
	userInfo, err := app.Auth.GetUserInfo(c.Request.Context(), code)
	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}
	if userInfo == nil || userInfo.ErrCode != 0 {
		c.String(http.StatusBadRequest, fmt.Sprintf("wecom oauth failed: %d %s", userInfo.ErrCode, userInfo.ErrMsg))
		return
	}
	userID := strings.TrimSpace(userInfo.UserID)
	if userID == "" {
		c.String(http.StatusBadRequest, "missing user id")
		return
	}
	var detail *workuser.UserDetail
	if strings.TrimSpace(userInfo.UserTicket) != "" {
		userDetail, err := app.Auth.GetUserDetail(c.Request.Context(), strings.TrimSpace(userInfo.UserTicket))
		if err == nil {
			detail = userDetail
		}
	}
	if err := h.upsertWeComMember(c, account, userID, detail); err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}
	redirect := decodeOAuthState(strings.TrimSpace(c.Query("state")))
	if redirect == "" {
		redirect = "/scrm/org_sync"
	}
	if strings.Contains(redirect, "?") {
		redirect = redirect + "&oauth=success"
	} else {
		redirect = redirect + "?oauth=success"
	}
	c.Redirect(http.StatusFound, redirect)
}

func newWeComWebhookApp(credentials map[string]string) (*work.Work, error) {
	corpID := strings.TrimSpace(credentials["corp_id"])
	secret := strings.TrimSpace(credentials["app_secret"])
	token := strings.TrimSpace(credentials["token"])
	aesKey := strings.TrimSpace(credentials["aes_key"])
	agentID := strings.TrimSpace(credentials["agent_id"])
	agentIDInt := 0
	if agentID != "" {
		if val, err := strconv.Atoi(agentID); err == nil {
			agentIDInt = val
		}
	}
	if corpID == "" || secret == "" || token == "" || aesKey == "" {
		return nil, errors.New("missing wecom callback credentials")
	}
	return work.NewWork(&work.UserConfig{
		CorpID:      corpID,
		AgentID:     agentIDInt,
		Secret:      secret,
		Token:       token,
		AESKey:      aesKey,
		CallbackURL: strings.TrimSpace(credentials["oauth_callback"]),
		OAuth: work.OAuth{
			Callback: strings.TrimSpace(credentials["oauth_callback"]),
			Scopes:   nil,
		},
		HttpDebug: true,
	})
}

func newWeComOAuthApp(credentials map[string]string, callback string) (*work.Work, error) {
	corpID := strings.TrimSpace(credentials["corp_id"])
	secret := strings.TrimSpace(credentials["app_secret"])
	agentID := strings.TrimSpace(credentials["agent_id"])
	agentIDInt := 0
	if agentID != "" {
		if val, err := strconv.Atoi(agentID); err == nil {
			agentIDInt = val
		}
	}
	if corpID == "" || secret == "" || agentIDInt == 0 {
		return nil, errors.New("missing wecom oauth credentials")
	}
	return work.NewWork(&work.UserConfig{
		CorpID:  corpID,
		AgentID: agentIDInt,
		Secret:  secret,
		OAuth: work.OAuth{
			Callback: callback,
			Scopes:   []string{"snsapi_privateinfo"},
		},
		HttpDebug: true,
	})
}

func credentialsToMap(input map[string]interface{}) map[string]string {
	out := make(map[string]string, len(input))
	for key, value := range input {
		if value == nil {
			continue
		}
		out[key] = strings.TrimSpace(strings.ReplaceAll(strings.TrimSpace(fmt.Sprintf("%v", value)), "\u0000", ""))
	}
	return out
}

func (h *WeComWebhookHandler) loadAccount(c *gin.Context) (*socialmodel.ChannelAccount, error) {
	if h == nil || h.repo == nil {
		return nil, errors.New("service unavailable")
	}
	accountUUID := strings.TrimSpace(c.Param("account_uuid"))
	if accountUUID == "" {
		return nil, errors.New("account_uuid is required")
	}
	account, err := h.repo.FindByUUID(c.Request.Context(), accountUUID)
	if err != nil {
		return nil, errors.New("account not found")
	}
	if account == nil || !strings.EqualFold(account.ChannelCode, "wechat") || !strings.EqualFold(account.AppType, "wecom") {
		return nil, errors.New("invalid account")
	}
	return account, nil
}

func (h *WeComWebhookHandler) upsertWeComMember(c *gin.Context, account *socialmodel.ChannelAccount, userID string, detail *workuser.UserDetail) error {
	if h == nil || h.sourceRepo == nil || h.memberRepo == nil {
		return errors.New("repository not configured")
	}
	sourceAccount, err := h.ensureSourceAccount(c.Request.Context(), account)
	if err != nil {
		return err
	}
	name := strings.TrimSpace(userID)
	phone := ""
	email := ""
	profileStatus := orgmodel.ProfileStatusLimited
	if detail != nil {
		if strings.TrimSpace(detail.Name) != "" {
			name = strings.TrimSpace(detail.Name)
		}
		if strings.TrimSpace(detail.Mobile) != "" {
			phone = strings.TrimSpace(detail.Mobile)
		}
		if strings.TrimSpace(detail.Email) != "" {
			email = strings.TrimSpace(detail.Email)
		}
		if email == "" && strings.TrimSpace(detail.BizMail) != "" {
			email = strings.TrimSpace(detail.BizMail)
		}
		profileStatus = orgmodel.ProfileStatusFull
	}
	now := time.Now().UTC()
	record := &orgmodel.SourceMember{
		TenantUUID:         sourceAccount.TenantUUID,
		SourceAccountUUID:  sourceAccount.SourceAccountUUID,
		ChannelAccountUUID: account.AccountUUID,
		ExternalMemberID:   strings.TrimSpace(userID),
		Name:               name,
		Phone:              phone,
		Email:              email,
		ProfileStatus:      profileStatus,
		Status:             "active",
		UpdatedAt:          now,
	}
	if err := h.memberRepo.DB.WithContext(c.Request.Context()).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "tenant_uuid"}, {Name: "source_account_uuid"}, {Name: "external_member_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"channel_account_uuid", "name", "phone", "email", "profile_status", "status", "updated_at"}),
		}).
		Create(record).Error; err != nil {
		return err
	}
	if profileStatus != orgmodel.ProfileStatusFull || h.profileRepo == nil {
		return nil
	}
	var member orgmodel.SourceMember
	if err := h.memberRepo.DB.WithContext(c.Request.Context()).
		Where("tenant_uuid = ? AND source_account_uuid = ? AND external_member_id = ?", sourceAccount.TenantUUID, sourceAccount.SourceAccountUUID, strings.TrimSpace(userID)).
		First(&member).Error; err != nil {
		return err
	}
	profile := &orgmodel.SourceMemberProfile{
		TenantUUID:         sourceAccount.TenantUUID,
		SourceMemberUUID:   member.SourceMemberUUID,
		ChannelAccountUUID: account.AccountUUID,
		ExternalMemberID:   strings.TrimSpace(userID),
		Name:               name,
		Phone:              phone,
		Email:              email,
		UpdatedAt:          now,
	}
	return h.profileRepo.DB.WithContext(c.Request.Context()).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "tenant_uuid"}, {Name: "source_member_uuid"}},
			DoUpdates: clause.AssignmentColumns([]string{"channel_account_uuid", "external_member_id", "name", "phone", "email", "avatar_url", "updated_at"}),
		}).
		Create(profile).Error
}

func (h *WeComWebhookHandler) ensureSourceAccount(ctx context.Context, account *socialmodel.ChannelAccount) (*orgmodel.SourceAccount, error) {
	if h == nil || h.sourceRepo == nil || h.sourceRepo.DB == nil {
		return nil, errors.New("source account repository not configured")
	}
	tenantUUID := strings.ToLower(strings.TrimSpace(account.TenantUuid))
	accountUUID := strings.ToLower(strings.TrimSpace(account.AccountUUID))
	if tenantUUID == "" || accountUUID == "" {
		return nil, errors.New("tenant_uuid is required")
	}
	existing, err := h.sourceRepo.FindByChannelAccount(ctx, tenantUUID, accountUUID)
	if err == nil && existing != nil {
		return existing, nil
	}
	if err != nil && !errors.Is(err, orgrepo.ErrSourceAccountNotFound) {
		return nil, err
	}
	record := &orgmodel.SourceAccount{
		SourceAccountUUID:  account.AccountUUID,
		TenantUUID:         tenantUUID,
		Provider:           strings.ToLower(strings.TrimSpace(account.ChannelCode)),
		AppType:            strings.ToLower(strings.TrimSpace(account.AppType)),
		ChannelAccountUUID: &account.AccountUUID,
		DisplayName:        account.DisplayName,
		Status:             orgmodel.SourceAccountStatusActive,
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}
	if err := h.sourceRepo.WithTenantTx(ctx, tenantUUID, func(tx *gorm.DB) error {
		return tx.Create(record).Error
	}); err != nil {
		return nil, err
	}
	return record, nil
}

func buildWeComOAuthURL(account *socialmodel.ChannelAccount, requestBaseURL, apiPrefix, redirect string, cfg *config.Config) (string, error) {
	if account == nil {
		return "", errors.New("account required")
	}
	credentials := credentialsToMap(account.Credentials)
	corpID := strings.TrimSpace(credentials["corp_id"])
	agentID := strings.TrimSpace(credentials["agent_id"])
	if agentID == "" {
		agentID = strings.TrimSpace(account.AccountID)
	}
	if corpID == "" || agentID == "" {
		return "", errors.New("missing corp_id or agent_id")
	}
	base := resolveCallbackBaseURL(requestBaseURL, credentials, cfg)
	callback := buildWeComOAuthCallback(base, apiPrefix, account.AccountUUID)
	if callback == "" {
		return "", errors.New("oauth callback url missing")
	}
	state := encodeOAuthState(redirect)
	values := url.Values{}
	values.Set("appid", corpID)
	values.Set("redirect_uri", callback)
	values.Set("response_type", "code")
	values.Set("scope", "snsapi_privateinfo")
	values.Set("agentid", agentID)
	if state != "" {
		values.Set("state", state)
	}
	return "https://open.weixin.qq.com/connect/oauth2/authorize?" + values.Encode() + "#wechat_redirect", nil
}

func buildWeComOAuthCallback(baseURL, apiPrefix, accountUUID string) string {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if base == "" || accountUUID == "" {
		return ""
	}
	prefix := strings.TrimRight(strings.TrimSpace(apiPrefix), "/")
	if prefix == "" {
		prefix = "/api/v1"
	}
	return fmt.Sprintf("%s%s/webhooks/wechat/wecom/%s/oauth/callback", base, prefix, accountUUID)
}

func resolveAPIPrefix(cfg *config.Config) string {
	if cfg == nil || cfg.Server == nil {
		return "/api/v1"
	}
	prefix := strings.TrimSpace(cfg.Server.APIPrefix)
	if prefix == "" {
		return "/api/v1"
	}
	if !strings.HasPrefix(prefix, "/") {
		prefix = "/" + prefix
	}
	return strings.TrimRight(prefix, "/")
}

func resolveCallbackBaseURL(requestBaseURL string, credentials map[string]string, cfg *config.Config) string {
	if val := strings.TrimSpace(credentials["callback_base_url"]); val != "" {
		if strings.HasPrefix(val, "http://") || strings.HasPrefix(val, "https://") {
			return strings.TrimRight(val, "/")
		}
		return "https://" + strings.TrimRight(val, "/")
	}
	if cfg != nil && cfg.Server != nil {
		if base := strings.TrimSpace(cfg.Server.CallbackBaseURL); base != "" {
			if strings.HasPrefix(base, "http://") || strings.HasPrefix(base, "https://") {
				return strings.TrimRight(base, "/")
			}
			return "https://" + strings.TrimRight(base, "/")
		}
	}
	return strings.TrimRight(strings.TrimSpace(requestBaseURL), "/")
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

func encodeOAuthState(redirect string) string {
	trimmed := strings.TrimSpace(redirect)
	if trimmed == "" {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString([]byte(trimmed))
}

func decodeOAuthState(state string) string {
	if strings.TrimSpace(state) == "" {
		return ""
	}
	value, err := base64.RawURLEncoding.DecodeString(state)
	if err != nil {
		return ""
	}
	return sanitizeRedirect(string(value))
}

func sanitizeRedirect(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(raw, "/") {
		return raw
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	if parsed.Path == "" {
		return ""
	}
	out := parsed.Path
	if parsed.RawQuery != "" {
		out = out + "?" + parsed.RawQuery
	}
	return out
}
