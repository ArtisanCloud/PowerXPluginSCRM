package lead_capture

import (
	"context"
	"strings"
	"testing"

	basemodels "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	leadrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/lead_capture"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestLeadService_ImportCSVWithMapping_PreserveExplicitSourceScope(t *testing.T) {
	db := openLeadServiceImportTestDB(t, "lead_service_import_preserve_scope")
	tenantUUID := "00000000-0000-0000-0000-000000000001"
	repo := leadrepo.NewLeadRepository(db)
	svc := NewLeadService(repo)

	csvData := strings.NewReader("name,phone,source_channel,source_app_type,source_account_uuid\nAlice,13900001234,douyin,short_video,\nAlice2,13900001234,douyin,short_video,\n")
	result, err := svc.ImportCSVWithMapping(context.Background(), tenantUUID, csvData, map[string]int{
		"display_name":        0,
		"phone":               1,
		"source_channel":      2,
		"source_app_type":     3,
		"source_account_uuid": 4,
	})
	require.NoError(t, err)
	require.Equal(t, 2, result.Total)
	require.Equal(t, 2, result.Success)
	require.Equal(t, 0, result.Failed)

	leads, err := repo.List(context.Background(), tenantUUID)
	require.NoError(t, err)
	require.Len(t, leads, 1)
	require.Equal(t, "douyin", leads[0].SourceChannel)
	require.Equal(t, "short_video", leads[0].SourceAppType)
	require.Nil(t, leads[0].SourceAccountUUID)
}

func TestLeadService_ImportCSV_AllowEmptySourceScopeWithoutDefaultFallback(t *testing.T) {
	db := openLeadServiceImportTestDB(t, "lead_service_import_empty_scope")
	tenantUUID := "00000000-0000-0000-0000-000000000001"
	repo := leadrepo.NewLeadRepository(db)
	svc := NewLeadService(repo)

	csvData := strings.NewReader("name,phone\nManual,13900009999\n")
	result, err := svc.ImportCSV(context.Background(), tenantUUID, csvData)
	require.NoError(t, err)
	require.Equal(t, 1, result.Total)
	require.Equal(t, 1, result.Success)
	require.Equal(t, 0, result.Failed)

	lead, err := repo.FindFirstByPhone(context.Background(), tenantUUID, "13900009999")
	require.NoError(t, err)
	require.NotNil(t, lead)
	require.Equal(t, "", lead.SourceChannel)
	require.Equal(t, "", lead.SourceAppType)
	require.Nil(t, lead.SourceAccountUUID)
}

func openLeadServiceImportTestDB(t *testing.T, name string) *gorm.DB {
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
