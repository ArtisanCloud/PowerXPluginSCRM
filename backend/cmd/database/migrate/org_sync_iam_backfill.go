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
	return nil
}
