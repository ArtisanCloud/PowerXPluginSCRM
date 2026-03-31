package social_channel_governance

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	basemodels "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	model "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/social_channel_governance"
	socialrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/social_channel_governance"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type fakeRoundTripper struct{}

func (fakeRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	path := req.URL.Path
	query := req.URL.RawQuery
	var body string
	switch {
	case strings.HasSuffix(path, "/get_provider_token"):
		body = `{"errcode":0,"errmsg":"ok","provider_access_token":"provider-token-001","expires_in":7200}`
	case strings.HasSuffix(path, "/get_customized_auth_url") && strings.Contains(query, "provider_access_token=provider-token-001"):
		body = `{"errcode":0,"errmsg":"ok","qrcode_url":"https://open.work.weixin.qq.com/3rdapp/install?suite_id=dk001"}`
	case strings.HasSuffix(path, "/get_suite_token"):
		body = `{"errcode":0,"errmsg":"ok","suite_access_token":"suite-token-001","expires_in":7200}`
	case strings.HasSuffix(path, "/get_permanent_code") && strings.Contains(query, "suite_access_token=suite-token-001"):
		body = `{
			"errcode":0,
			"errmsg":"ok",
			"corpid":"wwcorp001",
			"permanent_code":"perm-code-001",
			"auth_corp_info":{"corpid":"wwcorp001","corp_name":"Demo Corp"},
			"auth_info":{"agent":[{"agentid":1000002}]},
			"authorization_info":{"auth_user_info":{"userid":"admin001"}}
		}`
	default:
		body = `{"errcode":40001,"errmsg":"unknown endpoint"}`
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    req,
	}, nil
}

func TestOpenWorkFoundationService_StartAuthorization(t *testing.T) {
	db := openOpenWorkServiceTestDB(t, "openwork_start_authorize")
	repo := socialrepo.NewOpenWorkFoundationRepository(db)
	svc := NewOpenWorkFoundationService(repo, socialrepo.NewAccountRepository(db))
	svc.httpClient = &http.Client{Transport: fakeRoundTripper{}, Timeout: 5 * time.Second}

	resp, err := svc.StartAuthorization(context.Background(), OpenWorkAuthorizeStartInput{
		TenantUUID:     "00000000-0000-0000-0000-000000000001",
		TemplateID:     "dk001",
		TemplateSecret: "suite-secret-001",
		TemplateTicket: "ticket-001",
		ProviderCorpID: "ww-provider-001",
		ProviderSecret: "provider-secret-001",
		State:          "state-001",
	})
	require.NoError(t, err)
	require.Equal(t, "delegated_template", resp["auth_mode"])
	require.Equal(t, "dk001", resp["template_id"])
	require.Equal(t, "https://open.work.weixin.qq.com/3rdapp/install?suite_id=dk001", resp["authorize_url"])
}

func TestOpenWorkFoundationService_CompleteAuthorizationAndSwitchDefault(t *testing.T) {
	db := openOpenWorkServiceTestDB(t, "openwork_complete_authorize")
	repo := socialrepo.NewOpenWorkFoundationRepository(db)
	accountRepo := socialrepo.NewAccountRepository(db)
	svc := NewOpenWorkFoundationService(repo, accountRepo)
	svc.httpClient = &http.Client{Transport: fakeRoundTripper{}, Timeout: 5 * time.Second}

	tenantUUID := "00000000-0000-0000-0000-000000000002"
	accountA := &model.ChannelAccount{
		AccountUUID:     "11111111-1111-4111-8111-111111111111",
		TenantUuid:      tenantUUID,
		ChannelCode:     "wechat",
		AppType:         "wecom",
		AccountID:       "1000002",
		DisplayName:     "WeCom A",
		Status:          model.ChannelAccountStatusConnected,
		OwnerMemberUUID: "owner-001",
		MemberUserUUIDs: []string{},
		OrgSyncDefault:  false,
		Capabilities:    map[string]any{},
		Credentials:     map[string]any{},
	}
	accountB := &model.ChannelAccount{
		AccountUUID:     "22222222-2222-4222-8222-222222222222",
		TenantUuid:      tenantUUID,
		ChannelCode:     "wechat",
		AppType:         "wecom",
		AccountID:       "1000003",
		DisplayName:     "WeCom B",
		Status:          model.ChannelAccountStatusConnected,
		OwnerMemberUUID: "owner-001",
		MemberUserUUIDs: []string{},
		OrgSyncDefault:  true,
		Capabilities:    map[string]any{},
		Credentials:     map[string]any{},
	}
	require.NoError(t, db.Create(accountA).Error)
	require.NoError(t, db.Create(accountB).Error)

	binding, err := svc.CompleteAuthorization(context.Background(), OpenWorkAuthorizeCompleteInput{
		TenantUUID:         tenantUUID,
		TemplateID:         "dk001",
		TemplateSecret:     "suite-secret-001",
		TemplateTicket:     "ticket-001",
		ProviderCorpID:     "ww-provider-001",
		ProviderSecret:     "provider-secret-001",
		AuthCode:           "auth-code-001",
		ChannelAccountUUID: accountA.AccountUUID,
		SetDefault:         true,
	})
	require.NoError(t, err)
	require.Equal(t, "wwcorp001", binding.CorpID)
	require.True(t, binding.IsDefault)

	updatedA, err := accountRepo.GetByAccountUUID(context.Background(), tenantUUID, accountA.AccountUUID)
	require.NoError(t, err)
	require.True(t, updatedA.OrgSyncDefault)
	require.Equal(t, "dk001", strings.TrimSpace(updatedA.Credentials["template_id"].(string)))
	require.Equal(t, "perm-code-001", strings.TrimSpace(updatedA.Credentials["permanent_code"].(string)))

	updatedB, err := accountRepo.GetByAccountUUID(context.Background(), tenantUUID, accountB.AccountUUID)
	require.NoError(t, err)
	require.False(t, updatedB.OrgSyncDefault)
}

func TestOpenWorkFoundationService_DefaultSwitchPolicyForNewJobsOnly(t *testing.T) {
	db := openOpenWorkServiceTestDB(t, "openwork_default_switch_policy")
	repo := socialrepo.NewOpenWorkFoundationRepository(db)
	svc := NewOpenWorkFoundationService(repo, socialrepo.NewAccountRepository(db))

	tenantUUID := "00000000-0000-0000-0000-000000000003"
	now := time.Now().UTC()
	bindingA := &model.WeComOpenAuthBinding{
		BindingUUID: "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa",
		TenantUUID:  tenantUUID,
		ChannelCode: "wechat",
		AppType:     "wecom",
		SuiteID:     "suite-001",
		CorpID:      "corp-a",
		AgentID:     "10001",
		Status:      model.WeComAuthBindingStatusActive,
		IsDefault:   true,
		UpdatedAt:   now,
	}
	bindingB := &model.WeComOpenAuthBinding{
		BindingUUID: "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb",
		TenantUUID:  tenantUUID,
		ChannelCode: "wechat",
		AppType:     "wecom",
		SuiteID:     "suite-001",
		CorpID:      "corp-b",
		AgentID:     "10002",
		Status:      model.WeComAuthBindingStatusActive,
		IsDefault:   false,
		UpdatedAt:   now,
	}
	require.NoError(t, db.Create(bindingA).Error)
	require.NoError(t, db.Create(bindingB).Error)

	oldJob := &model.SyncBaselineJob{
		JobUUID:       "cccccccc-cccc-4ccc-8ccc-cccccccccccc",
		TenantUUID:    tenantUUID,
		BindingUUID:   bindingA.BindingUUID,
		Domain:        model.SyncDomainTags,
		Mode:          model.SyncModeIncremental,
		Status:        model.SyncJobStatusRunning,
		ResolveSource: "default",
	}
	require.NoError(t, db.Create(oldJob).Error)

	_, err := svc.SetDefaultBinding(context.Background(), tenantUUID, bindingB.BindingUUID, "")
	require.NoError(t, err)

	created, err := svc.CreateSyncJob(context.Background(), SyncBaselineJobCreateInput{
		TenantUUID: tenantUUID,
		Domain:     model.SyncDomainOrg,
		Mode:       model.SyncModeBootstrap,
		MaxRetries: 2,
	})
	require.NoError(t, err)
	require.Equal(t, bindingB.BindingUUID, created.BindingUUID)
	require.Equal(t, "default", created.ResolveSource)

	var stillOld model.SyncBaselineJob
	require.NoError(t, db.Where("job_uuid = ?", oldJob.JobUUID).First(&stillOld).Error)
	require.Equal(t, bindingA.BindingUUID, stillOld.BindingUUID)
}

func TestOpenWorkFoundationService_DeadLetterDashboardAndReplay(t *testing.T) {
	db := openOpenWorkServiceTestDB(t, "openwork_deadletter_replay")
	repo := socialrepo.NewOpenWorkFoundationRepository(db)
	svc := NewOpenWorkFoundationService(repo, socialrepo.NewAccountRepository(db))

	tenantUUID := "00000000-0000-0000-0000-000000000004"
	binding := &model.WeComOpenAuthBinding{
		BindingUUID: "dddddddd-dddd-4ddd-8ddd-dddddddddddd",
		TenantUUID:  tenantUUID,
		ChannelCode: "wechat",
		AppType:     "wecom",
		SuiteID:     "suite-001",
		CorpID:      "corp-d",
		AgentID:     "10003",
		Status:      model.WeComAuthBindingStatusActive,
		IsDefault:   true,
	}
	require.NoError(t, db.Create(binding).Error)

	job, err := svc.CreateSyncJob(context.Background(), SyncBaselineJobCreateInput{
		TenantUUID: tenantUUID,
		Domain:     model.SyncDomainExternalContacts,
		Mode:       model.SyncModePushback,
		MaxRetries: 1,
	})
	require.NoError(t, err)
	require.Equal(t, model.SyncJobStatusDeadLetter, job.Status)

	conflicts, err := svc.ListSyncConflicts(context.Background(), tenantUUID, model.SyncConflictStatusOpen, 20)
	require.NoError(t, err)
	require.Len(t, conflicts, 1)

	replayed, err := svc.ReplayConflict(context.Background(), tenantUUID, conflicts[0].ConflictUUID, "manual replay")
	require.NoError(t, err)
	require.Equal(t, model.SyncConflictStatusReplayed, replayed.Status)
	require.Equal(t, 1, replayed.ReplayCount)

	dashboard, err := svc.Dashboard(context.Background(), tenantUUID)
	require.NoError(t, err)
	jobs := dashboard["jobs"].(map[string]any)
	require.EqualValues(t, 1, jobs["dead_letter"])
	require.EqualValues(t, 0, dashboard["open_conflicts"])
}

func TestOpenWorkFoundationService_AuthorizationStatus(t *testing.T) {
	db := openOpenWorkServiceTestDB(t, "openwork_authorization_status")
	repo := socialrepo.NewOpenWorkFoundationRepository(db)
	svc := NewOpenWorkFoundationService(repo, socialrepo.NewAccountRepository(db))

	tenantUUID := "00000000-0000-0000-0000-000000000006"
	startedAt := time.Now().UTC().Add(-30 * time.Second).Unix()
	pending, err := svc.AuthorizationStatus(context.Background(), OpenWorkAuthorizeStatusInput{
		TenantUUID: tenantUUID,
		TemplateID: "suite-001",
		State:      "state-001",
		StartedAt:  startedAt,
	})
	require.NoError(t, err)
	require.Equal(t, "pending", pending["status"])

	require.NoError(t, db.Create(&model.WeComOpenAuthBinding{
		BindingUUID: "4f091f47-e3a5-4ce8-a03f-4493e26f0036",
		TenantUUID:  tenantUUID,
		ChannelCode: "wechat",
		AppType:     "wecom",
		SuiteID:     "suite-001",
		CorpID:      "corp-f",
		AgentID:     "10006",
		Status:      model.WeComAuthBindingStatusActive,
		IsDefault:   true,
	}).Error)
	success, err := svc.AuthorizationStatus(context.Background(), OpenWorkAuthorizeStatusInput{
		TenantUUID: tenantUUID,
		TemplateID: "suite-001",
		StartedAt:  startedAt,
	})
	require.NoError(t, err)
	require.Equal(t, "authorized", success["status"])

	failedTenantUUID := "00000000-0000-0000-0000-000000000007"
	require.NoError(t, db.Create(&model.WeComOpenAuthBinding{
		BindingUUID: "f4fcf6be-cfa5-4420-a26e-0f17a4fd66a7",
		TenantUUID:  failedTenantUUID,
		ChannelCode: "wechat",
		AppType:     "wecom",
		SuiteID:     "suite-001",
		CorpID:      "corp-g",
		AgentID:     "10007",
		Status:      model.WeComAuthBindingStatusCanceled,
		IsDefault:   false,
	}).Error)
	failed, err := svc.AuthorizationStatus(context.Background(), OpenWorkAuthorizeStatusInput{
		TenantUUID: failedTenantUUID,
		TemplateID: "suite-001",
		StartedAt:  startedAt,
	})
	require.NoError(t, err)
	require.Equal(t, "failed", failed["status"])
}

func openOpenWorkServiceTestDB(t *testing.T, name string) *gorm.DB {
	t.Helper()
	basemodels.ForceSchemaForTests("")
	db, err := gorm.Open(sqlite.Open("file:"+name+"?mode=memory&cache=shared"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	require.NoError(t, err)
	require.NoError(t, ensureOpenWorkServiceTables(db))
	return db
}

func ensureOpenWorkServiceTables(db *gorm.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS social_channel_accounts (
			account_uuid TEXT PRIMARY KEY,
			tenant_uuid TEXT NOT NULL,
			channel_code TEXT NOT NULL,
			app_type TEXT NOT NULL,
			account_id TEXT NOT NULL,
			display_name TEXT NOT NULL,
			status TEXT NOT NULL,
			org_sync_default BOOLEAN NOT NULL DEFAULT FALSE,
			owner_member_uuid TEXT NOT NULL,
			member_user_uuids TEXT,
			capabilities TEXT,
			credentials TEXT,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		);`,
		`CREATE TABLE IF NOT EXISTS social_wecom_auth_bindings (
			binding_uuid TEXT PRIMARY KEY,
			tenant_uuid TEXT NOT NULL,
			channel_account_uuid TEXT,
			channel_code TEXT NOT NULL,
			app_type TEXT NOT NULL,
			suite_id TEXT NOT NULL,
			corp_id TEXT NOT NULL,
			agent_id TEXT NOT NULL,
			corp_name TEXT NOT NULL DEFAULT '',
			permanent_code TEXT NOT NULL DEFAULT '',
			suite_access_token TEXT NOT NULL DEFAULT '',
			suite_ticket TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL,
			is_default BOOLEAN NOT NULL DEFAULT FALSE,
			default_switched_at DATETIME,
			last_event_type TEXT NOT NULL DEFAULT '',
			last_event_at DATETIME,
			auth_scope TEXT,
			metadata TEXT,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		);`,
		`CREATE UNIQUE INDEX IF NOT EXISTS uq_social_wecom_auth_binding_identity
			ON social_wecom_auth_bindings (tenant_uuid, corp_id, agent_id);`,
		`CREATE TABLE IF NOT EXISTS social_wecom_auth_events (
			event_uuid TEXT PRIMARY KEY,
			tenant_uuid TEXT NOT NULL,
			suite_id TEXT NOT NULL,
			event_type TEXT NOT NULL,
			corp_id TEXT NOT NULL DEFAULT '',
			agent_id TEXT NOT NULL DEFAULT '',
			event_time DATETIME,
			event_key TEXT NOT NULL,
			payload TEXT,
			created_at DATETIME
		);`,
		`CREATE UNIQUE INDEX IF NOT EXISTS uq_social_wecom_auth_event_key
			ON social_wecom_auth_events (event_key);`,
		`CREATE TABLE IF NOT EXISTS social_sync_baseline_jobs (
			job_uuid TEXT PRIMARY KEY,
			tenant_uuid TEXT NOT NULL,
			binding_uuid TEXT NOT NULL,
			domain TEXT NOT NULL,
			mode TEXT NOT NULL,
			status TEXT NOT NULL,
			idempotency_key TEXT NOT NULL DEFAULT '',
			resolve_source TEXT NOT NULL DEFAULT 'default',
			total_count INTEGER NOT NULL DEFAULT 0,
			success_count INTEGER NOT NULL DEFAULT 0,
			failed_count INTEGER NOT NULL DEFAULT 0,
			conflict_count INTEGER NOT NULL DEFAULT 0,
			retry_count INTEGER NOT NULL DEFAULT 0,
			max_retries INTEGER NOT NULL DEFAULT 3,
			last_error TEXT NOT NULL DEFAULT '',
			next_retry_at DATETIME,
			dead_letter_at DATETIME,
			started_at DATETIME,
			finished_at DATETIME,
			write_back_fields TEXT,
			context TEXT,
			created_at DATETIME,
			updated_at DATETIME
		);`,
		`CREATE TABLE IF NOT EXISTS social_sync_conflict_records (
			conflict_uuid TEXT PRIMARY KEY,
			tenant_uuid TEXT NOT NULL,
			job_uuid TEXT NOT NULL,
			binding_uuid TEXT NOT NULL,
			domain TEXT NOT NULL,
			conflict_key TEXT NOT NULL,
			status TEXT NOT NULL,
			reason TEXT NOT NULL DEFAULT '',
			external_version TEXT NOT NULL DEFAULT '',
			local_version TEXT NOT NULL DEFAULT '',
			resolution_note TEXT NOT NULL DEFAULT '',
			replay_count INTEGER NOT NULL DEFAULT 0,
			last_replayed_at DATETIME,
			payload TEXT,
			created_at DATETIME,
			updated_at DATETIME
		);`,
	}
	for _, stmt := range stmts {
		if err := db.Exec(stmt).Error; err != nil {
			return err
		}
	}
	return nil
}

func TestOpenWorkFoundationService_IngestEvent_Idempotent(t *testing.T) {
	db := openOpenWorkServiceTestDB(t, "openwork_ingest_event_idempotent")
	repo := socialrepo.NewOpenWorkFoundationRepository(db)
	svc := NewOpenWorkFoundationService(repo, socialrepo.NewAccountRepository(db))

	tenantUUID := "00000000-0000-0000-0000-000000000005"
	eventTime := time.Now().UTC().Unix()
	first, binding, err := svc.IngestEvent(context.Background(), OpenWorkEventIngestInput{
		TenantUUID:  tenantUUID,
		SuiteID:     "suite-001",
		EventType:   "create_auth",
		SuiteTicket: "ticket-001",
		CorpID:      "corp-e",
		AgentID:     "10005",
		EventTime:   eventTime,
		Payload:     map[string]any{"auth_corp_id": "corp-e"},
	})
	require.NoError(t, err)
	require.NotNil(t, first)
	require.NotNil(t, binding)
	require.Equal(t, model.WeComAuthBindingStatusActive, binding.Status)

	second, _, err := svc.IngestEvent(context.Background(), OpenWorkEventIngestInput{
		TenantUUID: tenantUUID,
		SuiteID:    "suite-001",
		EventType:  "create_auth",
		CorpID:     "corp-e",
		AgentID:    "10005",
		EventTime:  eventTime,
		Payload:    map[string]any{"auth_corp_id": "corp-e"},
	})
	require.NoError(t, err)
	require.Equal(t, first.EventUUID, second.EventUUID)

	var count int64
	require.NoError(t, db.Model(&model.WeComOpenAuthEvent{}).Where("tenant_uuid = ?", tenantUUID).Count(&count).Error)
	require.EqualValues(t, 1, count)

	var payloadMap map[string]any
	require.NoError(t, json.Unmarshal(mustJSONMarshal(t, second.Payload), &payloadMap))
	require.Equal(t, "corp-e", payloadMap["auth_corp_id"])
}

func mustJSONMarshal(t *testing.T, v any) []byte {
	t.Helper()
	raw, err := json.Marshal(v)
	require.NoError(t, err)
	return raw
}
