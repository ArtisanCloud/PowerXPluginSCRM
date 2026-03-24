package integration

import (
	"context"
	"testing"

	domainrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/repository/lead_capture"
	basemodels "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	leadsvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/lead_capture"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestChannelCodeWelcomePendingIntegration_SaveOnlyKeepsPending(t *testing.T) {
	db := openChannelCodeWelcomePendingDB(t, "channel_code_welcome_pending_integration")
	repos := domainrepo.NewBundle(db)
	channelSvc := leadsvc.NewChannelCodeService(repos.ChannelCodes, nil)
	welcomeSvc := leadsvc.NewWelcomeConfigService(repos.WelcomeConfigs, repos.ConfigChangeLogs, nil)

	code, err := channelSvc.Create(context.Background(), leadsvc.ChannelCodeCreateRequest{
		TenantUUID:         "00000000-0000-0000-0000-000000000031",
		Channel:            "wechat",
		AppType:            "wecom",
		ChannelAccountUUID: "11111111-1111-4111-8111-111111111131",
		CodeKey:            "pending-save-only",
		DisplayName:        "SaveOnly",
		TargetType:         "group",
		TargetID:           "group-pending",
	})
	require.NoError(t, err)

	cfg, err := welcomeSvc.Save(context.Background(), leadsvc.WelcomeConfigSaveRequest{
		TenantUUID:     code.TenantUUID,
		CodeUUID:       code.CodeUUID,
		WelcomeEnabled: true,
		MessageContent: []byte(`{"text":"仅保存不发布"}`),
	})
	require.NoError(t, err)
	require.Equal(t, "pending", cfg.SyncStatus)
	require.Equal(t, 1, cfg.Version)

	logs, err := welcomeSvc.ListChangeLogs(context.Background(), code.TenantUUID, code.CodeUUID, 10)
	require.NoError(t, err)
	require.Len(t, logs, 1)
	require.Equal(t, 1, logs[0].Version)
	require.Contains(t, logs[0].Summary, "待发布")
}

func openChannelCodeWelcomePendingDB(t *testing.T, name string) *gorm.DB {
	t.Helper()
	basemodels.ForceSchemaForTests("")
	db, err := gorm.Open(sqlite.Open("file:"+name+"?mode=memory&cache=shared"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	require.NoError(t, err)
	require.NoError(t, ensureChannelCodeWelcomePendingTables(db))
	return db
}

func ensureChannelCodeWelcomePendingTables(db *gorm.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS lead_capture_channel_codes (
			code_uuid TEXT PRIMARY KEY,
			tenant_uuid TEXT NOT NULL,
			channel TEXT NOT NULL,
			app_type TEXT NOT NULL,
			channel_account_uuid TEXT NOT NULL,
			code_key TEXT NOT NULL,
			display_name TEXT NOT NULL,
			target_type TEXT NOT NULL,
			target_id TEXT NOT NULL,
			status TEXT NOT NULL,
			created_by TEXT NOT NULL,
			updated_by TEXT NOT NULL,
			created_at DATETIME,
			updated_at DATETIME
		);`,
		`CREATE UNIQUE INDEX IF NOT EXISTS uq_lead_capture_channel_codes_tenant_channel_code
			ON lead_capture_channel_codes (tenant_uuid, channel, code_key);`,
		`CREATE TABLE IF NOT EXISTS lead_capture_code_welcome_configs (
			config_uuid TEXT PRIMARY KEY,
			tenant_uuid TEXT NOT NULL,
			code_uuid TEXT NOT NULL,
			welcome_enabled BOOLEAN NOT NULL DEFAULT FALSE,
			message_content TEXT NOT NULL,
			sync_status TEXT NOT NULL,
			last_sync_error TEXT,
			last_synced_at DATETIME,
			version INTEGER NOT NULL,
			created_by TEXT NOT NULL,
			updated_by TEXT NOT NULL,
			created_at DATETIME,
			updated_at DATETIME
		);`,
		`CREATE UNIQUE INDEX IF NOT EXISTS uq_lead_capture_code_welcome_cfg_code
			ON lead_capture_code_welcome_configs (code_uuid);`,
		`CREATE TABLE IF NOT EXISTS lead_capture_code_config_change_logs (
			change_uuid TEXT PRIMARY KEY,
			tenant_uuid TEXT NOT NULL,
			code_uuid TEXT NOT NULL,
			config_uuid TEXT NOT NULL,
			version INTEGER NOT NULL,
			summary TEXT NOT NULL,
			changed_fields TEXT,
			previous_content TEXT,
			next_content TEXT,
			changed_by TEXT NOT NULL,
			created_at DATETIME
		);`,
	}
	for _, stmt := range stmts {
		if err := db.Exec(stmt).Error; err != nil {
			return err
		}
	}
	return nil
}
