package lead_capture

import (
	"context"
	"testing"
	"time"

	domainmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/models/lead_capture"
	domainrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/repository/lead_capture"
	basemodels "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type stubWelcomeSyncAdapter struct {
	errs []error
	idx  int
}

func (a *stubWelcomeSyncAdapter) PublishWelcome(_ context.Context, _ WelcomeSyncPublishInput) error {
	if a == nil || len(a.errs) == 0 {
		return nil
	}
	if a.idx >= len(a.errs) {
		return a.errs[len(a.errs)-1]
	}
	err := a.errs[a.idx]
	a.idx++
	return err
}

func TestWelcomeSyncService_RetryToManualRequiredAndRecover(t *testing.T) {
	db := openWelcomeSyncServiceDB(t, "welcome_sync_service")
	tenantUUID := "00000000-0000-0000-0000-000000000071"
	codeUUID := uuid.NewString()

	require.NoError(t, seedWelcomeSyncServiceData(db, tenantUUID, codeUUID))
	repos := domainrepo.NewBundle(db)

	adapter := &stubWelcomeSyncAdapter{errs: []error{
		&WelcomeSyncAdapterError{Code: WelcomeSyncErrorChannelRateLimited, Message: "limited"},
		&WelcomeSyncAdapterError{Code: WelcomeSyncErrorChannelRateLimited, Message: "limited"},
		&WelcomeSyncAdapterError{Code: WelcomeSyncErrorChannelRateLimited, Message: "limited"},
		nil,
	}}
	svc := NewWelcomeSyncService(repos.ChannelCodes, repos.WelcomeConfigs, repos.WelcomeSyncAttempt, adapter, nil).
		WithRetryPolicy([]time.Duration{0, 0, 0}, 3)

	first, err := svc.TriggerSync(context.Background(), WelcomeSyncTriggerRequest{TenantUUID: tenantUUID, CodeUUID: codeUUID})
	require.NoError(t, err)
	require.Equal(t, domainmodel.WelcomeSyncStatusManualRequired, first.SyncStatus)
	require.Equal(t, WelcomeSyncErrorChannelRateLimited, first.ErrorCode)

	status1, err := svc.GetStatus(context.Background(), tenantUUID, codeUUID)
	require.NoError(t, err)
	require.Equal(t, domainmodel.WelcomeSyncStatusManualRequired, status1.SyncStatus)
	require.Equal(t, 3, status1.LatestAttemptNo)
	require.Contains(t, status1.LastSyncError, WelcomeSyncErrorChannelRateLimited)

	second, err := svc.TriggerSync(context.Background(), WelcomeSyncTriggerRequest{TenantUUID: tenantUUID, CodeUUID: codeUUID})
	require.NoError(t, err)
	require.Equal(t, domainmodel.WelcomeSyncStatusSuccess, second.SyncStatus)
	require.Equal(t, 4, second.AttemptNo)

	status2, err := svc.GetStatus(context.Background(), tenantUUID, codeUUID)
	require.NoError(t, err)
	require.Equal(t, domainmodel.WelcomeSyncStatusSuccess, status2.SyncStatus)
	require.Equal(t, 4, status2.LatestAttemptNo)
	require.Empty(t, status2.LastSyncError)
}

func openWelcomeSyncServiceDB(t *testing.T, name string) *gorm.DB {
	t.Helper()
	basemodels.ForceSchemaForTests("")
	db, err := gorm.Open(sqlite.Open("file:"+name+"?mode=memory&cache=shared"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	require.NoError(t, err)
	require.NoError(t, ensureWelcomeSyncServiceTables(db))
	return db
}

func ensureWelcomeSyncServiceTables(db *gorm.DB) error {
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

func seedWelcomeSyncServiceData(db *gorm.DB, tenantUUID, codeUUID string) error {
	now := time.Now().UTC()
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&domainmodel.ChannelCode{
			CodeUUID:           codeUUID,
			TenantUUID:         tenantUUID,
			Channel:            "wechat",
			AppType:            "wecom",
			ChannelAccountUUID: "11111111-1111-4111-8111-111111111171",
			CodeKey:            "welcome-sync-code",
			DisplayName:        "欢迎语同步",
			TargetType:         "group",
			TargetID:           "group-sync",
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
			MessageContent: []byte(`{"text":"hello"}`),
			SyncStatus:     domainmodel.WelcomeSyncStatusPending,
			Version:        1,
			CreatedBy:      "system",
			UpdatedBy:      "system",
			CreatedAt:      now,
			UpdatedAt:      now,
		}).Error
	})
}
