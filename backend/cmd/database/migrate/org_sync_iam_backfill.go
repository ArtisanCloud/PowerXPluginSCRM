package migrate

import (
	"context"

	orgsync "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/org_sync"
	"gorm.io/gorm"
)

// backfillOrgSyncIAMBindings migrates legacy mapping records into new org bindings tables.
func backfillOrgSyncIAMBindings(ctx context.Context, db *gorm.DB) error {
	if db == nil {
		return nil
	}
	if db.Dialector == nil || db.Dialector.Name() != "postgres" {
		return nil
	}
	if !db.Migrator().HasTable(&orgsync.UnitBinding{}) || !db.Migrator().HasTable(&orgsync.MemberBinding{}) {
		return nil
	}

	// unit mapping -> unit binding
	if db.Migrator().HasTable(&orgsync.UnitMapping{}) && db.Migrator().HasTable(&orgsync.SourceUnit{}) {
		if err := db.WithContext(ctx).Exec(`
INSERT INTO org_sync_unit_bindings (
  tenant_uuid,
  channel_account_uuid,
  main_unit_id,
  external_unit_id,
  parent_external_unit_id,
  sync_status,
  last_pulled_at,
  created_at,
  updated_at
)
SELECT
  um.tenant_uuid,
  su.channel_account_uuid,
  um.main_unit_id,
  su.external_unit_id,
  COALESCE(su.parent_external_unit_id, ''),
  CASE
    WHEN um.mapping_status = 'confirmed' THEN 'synced'
    WHEN um.mapping_status = 'conflict' THEN 'conflict'
    ELSE 'pending'
  END,
  NOW(),
  NOW(),
  NOW()
FROM org_sync_unit_mappings um
JOIN org_sync_source_units su ON su.source_unit_uuid = um.source_unit_uuid
WHERE COALESCE(um.main_unit_id, '') <> '' AND COALESCE(su.external_unit_id, '') <> ''
ON CONFLICT (tenant_uuid, channel_account_uuid, main_unit_id, external_unit_id)
DO UPDATE SET
  parent_external_unit_id = EXCLUDED.parent_external_unit_id,
  sync_status = EXCLUDED.sync_status,
  last_pulled_at = EXCLUDED.last_pulled_at,
  updated_at = NOW()
`).Error; err != nil {
			return err
		}
	}

	// member mapping -> member binding
	if db.Migrator().HasTable(&orgsync.MemberMapping{}) && db.Migrator().HasTable(&orgsync.SourceMember{}) {
		if err := db.WithContext(ctx).Exec(`
INSERT INTO org_sync_member_bindings (
  tenant_uuid,
  channel_account_uuid,
  main_member_id,
  external_member_id,
  sync_status,
  last_pulled_at,
  created_at,
  updated_at
)
SELECT
  mm.tenant_uuid,
  sm.channel_account_uuid,
  mm.main_member_id,
  sm.external_member_id,
  CASE
    WHEN mm.mapping_status = 'confirmed' THEN 'synced'
    WHEN mm.mapping_status = 'conflict' THEN 'conflict'
    ELSE 'pending'
  END,
  NOW(),
  NOW(),
  NOW()
FROM org_sync_member_mappings mm
JOIN org_sync_source_members sm ON sm.source_member_uuid = mm.source_member_uuid
WHERE COALESCE(mm.main_member_id, '') <> '' AND COALESCE(sm.external_member_id, '') <> ''
ON CONFLICT (tenant_uuid, channel_account_uuid, main_member_id, external_member_id)
DO UPDATE SET
  sync_status = EXCLUDED.sync_status,
  last_pulled_at = EXCLUDED.last_pulled_at,
  updated_at = NOW()
`).Error; err != nil {
			return err
		}
	}
	return nil
}
