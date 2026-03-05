package integration

import (
	"context"
	"testing"

	basemodels "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	leadmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/lead_capture"
	leadrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/lead_capture"
	leadsvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/lead_capture"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestLeadCaptureDedupMergeIntegration_RecordMergeActivityAndSourceTrace(t *testing.T) {
	db := openDedupMergeIntegrationDB(t, "lead_capture_dedup_merge_integration")
	repo := leadrepo.NewLeadRepository(db)
	svc := leadsvc.NewLeadService(repo)
	tenantUUID := "00000000-0000-0000-0000-000000000001"

	first, err := svc.Create(context.Background(), tenantUUID, leadsvc.LeadCreateRequest{
		DisplayName:       "初始线索",
		Phone:             "13800000001",
		SourceChannel:     "wechat",
		SourceAppType:     "wecom",
		SourceAccountUUID: "11111111-1111-4111-8111-111111111111",
	})
	require.NoError(t, err)
	require.NotEmpty(t, first.LeadUUID)

	second, err := svc.Create(context.Background(), tenantUUID, leadsvc.LeadCreateRequest{
		DisplayName:       "补全资料",
		Phone:             "13800000001",
		Email:             "merge@example.com",
		SourceChannel:     "wechat",
		SourceAppType:     "wecom",
		SourceAccountUUID: "11111111-1111-4111-8111-111111111111",
	})
	require.NoError(t, err)
	require.Equal(t, first.LeadUUID, second.LeadUUID)
	require.True(t, second.HasMerge)
	require.Equal(t, "merge@example.com", second.Email)

	leads, err := repo.List(context.Background(), tenantUUID)
	require.NoError(t, err)
	require.Len(t, leads, 1)

	activities, err := svc.ListActivities(context.Background(), tenantUUID, first.LeadUUID)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(activities), 2)
	require.Equal(t, leadmodel.LeadActivityTypeMerge, activities[0].ActivityType)

	sources, err := svc.ListSourceEvents(context.Background(), tenantUUID, first.LeadUUID)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(sources), 2)
	require.Equal(t, "wechat", sources[0].ChannelCode)
	require.Equal(t, "wecom", sources[0].AppType)
}

func openDedupMergeIntegrationDB(t *testing.T, name string) *gorm.DB {
	t.Helper()
	basemodels.ForceSchemaForTests("")
	db, err := gorm.Open(sqlite.Open("file:"+name+"?mode=memory&cache=shared"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	require.NoError(t, err)

	stmts := []string{
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
	}
	for _, stmt := range stmts {
		require.NoError(t, db.Exec(stmt).Error)
	}
	return db
}
