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

type runtimeConsistencySnapshot struct {
	SyncStatus      string
	AttemptNo       int
	ErrorCode       string
	LatestAttemptNo int
}

func TestChannelCodeRuntimeModeConsistency_WelcomeSyncBehavior(t *testing.T) {
	modes := []string{"0", "1"}
	results := make([]runtimeConsistencySnapshot, 0, len(modes))

	for _, mode := range modes {
		t.Run("powerx_proxy_"+mode, func(t *testing.T) {
			t.Setenv("POWERX_PROXY", mode)

			db := openRuntimeModeConsistencyDB(t, "channel_code_runtime_mode_consistency_"+mode)
			tenantUUID := "00000000-0000-0000-0000-000000000101"
			codeUUID := uuid.NewString()
			now := time.Now().UTC()
			require.NoError(t, seedRuntimeModeConsistencyData(db, tenantUUID, codeUUID, now))

			repos := domainrepo.NewBundle(db)
			svc := leadsvc.NewWelcomeSyncService(
				repos.ChannelCodes,
				repos.WelcomeConfigs,
				repos.WelcomeSyncAttempt,
				leadsvc.NewWeComWelcomeAdapter(),
				nil,
			).WithRetryPolicy([]time.Duration{0, 0, 0}, 3)

			trigger, err := svc.TriggerSync(context.Background(), leadsvc.WelcomeSyncTriggerRequest{
				TenantUUID: tenantUUID,
				CodeUUID:   codeUUID,
			})
			require.NoError(t, err)
			status, err := svc.GetStatus(context.Background(), tenantUUID, codeUUID)
			require.NoError(t, err)

			results = append(results, runtimeConsistencySnapshot{
				SyncStatus:      trigger.SyncStatus,
				AttemptNo:       trigger.AttemptNo,
				ErrorCode:       trigger.ErrorCode,
				LatestAttemptNo: status.LatestAttemptNo,
			})
		})
	}

	require.Len(t, results, 2)
	require.Equal(t, results[0], results[1], "standalone 与 host/proxy 模式输出应一致")
	require.Equal(t, domainmodel.WelcomeSyncStatusManualRequired, results[0].SyncStatus)
	require.Equal(t, leadsvc.WelcomeSyncErrorChannelUnavailable, results[0].ErrorCode)
	require.Equal(t, 3, results[0].AttemptNo)
	require.Equal(t, 3, results[0].LatestAttemptNo)
}

func openRuntimeModeConsistencyDB(t *testing.T, name string) *gorm.DB {
	t.Helper()
	basemodels.ForceSchemaForTests("")
	db, err := gorm.Open(sqlite.Open("file:"+name+"?mode=memory&cache=shared"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	require.NoError(t, err)
	require.NoError(t, ensureRuntimeModeConsistencyTables(db))
	return db
}

func ensureRuntimeModeConsistencyTables(db *gorm.DB) error {
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

func seedRuntimeModeConsistencyData(db *gorm.DB, tenantUUID, codeUUID string, now time.Time) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&domainmodel.ChannelCode{
			CodeUUID:           codeUUID,
			TenantUUID:         tenantUUID,
			Channel:            "wechat",
			AppType:            "wecom",
			ChannelAccountUUID: "11111111-1111-4111-8111-111111111201",
			CodeKey:            "runtime-mode-consistency",
			DisplayName:        "runtime-consistency",
			TargetType:         "group",
			TargetID:           "group-runtime",
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
			MessageContent: []byte(`{"text":"mode-check","mock_error_code":"CHANNEL_UNAVAILABLE"}`),
			SyncStatus:     domainmodel.WelcomeSyncStatusPending,
			Version:        1,
			CreatedBy:      "system",
			UpdatedBy:      "system",
			CreatedAt:      now,
			UpdatedAt:      now,
		}).Error
	})
}
