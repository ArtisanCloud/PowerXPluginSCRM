package lead_capture

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ArtisanCloud/PowerSocialite/v3/src/models"
	"github.com/ArtisanCloud/PowerWeChat/v3/src/kernel/response"
	"github.com/ArtisanCloud/PowerWeChat/v3/src/work"
	pwexternal "github.com/ArtisanCloud/PowerWeChat/v3/src/work/externalContact"
	pwexternalresp "github.com/ArtisanCloud/PowerWeChat/v3/src/work/externalContact/response"
	socialmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/social_channel_governance"
	socialrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/social_channel_governance"
)

// WeComLeadRecord is a normalized lead payload fetched from WeCom.
type WeComLeadRecord struct {
	ExternalLeadID string
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
	accountLoader weComChannelAccountLoader
	clientFactory weComExternalContactClientFactory
	batchLimit    int
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

	client, err := a.clientFactory(credentialsToStringMap(account.Credentials))
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
		record.Email = extractWeComExternalEmail(item.ExternalContact.ExternalProfile)
	}
	if item.FollowInfo != nil {
		if record.DisplayName == "" {
			record.DisplayName = strings.TrimSpace(item.FollowInfo.Remark)
		}
		record.Phone = firstNonEmpty(item.FollowInfo.RemarkMobiles...)
		if item.FollowInfo.CreateTime > 0 {
			record.OccurredAt = time.Unix(int64(item.FollowInfo.CreateTime), 0).UTC()
		}
	}
	return record
}

func extractWeComExternalEmail(profile *models.ExternalProfile) string {
	if profile == nil || len(profile.ExternalAttr) == 0 {
		return ""
	}
	for _, attr := range profile.ExternalAttr {
		if attr == nil || attr.Text == nil {
			continue
		}
		name := strings.ToLower(strings.TrimSpace(attr.Name))
		if name == "email" || strings.Contains(name, "邮箱") {
			return strings.ToLower(strings.TrimSpace(attr.Text.Value))
		}
	}
	return ""
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

func newWeComLeadSyncApp(credentials map[string]string) (*work.Work, error) {
	corpID := strings.TrimSpace(credentials["corp_id"])
	appSecret := strings.TrimSpace(credentials["app_secret"])
	agentIDRaw := strings.TrimSpace(credentials["agent_id"])
	if corpID == "" || appSecret == "" {
		return nil, errors.New("wecom credentials missing corp_id/app_secret")
	}
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

func parseCredentialBool(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}
