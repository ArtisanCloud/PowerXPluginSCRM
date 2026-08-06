package lead_capture_test

import (
	"context"
	"testing"

	pwresponse "github.com/ArtisanCloud/PowerWeChat/v3/src/kernel/response"
	pwexternalreq "github.com/ArtisanCloud/PowerWeChat/v3/src/work/externalContact/request"
	basemodels "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	leadmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/lead_capture"
	socialmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/social_channel_governance"
	leadrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/lead_capture"
	socialrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/social_channel_governance"
	leadsvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/lead_capture"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type us3ProviderAdapter struct{}

func (us3ProviderAdapter) SubmitSyncTask(_ context.Context, _ leadsvc.TriggerSyncRequest, _ string) leadsvc.SyncTaskSubmitResult {
	return leadsvc.SyncTaskSubmitResult{Provider: leadmodel.LeadSyncTaskProviderLocalFallback, Status: "queued"}
}

type us3ExternalContactAdapter struct{}

func (us3ExternalContactAdapter) FetchLeads(_ context.Context, _ leadsvc.TriggerSyncRequest, _ string) ([]leadsvc.WeComLeadRecord, error) {
	return []leadsvc.WeComLeadRecord{
		{ExternalLeadID: "ext-001", CorpID: "corp-001", DisplayName: "客户A", Phone: "13800000021", Email: "a@example.com"},
		{ExternalLeadID: "ext-001", CorpID: "corp-001", DisplayName: "客户A-重复", Phone: "13800000021", Email: "dup@example.com"},
	}, nil
}

type us3RemarkClient struct{}

func (us3RemarkClient) Remark(_ context.Context, _ *pwexternalreq.RequestExternalContactRemark) (*pwresponse.ResponseWork, error) {
	return &pwresponse.ResponseWork{ErrCode: 0, ErrMsg: "ok"}, nil
}

func TestExternalContactBidirectionalIntegration_DedupWritebackAndDeadLetter(t *testing.T) {
	tenantUUID := "00000000-0000-0000-0000-000000000001"
	accountUUID := "11111111-1111-4111-8111-111111111111"

	db := openUS3IntegrationDB(t, "external_contact_bidirectional_integration")
	require.NoError(t, db.Create(&socialmodel.ChannelAccount{
		AccountUUID:     accountUUID,
		TenantUuid:      tenantUUID,
		ChannelCode:     "wechat",
		AppType:         "wecom",
		AccountID:       "wecom-us3",
		DisplayName:     "US3测试账号",
		Status:          socialmodel.ChannelAccountStatusConnected,
		OrgSyncDefault:  true,
		OwnerMemberUUID: "00000000-0000-0000-0000-000000000141",
	}).Error)

	taskRepo := leadrepo.NewLeadSyncTaskRepository(db)
	leadRepo := leadrepo.NewLeadRepository(db)
	syncRepo := socialrepo.NewSyncFoundationRepository(db)
	svc := leadsvc.NewWeComSyncService(taskRepo, nil, us3ProviderAdapter{}).
		WithLeadIngestion(leadRepo, us3ExternalContactAdapter{}).
		WithLeadService(leadsvc.NewLeadService(leadRepo)).
		WithRemarkClient(us3RemarkClient{}).
		WithSyncFoundation(syncRepo)

	pullTask, err := svc.TriggerSync(context.Background(), leadsvc.TriggerSyncRequest{
		TenantUUID:       tenantUUID,
		Channel:          "wechat",
		AppType:          "wecom",
		Domain:           "external_contacts",
		Direction:        "pull",
		Mode:             "incremental",
		CheckpointCursor: "us3-pull-cursor-1",
	})
	require.NoError(t, err)
	require.Equal(t, "success", pullTask.Status)
	require.Equal(t, 2, pullTask.StatsTotal)
	require.Equal(t, 1, pullTask.StatsCreated)

	leads, err := leadRepo.List(context.Background(), tenantUUID)
	require.NoError(t, err)
	require.Len(t, leads, 1)

	cp, err := syncRepo.GetCheckpoint(context.Background(), tenantUUID, "external_contacts", "pull")
	require.NoError(t, err)
	require.Equal(t, "us3-pull-cursor-1", cp.Cursor)

	_, err = svc.UpdateLeadWritebackPolicy(
		context.Background(),
		tenantUUID,
		"wechat",
		"wecom",
		map[string]any{"whitelist": []string{"phone"}},
		map[string]any{"fields": []string{"status"}},
		"safe",
		true,
	)
	require.NoError(t, err)

	pushTaskFirst, err := svc.TriggerSync(context.Background(), leadsvc.TriggerSyncRequest{
		TenantUUID: tenantUUID,
		Channel:    "wechat",
		AppType:    "wecom",
		Domain:     "leads",
		Direction:  "push",
		LeadWriteback: []leadsvc.LeadWritebackRecord{
			{ExternalUserID: "ext-write-1", CorpID: "corp-001", Phone: "13800000022", OrderVersion: 2, Fields: map[string]any{"phone": "13800000022", "status": "converted", "userid": "owner-001"}},
		},
	})
	require.NoError(t, err)
	require.Equal(t, "success", pushTaskFirst.Status)
	require.Equal(t, 1, pushTaskFirst.StatsUpdated)

	pushTaskDuplicate, err := svc.TriggerSync(context.Background(), leadsvc.TriggerSyncRequest{
		TenantUUID: tenantUUID,
		Channel:    "wechat",
		AppType:    "wecom",
		Domain:     "leads",
		Direction:  "push",
		LeadWriteback: []leadsvc.LeadWritebackRecord{
			{ExternalUserID: "ext-write-1", CorpID: "corp-001", Phone: "13800000022", OrderVersion: 2, Fields: map[string]any{"phone": "13800000022", "userid": "owner-001"}},
		},
	})
	require.NoError(t, err)
	require.Equal(t, "success", pushTaskDuplicate.Status)
	require.Equal(t, 0, pushTaskDuplicate.StatsUpdated)

	_, err = svc.UpdateLeadWritebackPolicy(
		context.Background(),
		tenantUUID,
		"wechat",
		"wecom",
		map[string]any{"whitelist": []string{"phone", "force_fail"}},
		map[string]any{"fields": []string{}},
		"safe",
		true,
	)
	require.NoError(t, err)

	failedPushTask, err := svc.TriggerSync(context.Background(), leadsvc.TriggerSyncRequest{
		TenantUUID: tenantUUID,
		Channel:    "wechat",
		AppType:    "wecom",
		Domain:     "leads",
		Direction:  "push",
		LeadWriteback: []leadsvc.LeadWritebackRecord{
			{ExternalUserID: "ext-write-2", CorpID: "corp-001", Phone: "13800000023", OrderVersion: 1, Fields: map[string]any{"phone": "13800000023", "force_fail": true, "userid": "owner-001"}},
		},
	})
	require.NoError(t, err)
	require.Equal(t, "failed", failedPushTask.Status)

	deadLetters, err := svc.ListLeadWritebackDeadLetters(context.Background(), tenantUUID, 20)
	require.NoError(t, err)
	require.Len(t, deadLetters, 1)
	require.Equal(t, "pending", deadLetters[0].ReplayStatus)

	replayed, err := svc.ReplayLeadWritebackDeadLetter(context.Background(), tenantUUID, deadLetters[0].DeadLetterUUID)
	require.NoError(t, err)
	require.Equal(t, "replayed", replayed.ReplayStatus)
}

func openUS3IntegrationDB(t *testing.T, name string) *gorm.DB {
	t.Helper()
	basemodels.ForceSchemaForTests("")
	db, err := gorm.Open(sqlite.Open("file:"+name+"?mode=memory&cache=shared"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	require.NoError(t, err)

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
		`CREATE TABLE IF NOT EXISTS lead_capture_sync_tasks (
			task_uuid TEXT PRIMARY KEY,
			external_task_id TEXT,
			tenant_uuid TEXT NOT NULL,
			channel TEXT NOT NULL,
			app_type TEXT NOT NULL,
			channel_account_uuid TEXT NOT NULL,
			account_resolve_source TEXT NOT NULL,
			task_provider TEXT NOT NULL,
			trigger_type TEXT NOT NULL,
			status TEXT NOT NULL,
			progress_total INTEGER NOT NULL DEFAULT 0,
			progress_current INTEGER NOT NULL DEFAULT 0,
			progress_percent INTEGER NOT NULL DEFAULT 0,
			stats_total INTEGER NOT NULL DEFAULT 0,
			stats_created INTEGER NOT NULL DEFAULT 0,
			stats_updated INTEGER NOT NULL DEFAULT 0,
			stats_merged INTEGER NOT NULL DEFAULT 0,
			stats_failed INTEGER NOT NULL DEFAULT 0,
			error_code TEXT,
			error_message TEXT,
			started_at DATETIME,
			finished_at DATETIME,
			created_at DATETIME,
			updated_at DATETIME
		);`,
		`CREATE TABLE IF NOT EXISTS lead_capture_leads (
			lead_uuid TEXT PRIMARY KEY,
			tenant_uuid TEXT NOT NULL,
			display_name TEXT,
			phone TEXT,
			email TEXT,
			status TEXT NOT NULL,
			owner_user_uuid TEXT,
			source_channel TEXT,
			source_app_type TEXT,
			source_account_uuid TEXT,
			created_at DATETIME,
			updated_at DATETIME
		);`,
		`CREATE TABLE IF NOT EXISTS lead_capture_sources (
			source_uuid TEXT PRIMARY KEY,
			lead_uuid TEXT NOT NULL,
			tenant_uuid TEXT NOT NULL,
			channel_code TEXT,
			app_type TEXT,
			account_uuid TEXT,
			campaign_code TEXT,
			utm_source TEXT,
			utm_medium TEXT,
			utm_campaign TEXT,
			created_at DATETIME,
			updated_at DATETIME
		);`,
		`CREATE TABLE IF NOT EXISTS lead_capture_activities (
			activity_uuid TEXT PRIMARY KEY,
			lead_uuid TEXT NOT NULL,
			tenant_uuid TEXT NOT NULL,
			activity_type TEXT NOT NULL,
			payload TEXT,
			created_at DATETIME,
			updated_at DATETIME
		);`,
		`CREATE TABLE IF NOT EXISTS social_sync_checkpoints (
			checkpoint_uuid TEXT PRIMARY KEY,
			tenant_uuid TEXT NOT NULL,
			domain TEXT NOT NULL,
			direction TEXT NOT NULL,
			cursor TEXT,
			snapshot_version TEXT,
			last_event_time DATETIME,
			created_at DATETIME,
			updated_at DATETIME
		);`,
		`CREATE UNIQUE INDEX IF NOT EXISTS uq_social_sync_checkpoints_scope
			ON social_sync_checkpoints (tenant_uuid, domain, direction);`,
		`CREATE TABLE IF NOT EXISTS social_sync_writeback_policies (
			policy_uuid TEXT PRIMARY KEY,
			tenant_uuid TEXT NOT NULL,
			domain TEXT NOT NULL,
			mapping_rules TEXT,
			protected_fields TEXT,
			overwrite_mode TEXT,
			enabled BOOLEAN NOT NULL DEFAULT 1,
			capability_status TEXT,
			created_at DATETIME,
			updated_at DATETIME
		);`,
		`CREATE UNIQUE INDEX IF NOT EXISTS uq_social_sync_writeback_policies_scope
			ON social_sync_writeback_policies (tenant_uuid, domain);`,
		`CREATE TABLE IF NOT EXISTS social_sync_jobs (
			job_uuid TEXT PRIMARY KEY,
			tenant_uuid TEXT NOT NULL,
			domain TEXT NOT NULL,
			direction TEXT,
			mode TEXT,
			status TEXT,
			attempt_no INTEGER,
			max_attempts INTEGER,
			idempotency_key TEXT,
			payload TEXT,
			error_code TEXT,
			error_message TEXT,
			started_at DATETIME,
			finished_at DATETIME,
			capability_status TEXT,
			created_at DATETIME,
			updated_at DATETIME
		);`,
		`CREATE TABLE IF NOT EXISTS social_sync_dead_letters (
			dead_letter_uuid TEXT PRIMARY KEY,
			tenant_uuid TEXT NOT NULL,
			job_uuid TEXT NOT NULL,
			domain TEXT NOT NULL,
			direction TEXT,
			last_error_code TEXT,
			last_error_message TEXT,
			retry_exhausted_at DATETIME,
			replay_status TEXT,
			replayed_by TEXT,
			replayed_at DATETIME,
			payload TEXT,
			created_at DATETIME,
			updated_at DATETIME
		);`,
		`CREATE TABLE IF NOT EXISTS social_wecom_auth_bindings (
			binding_uuid TEXT PRIMARY KEY,
			tenant_uuid TEXT NOT NULL,
			channel_account_uuid TEXT,
			channel_code TEXT NOT NULL DEFAULT 'wechat',
			app_type TEXT NOT NULL DEFAULT 'wecom',
			suite_id TEXT NOT NULL DEFAULT '',
			corp_id TEXT NOT NULL DEFAULT '',
			agent_id TEXT NOT NULL DEFAULT '',
			corp_name TEXT NOT NULL DEFAULT '',
			permanent_code TEXT NOT NULL DEFAULT '',
			suite_access_token TEXT NOT NULL DEFAULT '',
			suite_ticket TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT 'active',
			is_default BOOLEAN NOT NULL DEFAULT 0,
			default_switched_at DATETIME,
			last_event_type TEXT NOT NULL DEFAULT '',
			last_event_at DATETIME,
			auth_scope TEXT,
			metadata TEXT,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		);`,
		`CREATE TABLE IF NOT EXISTS social_channel_platform_settings (
			setting_uuid TEXT PRIMARY KEY,
			channel_code TEXT NOT NULL,
			provider_code TEXT NOT NULL,
			enabled BOOLEAN NOT NULL DEFAULT 0,
			config TEXT,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		);`,
		`CREATE UNIQUE INDEX IF NOT EXISTS uq_social_channel_platform_settings
			ON social_channel_platform_settings (channel_code, provider_code);`,
	}
	for _, stmt := range stmts {
		require.NoError(t, db.Exec(stmt).Error)
	}
	return db
}
