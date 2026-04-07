package lead_capture

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	plcache "github.com/ArtisanCloud/PowerLibs/v3/cache"
	"github.com/ArtisanCloud/PowerSocialite/v3/src/models"
	"github.com/ArtisanCloud/PowerWeChat/v3/src/kernel"
	"github.com/ArtisanCloud/PowerWeChat/v3/src/kernel/response"
	openwork "github.com/ArtisanCloud/PowerWeChat/v3/src/openWork"
	openworksuit "github.com/ArtisanCloud/PowerWeChat/v3/src/openWork/suitAuth"
	"github.com/ArtisanCloud/PowerWeChat/v3/src/work"
	pwexternal "github.com/ArtisanCloud/PowerWeChat/v3/src/work/externalContact"
	pwexternalresp "github.com/ArtisanCloud/PowerWeChat/v3/src/work/externalContact/response"
	socialmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/social_channel_governance"
	socialrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/social_channel_governance"
)

// WeComLeadRecord is a normalized lead payload fetched from WeCom.
type WeComLeadRecord struct {
	ExternalLeadID string
	WechatID       string
	DisplayName    string
	Phone          string
	Email          string
	OccurredAt     time.Time
}

// WeComLeadAdapter fetches leads from WeCom channel accounts.
type WeComLeadAdapter interface {
	FetchLeads(ctx context.Context, req TriggerSyncRequest, channelAccountUUID string) ([]WeComLeadRecord, error)
}

type weComChannelAccountLoader interface {
	GetByAccountUUID(ctx context.Context, tenantUUID, accountUUID string) (*socialmodel.ChannelAccount, error)
}

type weComOpenWorkBindingLoader interface {
	ResolveBindingByChannelAccount(ctx context.Context, tenantUUID, channelAccountUUID string) (*socialmodel.WeComOpenAuthBinding, error)
	GetLatestSuiteTicket(ctx context.Context, tenantUUID, suiteID string) (string, error)
}

type weComPlatformConfigLoader interface {
	GetByChannelProvider(ctx context.Context, channelCode, providerCode string) (*socialmodel.ChannelPlatformSetting, error)
}

type weComExternalContactClient interface {
	GetFollowUsers(ctx context.Context) (*pwexternalresp.ResponseGetFollowUserList, error)
	BatchGet(ctx context.Context, userID []string, cursor string, limit int) (*pwexternalresp.ResponseBatchGetByUser, error)
}

type weComExternalContactClientFactory func(credentials map[string]string) (weComExternalContactClient, error)

type powerWeComExternalContactClient struct {
	client *pwexternal.Client
}

func (c *powerWeComExternalContactClient) GetFollowUsers(ctx context.Context) (*pwexternalresp.ResponseGetFollowUserList, error) {
	return c.client.GetFollowUsers(ctx)
}

func (c *powerWeComExternalContactClient) BatchGet(ctx context.Context, userID []string, cursor string, limit int) (*pwexternalresp.ResponseBatchGetByUser, error) {
	return c.client.BatchGet(ctx, userID, cursor, limit)
}

var defaultWeComExternalContactClientFactory weComExternalContactClientFactory = func(credentials map[string]string) (weComExternalContactClient, error) {
	app, err := newWeComLeadSyncApp(credentials)
	if err != nil {
		return nil, err
	}
	if app == nil || app.ExternalContact == nil {
		return nil, errors.New("wecom external contact client unavailable")
	}
	return &powerWeComExternalContactClient{client: app.ExternalContact}, nil
}

// DefaultWeComLeadAdapter fetches external contacts via WeCom SDK.
type DefaultWeComLeadAdapter struct {
	accountLoader  weComChannelAccountLoader
	openworkLoader weComOpenWorkBindingLoader
	platformLoader weComPlatformConfigLoader
	clientFactory  weComExternalContactClientFactory
	batchLimit     int
}

func NewDefaultWeComLeadAdapter() *DefaultWeComLeadAdapter {
	return &DefaultWeComLeadAdapter{
		clientFactory: defaultWeComExternalContactClientFactory,
		batchLimit:    100,
	}
}

func NewDefaultWeComLeadAdapterWithAccountRepo(repo *socialrepo.AccountRepository) *DefaultWeComLeadAdapter {
	adapter := NewDefaultWeComLeadAdapter()
	adapter.accountLoader = repo
	return adapter
}

func NewDefaultWeComLeadAdapterWithResolvers(
	accountRepo *socialrepo.AccountRepository,
	openworkRepo *socialrepo.OpenWorkFoundationRepository,
	platformRepo *socialrepo.ChannelPlatformSettingRepository,
) *DefaultWeComLeadAdapter {
	adapter := NewDefaultWeComLeadAdapter()
	adapter.accountLoader = accountRepo
	adapter.openworkLoader = openworkRepo
	adapter.platformLoader = platformRepo
	return adapter
}

func (a *DefaultWeComLeadAdapter) FetchLeads(ctx context.Context, req TriggerSyncRequest, channelAccountUUID string) ([]WeComLeadRecord, error) {
	if a == nil {
		return nil, errors.New("wecom lead adapter is nil")
	}
	if a.accountLoader == nil {
		return nil, errors.New("wecom account loader not configured")
	}
	if a.clientFactory == nil {
		return nil, errors.New("wecom client factory not configured")
	}

	tenantUUID := strings.ToLower(strings.TrimSpace(req.TenantUUID))
	channelAccountUUID = strings.ToLower(strings.TrimSpace(channelAccountUUID))
	if tenantUUID == "" || channelAccountUUID == "" {
		return nil, errors.New("tenant_uuid and channel_account_uuid are required")
	}

	account, err := a.accountLoader.GetByAccountUUID(ctx, tenantUUID, channelAccountUUID)
	if err != nil {
		return nil, err
	}
	if account == nil {
		return nil, socialrepo.ErrAccountNotFound
	}
	if !strings.EqualFold(strings.TrimSpace(account.ChannelCode), "wechat") || !strings.EqualFold(strings.TrimSpace(account.AppType), "wecom") {
		return nil, fmt.Errorf("unsupported wecom account identity: %s/%s", strings.TrimSpace(account.ChannelCode), strings.TrimSpace(account.AppType))
	}

	credentials := credentialsToStringMap(account.Credentials)
	credentials = a.mergeDelegatedCredentials(ctx, tenantUUID, channelAccountUUID, credentials)
	client, err := a.clientFactory(credentials)
	if err != nil {
		return nil, err
	}
	if client == nil {
		return nil, errors.New("wecom external contact client unavailable")
	}

	followUsersResp, err := client.GetFollowUsers(ctx)
	if err != nil {
		return nil, err
	}
	if err := validateWeComResponseCode("get_follow_user_list", followUsersResp.ResponseWork); err != nil {
		return nil, err
	}
	followUsers := compactNonEmptyStrings(followUsersResp.FollowUser)
	if len(followUsers) == 0 {
		return []WeComLeadRecord{}, nil
	}

	limit := a.batchLimit
	if limit <= 0 {
		limit = 100
	}
	leads := make([]WeComLeadRecord, 0, 256)
	seenExternalID := map[string]struct{}{}
	cursor := ""
	for {
		batchResp, batchErr := client.BatchGet(ctx, followUsers, cursor, limit)
		if batchErr != nil {
			return nil, batchErr
		}
		if err := validateWeComResponseCode("batch_get_by_user", batchResp.ResponseWork); err != nil {
			return nil, err
		}
		for _, item := range batchResp.ExternalContactList {
			record := mapWeComExternalContactRecord(item)
			if record.ExternalLeadID == "" {
				continue
			}
			if _, ok := seenExternalID[record.ExternalLeadID]; ok {
				continue
			}
			seenExternalID[record.ExternalLeadID] = struct{}{}
			leads = append(leads, normalizeWeComLeadRecord(record))
		}
		nextCursor := strings.TrimSpace(batchResp.NextCursor)
		if nextCursor == "" || nextCursor == cursor {
			break
		}
		cursor = nextCursor
	}

	return leads, nil
}

func normalizeWeComLeadRecord(in WeComLeadRecord) WeComLeadRecord {
	in.ExternalLeadID = strings.TrimSpace(in.ExternalLeadID)
	in.DisplayName = strings.TrimSpace(in.DisplayName)
	in.Phone = strings.TrimSpace(in.Phone)
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	if in.OccurredAt.IsZero() {
		in.OccurredAt = time.Now().UTC()
	}
	return in
}

func validateWeComResponseCode(action string, res response.ResponseWork) error {
	if res.ErrCode != 0 {
		return fmt.Errorf("wecom %s failed: %d %s", action, res.ErrCode, strings.TrimSpace(res.ErrMsg))
	}
	return nil
}

func mapWeComExternalContactRecord(item *pwexternalresp.ResponseExternalContact) WeComLeadRecord {
	if item == nil {
		return WeComLeadRecord{}
	}
	record := WeComLeadRecord{}
	if item.ExternalContact != nil {
		record.ExternalLeadID = strings.TrimSpace(item.ExternalContact.ExternalUserID)
		record.DisplayName = strings.TrimSpace(item.ExternalContact.Name)
		record.Email, record.Phone, record.WechatID = extractWeComExternalProfileAttrs(item.ExternalContact.ExternalProfile)
	}
	if item.FollowInfo != nil {
		if record.DisplayName == "" {
			record.DisplayName = strings.TrimSpace(item.FollowInfo.Remark)
		}
		record.Phone = firstNonEmpty(append(item.FollowInfo.RemarkMobiles, record.Phone)...)
		if item.FollowInfo.CreateTime > 0 {
			record.OccurredAt = time.Unix(int64(item.FollowInfo.CreateTime), 0).UTC()
		}
	}
	return record
}

func extractWeComExternalProfileAttrs(profile *models.ExternalProfile) (email string, phone string, wechatID string) {
	if profile == nil || len(profile.ExternalAttr) == 0 {
		return "", "", ""
	}
	for _, attr := range profile.ExternalAttr {
		if attr == nil || attr.Text == nil {
			continue
		}
		name := strings.ToLower(strings.TrimSpace(attr.Name))
		value := strings.TrimSpace(attr.Text.Value)
		if value == "" {
			continue
		}
		switch {
		case name == "email" || strings.Contains(name, "邮箱"):
			if email == "" {
				email = strings.ToLower(value)
			}
		case strings.Contains(name, "phone") || strings.Contains(name, "mobile") || strings.Contains(name, "手机号") || strings.Contains(name, "电话"):
			if phone == "" {
				phone = value
			}
		case strings.Contains(name, "微信") || strings.Contains(name, "wechat") || strings.Contains(name, "weixin"):
			if wechatID == "" {
				wechatID = value
			}
		}
	}
	return email, phone, wechatID
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if s := strings.TrimSpace(v); s != "" {
			return s
		}
	}
	return ""
}

func compactNonEmptyStrings(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	out := make([]string, 0, len(values))
	for _, v := range values {
		if s := strings.TrimSpace(v); s != "" {
			out = append(out, s)
		}
	}
	return out
}

func credentialsToStringMap(input map[string]interface{}) map[string]string {
	out := make(map[string]string, len(input))
	for key, value := range input {
		if value == nil {
			continue
		}
		out[key] = fmt.Sprintf("%v", value)
	}
	return out
}

func (a *DefaultWeComLeadAdapter) mergeDelegatedCredentials(
	ctx context.Context,
	tenantUUID, channelAccountUUID string,
	input map[string]string,
) map[string]string {
	out := make(map[string]string, len(input)+10)
	for k, v := range input {
		out[k] = strings.TrimSpace(v)
	}
	if strings.TrimSpace(out["app_secret"]) != "" {
		return out
	}

	if a != nil && a.openworkLoader != nil && strings.TrimSpace(tenantUUID) != "" && strings.TrimSpace(channelAccountUUID) != "" {
		binding, err := a.openworkLoader.ResolveBindingByChannelAccount(ctx, tenantUUID, channelAccountUUID)
		if err == nil && binding != nil && strings.TrimSpace(binding.Status) == socialmodel.WeComAuthBindingStatusActive {
			if strings.TrimSpace(out["corp_id"]) == "" {
				out["corp_id"] = strings.TrimSpace(binding.CorpID)
			}
			if strings.TrimSpace(out["permanent_code"]) == "" {
				out["permanent_code"] = strings.TrimSpace(binding.PermanentCode)
			}
			if strings.TrimSpace(out["template_id"]) == "" {
				out["template_id"] = strings.TrimSpace(binding.SuiteID)
			}
			if strings.TrimSpace(out["suite_id"]) == "" {
				out["suite_id"] = strings.TrimSpace(binding.SuiteID)
			}
			if strings.TrimSpace(out["agent_id"]) == "" {
				out["agent_id"] = strings.TrimSpace(binding.AgentID)
			}
			if strings.TrimSpace(out["template_ticket"]) == "" {
				out["template_ticket"] = strings.TrimSpace(binding.SuiteTicket)
			}
			if strings.TrimSpace(out["suite_ticket"]) == "" {
				out["suite_ticket"] = strings.TrimSpace(binding.SuiteTicket)
			}
			if strings.TrimSpace(binding.SuiteID) != "" {
				if latest, latestErr := a.openworkLoader.GetLatestSuiteTicket(ctx, tenantUUID, strings.TrimSpace(binding.SuiteID)); latestErr == nil {
					latest = strings.TrimSpace(latest)
					if latest != "" {
						out["template_ticket"] = latest
						out["suite_ticket"] = latest
					}
				}
			}
		}
	}

	if a == nil || a.platformLoader == nil {
		return out
	}
	record, err := a.platformLoader.GetByChannelProvider(ctx, "wechat", "openwork")
	if err != nil || record == nil || record.Config == nil {
		return out
	}

	cfg := record.Config
	defaultTemplateID := strings.TrimSpace(fmt.Sprintf("%v", cfg["default_template_id"]))
	if defaultTemplateID == "" {
		defaultTemplateID = strings.TrimSpace(fmt.Sprintf("%v", cfg["template_id"]))
	}

	pick := map[string]any{}
	if rows, ok := cfg["templates"].([]any); ok {
		for _, raw := range rows {
			row, ok := raw.(map[string]any)
			if !ok || row == nil {
				continue
			}
			rowTpl := strings.TrimSpace(fmt.Sprintf("%v", row["template_id"]))
			if rowTpl == "" {
				continue
			}
			if len(pick) == 0 {
				pick = row
			}
			if defaultTemplateID != "" && rowTpl == defaultTemplateID {
				pick = row
				break
			}
		}
	}

	applyIfMissing := func(key string, values ...string) {
		if strings.TrimSpace(out[key]) != "" {
			return
		}
		for _, val := range values {
			val = strings.TrimSpace(val)
			if val != "" {
				out[key] = val
				return
			}
		}
	}

	applyIfMissing("template_id",
		strings.TrimSpace(fmt.Sprintf("%v", pick["template_id"])),
		strings.TrimSpace(fmt.Sprintf("%v", cfg["template_id"])),
	)
	applyIfMissing("template_secret",
		strings.TrimSpace(fmt.Sprintf("%v", pick["template_secret"])),
		strings.TrimSpace(fmt.Sprintf("%v", cfg["template_secret"])),
	)
	applyIfMissing("template_ticket",
		strings.TrimSpace(fmt.Sprintf("%v", pick["template_ticket"])),
		strings.TrimSpace(fmt.Sprintf("%v", cfg["template_ticket"])),
	)
	applyIfMissing("provider_corpid",
		strings.TrimSpace(fmt.Sprintf("%v", pick["provider_corpid"])),
		strings.TrimSpace(fmt.Sprintf("%v", cfg["provider_corpid"])),
	)
	applyIfMissing("provider_secret",
		strings.TrimSpace(fmt.Sprintf("%v", pick["provider_secret"])),
		strings.TrimSpace(fmt.Sprintf("%v", cfg["provider_secret"])),
	)
	applyIfMissing("http_debug",
		strings.TrimSpace(fmt.Sprintf("%v", pick["http_debug"])),
		strings.TrimSpace(fmt.Sprintf("%v", cfg["http_debug"])),
	)
	return out
}

func newWeComLeadSyncApp(credentials map[string]string) (*work.Work, error) {
	corpID := strings.TrimSpace(credentials["corp_id"])
	appSecret := strings.TrimSpace(credentials["app_secret"])
	agentIDRaw := strings.TrimSpace(credentials["agent_id"])
	if corpID != "" && appSecret != "" {
		agentID := 0
		if agentIDRaw != "" {
			parsed, err := strconv.Atoi(agentIDRaw)
			if err != nil {
				return nil, errors.New("wecom credentials invalid agent_id")
			}
			agentID = parsed
		}
		return work.NewWork(&work.UserConfig{
			CorpID:    corpID,
			AgentID:   agentID,
			Secret:    appSecret,
			Token:     strings.TrimSpace(credentials["token"]),
			HttpDebug: parseCredentialBool(credentials["http_debug"]),
			OAuth: work.OAuth{
				Callback: strings.TrimSpace(credentials["oauth_callback"]),
				Scopes:   nil,
			},
		})
	}

	templateID := strings.TrimSpace(firstNonEmpty(
		credentials["template_id"],
		credentials["suite_id"],
	))
	templateSecret := strings.TrimSpace(firstNonEmpty(
		credentials["template_secret"],
		credentials["suite_secret"],
	))
	templateTicket := strings.TrimSpace(firstNonEmpty(
		credentials["template_ticket"],
		credentials["suite_ticket"],
	))
	providerCorpID := strings.TrimSpace(firstNonEmpty(
		credentials["provider_corpid"],
		credentials["provider_corpid"],
		credentials["provider_corp_id"],
	))
	providerSecret := strings.TrimSpace(credentials["provider_secret"])
	permanentCode := strings.TrimSpace(credentials["permanent_code"])
	if corpID == "" {
		corpID = strings.TrimSpace(credentials["auth_corp_id"])
	}

	if corpID == "" || templateID == "" || templateSecret == "" || providerCorpID == "" || providerSecret == "" || permanentCode == "" {
		return nil, errors.New("wecom credentials missing corp_id/app_secret or delegated_template credentials")
	}
	if templateTicket == "" {
		return nil, errors.New("wecom delegated credentials missing template_ticket")
	}

	callback := strings.TrimSpace(credentials["oauth_callback"])
	if callback == "" {
		callback = "http://localhost"
	}
	memCache := plcache.NewMemCache("scrm_lead_sync_openwork", 10*time.Minute, os.TempDir())
	if memCache == nil {
		return nil, errors.New("wecom delegated init cache failed")
	}
	httpDebug := parseCredentialBool(credentials["http_debug"])
	openWorkApp, err := openwork.NewOpenWork(&openwork.UserConfig{
		AppID:          templateID,
		Secret:         templateSecret,
		ProviderCorpID: providerCorpID,
		ProviderSecret: providerSecret,
		CallbackURL:    callback,
		Cache:          kernel.CacheInterface(memCache),
		HttpDebug:      httpDebug,
		Log: openwork.Log{
			Level:  "debug",
			Stdout: httpDebug,
		},
	})
	if err != nil {
		return nil, err
	}
	suiteTicketComponent, ok := openWorkApp.GetComponent("SuiteTicket").(*openworksuit.SuiteTicket)
	if !ok || suiteTicketComponent == nil {
		return nil, errors.New("wecom delegated SuiteTicket component unavailable")
	}
	if err := suiteTicketComponent.SetTicket(templateTicket); err != nil {
		return nil, err
	}
	return openWorkApp.ProviderClient(corpID, permanentCode, nil)
}

func parseCredentialBool(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}
