package social_channel_governance_test

import (
	"context"
	"testing"
	"time"

	basemodels "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	leadmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/lead_capture"
	socialsvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/social_channel_governance"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestBaselineRepairIntegration_RepairDirtyData(t *testing.T) {
	tenantUUID := "00000000-0000-0000-0000-000000000001"
	db := openBaselineRepairIntegrationDB(t, "baseline_repair_integration")
	now := time.Now().UTC()

	require.NoError(t, db.Exec(`
		INSERT INTO org_sync_source_members (
			source_member_uuid, tenant_uuid, source_account_uuid, channel_account_uuid,
			external_member_id, name, created_at, updated_at
		) VALUES
			('11111111-1111-4111-8111-111111111111', ?, 'acc-1', 'ch-1', 'ext-member-1', '张三旧', ?, ?),
			('22222222-2222-4222-8222-222222222222', ?, 'acc-1', 'ch-1', 'ext-member-1', '张三新', ?, ?)
	`, tenantUUID, now.Add(-2*time.Hour), now.Add(-2*time.Hour), tenantUUID, now, now).Error)

	require.NoError(t, db.Exec(`
		INSERT INTO social_sync_conflicts (
			conflict_uuid, tenant_uuid, domain, entity_type, entity_key, status, created_at, updated_at
		) VALUES
			('33333333-3333-4333-8333-333333333333', ?, 'tags', 'tag', 'tag-key-1', 'open', ?, ?),
			('44444444-4444-4444-8444-444444444444', ?, 'tags', 'tag', 'tag-key-1', 'open', ?, ?)
	`, tenantUUID, now.Add(-3*time.Hour), now.Add(-3*time.Hour), tenantUUID, now, now).Error)

	require.NoError(t, db.Exec(`
		INSERT INTO lead_capture_activities (
			activity_uuid, lead_uuid, tenant_uuid, activity_type, payload, created_at, updated_at
		) VALUES
			('55555555-5555-4555-8555-555555555555', 'lead-1', ?, 'sync_trace', '{"external_lead_id":"bad id with space"}', ?, ?),
			('66666666-6666-4666-8666-666666666666', 'lead-2', ?, 'sync_trace', '{"external_lead_id":"ok_userid_1"}', ?, ?)
	`, tenantUUID, now, now, tenantUUID, now, now).Error)

	svc := socialsvc.NewBaselineRepairService(db)
	result, err := svc.Repair(context.Background(), tenantUUID)
	require.NoError(t, err)
	require.Equal(t, 1, result.DuplicateMembersFixed)
	require.Equal(t, 1, result.DuplicateTagsFixed)
	require.Equal(t, 1, result.InvalidExternalUserFixed)

	var memberCount int64
	require.NoError(t, db.Table("org_sync_source_members").
		Where("tenant_uuid = ? AND source_account_uuid = ? AND external_member_id = ?", tenantUUID, "acc-1", "ext-member-1").
		Count(&memberCount).Error)
	require.Equal(t, int64(1), memberCount)

	var resolvedCount int64
	require.NoError(t, db.Table("social_sync_conflicts").
		Where("tenant_uuid = ? AND domain = ? AND status = ?", tenantUUID, "tags", "resolved").
		Count(&resolvedCount).Error)
	require.Equal(t, int64(1), resolvedCount)

	var activity leadmodel.LeadActivity
	require.NoError(t, db.Where("activity_uuid = ?", "55555555-5555-4555-8555-555555555555").First(&activity).Error)
	require.Equal(t, "", activity.Payload["external_lead_id"])
	require.Equal(t, "invalid_external_userid_fixed", activity.Payload["repair_mark"])
}

func openBaselineRepairIntegrationDB(t *testing.T, name string) *gorm.DB {
	t.Helper()
	basemodels.ForceSchemaForTests("")
	db, err := gorm.Open(sqlite.Open("file:"+name+"?mode=memory&cache=shared"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	require.NoError(t, err)

	stmts := []string{
		`CREATE TABLE IF NOT EXISTS org_sync_source_members (
			source_member_uuid TEXT PRIMARY KEY,
			tenant_uuid TEXT NOT NULL,
			source_account_uuid TEXT NOT NULL,
			channel_account_uuid TEXT NOT NULL,
			external_member_id TEXT NOT NULL,
			name TEXT NOT NULL,
			phone TEXT,
			email TEXT,
			profile_status TEXT,
			status TEXT,
			created_at DATETIME,
			updated_at DATETIME
		);`,
		`CREATE TABLE IF NOT EXISTS social_sync_conflicts (
			conflict_uuid TEXT PRIMARY KEY,
			tenant_uuid TEXT NOT NULL,
			domain TEXT NOT NULL,
			entity_type TEXT,
			entity_key TEXT,
			resolution_strategy TEXT,
			status TEXT NOT NULL,
			local_value TEXT,
			remote_value TEXT,
			resolved_by TEXT,
			resolved_at DATETIME,
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
	}
	for _, stmt := range stmts {
		require.NoError(t, db.Exec(stmt).Error)
	}
	return db
}
