package lead_capture

import (
	"context"
	"errors"
	"testing"
	"time"

	basemodels "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	leadmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/lead_capture"
	socialmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/social_channel_governance"
	leadrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/lead_capture"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type mockProviderAdapter struct{}

func (m mockProviderAdapter) SubmitSyncTask(_ context.Context, _ TriggerSyncRequest, _ string) SyncTaskSubmitResult {
	return SyncTaskSubmitResult{Provider: leadmodel.LeadSyncTaskProviderLocalFallback, Status: "queued"}
}

type mockWeComLeadAdapter struct {
	items []WeComLeadRecord
	err   error
}

func (m mockWeComLeadAdapter) FetchLeads(_ context.Context, _ TriggerSyncRequest, _ string) ([]WeComLeadRecord, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.items, nil
}

func TestWeComSyncService_ProviderFallbackAndStatusFlow(t *testing.T) {
	tenantUUID := "00000000-0000-0000-0000-000000000001"
	accountUUID := "11111111-1111-4111-8111-111111111111"

	db := openWeComSyncServiceTestDB(t, "wecom_sync_service_status_flow")
	require.NoError(t, db.Create(&socialmodel.ChannelAccount{
		AccountUUID:     accountUUID,
		TenantUuid:      tenantUUID,
		ChannelCode:     "wechat",
		AppType:         "wecom",
		AccountID:       "wecom-main",
		DisplayName:     "企微主账号",
		Status:          socialmodel.ChannelAccountStatusConnected,
		OrgSyncDefault:  true,
		OwnerMemberUUID: "owner-001",
	}).Error)

	taskRepo := leadrepo.NewLeadSyncTaskRepository(db)
	leadRepo := leadrepo.NewLeadRepository(db)
	svc := NewWeComSyncService(taskRepo, nil, mockProviderAdapter{}).WithLeadIngestion(leadRepo, mockWeComLeadAdapter{
		items: []WeComLeadRecord{{DisplayName: "Alice", Phone: "13800000001"}},
	})

	task, err := svc.TriggerSync(context.Background(), TriggerSyncRequest{
		TenantUUID: tenantUUID,
		Channel:    "wechat",
		AppType:    "wecom",
		TraceID:    "trace-001",
	})
	require.NoError(t, err)
	require.Equal(t, leadmodel.LeadSyncTaskProviderLocalFallback, task.TaskProvider)
	require.Equal(t, "success", task.Status)
	require.Equal(t, 1, task.StatsTotal)
	require.Equal(t, 1, task.StatsCreated)
	require.Equal(t, 0, task.StatsUpdated)

	list, err := taskRepo.ListByFilter(context.Background(), tenantUUID, accountUUID, "success", 20)
	require.NoError(t, err)
	require.Len(t, list, 1)
}

func TestWeComSyncService_FailedThenRetryToQueued(t *testing.T) {
	tenantUUID := "00000000-0000-0000-0000-000000000001"
	accountUUID := "11111111-1111-4111-8111-111111111111"

	db := openWeComSyncServiceTestDB(t, "wecom_sync_service_retry_flow")
	require.NoError(t, db.Create(&socialmodel.ChannelAccount{
		AccountUUID:     accountUUID,
		TenantUuid:      tenantUUID,
		ChannelCode:     "wechat",
		AppType:         "wecom",
		AccountID:       "wecom-main",
		DisplayName:     "企微主账号",
		Status:          socialmodel.ChannelAccountStatusConnected,
		OrgSyncDefault:  true,
		OwnerMemberUUID: "owner-001",
	}).Error)

	taskRepo := leadrepo.NewLeadSyncTaskRepository(db)
	leadRepo := leadrepo.NewLeadRepository(db)
	svc := NewWeComSyncService(taskRepo, nil, mockProviderAdapter{}).WithLeadIngestion(leadRepo, mockWeComLeadAdapter{
		err: errors.New("upstream rate limited"),
	})

	task, err := svc.TriggerSync(context.Background(), TriggerSyncRequest{
		TenantUUID: tenantUUID,
		Channel:    "wechat",
		AppType:    "wecom",
	})
	require.NoError(t, err)
	require.Equal(t, "failed", task.Status)
	require.Contains(t, task.ErrorMessage, "upstream rate limited")

	require.NoError(t, svc.RetryTask(context.Background(), tenantUUID, task.TaskUUID))

	items, err := taskRepo.ListByFilter(context.Background(), tenantUUID, accountUUID, "queued", 10)
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, task.TaskUUID, items[0].TaskUUID)
}

func TestWeComSyncService_WriteExternalUserSourceTraceActivity(t *testing.T) {
	tenantUUID := "00000000-0000-0000-0000-000000000001"
	accountUUID := "11111111-1111-4111-8111-111111111111"
	traceID := "trace-ext-001"

	db := openWeComSyncServiceTestDB(t, "wecom_sync_service_source_trace")
	require.NoError(t, db.Create(&socialmodel.ChannelAccount{
		AccountUUID:     accountUUID,
		TenantUuid:      tenantUUID,
		ChannelCode:     "wechat",
		AppType:         "wecom",
		AccountID:       "wecom-main",
		DisplayName:     "企微主账号",
		Status:          socialmodel.ChannelAccountStatusConnected,
		OrgSyncDefault:  true,
		OwnerMemberUUID: "owner-001",
	}).Error)

	taskRepo := leadrepo.NewLeadSyncTaskRepository(db)
	leadRepo := leadrepo.NewLeadRepository(db)
	leadSvc := NewLeadService(leadRepo)
	svc := NewWeComSyncService(taskRepo, nil, mockProviderAdapter{}).
		WithLeadIngestion(leadRepo, mockWeComLeadAdapter{
			items: []WeComLeadRecord{{
				ExternalLeadID: "ext-user-001",
				DisplayName:    "External User",
				Phone:          "13800000009",
				Email:          "ext-user@example.com",
				OccurredAt:     time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC),
			}},
		}).
		WithLeadService(leadSvc)

	_, err := svc.TriggerSync(context.Background(), TriggerSyncRequest{
		TenantUUID: tenantUUID,
		Channel:    "wechat",
		AppType:    "wecom",
		TraceID:    traceID,
	})
	require.NoError(t, err)

	lead, err := leadRepo.FindFirstByPhone(context.Background(), tenantUUID, "13800000009")
	require.NoError(t, err)
	require.NotNil(t, lead)

	activities, err := leadSvc.ListActivities(context.Background(), tenantUUID, lead.LeadUUID)
	require.NoError(t, err)
	require.NotEmpty(t, activities)

	var syncTrace *leadmodel.LeadActivity
	for _, item := range activities {
		if item != nil && item.ActivityType == leadmodel.LeadActivityTypeSyncTrace {
			syncTrace = item
			break
		}
	}
	require.NotNil(t, syncTrace)
	require.Equal(t, "ext-user-001", syncTrace.Payload["external_lead_id"])
	require.Equal(t, "wechat", syncTrace.Payload["source_channel"])
	require.Equal(t, "wecom", syncTrace.Payload["source_app_type"])
	require.Equal(t, accountUUID, syncTrace.Payload["source_account_uuid"])
	require.Equal(t, traceID, syncTrace.Payload["trace_id"])
}

func TestWeComSyncService_PullAutoBindOwnerFromMemberBinding(t *testing.T) {
	tenantUUID := "00000000-0000-0000-0000-000000000001"
	accountUUID := "11111111-1111-4111-8111-111111111111"
	mainMemberID := "member-local-001"
	externalMemberID := "wosdnEDAAAgy3CKPLG5gxsi5ByObeabg"

	db := openWeComSyncServiceTestDB(t, "wecom_sync_service_owner_auto_bind")
	require.NoError(t, db.Create(&socialmodel.ChannelAccount{
		AccountUUID:     accountUUID,
		TenantUuid:      tenantUUID,
		ChannelCode:     "wechat",
		AppType:         "wecom",
		AccountID:       "wecom-main",
		DisplayName:     "企微主账号",
		Status:          socialmodel.ChannelAccountStatusConnected,
		OrgSyncDefault:  true,
		OwnerMemberUUID: "owner-001",
	}).Error)
	require.NoError(t, db.Exec(
		`INSERT INTO org_sync_member_bindings (member_binding_uuid, tenant_uuid, channel_account_uuid, main_member_id, external_member_id, sync_status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		"mb-0001", tenantUUID, accountUUID, mainMemberID, externalMemberID, "synced", time.Now().UTC(), time.Now().UTC(),
	).Error)

	taskRepo := leadrepo.NewLeadSyncTaskRepository(db)
	leadRepo := leadrepo.NewLeadRepository(db)
	leadSvc := NewLeadService(leadRepo)
	svc := NewWeComSyncService(taskRepo, nil, mockProviderAdapter{}).
		WithLeadIngestion(leadRepo, mockWeComLeadAdapter{
			items: []WeComLeadRecord{{
				ExternalLeadID:  "ext-user-bind-owner-001",
				DisplayName:     "Auto Bind Owner",
				Phone:           "13800000039",
				Email:           "owner-bind@example.com",
				OwnerMemberUUID: externalMemberID,
				OccurredAt:      time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC),
			}},
		}).
		WithLeadService(leadSvc)

	_, err := svc.TriggerSync(context.Background(), TriggerSyncRequest{
		TenantUUID: tenantUUID,
		Channel:    "wechat",
		AppType:    "wecom",
		TraceID:    "trace-owner-auto-bind",
	})
	require.NoError(t, err)

	lead, err := leadRepo.FindFirstByPhone(context.Background(), tenantUUID, "13800000039")
	require.NoError(t, err)
	require.NotNil(t, lead)
	require.Equal(t, mainMemberID, lead.OwnerUserUUID)
}

func TestWeComSyncService_UpsertSyncTraceActivity(t *testing.T) {
	tenantUUID := "00000000-0000-0000-0000-000000000001"
	accountUUID := "11111111-1111-4111-8111-111111111111"

	db := openWeComSyncServiceTestDB(t, "wecom_sync_service_sync_trace_upsert")
	require.NoError(t, db.Create(&socialmodel.ChannelAccount{
		AccountUUID:     accountUUID,
		TenantUuid:      tenantUUID,
		ChannelCode:     "wechat",
		AppType:         "wecom",
		AccountID:       "wecom-main",
		DisplayName:     "企微主账号",
		Status:          socialmodel.ChannelAccountStatusConnected,
		OrgSyncDefault:  true,
		OwnerMemberUUID: "owner-001",
	}).Error)

	taskRepo := leadrepo.NewLeadSyncTaskRepository(db)
	leadRepo := leadrepo.NewLeadRepository(db)
	leadSvc := NewLeadService(leadRepo)
	svc := NewWeComSyncService(taskRepo, nil, mockProviderAdapter{}).
		WithLeadIngestion(leadRepo, mockWeComLeadAdapter{
			items: []WeComLeadRecord{{
				ExternalLeadID: "ext-user-upsert-001",
				WechatID:       "wx-upsert-001",
				DisplayName:    "Upsert User",
				Phone:          "13800000029",
				Email:          "upsert-user@example.com",
				OccurredAt:     time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC),
			}},
		}).
		WithLeadService(leadSvc)

	_, err := svc.TriggerSync(context.Background(), TriggerSyncRequest{
		TenantUUID: tenantUUID,
		Channel:    "wechat",
		AppType:    "wecom",
		TraceID:    "trace-sync-upsert-1",
	})
	require.NoError(t, err)

	_, err = svc.TriggerSync(context.Background(), TriggerSyncRequest{
		TenantUUID: tenantUUID,
		Channel:    "wechat",
		AppType:    "wecom",
		TraceID:    "trace-sync-upsert-2",
	})
	require.NoError(t, err)

	lead, err := leadRepo.FindFirstByPhone(context.Background(), tenantUUID, "13800000029")
	require.NoError(t, err)
	require.NotNil(t, lead)

	activities, err := leadSvc.ListActivities(context.Background(), tenantUUID, lead.LeadUUID)
	require.NoError(t, err)
	require.NotEmpty(t, activities)

	syncCount := 0
	var syncTrace *leadmodel.LeadActivity
	for _, item := range activities {
		if item != nil && item.ActivityType == leadmodel.LeadActivityTypeSyncTrace {
			syncCount++
			syncTrace = item
		}
	}
	require.Equal(t, 1, syncCount)
	require.NotNil(t, syncTrace)
	require.Equal(t, "ext-user-upsert-001", syncTrace.Payload["external_lead_id"])
	require.Equal(t, "wx-upsert-001", syncTrace.Payload["external_wechat_id"])
}

func TestWeComSyncService_NoContactInfo_ShouldUpsertByExternalIdentity(t *testing.T) {
	tenantUUID := "00000000-0000-0000-0000-000000000001"
	accountUUID := "11111111-1111-4111-8111-111111111111"

	db := openWeComSyncServiceTestDB(t, "wecom_sync_service_external_identity_upsert")
	require.NoError(t, db.Create(&socialmodel.ChannelAccount{
		AccountUUID:     accountUUID,
		TenantUuid:      tenantUUID,
		ChannelCode:     "wechat",
		AppType:         "openwork",
		AccountID:       "openwork-main",
		DisplayName:     "企微代开发账号",
		Status:          socialmodel.ChannelAccountStatusConnected,
		OrgSyncDefault:  true,
		OwnerMemberUUID: "owner-001",
	}).Error)

	taskRepo := leadrepo.NewLeadSyncTaskRepository(db)
	leadRepo := leadrepo.NewLeadRepository(db)
	leadSvc := NewLeadService(leadRepo)
	svc := NewWeComSyncService(taskRepo, nil, mockProviderAdapter{}).
		WithLeadIngestion(leadRepo, mockWeComLeadAdapter{
			items: []WeComLeadRecord{{
				ExternalLeadID: "ext-no-contact-001",
				WechatID:       "wx-no-contact-001",
				DisplayName:    "无联系方式线索",
				OccurredAt:     time.Date(2026, 3, 21, 0, 0, 0, 0, time.UTC),
			}},
		}).
		WithLeadService(leadSvc)

	_, err := svc.TriggerSync(context.Background(), TriggerSyncRequest{
		TenantUUID: tenantUUID,
		Channel:    "wechat",
		AppType:    "openwork",
		TraceID:    "trace-no-contact-1",
	})
	require.NoError(t, err)
	var traces []leadmodel.LeadActivity
	require.NoError(t, db.Where("tenant_uuid = ? AND activity_type = ?", tenantUUID, leadmodel.LeadActivityTypeSyncTrace).Find(&traces).Error)
	require.NotEmpty(t, traces)
	require.Equal(t, "ext-no-contact-001", traces[0].Payload["external_lead_id"])

	_, err = svc.TriggerSync(context.Background(), TriggerSyncRequest{
		TenantUUID: tenantUUID,
		Channel:    "wechat",
		AppType:    "openwork",
		TraceID:    "trace-no-contact-2",
	})
	require.NoError(t, err)

	var count int64
	require.NoError(t, db.Model(&leadmodel.Lead{}).Where("tenant_uuid = ?", tenantUUID).Count(&count).Error)
	require.EqualValues(t, 1, count)
}

func TestWeComSyncService_TriggerSyncAsync_ReturnQueuedThenFinish(t *testing.T) {
	tenantUUID := "00000000-0000-0000-0000-000000000001"
	accountUUID := "11111111-1111-4111-8111-111111111111"

	db := openWeComSyncServiceTestDB(t, "wecom_sync_service_async")
	require.NoError(t, db.Create(&socialmodel.ChannelAccount{
		AccountUUID:     accountUUID,
		TenantUuid:      tenantUUID,
		ChannelCode:     "wechat",
		AppType:         "wecom",
		AccountID:       "wecom-main",
		DisplayName:     "企微主账号",
		Status:          socialmodel.ChannelAccountStatusConnected,
		OrgSyncDefault:  true,
		OwnerMemberUUID: "owner-001",
	}).Error)

	taskRepo := leadrepo.NewLeadSyncTaskRepository(db)
	leadRepo := leadrepo.NewLeadRepository(db)
	leadSvc := NewLeadService(leadRepo)
	svc := NewWeComSyncService(taskRepo, nil, mockProviderAdapter{}).
		WithLeadIngestion(leadRepo, mockWeComLeadAdapter{
			items: []WeComLeadRecord{{DisplayName: "Async Lead", Phone: "13800000019"}},
		}).
		WithLeadService(leadSvc)

	task, err := svc.TriggerSyncAsync(context.Background(), TriggerSyncRequest{
		TenantUUID: tenantUUID,
		Channel:    "wechat",
		AppType:    "wecom",
	})
	require.NoError(t, err)
	require.Equal(t, "queued", task.Status)

	require.Eventually(t, func() bool {
		list, listErr := taskRepo.ListByFilter(context.Background(), tenantUUID, accountUUID, "", 1)
		if listErr != nil || len(list) == 0 || list[0] == nil {
			return false
		}
		return list[0].Status == "success" && list[0].ProgressPercent == 100
	}, 2*time.Second, 30*time.Millisecond)
}

func openWeComSyncServiceTestDB(t *testing.T, name string) *gorm.DB {
	t.Helper()
	basemodels.ForceSchemaForTests("")
	db, err := gorm.Open(sqlite.Open("file:"+name+"?mode=memory&cache=shared"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	require.NoError(t, err)
	require.NoError(t, createWeComSyncTestSchema(db))
	return db
}

func createWeComSyncTestSchema(db *gorm.DB) error {
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
		`CREATE TABLE IF NOT EXISTS lead_capture_activities (
			activity_uuid TEXT PRIMARY KEY,
			lead_uuid TEXT NOT NULL,
			tenant_uuid TEXT NOT NULL,
			activity_type TEXT NOT NULL,
			payload TEXT,
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
		`CREATE TABLE IF NOT EXISTS org_sync_member_bindings (
			member_binding_uuid TEXT PRIMARY KEY,
			tenant_uuid TEXT NOT NULL,
			channel_account_uuid TEXT NOT NULL,
			main_member_id TEXT NOT NULL,
			external_member_id TEXT NOT NULL,
			sync_status TEXT NOT NULL,
			last_pulled_at DATETIME,
			last_pushed_at DATETIME,
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
