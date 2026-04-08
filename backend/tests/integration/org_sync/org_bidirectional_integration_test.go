package org_sync_test

import (
	"context"
	"testing"

	basemodels "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	socialrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/social_channel_governance"
	orgsvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/org_sync"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestOrgBidirectionalAndConflictQueue(t *testing.T) {
	db := openOrgBidirectionalDB(t, "org_bidirectional_integration")
	openworkRepo := socialrepo.NewOpenWorkFoundationRepository(db)
	svc := orgsvc.NewSyncService(nil, nil, nil, nil, nil, nil, openworkRepo, nil, nil)
	tenantUUID := "00000000-0000-0000-0000-000000000001"

	pullRes, err := svc.SyncOrgRemoteToLocal(context.Background(), tenantUUID, "__dry_run__")
	require.NoError(t, err)
	require.Equal(t, "pull", pullRes.Direction)
	require.Equal(t, "incremental", pullRes.Mode)

	pushRes, err := svc.SyncOrgLocalToRemote(context.Background(), tenantUUID, "source-001", []orgsvc.OrgWritebackChange{
		{
			EntityType: "department",
			EntityID:   "dept-100",
			Action:     "move",
			Payload: map[string]any{
				"conflict": true,
				"from":     1,
				"to":       2,
			},
		},
		{
			EntityType: "member",
			EntityID:   "user-200",
			Action:     "update",
			Payload: map[string]any{
				"conflict": false,
				"name":     "张三",
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, "push", pushRes.Direction)
	require.Equal(t, 1, pushRes.Conflicts)
	require.Equal(t, 1, pushRes.Applied)

	syncRepo := socialrepo.NewSyncFoundationRepository(db)
	conflicts, err := syncRepo.ListConflicts(context.Background(), tenantUUID, "org", "open", 20)
	require.NoError(t, err)
	require.Len(t, conflicts, 1)
}

func openOrgBidirectionalDB(t *testing.T, name string) *gorm.DB {
	t.Helper()
	basemodels.ForceSchemaForTests("")
	db, err := gorm.Open(sqlite.Open("file:"+name+"?mode=memory&cache=shared"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE IF NOT EXISTS social_sync_conflicts (
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
	);`).Error)
	return db
}
