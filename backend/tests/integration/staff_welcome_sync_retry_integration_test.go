package integration

import (
	"context"
	"testing"
	"time"

	acqmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/models/acquisition"
	acqrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/repository/acquisition"
	basemodels "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	acqsvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/acquisition"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestStaffWelcomeSyncRetryIntegration_ReadyAfterConfigSync(t *testing.T) {
	db := openStaffWelcomeIntegrationDB(t, "staff_welcome_sync_retry_integration")
	tenantUUID := "00000000-0000-0000-0000-000000000411"
	staffCodeUUID := "33333333-3333-4333-8333-333333333411"
	now := time.Now().UTC()

	require.NoError(t, seedStaffWelcomeIntegrationData(db, tenantUUID, staffCodeUUID, now))
	repos := acqrepo.NewBundle(db)
	svc := acqsvc.NewStaffWelcomeService(repos.StaffLiveCodes, repos.StaffWelcomeConfigs, repos.StaffWelcomeAttempt)

	_, err := svc.Save(context.Background(), acqsvc.StaffWelcomeSaveRequest{
		TenantUUID:    tenantUUID,
		StaffCodeUUID: staffCodeUUID,
		WelcomeMode:   acqmodel.WelcomeModeSend,
		ContentBlocks: datatypes.JSON([]byte(`[{"type":"text","text":"hello"}]`)),
	})
	require.NoError(t, err)

	result, err := svc.TriggerSync(context.Background(), tenantUUID, staffCodeUUID, "")
	require.NoError(t, err)
	require.Equal(t, acqmodel.WelcomeSyncStatusSuccess, result.SyncStatus)
	require.Equal(t, 1, result.AttemptNo)
	require.Contains(t, result.Message, "welcome_code")

	status, err := svc.GetSyncStatus(context.Background(), tenantUUID, staffCodeUUID)
	require.NoError(t, err)
	require.Equal(t, acqmodel.WelcomeSyncStatusSuccess, status.SyncStatus)
	require.Equal(t, 1, status.LatestAttemptNo)
	require.Empty(t, status.LastSyncError)

	attempts, err := repos.StaffWelcomeAttempt.ListByStaffCodeUUID(context.Background(), tenantUUID, staffCodeUUID, 10)
	require.NoError(t, err)
	require.Len(t, attempts, 1)
}

func openStaffWelcomeIntegrationDB(t *testing.T, name string) *gorm.DB {
	t.Helper()
	basemodels.ForceSchemaForTests("")
	db, err := gorm.Open(sqlite.Open("file:"+name+"?mode=memory&cache=shared"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	require.NoError(t, err)
	require.NoError(t, ensureStaffWelcomeIntegrationTables(db))
	return db
}

func ensureStaffWelcomeIntegrationTables(db *gorm.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS acquisition_staff_live_codes (
			staff_code_uuid TEXT PRIMARY KEY,
			tenant_uuid TEXT NOT NULL,
			channel TEXT NOT NULL,
			app_type TEXT NOT NULL,
			channel_account_uuid TEXT NOT NULL,
			activity_name TEXT NOT NULL,
			code_key TEXT NOT NULL,
			state TEXT,
			config_id TEXT,
			qr_code TEXT,
			member_uuids TEXT NOT NULL,
			corp_tag_ids TEXT NOT NULL,
			new_customer_remark_enabled BOOLEAN NOT NULL DEFAULT FALSE,
			status TEXT NOT NULL,
			created_by TEXT NOT NULL,
			updated_by TEXT NOT NULL,
			created_at DATETIME,
			updated_at DATETIME
		);`,
		`CREATE UNIQUE INDEX IF NOT EXISTS uq_acq_staff_codes_tenant_code_key
			ON acquisition_staff_live_codes (tenant_uuid, code_key);`,
		`CREATE TABLE IF NOT EXISTS acquisition_staff_welcome_configs (
			config_uuid TEXT PRIMARY KEY,
			tenant_uuid TEXT NOT NULL,
			staff_code_uuid TEXT NOT NULL,
			welcome_mode TEXT NOT NULL,
			content_blocks TEXT NOT NULL,
			payload_preview TEXT NOT NULL,
			sync_status TEXT NOT NULL,
			last_sync_error TEXT,
			last_synced_at DATETIME,
			version INTEGER NOT NULL,
			created_by TEXT NOT NULL,
			updated_by TEXT NOT NULL,
			created_at DATETIME,
			updated_at DATETIME
		);`,
		`CREATE UNIQUE INDEX IF NOT EXISTS uq_acq_staff_welcome_code
			ON acquisition_staff_welcome_configs (staff_code_uuid);`,
		`CREATE TABLE IF NOT EXISTS acquisition_staff_welcome_sync_attempts (
			attempt_uuid TEXT PRIMARY KEY,
			tenant_uuid TEXT NOT NULL,
			staff_code_uuid TEXT NOT NULL,
			config_version INTEGER NOT NULL,
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

func seedStaffWelcomeIntegrationData(db *gorm.DB, tenantUUID, staffCodeUUID string, now time.Time) error {
	return db.Exec(
		`INSERT INTO acquisition_staff_live_codes
			(staff_code_uuid, tenant_uuid, channel, app_type, channel_account_uuid, activity_name, code_key, member_uuids, corp_tag_ids, new_customer_remark_enabled, status, created_by, updated_by, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		staffCodeUUID,
		tenantUUID,
		"wechat",
		"wecom",
		"22222222-2222-4222-8222-222222222411",
		"欢迎语重试",
		"staff-retry-001",
		`["11111111-1111-4111-8111-111111111411"]`,
		`[]`,
		false,
		"active",
		"system",
		"system",
		now,
		now,
	).Error
}
