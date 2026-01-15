package console_test

import (
	"context"
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/config"
	model "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/admin_console"
	adminmetrics "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/observability/admin_console"
	consolesvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/console"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
)

func setupConfigService(t *testing.T) (*consolesvc.ConfigService, *gorm.DB) {
	t.Helper()
	ginDB, err := gorm.Open(sqlite.Open("file:admin_console_config?mode=memory&cache=shared"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	require.NoError(t, err)
	models.ForceSchemaForTests("")
	require.NoError(t, ginDB.Exec(`CREATE TABLE IF NOT EXISTS admin_console_audit_events (
		id TEXT PRIMARY KEY,
		plugin_id TEXT NOT NULL,
		tenant_uuid TEXT,
		actor_id TEXT NOT NULL,
		actor_name TEXT,
		actor_email TEXT,
		permission_code TEXT NOT NULL,
		action TEXT NOT NULL,
		resource_type TEXT NOT NULL,
		resource_ref TEXT,
		summary TEXT,
		diff TEXT,
		occurred_at DATETIME,
		created_at DATETIME
	);`).Error)
	require.NoError(t, ginDB.Exec(`CREATE TABLE IF NOT EXISTS admin_console_config_changes (
		id TEXT PRIMARY KEY,
		plugin_id TEXT NOT NULL,
		tenant_uuid TEXT,
		section_key TEXT NOT NULL,
		change_type TEXT NOT NULL,
		previous_snapshot TEXT,
		next_snapshot TEXT,
		validation_summary TEXT,
		audit_event_id TEXT,
		applied_at DATETIME
	);`).Error)
	cfg := &config.Config{
		AdminConsole: &config.AdminConsoleConfig{
			AuditRetentionDays:    180,
			ConfigChangeRetention: 90,
			JobHistoryDays:        30,
			Export: config.AdminConsoleExportConfig{
				MaxRangeDays:  21,
				DefaultFormat: "json",
			},
			Troubleshooting: config.AdminConsoleTroubleshooting{
				RefreshIntervalSeconds: 240,
				CacheTTLSeconds:        120,
			},
			SafeOps: config.AdminConsoleSafeOps{
				LockTTLSeconds:   90,
				MaxConcurrentOps: 2,
			},
		},
	}
	deps := &app.Deps{DB: ginDB, Config: cfg, AdminConsoleMetrics: adminmetrics.NewMetrics()}
	return consolesvc.NewConfigService(deps), ginDB
}

func TestConfigService_ListSectionsDefaults(t *testing.T) {
	svc, _ := setupConfigService(t)
	sections, err := svc.ListSections(context.Background(), nil)
	require.NoError(t, err)
	require.NotEmpty(t, sections)
	first := sections[0]
	require.Equal(t, "admin_console.retention", first.Key)
	require.Equal(t, 180, first.CurrentValues["audit_retention_days"])
}

func TestConfigService_UpdateSection(t *testing.T) {
	svc, db := setupConfigService(t)
	input := consolesvc.UpdateSectionInput{
		SectionKey: "admin_console.retention",
		Values: map[string]any{
			"audit_retention_days":         200,
			"config_change_retention_days": 120,
			"job_history_days":             40,
		},
		Actor: consolesvc.Actor{ID: "user:123", Name: "Jane Admin", PermissionCode: "operations.plugin.admin"},
	}
	updated, err := svc.UpdateSection(context.Background(), input)
	require.NoError(t, err)
	require.NotNil(t, updated)
	require.EqualValues(t, 200, updated.CurrentValues["audit_retention_days"])

	var changeCount int64
	require.NoError(t, db.Model(&model.ConfigChange{}).Count(&changeCount).Error)
	require.Equal(t, int64(1), changeCount)
}
