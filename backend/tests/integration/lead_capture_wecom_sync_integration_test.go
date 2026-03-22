package integration

import (
	"context"
	"testing"

	basemodels "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	leadmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/lead_capture"
	socialmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/social_channel_governance"
	leadrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/lead_capture"
	leadsvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/lead_capture"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type integrationProviderAdapter struct{}

func (i integrationProviderAdapter) SubmitSyncTask(_ context.Context, _ leadsvc.TriggerSyncRequest, _ string) leadsvc.SyncTaskSubmitResult {
	return leadsvc.SyncTaskSubmitResult{Provider: leadmodel.LeadSyncTaskProviderLocalFallback, Status: "queued"}
}

type integrationLeadAdapter struct{}

func (a integrationLeadAdapter) FetchLeads(_ context.Context, _ leadsvc.TriggerSyncRequest, _ string) ([]leadsvc.WeComLeadRecord, error) {
	return []leadsvc.WeComLeadRecord{
		{DisplayName: "新线索", Phone: "13800000001", Email: "new@example.com"},
		{DisplayName: "补全邮箱", Phone: "13800000002", Email: "merged@example.com"},
	}, nil
}

func TestLeadCaptureWeComSyncIntegration_TenantIsolationAndStats(t *testing.T) {
	tenantA := "00000000-0000-0000-0000-000000000001"
	tenantB := "00000000-0000-0000-0000-000000000002"
	accountA := "11111111-1111-4111-8111-111111111111"
	accountB := "22222222-2222-4222-8222-222222222222"

	db := openWeComSyncIntegrationDB(t, "lead_capture_wecom_sync_integration")
	require.NoError(t, db.Create(&socialmodel.ChannelAccount{
		AccountUUID:     accountA,
		TenantUuid:      tenantA,
		ChannelCode:     "wechat",
		AppType:         "wecom",
		AccountID:       "wecom-a",
		DisplayName:     "租户A账号",
		Status:          socialmodel.ChannelAccountStatusConnected,
		OrgSyncDefault:  true,
		OwnerMemberUUID: "owner-a",
	}).Error)
	require.NoError(t, db.Create(&socialmodel.ChannelAccount{
		AccountUUID:     accountB,
		TenantUuid:      tenantB,
		ChannelCode:     "wechat",
		AppType:         "wecom",
		AccountID:       "wecom-b",
		DisplayName:     "租户B账号",
		Status:          socialmodel.ChannelAccountStatusConnected,
		OrgSyncDefault:  true,
		OwnerMemberUUID: "owner-b",
	}).Error)

	require.NoError(t, db.Create(&leadmodel.Lead{
		LeadUUID:      "33333333-3333-4333-8333-333333333333",
		TenantUUID:    tenantA,
		DisplayName:   "历史线索",
		Phone:         "13800000002",
		Email:         "",
		Status:        leadmodel.LeadStatusNew,
		SourceChannel: "wechat",
		SourceAppType: "wecom",
	}).Error)

	taskRepo := leadrepo.NewLeadSyncTaskRepository(db)
	leadRepo := leadrepo.NewLeadRepository(db)
	svc := leadsvc.NewWeComSyncService(taskRepo, nil, integrationProviderAdapter{}).WithLeadIngestion(leadRepo, integrationLeadAdapter{})

	task, err := svc.TriggerSync(context.Background(), leadsvc.TriggerSyncRequest{
		TenantUUID: tenantA,
		Channel:    "wechat",
		AppType:    "wecom",
		TraceID:    "integration-us1",
	})
	require.NoError(t, err)
	require.Equal(t, "success", task.Status)
	require.Equal(t, 2, task.StatsTotal)
	require.Equal(t, 1, task.StatsCreated)
	require.Equal(t, 1, task.StatsUpdated)
	require.Equal(t, 1, task.StatsMerged)

	leadsA, err := leadRepo.List(context.Background(), tenantA)
	require.NoError(t, err)
	require.Len(t, leadsA, 2)

	mergedLead, err := leadRepo.FindFirstByPhone(context.Background(), tenantA, "13800000002")
	require.NoError(t, err)
	require.Equal(t, "merged@example.com", mergedLead.Email)

	leadsB, err := leadRepo.List(context.Background(), tenantB)
	require.NoError(t, err)
	require.Len(t, leadsB, 0)

	tasksB, err := taskRepo.ListByFilter(context.Background(), tenantB, "", "", 20)
	require.NoError(t, err)
	require.Len(t, tasksB, 0)
}

func openWeComSyncIntegrationDB(t *testing.T, name string) *gorm.DB {
	t.Helper()
	basemodels.ForceSchemaForTests("")
	db, err := gorm.Open(sqlite.Open("file:"+name+"?mode=memory&cache=shared"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	require.NoError(t, err)
	require.NoError(t, createIntegrationTestSchema(db))
	return db
}

func createIntegrationTestSchema(db *gorm.DB) error {
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
	}
	for _, stmt := range stmts {
		if err := db.Exec(stmt).Error; err != nil {
			return err
		}
	}
	return nil
}
