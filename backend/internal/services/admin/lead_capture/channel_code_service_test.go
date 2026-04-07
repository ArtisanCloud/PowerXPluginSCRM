package lead_capture

import (
	"context"
	"testing"

	domainrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/repository/lead_capture"
	basemodels "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestChannelCodeService_TenantIsolationAndUniqueConstraint(t *testing.T) {
	db := openChannelCodeServiceDB(t, "channel_code_service_isolation")
	repos := domainrepo.NewBundle(db)
	svc := NewChannelCodeService(repos.ChannelCodes, nil)

	_, err := svc.Create(context.Background(), ChannelCodeCreateRequest{
		TenantUUID:         "00000000-0000-0000-0000-000000000021",
		Channel:            "wechat",
		AppType:            "wecom",
		ChannelAccountUUID: "11111111-1111-4111-8111-111111111121",
		CodeKey:            "sales-campaign-a",
		DisplayName:        "A",
		TargetType:         "group",
		TargetID:           "group-a",
	})
	require.NoError(t, err)

	_, err = svc.Create(context.Background(), ChannelCodeCreateRequest{
		TenantUUID:         "00000000-0000-0000-0000-000000000022",
		Channel:            "wechat",
		AppType:            "wecom",
		ChannelAccountUUID: "11111111-1111-4111-8111-111111111122",
		CodeKey:            "sales-campaign-a",
		DisplayName:        "B",
		TargetType:         "group",
		TargetID:           "group-b",
	})
	require.NoError(t, err)

	listA, err := svc.List(context.Background(), ChannelCodeListRequest{TenantUUID: "00000000-0000-0000-0000-000000000021", Limit: 20})
	require.NoError(t, err)
	require.Len(t, listA, 1)
	require.Equal(t, "sales-campaign-a", listA[0].CodeKey)

	_, err = svc.Create(context.Background(), ChannelCodeCreateRequest{
		TenantUUID:         "00000000-0000-0000-0000-000000000021",
		Channel:            "wechat",
		AppType:            "wecom",
		ChannelAccountUUID: "11111111-1111-4111-8111-111111111123",
		CodeKey:            "sales-campaign-a",
		DisplayName:        "Duplicate",
		TargetType:         "group",
		TargetID:           "group-c",
	})
	require.ErrorIs(t, err, ErrChannelCodeAlreadyExists)
}

func TestChannelCodeService_WelcomeConfigChangeLogSummary(t *testing.T) {
	db := openChannelCodeServiceDB(t, "channel_code_service_welcome_log")
	repos := domainrepo.NewBundle(db)
	channelSvc := NewChannelCodeService(repos.ChannelCodes, nil)
	welcomeSvc := NewWelcomeConfigService(repos.WelcomeConfigs, repos.ConfigChangeLogs, nil)

	code, err := channelSvc.Create(context.Background(), ChannelCodeCreateRequest{
		TenantUUID:         "00000000-0000-0000-0000-000000000023",
		Channel:            "wechat",
		AppType:            "wecom",
		ChannelAccountUUID: "11111111-1111-4111-8111-111111111124",
		CodeKey:            "welcome-log-campaign",
		DisplayName:        "Welcome",
		TargetType:         "group",
		TargetID:           "group-d",
	})
	require.NoError(t, err)

	cfg, err := welcomeSvc.Save(context.Background(), WelcomeConfigSaveRequest{
		TenantUUID:     code.TenantUUID,
		CodeUUID:       code.CodeUUID,
		WelcomeEnabled: true,
		MessageContent: []byte(`{"text":"欢迎1"}`),
	})
	require.NoError(t, err)
	require.Equal(t, 1, cfg.Version)
	require.Equal(t, "pending", cfg.SyncStatus)

	cfg, err = welcomeSvc.Save(context.Background(), WelcomeConfigSaveRequest{
		TenantUUID:     code.TenantUUID,
		CodeUUID:       code.CodeUUID,
		WelcomeEnabled: true,
		MessageContent: []byte(`{"text":"欢迎2"}`),
	})
	require.NoError(t, err)
	require.Equal(t, 2, cfg.Version)

	logs, err := welcomeSvc.ListChangeLogs(context.Background(), code.TenantUUID, code.CodeUUID, 10)
	require.NoError(t, err)
	require.Len(t, logs, 2)
	require.Contains(t, logs[0].Summary, "待发布")
}

func openChannelCodeServiceDB(t *testing.T, name string) *gorm.DB {
	t.Helper()
	basemodels.ForceSchemaForTests("")
	db, err := gorm.Open(sqlite.Open("file:"+name+"?mode=memory&cache=shared"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	require.NoError(t, err)
	require.NoError(t, ensureChannelCodeServiceTables(db))
	return db
}

func ensureChannelCodeServiceTables(db *gorm.DB) error {
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
