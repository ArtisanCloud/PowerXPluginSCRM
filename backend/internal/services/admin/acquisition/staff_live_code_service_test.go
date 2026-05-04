package acquisition

import (
	"context"
	"testing"
	"time"

	acqrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/repository/acquisition"
	basemodels "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestStaffLiveCodeService_RequireConfirmedMemberMappings(t *testing.T) {
	db := openStaffLiveCodeServiceTestDB(t, "staff_live_code_service_test")
	tenantUUID := "00000000-0000-0000-0000-000000000311"
	memberA := "11111111-1111-4111-8111-111111111311"
	memberB := "11111111-1111-4111-8111-111111111312"
	now := time.Now().UTC()

	require.NoError(t, db.Exec(
		`INSERT INTO org_sync_member_bindings (member_binding_uuid, tenant_uuid, source_member_id, main_member_id, mapping_status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"map-a", tenantUUID, memberA, memberA, "confirmed", now, now,
	).Error)
	require.NoError(t, db.Exec(
		`INSERT INTO org_sync_member_bindings (member_binding_uuid, tenant_uuid, source_member_id, main_member_id, mapping_status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"map-b", tenantUUID, memberB, memberB, "pending", now, now,
	).Error)

	repo := acqrepo.NewBundle(db).StaffLiveCodes
	svc := NewStaffLiveCodeService(repo)

	first, err := svc.Create(context.Background(), StaffLiveCodeCreateRequest{
		TenantUUID:         tenantUUID,
		Channel:            "wechat",
		AppType:            "wecom",
		ChannelAccountUUID: "22222222-2222-4222-8222-222222222311",
		ActivityName:       "映射校验",
		CodeKey:            "mapping-check-001",
		MemberUUIDs:        []string{memberA, memberB},
	})
	require.NoError(t, err)
	require.NotEmpty(t, first.StaffCodeUUID)

	require.NoError(t, db.Exec(
		`UPDATE org_sync_member_bindings SET mapping_status = ? WHERE tenant_uuid = ? AND source_member_id = ?`,
		"confirmed", tenantUUID, memberB,
	).Error)
	created, err := svc.Create(context.Background(), StaffLiveCodeCreateRequest{
		TenantUUID:         tenantUUID,
		Channel:            "wechat",
		AppType:            "wecom",
		ChannelAccountUUID: "22222222-2222-4222-8222-222222222311",
		ActivityName:       "映射校验",
		CodeKey:            "mapping-check-002",
		MemberUUIDs:        []string{memberA, memberB},
	})
	require.NoError(t, err)
	require.NotEmpty(t, created.StaffCodeUUID)
	require.Equal(t, 2, len(created.MemberUUIDs))
}

func openStaffLiveCodeServiceTestDB(t *testing.T, name string) *gorm.DB {
	t.Helper()
	basemodels.ForceSchemaForTests("")
	db, err := gorm.Open(sqlite.Open("file:"+name+"?mode=memory&cache=shared"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	require.NoError(t, err)
	require.NoError(t, ensureStaffLiveCodeServiceTables(db))
	return db
}

func ensureStaffLiveCodeServiceTables(db *gorm.DB) error {
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
		`CREATE TABLE IF NOT EXISTS org_sync_member_bindings (
			member_binding_uuid TEXT PRIMARY KEY,
			tenant_uuid TEXT NOT NULL,
			channel_account_uuid TEXT,
			source_member_id TEXT NOT NULL,
			main_member_id TEXT NOT NULL,
			external_member_id TEXT,
			mapping_status TEXT NOT NULL,
			sync_status TEXT,
			created_at DATETIME,
			updated_at DATETIME
		);`,
	}
	for _, stmt := range stmts {
		if err := db.Exec(stmt).Error; err != nil {
			return err
		}
	}
	return nil
}
