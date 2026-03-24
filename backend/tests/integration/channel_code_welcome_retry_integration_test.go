package integration

import (
	"context"
	"testing"
	"time"

	domainmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/models/lead_capture"
	domainrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/repository/lead_capture"
	basemodels "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	leadsvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/lead_capture"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestChannelCodeWelcomeRetryIntegration_ManualRequiredThenRecover(t *testing.T) {
	db := openWelcomeRetryIntegrationDB(t, "channel_code_welcome_retry_integration")
	tenantUUID := "00000000-0000-0000-0000-000000000091"
	codeUUID := uuid.NewString()
	now := time.Now().UTC()

	require.NoError(t, seedWelcomeRetryIntegrationData(db, tenantUUID, codeUUID, now))
	repos := domainrepo.NewBundle(db)
	svc := leadsvc.NewWelcomeSyncService(
		repos.ChannelCodes,
		repos.WelcomeConfigs,
		repos.WelcomeSyncAttempt,
		leadsvc.NewWeComWelcomeAdapter(),
		nil,
	).WithRetryPolicy([]time.Duration{0, 0, 0}, 3)

	first, err := svc.TriggerSync(context.Background(), leadsvc.WelcomeSyncTriggerRequest{TenantUUID: tenantUUID, CodeUUID: codeUUID})
	require.NoError(t, err)
	require.Equal(t, domainmodel.WelcomeSyncStatusManualRequired, first.SyncStatus)
	require.Equal(t, leadsvc.WelcomeSyncErrorChannelUnavailable, first.ErrorCode)
	require.Equal(t, 3, first.AttemptNo)

	cfg, err := repos.WelcomeConfigs.GetByCodeUUID(context.Background(), tenantUUID, codeUUID)
	require.NoError(t, err)
	cfg.MessageContent = []byte(`{"text":"recover"}`)
	cfg.SyncStatus = domainmodel.WelcomeSyncStatusPending
	cfg.UpdatedBy = "system"
	require.NoError(t, repos.WelcomeConfigs.Save(context.Background(), cfg))

	second, err := svc.TriggerSync(context.Background(), leadsvc.WelcomeSyncTriggerRequest{TenantUUID: tenantUUID, CodeUUID: codeUUID})
	require.NoError(t, err)
	require.Equal(t, domainmodel.WelcomeSyncStatusSuccess, second.SyncStatus)
	require.Equal(t, 4, second.AttemptNo)

	status, err := svc.GetStatus(context.Background(), tenantUUID, codeUUID)
	require.NoError(t, err)
	require.Equal(t, domainmodel.WelcomeSyncStatusSuccess, status.SyncStatus)
	require.Equal(t, 4, status.LatestAttemptNo)
	require.Empty(t, status.LastSyncError)
}

func openWelcomeRetryIntegrationDB(t *testing.T, name string) *gorm.DB {
	t.Helper()
	basemodels.ForceSchemaForTests("")
	db, err := gorm.Open(sqlite.Open("file:"+name+"?mode=memory&cache=shared"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	require.NoError(t, err)
	require.NoError(t, ensureWelcomeRetryIntegrationTables(db))
	return db
}

func ensureWelcomeRetryIntegrationTables(db *gorm.DB) error {
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
		`CREATE TABLE IF NOT EXISTS lead_capture_code_welcome_sync_attempts (
			attempt_uuid TEXT PRIMARY KEY,
			tenant_uuid TEXT NOT NULL,
			code_uuid TEXT NOT NULL,
			config_version INTEGER NOT NULL,
			trigger_source TEXT NOT NULL,
			attempt_no INTEGER NOT NULL,
			result TEXT NOT NULL,
			error_code TEXT,
			error_message TEXT,
			started_at DATETIME,
			finished_at DATETIME,
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

func seedWelcomeRetryIntegrationData(db *gorm.DB, tenantUUID, codeUUID string, now time.Time) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&domainmodel.ChannelCode{
			CodeUUID:           codeUUID,
			TenantUUID:         tenantUUID,
			Channel:            "wechat",
			AppType:            "wecom",
			ChannelAccountUUID: "11111111-1111-4111-8111-111111111191",
			CodeKey:            "welcome-retry",
			DisplayName:        "欢迎语重试",
			TargetType:         "group",
			TargetID:           "group-retry",
			Status:             domainmodel.ChannelCodeStatusActive,
			CreatedBy:          "system",
			UpdatedBy:          "system",
			CreatedAt:          now,
			UpdatedAt:          now,
		}).Error; err != nil {
			return err
		}
		return tx.Create(&domainmodel.CodeWelcomeConfig{
			ConfigUUID:     uuid.NewString(),
			TenantUUID:     tenantUUID,
			CodeUUID:       codeUUID,
			WelcomeEnabled: true,
			MessageContent: []byte(`{"text":"fail","mock_error_code":"CHANNEL_UNAVAILABLE"}`),
			SyncStatus:     domainmodel.WelcomeSyncStatusPending,
			Version:        1,
			CreatedBy:      "system",
			UpdatedBy:      "system",
			CreatedAt:      now,
			UpdatedAt:      now,
		}).Error
	})
}
