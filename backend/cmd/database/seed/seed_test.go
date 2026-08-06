package seed

import (
	"context"
	"testing"

	basemodels "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	iammodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/iam"
	leadmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/lead_capture"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

const seedTestTenantUUID = "00000000-0000-0000-0000-000000000777"

func TestSeedPluginDataLocalDemoLeadGate(t *testing.T) {
	tests := []struct {
		name string
		opts PluginSeedOptions
		want int64
	}{
		{
			name: "delegated mode does not seed demo leads",
			opts: PluginSeedOptions{ProviderMode: "delegated", DevMode: true, TenantUUID: seedTestTenantUUID},
			want: 0,
		},
		{
			name: "non dev local mode does not seed demo leads",
			opts: PluginSeedOptions{ProviderMode: "local", DevMode: false, TenantUUID: seedTestTenantUUID},
			want: 0,
		},
		{
			name: "local dev mode seeds channel leads",
			opts: PluginSeedOptions{ProviderMode: "local", DevMode: true, TenantUUID: seedTestTenantUUID},
			want: 8,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := openSeedTestDB(t, tt.name)

			require.NoError(t, SeedPluginData(context.Background(), db, tt.opts))

			var leads int64
			require.NoError(t, db.Model(&leadmodel.Lead{}).Count(&leads).Error)
			require.Equal(t, tt.want, leads)

			var catalogs int64
			require.NoError(t, db.Model(&leadmodel.LeadSourceCatalog{}).Count(&catalogs).Error)
			require.Equal(t, int64(8), catalogs)

			if tt.want == 0 {
				return
			}

			for _, status := range []string{
				leadmodel.LeadStatusCaptured,
				leadmodel.LeadStatusEnriched,
				leadmodel.LeadStatusDeduplicated,
				leadmodel.LeadStatusRouted,
				leadmodel.LeadStatusEngaging,
				leadmodel.LeadStatusQualifiedForHandoff,
				leadmodel.LeadStatusHandoffPending,
				leadmodel.LeadStatusHandoffAccepted,
			} {
				var count int64
				require.NoError(t, db.Model(&leadmodel.Lead{}).Where("status = ?", status).Count(&count).Error)
				require.Equal(t, int64(1), count, status)
			}

			var sources int64
			require.NoError(t, db.Model(&leadmodel.LeadSource{}).Count(&sources).Error)
			require.Equal(t, tt.want, sources)

			var traces int64
			require.NoError(t, db.Model(&leadmodel.LeadActivity{}).Where("activity_type = ?", leadmodel.LeadActivityTypeSyncTrace).Count(&traces).Error)
			require.Equal(t, tt.want, traces)

			var histories int64
			require.NoError(t, db.Model(&leadmodel.LeadStatusHistory{}).Count(&histories).Error)
			require.Equal(t, tt.want, histories)
		})
	}
}

func TestSeedPluginDataResolvesLocalTenant(t *testing.T) {
	db := openSeedTestDB(t, "resolve tenant")
	require.NoError(t, db.Create(&iammodel.Tenant{
		UUID:   seedTestTenantUUID,
		Key:    "seed-test",
		Name:   "Seed Test",
		Status: iammodel.StatusActive,
		Plan:   "free",
	}).Error)

	require.NoError(t, SeedPluginData(context.Background(), db, PluginSeedOptions{
		ProviderMode: "local",
		DevMode:      true,
	}))

	var leads int64
	require.NoError(t, db.Model(&leadmodel.Lead{}).Where("tenant_uuid = ?", seedTestTenantUUID).Count(&leads).Error)
	require.Equal(t, int64(8), leads)
}

func openSeedTestDB(t *testing.T, name string) *gorm.DB {
	t.Helper()
	basemodels.ForceSchemaForTests("")
	db, err := gorm.Open(sqlite.Open("file:"+name+"?mode=memory&cache=shared"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	require.NoError(t, err)
	for _, stmt := range seedTestSchema() {
		require.NoError(t, db.Exec(stmt).Error)
	}
	return db
}

func seedTestSchema() []string {
	return []string{
		`CREATE TABLE IF NOT EXISTS iam_tenants (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			uuid TEXT NOT NULL UNIQUE,
			key TEXT NOT NULL UNIQUE,
			name TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'active',
			plan TEXT NOT NULL DEFAULT 'free',
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		);`,
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
		`CREATE TABLE IF NOT EXISTS lead_capture_status_history (
			history_uuid TEXT PRIMARY KEY,
			tenant_uuid TEXT NOT NULL,
			lead_uuid TEXT NOT NULL,
			from_status TEXT NOT NULL,
			to_status TEXT NOT NULL,
			changed_at DATETIME
		);`,
		`CREATE TABLE IF NOT EXISTS lead_capture_source_catalogs (
			catalog_uuid TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
			category TEXT NOT NULL,
			code TEXT NOT NULL,
			label TEXT NOT NULL,
			sort INTEGER NOT NULL DEFAULT 100,
			enabled BOOLEAN NOT NULL DEFAULT TRUE,
			created_at DATETIME,
			updated_at DATETIME,
			UNIQUE(category, code)
		);`,
	}
}
