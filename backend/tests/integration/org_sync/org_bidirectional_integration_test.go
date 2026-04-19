package org_sync_test

import (
	"context"
	"strconv"
	"testing"

	basemodels "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	iammodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/iam"
	orgmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/org_sync"
	socialmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/social_channel_governance"
	orgrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/org_sync"
	orgsvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/org_sync"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestOrgSingleMasterPullWritesIAMAndBindings(t *testing.T) {
	db := openOrgBidirectionalDB(t, "org_bidirectional_single_master_pull")
	tenantUUID := "00000000-0000-0000-0000-000000000001"
	sourceAccountUUID := "30aa4d9f-6768-4fdf-97c8-66844422a7a5"
	channelAccountUUID := "30aa4d9f-6768-4fdf-97c8-66844422a7a5"

	require.NoError(t, db.Create(&orgmodel.SourceAccount{
		SourceAccountUUID:  sourceAccountUUID,
		TenantUUID:         tenantUUID,
		Provider:           "wechat",
		AppType:            "wecom",
		ChannelAccountUUID: &channelAccountUUID,
		DisplayName:        "default",
		Status:             orgmodel.SourceAccountStatusActive,
	}).Error)
	rootExternal := "1"
	require.NoError(t, db.Create(&orgmodel.SourceUnit{
		SourceUnitUUID:     "unit-root-uuid",
		TenantUUID:         tenantUUID,
		SourceAccountUUID:  sourceAccountUUID,
		ChannelAccountUUID: channelAccountUUID,
		ExternalUnitID:     rootExternal,
		Name:               "总部门",
		Order:              1,
		Status:             "active",
	}).Error)
	parentExternal := "101"
	require.NoError(t, db.Create(&orgmodel.SourceUnit{
		SourceUnitUUID:       "unit-101-uuid",
		TenantUUID:           tenantUUID,
		SourceAccountUUID:    sourceAccountUUID,
		ChannelAccountUUID:   channelAccountUUID,
		ExternalUnitID:       parentExternal,
		ParentExternalUnitID: &rootExternal,
		Name:                 "销售部",
		Order:                2,
		Status:               "active",
	}).Error)
	require.NoError(t, db.Create(&orgmodel.SourceMember{
		SourceMemberUUID:   "member-001-uuid",
		TenantUUID:         tenantUUID,
		SourceAccountUUID:  sourceAccountUUID,
		ChannelAccountUUID: channelAccountUUID,
		ExternalMemberID:   "wosdnEDAAA001",
		Name:               "张三",
		Phone:              "13800001111",
		Email:              "zhangsan@example.com",
		ProfileStatus:      orgmodel.ProfileStatusFull,
		Status:             "active",
	}).Error)
	require.NoError(t, db.Create(&orgmodel.SourceMemberUnit{
		SourceMemberUnitUUID: "member-unit-001",
		TenantUUID:           tenantUUID,
		SourceAccountUUID:    sourceAccountUUID,
		ChannelAccountUUID:   channelAccountUUID,
		SourceMemberUUID:     "member-001-uuid",
		SourceUnitUUID:       "unit-101-uuid",
		ExternalMemberID:     "wosdnEDAAA001",
		ExternalUnitID:       parentExternal,
		Order:                1,
	}).Error)

	svc := orgsvc.NewSyncService(
		orgrepo.NewSourceAccountRepository(db),
		orgrepo.NewSourceUnitRepository(db),
		orgrepo.NewSourceMemberRepository(db),
		orgrepo.NewSyncLogRepository(db),
		nil,
		nil,
		nil,
	)
	require.NoError(t, svc.SyncIAMAndBindingsFromPull(context.Background(), tenantUUID, channelAccountUUID))

	var departments []iammodel.Department
	require.NoError(t, db.Where("tenant_uuid = ?", tenantUUID).Order("id asc").Find(&departments).Error)
	require.GreaterOrEqual(t, len(departments), 2)

	var unitBinding orgmodel.UnitBinding
	require.NoError(t, db.Where("tenant_uuid = ? AND channel_account_uuid = ? AND external_unit_id = ?", tenantUUID, channelAccountUUID, parentExternal).First(&unitBinding).Error)
	require.NotEmpty(t, unitBinding.MainUnitID)
	mainDeptID, err := strconv.ParseUint(unitBinding.MainUnitID, 10, 64)
	require.NoError(t, err)

	var memberBinding orgmodel.MemberBinding
	require.NoError(t, db.Where("tenant_uuid = ? AND channel_account_uuid = ? AND external_member_id = ?", tenantUUID, channelAccountUUID, "wosdnEDAAA001").First(&memberBinding).Error)
	require.NotEmpty(t, memberBinding.MainMemberID)
	mainMemberID, err := strconv.ParseUint(memberBinding.MainMemberID, 10, 64)
	require.NoError(t, err)

	var member iammodel.Member
	require.NoError(t, db.Where("id = ? AND tenant_uuid = ?", mainMemberID, tenantUUID).First(&member).Error)
	require.Equal(t, "wosdnEDAAA001", member.Username)
	require.NotNil(t, member.DepartmentID)
	require.Equal(t, mainDeptID, *member.DepartmentID)

	var checkpoint socialmodel.SyncCheckpoint
	require.NoError(t, db.Where("tenant_uuid = ? AND domain = ? AND direction = ?", tenantUUID, socialmodel.SyncDomainOrg, "pull").First(&checkpoint).Error)
	require.NotEmpty(t, checkpoint.Cursor)
	require.NotEmpty(t, checkpoint.SnapshotVersion)
}

func TestOrgSingleMasterPushBindingBackfillTimestamps(t *testing.T) {
	db := openOrgBidirectionalDB(t, "org_bidirectional_single_master_push")
	tenantUUID := "00000000-0000-0000-0000-000000000001"
	channelAccountUUID := "30aa4d9f-6768-4fdf-97c8-66844422a7a5"
	svc := orgsvc.NewSyncService(orgrepo.NewSourceAccountRepository(db), nil, nil, nil, nil, nil, nil)

	require.NoError(t, svc.UpsertUnitBinding(context.Background(), tenantUUID, channelAccountUUID, "1001", "201", "1", "synced", true, false))
	require.NoError(t, svc.UpsertMemberBinding(context.Background(), tenantUUID, channelAccountUUID, "2001", "wx_user_2001", "synced", true, false))

	var unitBinding orgmodel.UnitBinding
	require.NoError(t, db.Where("tenant_uuid = ? AND channel_account_uuid = ? AND main_unit_id = ? AND external_unit_id = ?", tenantUUID, channelAccountUUID, "1001", "201").First(&unitBinding).Error)
	require.NotNil(t, unitBinding.LastPushedAt)

	var memberBinding orgmodel.MemberBinding
	require.NoError(t, db.Where("tenant_uuid = ? AND channel_account_uuid = ? AND main_member_id = ? AND external_member_id = ?", tenantUUID, channelAccountUUID, "2001", "wx_user_2001").First(&memberBinding).Error)
	require.NotNil(t, memberBinding.LastPushedAt)
}

func openOrgBidirectionalDB(t *testing.T, name string) *gorm.DB {
	t.Helper()
	basemodels.ForceSchemaForTests("")
	db, err := gorm.Open(sqlite.Open("file:"+name+"?mode=memory&cache=shared"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE IF NOT EXISTS org_sync_source_accounts (
		source_account_uuid TEXT PRIMARY KEY,
		tenant_uuid TEXT NOT NULL,
		provider TEXT NOT NULL,
		app_type TEXT NOT NULL,
		channel_account_uuid TEXT,
		display_name TEXT NOT NULL,
		status TEXT NOT NULL,
		last_sync_at DATETIME,
		last_sync_status TEXT,
		last_sync_message TEXT,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	);`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE IF NOT EXISTS org_sync_source_units (
		source_unit_uuid TEXT PRIMARY KEY,
		tenant_uuid TEXT NOT NULL,
		source_account_uuid TEXT NOT NULL,
		channel_account_uuid TEXT NOT NULL,
		external_unit_id TEXT NOT NULL,
		parent_external_unit_id TEXT,
		name TEXT NOT NULL,
		"order" INTEGER NOT NULL DEFAULT 0,
		status TEXT NOT NULL,
		created_at DATETIME,
		updated_at DATETIME
	);`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE IF NOT EXISTS org_sync_source_members (
		source_member_uuid TEXT PRIMARY KEY,
		tenant_uuid TEXT NOT NULL,
		source_account_uuid TEXT NOT NULL,
		channel_account_uuid TEXT NOT NULL,
		external_member_id TEXT NOT NULL,
		name TEXT NOT NULL,
		phone TEXT,
		email TEXT,
		profile_status TEXT,
		status TEXT NOT NULL,
		created_at DATETIME,
		updated_at DATETIME
	);`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE IF NOT EXISTS org_sync_source_member_units (
		source_member_unit_uuid TEXT PRIMARY KEY,
		tenant_uuid TEXT NOT NULL,
		source_account_uuid TEXT NOT NULL,
		channel_account_uuid TEXT NOT NULL,
		source_member_uuid TEXT NOT NULL,
		source_unit_uuid TEXT NOT NULL,
		external_member_id TEXT NOT NULL,
		external_unit_id TEXT NOT NULL,
		"order" INTEGER NOT NULL DEFAULT 0,
		created_at DATETIME,
		updated_at DATETIME
	);`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE IF NOT EXISTS org_sync_unit_bindings (
		unit_binding_uuid TEXT PRIMARY KEY,
		tenant_uuid TEXT NOT NULL,
		channel_account_uuid TEXT NOT NULL,
		main_unit_id TEXT NOT NULL,
		external_unit_id TEXT NOT NULL,
		parent_external_unit_id TEXT,
		sync_status TEXT,
		last_pulled_at DATETIME,
		last_pushed_at DATETIME,
		created_at DATETIME,
		updated_at DATETIME
	);`).Error)
	require.NoError(t, db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS uq_org_sync_unit_bindings_identity
		ON org_sync_unit_bindings (tenant_uuid, channel_account_uuid, main_unit_id, external_unit_id);`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE IF NOT EXISTS org_sync_member_bindings (
		member_binding_uuid TEXT PRIMARY KEY,
		tenant_uuid TEXT NOT NULL,
		channel_account_uuid TEXT NOT NULL,
		main_member_id TEXT NOT NULL,
		external_member_id TEXT NOT NULL,
		sync_status TEXT,
		last_pulled_at DATETIME,
		last_pushed_at DATETIME,
		created_at DATETIME,
		updated_at DATETIME
	);`).Error)
	require.NoError(t, db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS uq_org_sync_member_bindings_identity
		ON org_sync_member_bindings (tenant_uuid, channel_account_uuid, main_member_id, external_member_id);`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE IF NOT EXISTS iam_departments (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		tenant_uuid TEXT NOT NULL,
		name TEXT NOT NULL,
		code TEXT NOT NULL,
		parent_id INTEGER,
		description TEXT,
		path TEXT NOT NULL,
		sort_order INTEGER,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	);`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE IF NOT EXISTS iam_users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		email TEXT,
		phone TEXT,
		display_name TEXT,
		avatar_url TEXT,
		status TEXT NOT NULL,
		is_root BOOLEAN NOT NULL DEFAULT 0,
		password_hash TEXT NOT NULL,
		meta TEXT,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	);`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE IF NOT EXISTS iam_members (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		tenant_uuid TEXT NOT NULL,
		user_id INTEGER NOT NULL,
		username TEXT NOT NULL,
		display_name TEXT,
		avatar_url TEXT,
		status TEXT NOT NULL,
		department_id INTEGER,
		meta TEXT,
		last_login_at DATETIME,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	);`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE IF NOT EXISTS social_sync_checkpoints (
		checkpoint_uuid TEXT PRIMARY KEY,
		tenant_uuid TEXT NOT NULL,
		domain TEXT NOT NULL,
		direction TEXT NOT NULL,
		cursor TEXT,
		snapshot_version TEXT,
		last_event_time DATETIME,
		updated_at DATETIME,
		created_at DATETIME
	);`).Error)
	require.NoError(t, db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS uq_social_sync_checkpoints_scope
		ON social_sync_checkpoints (tenant_uuid, domain, direction);`).Error)
	return db
}
