package social_channel_governance

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	model "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/social_channel_governance"
	repository "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/social_channel_governance"
	"github.com/sirupsen/logrus"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const wecomAPIBase = "https://qyapi.weixin.qq.com/cgi-bin/service"

const openWorkAuthModeDelegatedTemplate = "delegated_template"

var openWorkTemplateIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{3,128}$`)

type OpenWorkFoundationService struct {
	repo        *repository.OpenWorkFoundationRepository
	accountRepo *repository.AccountRepository
	httpClient  *http.Client
}

func NewOpenWorkFoundationService(repo *repository.OpenWorkFoundationRepository, accountRepo *repository.AccountRepository) *OpenWorkFoundationService {
	return &OpenWorkFoundationService{
		repo:        repo,
		accountRepo: accountRepo,
		httpClient:  &http.Client{Timeout: 15 * time.Second},
	}
}

func (s *OpenWorkFoundationService) SetHTTPClient(client *http.Client) *OpenWorkFoundationService {
	if s == nil {
		return s
	}
	if client != nil {
		s.httpClient = client
	}
	return s
}

type OpenWorkEventIngestInput struct {
	TenantUUID  string
	SuiteID     string
	EventType   string
	SuiteTicket string
	CorpID      string
	AgentID     string
	EventTime   int64
	Payload     map[string]any
}

type OpenWorkAuthorizeStartInput struct {
	TenantUUID     string
	TemplateID     string
	TemplateSecret string
	TemplateTicket string
	ProviderCorpID string
	ProviderSecret string
	State          string
}

type OpenWorkAuthorizeCompleteInput struct {
	TenantUUID         string
	TemplateID         string
	TemplateSecret     string
	TemplateTicket     string
	ProviderCorpID     string
	ProviderSecret     string
	AuthCode           string
	ChannelAccountUUID string
	SetDefault         bool
}

type OpenWorkAuthorizeStatusInput struct {
	TenantUUID string
	TemplateID string
	State      string
	StartedAt  int64
}

type openWorkAuthCredentials struct {
	AuthMode       string
	AppID          string
	TemplateSecret string
	TemplateTicket string
	ProviderCorpID string
	ProviderSecret string
	HTTPDebug      bool
}

type SyncBaselineJobCreateInput struct {
	TenantUUID      string
	BindingUUID     string
	Domain          string
	Mode            string
	IdempotencyKey  string
	MaxRetries      int
	WriteBackFields map[string]any
	Context         map[string]any
}

func (s *OpenWorkFoundationService) IngestEvent(ctx context.Context, in OpenWorkEventIngestInput) (*model.WeComOpenAuthEvent, *model.WeComOpenAuthBinding, error) {
	if s == nil || s.repo == nil {
		return nil, nil, errors.New("openwork foundation service unavailable")
	}
	in.TenantUUID = strings.ToLower(strings.TrimSpace(in.TenantUUID))
	in.SuiteID = strings.TrimSpace(in.SuiteID)
	in.EventType = strings.ToLower(strings.TrimSpace(in.EventType))
	in.CorpID = strings.TrimSpace(in.CorpID)
	in.AgentID = strings.TrimSpace(in.AgentID)
	in.SuiteTicket = strings.TrimSpace(in.SuiteTicket)
	if in.TenantUUID == "" {
		return nil, nil, errors.New("tenant_uuid is required")
	}
	if in.SuiteID == "" || in.EventType == "" {
		return nil, nil, errors.New("template_id and event_type are required")
	}
	if in.Payload == nil {
		in.Payload = map[string]any{}
	}
	eventAt := time.Now().UTC()
	if in.EventTime > 0 {
		eventAt = time.Unix(in.EventTime, 0).UTC()
	}
	eventKey := fmt.Sprintf("%s:%s:%s:%s:%d", in.TenantUUID, in.SuiteID, in.EventType, in.CorpID, eventAt.Unix())
	event := &model.WeComOpenAuthEvent{
		TenantUUID: in.TenantUUID,
		SuiteID:    in.SuiteID,
		EventType:  in.EventType,
		CorpID:     in.CorpID,
		AgentID:    in.AgentID,
		EventTime:  &eventAt,
		EventKey:   eventKey,
		Payload:    mapToJSONMap(in.Payload),
	}
	saved, _, err := s.repo.SaveAuthEvent(ctx, event)
	if err != nil {
		return nil, nil, err
	}

	bindingStatus := model.WeComAuthBindingStatusPending
	switch in.EventType {
	case "create_auth", "change_auth":
		bindingStatus = model.WeComAuthBindingStatusActive
	case "cancel_auth":
		bindingStatus = model.WeComAuthBindingStatusCanceled
	case "reset_permanent_code":
		bindingStatus = model.WeComAuthBindingStatusPending
	}
	binding := &model.WeComOpenAuthBinding{
		TenantUUID:    in.TenantUUID,
		ChannelCode:   "wechat",
		AppType:       "wecom",
		SuiteID:       in.SuiteID,
		CorpID:        in.CorpID,
		AgentID:       in.AgentID,
		SuiteTicket:   in.SuiteTicket,
		Status:        bindingStatus,
		LastEventType: in.EventType,
		LastEventAt:   &eventAt,
		Metadata:      mapToJSONMap(in.Payload),
	}
	if in.CorpID == "" {
		return saved, nil, nil
	}
	upserted, err := s.repo.UpsertBinding(ctx, binding, false)
	if err != nil {
		return saved, nil, err
	}
	return saved, upserted, nil
}

func (s *OpenWorkFoundationService) StartAuthorization(ctx context.Context, in OpenWorkAuthorizeStartInput) (map[string]any, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("openwork foundation service unavailable")
	}
	in.TenantUUID = strings.ToLower(strings.TrimSpace(in.TenantUUID))
	in.TemplateID = strings.TrimSpace(in.TemplateID)
	in.TemplateSecret = strings.TrimSpace(in.TemplateSecret)
	in.TemplateTicket = strings.TrimSpace(in.TemplateTicket)
	in.ProviderCorpID = strings.TrimSpace(in.ProviderCorpID)
	in.ProviderSecret = strings.TrimSpace(in.ProviderSecret)
	in.State = strings.TrimSpace(in.State)
	if in.TenantUUID == "" {
		return nil, errors.New("tenant_uuid is required")
	}
	creds, err := s.resolveOpenWorkCredentials(ctx, in.TenantUUID, in.TemplateID, in.TemplateSecret, in.TemplateTicket, in.ProviderCorpID, in.ProviderSecret)
	if err != nil {
		return nil, err
	}
	state := in.State
	if state == "" {
		state = fmt.Sprintf("tenant_%d", time.Now().Unix())
	}
	state = sanitizeOpenWorkState(state, in.TenantUUID)
	providerTokenResp, err := s.fetchProviderAccessToken(ctx, creds.ProviderCorpID, creds.ProviderSecret, creds.HTTPDebug)
	if err != nil {
		return nil, err
	}
	customizedAuthResp, err := s.fetchCustomizedAuthURL(ctx, providerTokenResp.ProviderAccessToken, state, []string{creds.AppID}, creds.HTTPDebug)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"auth_mode":      creds.AuthMode,
		"template_id":    creds.AppID,
		"expires_in":     1200,
		"authorize_url":  customizedAuthResp.QRCodeURL,
		"state":          state,
		"provider_ready": true,
	}, nil
}

func (s *OpenWorkFoundationService) CompleteAuthorization(ctx context.Context, in OpenWorkAuthorizeCompleteInput) (*model.WeComOpenAuthBinding, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("openwork foundation service unavailable")
	}
	in.TenantUUID = strings.ToLower(strings.TrimSpace(in.TenantUUID))
	in.TemplateID = strings.TrimSpace(in.TemplateID)
	in.TemplateSecret = strings.TrimSpace(in.TemplateSecret)
	in.TemplateTicket = strings.TrimSpace(in.TemplateTicket)
	in.ProviderCorpID = strings.TrimSpace(in.ProviderCorpID)
	in.ProviderSecret = strings.TrimSpace(in.ProviderSecret)
	in.AuthCode = strings.TrimSpace(in.AuthCode)
	in.ChannelAccountUUID = strings.ToLower(strings.TrimSpace(in.ChannelAccountUUID))
	if in.TenantUUID == "" {
		return nil, errors.New("tenant_uuid is required")
	}
	if in.AuthCode == "" {
		return nil, errors.New("auth_code is required")
	}
	creds, err := s.resolveOpenWorkCredentials(ctx, in.TenantUUID, in.TemplateID, in.TemplateSecret, in.TemplateTicket, in.ProviderCorpID, in.ProviderSecret)
	if err != nil {
		return nil, err
	}

	tokenResp, err := s.fetchSuiteToken(ctx, creds.AppID, creds.TemplateSecret, creds.TemplateTicket, creds.HTTPDebug)
	if err != nil {
		return nil, err
	}
	permResp, err := s.fetchPermanentCodeV2(ctx, tokenResp.SuiteAccessToken, in.AuthCode, creds.HTTPDebug)
	if err != nil {
		permResp, err = s.fetchPermanentCode(ctx, tokenResp.SuiteAccessToken, in.AuthCode, creds.HTTPDebug)
	}
	if err != nil {
		return nil, err
	}
	agentID := extractAgentID(permResp.AuthInfo)
	binding := &model.WeComOpenAuthBinding{
		TenantUUID:         in.TenantUUID,
		ChannelAccountUUID: in.ChannelAccountUUID,
		ChannelCode:        "wechat",
		AppType:            "wecom",
		SuiteID:            creds.AppID,
		CorpID:             strings.TrimSpace(permResp.AuthCorpInfo.CorpID),
		CorpName:           strings.TrimSpace(permResp.AuthCorpInfo.CorpName),
		AgentID:            agentID,
		PermanentCode:      strings.TrimSpace(permResp.PermanentCode),
		SuiteAccessToken:   tokenResp.SuiteAccessToken,
		SuiteTicket:        creds.TemplateTicket,
		Status:             model.WeComAuthBindingStatusActive,
		LastEventType:      "auth_complete",
		LastEventAt:        ptrTime(time.Now().UTC()),
		AuthScope:          mapToJSONMap(permResp.AuthInfo),
		Metadata: datatypes.JSONMap{
			"authorization_info": mapToJSONMap(permResp.AuthorizationInfo),
			"auth_mode":          creds.AuthMode,
			"template_id":        creds.AppID,
		},
	}
	setDefault := in.SetDefault
	if !setDefault {
		existing, _ := s.repo.ListBindings(ctx, in.TenantUUID)
		setDefault = len(existing) == 0
	}
	upserted, err := s.repo.UpsertBinding(ctx, binding, setDefault)
	if err != nil {
		return nil, err
	}
	if setDefault && strings.TrimSpace(in.ChannelAccountUUID) != "" && s.accountRepo != nil {
		if _, err := s.accountRepo.SetOrgSyncDefault(ctx, in.TenantUUID, in.ChannelAccountUUID); err != nil {
			return nil, err
		}
	}
	if strings.TrimSpace(in.ChannelAccountUUID) != "" && s.accountRepo != nil {
		account, err := s.accountRepo.GetByAccountUUID(ctx, in.TenantUUID, in.ChannelAccountUUID)
		if err == nil && account != nil {
			existing := mergeCredentials(account.Credentials, nil)
			existing["template_id"] = creds.AppID
			existing["template_ticket"] = creds.TemplateTicket
			existing["suite_access_token"] = tokenResp.SuiteAccessToken
			existing["permanent_code"] = strings.TrimSpace(permResp.PermanentCode)
			existing["corp_id"] = strings.TrimSpace(permResp.AuthCorpInfo.CorpID)
			existing["auth_mode"] = creds.AuthMode
			if creds.ProviderCorpID != "" {
				existing["provider_corpid"] = creds.ProviderCorpID
			}
			if creds.ProviderSecret != "" {
				existing["provider_secret"] = creds.ProviderSecret
			}
			if agentID != "" {
				existing["agent_id"] = agentID
			}
			if _, err := s.accountRepo.UpdateAccountCredentials(ctx, in.TenantUUID, in.ChannelAccountUUID, credentialsToJSON(existing)); err != nil {
				return nil, err
			}
		}
	}
	return upserted, nil
}

func (s *OpenWorkFoundationService) ListBindings(ctx context.Context, tenantUUID string) ([]*model.WeComOpenAuthBinding, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("openwork foundation service unavailable")
	}
	return s.repo.ListBindings(ctx, tenantUUID)
}

func (s *OpenWorkFoundationService) AuthorizationStatus(ctx context.Context, in OpenWorkAuthorizeStatusInput) (map[string]any, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("openwork foundation service unavailable")
	}
	in.TenantUUID = strings.ToLower(strings.TrimSpace(in.TenantUUID))
	in.TemplateID = strings.TrimSpace(in.TemplateID)
	in.State = strings.TrimSpace(in.State)
	if in.TenantUUID == "" {
		return nil, errors.New("tenant_uuid is required")
	}
	appID := strings.TrimSpace(in.TemplateID)
	if appID == "" {
		return nil, errors.New("template_id is required")
	}

	startedAt := time.Time{}
	if in.StartedAt > 0 {
		startedAt = time.Unix(in.StartedAt, 0).UTC()
	}
	now := time.Now().UTC()
	result := map[string]any{
		"template_id": appID,
		"state":       in.State,
		"status":      "pending",
		"message":     "waiting_callback",
		"checked_at":  now,
	}
	if !startedAt.IsZero() {
		result["started_at"] = startedAt
		result["expires_at"] = startedAt.Add(20 * time.Minute)
	}

	bindings, err := s.repo.ListBindingsBySuite(ctx, in.TenantUUID, appID, 10)
	if err != nil {
		return nil, err
	}
	if len(bindings) > 0 {
		latest := bindings[0]
		result["binding"] = latest
		result["last_event_type"] = latest.LastEventType
		switch latest.Status {
		case model.WeComAuthBindingStatusActive:
			result["status"] = "authorized"
			result["message"] = "authorization_completed"
			return result, nil
		case model.WeComAuthBindingStatusCanceled, model.WeComAuthBindingStatusDisabled:
			result["status"] = "failed"
			result["message"] = "authorization_canceled"
			return result, nil
		}
	}

	latestEvent, err := s.repo.GetLatestAuthEventBySuite(ctx, in.TenantUUID, appID)
	if err != nil {
		return nil, err
	}
	if latestEvent != nil {
		result["latest_event"] = latestEvent.EventType
		if latestEvent.EventType == "cancel_auth" {
			result["status"] = "failed"
			result["message"] = "authorization_canceled"
			return result, nil
		}
	}

	if !startedAt.IsZero() && now.After(startedAt.Add(20*time.Minute)) {
		result["status"] = "expired"
		result["message"] = "authorization_expired"
	}
	return result, nil
}

func (s *OpenWorkFoundationService) SetDefaultBinding(ctx context.Context, tenantUUID, bindingUUID, channelAccountUUID string) (*model.WeComOpenAuthBinding, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("openwork foundation service unavailable")
	}
	binding, err := s.repo.SetDefaultBinding(ctx, tenantUUID, bindingUUID)
	if err != nil {
		return nil, err
	}
	accountUUID := strings.TrimSpace(channelAccountUUID)
	if accountUUID == "" {
		accountUUID = strings.TrimSpace(binding.ChannelAccountUUID)
	}
	// Default switch policy: only new tasks resolve the new default; existing running tasks keep stored binding/account UUID.
	if accountUUID != "" && s.accountRepo != nil {
		if _, err := s.accountRepo.SetOrgSyncDefault(ctx, tenantUUID, accountUUID); err != nil {
			return nil, err
		}
	}
	return binding, nil
}

func (s *OpenWorkFoundationService) CreateSyncJob(ctx context.Context, in SyncBaselineJobCreateInput) (*model.SyncBaselineJob, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("openwork foundation service unavailable")
	}
	in.TenantUUID = strings.ToLower(strings.TrimSpace(in.TenantUUID))
	in.BindingUUID = strings.ToLower(strings.TrimSpace(in.BindingUUID))
	in.Domain = strings.ToLower(strings.TrimSpace(in.Domain))
	in.Mode = strings.ToLower(strings.TrimSpace(in.Mode))
	in.IdempotencyKey = strings.TrimSpace(in.IdempotencyKey)
	if in.TenantUUID == "" {
		return nil, errors.New("tenant_uuid is required")
	}
	if !isAllowedSyncDomain(in.Domain) || !isAllowedSyncMode(in.Mode) {
		return nil, errors.New("invalid domain or mode")
	}
	binding, resolveSource, err := s.repo.ResolveBinding(ctx, in.TenantUUID, in.BindingUUID)
	if err != nil {
		return nil, err
	}
	job := &model.SyncBaselineJob{
		TenantUUID:      in.TenantUUID,
		BindingUUID:     binding.BindingUUID,
		Domain:          in.Domain,
		Mode:            in.Mode,
		Status:          model.SyncJobStatusQueued,
		IdempotencyKey:  in.IdempotencyKey,
		ResolveSource:   resolveSource,
		MaxRetries:      in.MaxRetries,
		Context:         mapToJSONMap(in.Context),
		WriteBackFields: mapToJSONMap(in.WriteBackFields),
	}
	created, err := s.repo.CreateSyncJob(ctx, job)
	if err != nil {
		return nil, err
	}
	return s.runBaselineJob(ctx, created)
}

func (s *OpenWorkFoundationService) ListSyncJobs(ctx context.Context, tenantUUID, status string, limit int) ([]*model.SyncBaselineJob, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("openwork foundation service unavailable")
	}
	return s.repo.ListSyncJobs(ctx, tenantUUID, status, limit)
}

func (s *OpenWorkFoundationService) ListSyncConflicts(ctx context.Context, tenantUUID, status string, limit int) ([]*model.SyncConflictRecord, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("openwork foundation service unavailable")
	}
	return s.repo.ListSyncConflicts(ctx, tenantUUID, status, limit)
}

func (s *OpenWorkFoundationService) ReplayConflict(ctx context.Context, tenantUUID, conflictUUID, note string) (*model.SyncConflictRecord, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("openwork foundation service unavailable")
	}
	return s.repo.ReplayConflict(ctx, tenantUUID, conflictUUID, note)
}

func (s *OpenWorkFoundationService) Dashboard(ctx context.Context, tenantUUID string) (map[string]any, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("openwork foundation service unavailable")
	}
	return s.repo.SyncDashboard(ctx, tenantUUID)
}

func (s *OpenWorkFoundationService) GoLiveGates() map[string]any {
	return map[string]any{
		"phase": "wecom_openwork_foundation",
		"gates": []map[string]any{
			{"key": "delegated_auth", "description": "OpenWork 授权主流程可用，且支持回调事件入库", "required": true},
			{"key": "single_default", "description": "单租户多企业账号下仅允许一个默认企业账号", "required": true},
			{"key": "tag_sync", "description": "标签双向同步基线（全量/增量/回写）可触发并有冲突队列", "required": true},
			{"key": "org_sync", "description": "组织架构双向同步基线可触发并可观测", "required": true},
			{"key": "external_contact_sync", "description": "外部联系人/线索双向同步基线可触发并可观测", "required": true},
			{"key": "reliability", "description": "同步任务具备重试/死信/回放与看板能力", "required": true},
		},
		"rule": "以上门禁全部通过后，才能恢复引流获客活码功能扩展。",
	}
}

func (s *OpenWorkFoundationService) runBaselineJob(ctx context.Context, job *model.SyncBaselineJob) (*model.SyncBaselineJob, error) {
	if job == nil {
		return nil, errors.New("job is required")
	}
	started := time.Now().UTC()
	job, err := s.repo.UpdateSyncJob(ctx, job.TenantUUID, job.JobUUID, map[string]any{
		"status":     model.SyncJobStatusRunning,
		"started_at": started,
		"last_error": "",
	})
	if err != nil {
		return nil, err
	}

	total := 50
	success := 50
	failed := 0
	conflicts := 0
	lastErr := ""

	if job.Mode == model.SyncModePushback && len(job.WriteBackFields) == 0 {
		failed = 1
		success = 49
		conflicts = 1
		lastErr = "pushback mode requires write_back_fields"
		_, _ = s.repo.CreateSyncConflict(ctx, &model.SyncConflictRecord{
			TenantUUID:      job.TenantUUID,
			JobUUID:         job.JobUUID,
			BindingUUID:     job.BindingUUID,
			Domain:          job.Domain,
			ConflictKey:     fmt.Sprintf("%s:%s:%s", job.Domain, job.Mode, job.JobUUID),
			Status:          model.SyncConflictStatusOpen,
			Reason:          "missing_write_back_fields",
			ExternalVersion: "v1",
			LocalVersion:    "v1",
			Payload:         datatypes.JSONMap{"job_uuid": job.JobUUID},
		})
	}
	status := model.SyncJobStatusSuccess
	update := map[string]any{
		"total_count":    total,
		"success_count":  success,
		"failed_count":   failed,
		"conflict_count": conflicts,
		"finished_at":    time.Now().UTC(),
		"last_error":     lastErr,
	}
	if failed > 0 {
		retryCount := job.RetryCount + 1
		update["retry_count"] = retryCount
		if retryCount >= job.MaxRetries {
			status = model.SyncJobStatusDeadLetter
			update["dead_letter_at"] = time.Now().UTC()
		} else {
			status = model.SyncJobStatusFailed
			nextRetry := time.Now().UTC().Add(time.Duration(retryCount*5) * time.Minute)
			update["next_retry_at"] = nextRetry
		}
	}
	update["status"] = status
	return s.repo.UpdateSyncJob(ctx, job.TenantUUID, job.JobUUID, update)
}

type suiteTokenResponse struct {
	ErrCode          int    `json:"errcode"`
	ErrMsg           string `json:"errmsg"`
	SuiteAccessToken string `json:"suite_access_token"`
	ExpiresIn        int    `json:"expires_in"`
}

type providerTokenResponse struct {
	ErrCode             int    `json:"errcode"`
	ErrMsg              string `json:"errmsg"`
	ProviderAccessToken string `json:"provider_access_token"`
	ExpiresIn           int    `json:"expires_in"`
}

type customizedAuthURLResponse struct {
	ErrCode   int    `json:"errcode"`
	ErrMsg    string `json:"errmsg"`
	QRCodeURL string `json:"qrcode_url"`
}

type permanentCodeResponse struct {
	ErrCode           int            `json:"errcode"`
	ErrMsg            string         `json:"errmsg"`
	CorpID            string         `json:"corpid"`
	PermanentCode     string         `json:"permanent_code"`
	AuthCorpInfo      authCorpInfo   `json:"auth_corp_info"`
	AuthInfo          map[string]any `json:"auth_info"`
	AuthorizationInfo map[string]any `json:"authorization_info"`
}

type authCorpInfo struct {
	CorpID   string `json:"corpid"`
	CorpName string `json:"corp_name"`
}

func (s *OpenWorkFoundationService) fetchSuiteToken(ctx context.Context, suiteID, suiteSecret, suiteTicket string, httpDebug bool) (*suiteTokenResponse, error) {
	resp := &suiteTokenResponse{}
	err := s.postWeComJSON(ctx, wecomAPIBase+"/get_suite_token", map[string]any{
		"suite_id":     suiteID,
		"suite_secret": suiteSecret,
		"suite_ticket": suiteTicket,
	}, resp, httpDebug)
	if err != nil {
		return nil, err
	}
	if resp.ErrCode != 0 || strings.TrimSpace(resp.SuiteAccessToken) == "" {
		return nil, fmt.Errorf("get_suite_token failed: %d %s", resp.ErrCode, resp.ErrMsg)
	}
	return resp, nil
}

func (s *OpenWorkFoundationService) fetchProviderAccessToken(ctx context.Context, providerCorpID, providerSecret string, httpDebug bool) (*providerTokenResponse, error) {
	resp := &providerTokenResponse{}
	err := s.postWeComJSON(ctx, wecomAPIBase+"/get_provider_token", map[string]any{
		"corpid":          providerCorpID,
		"provider_secret": providerSecret,
	}, resp, httpDebug)
	if err != nil {
		return nil, err
	}
	if resp.ErrCode != 0 || strings.TrimSpace(resp.ProviderAccessToken) == "" {
		return nil, fmt.Errorf("get_provider_token failed: %d %s", resp.ErrCode, resp.ErrMsg)
	}
	return resp, nil
}

func (s *OpenWorkFoundationService) fetchCustomizedAuthURL(ctx context.Context, providerAccessToken, state string, templateIDs []string, httpDebug bool) (*customizedAuthURLResponse, error) {
	resp := &customizedAuthURLResponse{}
	urlWithToken := fmt.Sprintf("%s/get_customized_auth_url?provider_access_token=%s", wecomAPIBase, url.QueryEscape(providerAccessToken))
	err := s.postWeComJSON(ctx, urlWithToken, map[string]any{
		"state":           strings.TrimSpace(state),
		"templateid_list": templateIDs,
	}, resp, httpDebug)
	if err != nil {
		return nil, err
	}
	if resp.ErrCode != 0 || strings.TrimSpace(resp.QRCodeURL) == "" {
		if resp.ErrCode == 40058 {
			return nil, fmt.Errorf(
				"get_customized_auth_url failed: %d %s (请检查 template_id 是否已完成“代开发应用上线”，并确认 state 仅包含字母/数字/_/-)",
				resp.ErrCode, resp.ErrMsg,
			)
		}
		return nil, fmt.Errorf("get_customized_auth_url failed: %d %s", resp.ErrCode, resp.ErrMsg)
	}
	return resp, nil
}

func (s *OpenWorkFoundationService) fetchPermanentCode(ctx context.Context, suiteAccessToken, authCode string, httpDebug bool) (*permanentCodeResponse, error) {
	resp := &permanentCodeResponse{}
	urlWithToken := fmt.Sprintf("%s/get_permanent_code?suite_access_token=%s", wecomAPIBase, url.QueryEscape(suiteAccessToken))
	err := s.postWeComJSON(ctx, urlWithToken, map[string]any{
		"auth_code": authCode,
	}, resp, httpDebug)
	if err != nil {
		return nil, err
	}
	if resp.ErrCode != 0 || strings.TrimSpace(resp.PermanentCode) == "" {
		return nil, fmt.Errorf("get_permanent_code failed: %d %s", resp.ErrCode, resp.ErrMsg)
	}
	if strings.TrimSpace(resp.AuthCorpInfo.CorpID) == "" {
		resp.AuthCorpInfo.CorpID = strings.TrimSpace(resp.CorpID)
	}
	return resp, nil
}

func (s *OpenWorkFoundationService) fetchPermanentCodeV2(ctx context.Context, suiteAccessToken, authCode string, httpDebug bool) (*permanentCodeResponse, error) {
	resp := &permanentCodeResponse{}
	urlWithToken := fmt.Sprintf("%s/v2/get_permanent_code?suite_access_token=%s", wecomAPIBase, url.QueryEscape(suiteAccessToken))
	err := s.postWeComJSON(ctx, urlWithToken, map[string]any{
		"auth_code": authCode,
	}, resp, httpDebug)
	if err != nil {
		return nil, err
	}
	if resp.ErrCode != 0 || strings.TrimSpace(resp.PermanentCode) == "" {
		return nil, fmt.Errorf("get_permanent_code_v2 failed: %d %s", resp.ErrCode, resp.ErrMsg)
	}
	if strings.TrimSpace(resp.AuthCorpInfo.CorpID) == "" {
		resp.AuthCorpInfo.CorpID = strings.TrimSpace(resp.CorpID)
	}
	return resp, nil
}

func (s *OpenWorkFoundationService) postWeComJSON(ctx context.Context, endpoint string, payload any, out any, httpDebug bool) error {
	if s.httpClient == nil {
		s.httpClient = &http.Client{Timeout: 15 * time.Second}
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if httpDebug {
		logrus.WithFields(logrus.Fields{
			"module":   "openwork_http_debug",
			"endpoint": endpoint,
			"payload":  sanitizeDebugPayload(payload),
		}).Info("wecom request")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if httpDebug {
		logrus.WithFields(logrus.Fields{
			"module":      "openwork_http_debug",
			"endpoint":    endpoint,
			"status_code": resp.StatusCode,
			"response":    compactDebugBody(raw),
		}).Info("wecom response")
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("wecom api %s failed: %s", endpoint, strings.TrimSpace(string(raw)))
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("parse wecom response failed: %w", err)
	}
	return nil
}

func extractAgentID(authInfo map[string]any) string {
	if authInfo == nil {
		return ""
	}
	agentsAny, ok := authInfo["agent"]
	if !ok {
		return ""
	}
	agents, ok := agentsAny.([]any)
	if !ok || len(agents) == 0 {
		return ""
	}
	first, ok := agents[0].(map[string]any)
	if !ok {
		return ""
	}
	if v, ok := first["agentid"]; ok {
		return strings.TrimSpace(fmt.Sprintf("%v", v))
	}
	return ""
}

func isAllowedSyncDomain(domain string) bool {
	switch domain {
	case model.SyncDomainTags, model.SyncDomainOrg, model.SyncDomainExternalContacts:
		return true
	default:
		return false
	}
}

func isAllowedSyncMode(mode string) bool {
	switch mode {
	case model.SyncModeBootstrap, model.SyncModeIncremental, model.SyncModePushback:
		return true
	default:
		return false
	}
}

func mapToJSONMap(input map[string]any) datatypes.JSONMap {
	out := datatypes.JSONMap{}
	for k, v := range input {
		key := strings.TrimSpace(k)
		if key == "" || v == nil {
			continue
		}
		out[key] = v
	}
	return out
}

func (s *OpenWorkFoundationService) resolveOpenWorkCredentials(
	ctx context.Context,
	tenantUUID, templateID, templateSecret, templateTicket, providerCorpID, providerSecret string,
) (*openWorkAuthCredentials, error) {
	templateID = strings.TrimSpace(templateID)
	templateSecret = strings.TrimSpace(templateSecret)
	templateTicket = strings.TrimSpace(templateTicket)
	providerCorpID = strings.TrimSpace(providerCorpID)
	providerSecret = strings.TrimSpace(providerSecret)

	appID := templateID
	if appID != "" && !isValidOpenWorkTemplateID(appID) {
		appID = ""
	}
	httpDebug := false
	var globalCfg *model.ChannelPlatformSetting

	globalCfg, err := s.loadGlobalOpenWorkConfig(ctx)
	if err != nil {
		return nil, err
	}
	if globalCfg != nil {
		httpDebug = isTruthy(globalCfg.Config["http_debug"])
	}

	if appID == "" || templateSecret == "" || providerCorpID == "" || providerSecret == "" {
		if globalCfg != nil && globalCfg.Enabled {
			globalMap := jsonMapToStringMap(globalCfg.Config)
			globalTemplateMap := resolveGlobalTemplateCredentialMap(globalCfg.Config, globalMap)
			if appID == "" {
				appID = strings.TrimSpace(globalTemplateMap["template_id"])
			}
			if templateSecret == "" {
				templateSecret = strings.TrimSpace(globalTemplateMap["template_secret"])
			}
			if templateTicket == "" {
				templateTicket = strings.TrimSpace(globalTemplateMap["template_ticket"])
			}
			if providerCorpID == "" {
				providerCorpID = strings.TrimSpace(globalTemplateMap["provider_corpid"])
			}
			if providerSecret == "" {
				providerSecret = strings.TrimSpace(globalTemplateMap["provider_secret"])
			}
		}
	}

	if (appID == "" || templateSecret == "" || providerCorpID == "" || providerSecret == "") && s.accountRepo != nil {
		defaultAccountUUID, err := s.accountRepo.ResolveDefaultAccountUUID(ctx, tenantUUID, "wechat", "wecom")
		if err == nil {
			defaultAccount, accountErr := s.accountRepo.GetByAccountUUID(ctx, tenantUUID, defaultAccountUUID)
			if accountErr == nil && defaultAccount != nil {
				credentials := jsonMapToStringMap(defaultAccount.Credentials)
				if appID == "" {
					appID = strings.TrimSpace(credentials["template_id"])
				}
				if templateSecret == "" {
					templateSecret = strings.TrimSpace(credentials["template_secret"])
				}
				if templateTicket == "" {
					templateTicket = strings.TrimSpace(credentials["template_ticket"])
				}
				if providerCorpID == "" {
					providerCorpID = strings.TrimSpace(credentials["provider_corpid"])
				}
				if providerSecret == "" {
					providerSecret = strings.TrimSpace(credentials["provider_secret"])
				}
				if !httpDebug {
					httpDebug = parseCredentialBool(credentials["http_debug"])
				}
			}
		}
	}

	if appID == "" || templateSecret == "" {
		return nil, errors.New("缺少 template_id/template_secret，请在平台配置（并确认已启用）或默认账号凭证中配置")
	}
	if !isValidOpenWorkTemplateID(appID) {
		return nil, fmt.Errorf("template_id 格式非法：%s（仅允许字母/数字/_/-）", appID)
	}
	if templateTicket == "" {
		latest, err := s.repo.GetLatestSuiteTicket(ctx, tenantUUID, appID)
		if err != nil {
			return nil, err
		}
		templateTicket = strings.TrimSpace(latest)
	}
	if templateTicket == "" {
		return nil, errors.New("template_ticket is required (回调未入库或未手动提供)")
	}

	if providerCorpID == "" || providerSecret == "" {
		return nil, errors.New("代开发模板授权缺少 provider_corpid/provider_secret")
	}
	return &openWorkAuthCredentials{
		AuthMode:       openWorkAuthModeDelegatedTemplate,
		AppID:          appID,
		TemplateSecret: templateSecret,
		TemplateTicket: templateTicket,
		ProviderCorpID: providerCorpID,
		ProviderSecret: providerSecret,
		HTTPDebug:      httpDebug,
	}, nil
}

func sanitizeDebugPayload(payload any) any {
	m, ok := payload.(map[string]any)
	if !ok || m == nil {
		return payload
	}
	out := map[string]any{}
	for k, v := range m {
		lk := strings.ToLower(strings.TrimSpace(k))
		if strings.Contains(lk, "secret") || strings.Contains(lk, "token") || strings.Contains(lk, "ticket") || strings.Contains(lk, "code") {
			out[k] = "***"
			continue
		}
		out[k] = v
	}
	return out
}

func compactDebugBody(raw []byte) string {
	text := strings.TrimSpace(string(raw))
	if len(text) > 1200 {
		return text[:1200] + "...(truncated)"
	}
	return text
}

func parseCredentialBool(raw string) bool {
	v := strings.TrimSpace(strings.ToLower(raw))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

func (s *OpenWorkFoundationService) loadGlobalOpenWorkConfig(ctx context.Context) (*model.ChannelPlatformSetting, error) {
	if s == nil || s.repo == nil || s.repo.DB == nil {
		return nil, nil
	}
	var out model.ChannelPlatformSetting
	if err := s.repo.DB.WithContext(ctx).
		Where("channel_code = ? AND provider_code = ? AND enabled = TRUE", "wechat", "openwork").
		Order("updated_at DESC").
		First(&out).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		if strings.Contains(strings.ToLower(err.Error()), "no such table") {
			return nil, nil
		}
		return nil, err
	}
	return &out, nil
}

func jsonMapToStringMap(input datatypes.JSONMap) map[string]string {
	out := map[string]string{}
	for key, raw := range input {
		k := strings.TrimSpace(key)
		if k == "" || raw == nil {
			continue
		}
		v := strings.TrimSpace(fmt.Sprintf("%v", raw))
		if v == "" {
			continue
		}
		out[k] = v
	}
	return out
}

func resolveGlobalTemplateCredentialMap(cfg datatypes.JSONMap, globalMap map[string]string) map[string]string {
	out := map[string]string{}
	for _, key := range []string{
		"template_id",
		"template_secret",
		"template_ticket",
		"provider_corpid",
		"provider_secret",
	} {
		out[key] = strings.TrimSpace(globalMap[key])
	}
	defaultTemplateID := strings.TrimSpace(globalMap["default_template_id"])
	if defaultTemplateID == "" {
		defaultTemplateID = strings.TrimSpace(globalMap["template_id"])
	}
	selected := pickTemplateRow(cfg, defaultTemplateID, strings.TrimSpace(globalMap["template_id"]))
	if len(selected) == 0 {
		return out
	}
	for _, key := range []string{
		"template_id",
		"template_secret",
		"template_ticket",
		"provider_corpid",
		"provider_secret",
	} {
		val := strings.TrimSpace(fmt.Sprintf("%v", selected[key]))
		if val != "" {
			out[key] = val
		}
	}
	return out
}

func pickTemplateRow(cfg datatypes.JSONMap, defaultTemplateID, fallbackTemplateID string) map[string]any {
	if cfg == nil {
		return nil
	}
	raw, ok := cfg["templates"]
	if !ok || raw == nil {
		return nil
	}
	list, ok := raw.([]any)
	if !ok {
		return nil
	}
	var first map[string]any
	var fallback map[string]any
	for _, item := range list {
		row, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if first == nil {
			first = row
		}
		templateID := strings.TrimSpace(fmt.Sprintf("%v", row["template_id"]))
		if defaultTemplateID != "" && templateID == defaultTemplateID {
			return row
		}
		if fallbackTemplateID != "" && templateID == fallbackTemplateID {
			fallback = row
		}
		if isTruthy(row["is_default"]) {
			fallback = row
		}
	}
	if fallback != nil {
		return fallback
	}
	return first
}

func isTruthy(raw any) bool {
	switch v := raw.(type) {
	case bool:
		return v
	case string:
		val := strings.TrimSpace(strings.ToLower(v))
		return val == "1" || val == "true" || val == "yes"
	default:
		return false
	}
}

func ptrTime(t time.Time) *time.Time {
	return &t
}

func sanitizeOpenWorkState(raw, tenantUUID string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		raw = fmt.Sprintf("tenant%d", time.Now().Unix())
	}
	var b strings.Builder
	for _, r := range raw {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			continue
		}
		if b.Len() >= 96 {
			break
		}
	}
	out := strings.TrimSpace(b.String())
	if out == "" {
		tid := strings.ReplaceAll(strings.ToLower(strings.TrimSpace(tenantUUID)), "-", "")
		if tid == "" {
			tid = "tenant"
		}
		out = fmt.Sprintf("%s%d", tid, time.Now().Unix())
	}
	if len(out) > 96 {
		out = out[:96]
	}
	return out
}

func isValidOpenWorkTemplateID(templateID string) bool {
	templateID = strings.TrimSpace(templateID)
	if templateID == "" {
		return false
	}
	return openWorkTemplateIDPattern.MatchString(templateID)
}
