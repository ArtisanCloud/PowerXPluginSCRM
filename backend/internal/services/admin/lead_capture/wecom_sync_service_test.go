package lead_capture

import (
	"context"
	"errors"
	"testing"

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
	}
	for _, stmt := range stmts {
		if err := db.Exec(stmt).Error; err != nil {
			return err
		}
	}
	return nil
}
