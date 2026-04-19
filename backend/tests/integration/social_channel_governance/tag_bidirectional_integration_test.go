package social_channel_governance_test

import (
	"context"
	"testing"
	"time"

	basemodels "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	socialrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/social_channel_governance"
	socialsvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/social_channel_governance"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestTagBidirectionalSyncAndConflictQueue(t *testing.T) {
	db := openTagSyncIntegrationDB(t, "tag_bidirectional_integration")
	tenantUUID := "00000000-0000-0000-0000-000000000001"
	syncRepo := socialrepo.NewSyncFoundationRepository(db)
	tagRepo := socialrepo.NewTagMappingRepository(syncRepo)
	tagSvc := socialsvc.NewTagSyncService(tagRepo, socialsvc.NewConflictResolutionService(syncRepo))

	pullRes, err := tagSvc.SyncRemoteToLocal(context.Background(), tenantUUID,
		[]socialsvc.TagRecord{
			{TagID: "tag-a", Name: "远端-A", Version: "r1"},
			{TagID: "tag-b", Name: "远端-B", Version: "r1"},
		},
		[]socialsvc.TagRecord{
			{TagID: "tag-a", Name: "本地-A", Version: "l1"}, // 冲突
		},
		"cursor-pull-1",
	)
	require.NoError(t, err)
	require.Equal(t, 2, pullRes.Pulled)
	require.Equal(t, 1, pullRes.Conflicts)
	require.Equal(t, 1, pullRes.Created)
	require.Equal(t, 1, pullRes.Updated)
	require.Equal(t, "v1", pullRes.SnapshotVersion)

	pushRes, err := tagSvc.SyncLocalToRemote(context.Background(), tenantUUID,
		[]socialsvc.TagRecord{
			{TagID: "tag-c", Name: "本地-C", Version: "l1"},
		},
		[]socialsvc.TagRecord{
			{TagID: "tag-a", Name: "远端-A", Version: "r1"},
		},
		"cursor-push-1",
	)
	require.NoError(t, err)
	require.Equal(t, 1, pushRes.Pushed)
	require.Equal(t, 0, pushRes.Conflicts)
	require.Equal(t, 1, pushRes.Created)
	require.Equal(t, "v1", pushRes.SnapshotVersion)

	conflicts, err := syncRepo.ListConflicts(context.Background(), tenantUUID, "tags", "open", 20)
	require.NoError(t, err)
	require.Len(t, conflicts, 1)

	cpPull, err := tagRepo.GetCheckpoint(context.Background(), tenantUUID, "pull")
	require.NoError(t, err)
	require.Equal(t, "cursor-pull-1", cpPull.Cursor)
	require.Equal(t, "v1", cpPull.SnapshotVersion)
	require.WithinDuration(t, time.Now().UTC(), cpPull.LastEventTime, 2*time.Second)

	// no-op sync should keep snapshot version unchanged
	noopRes, err := tagSvc.SyncRemoteToLocal(context.Background(), tenantUUID,
		[]socialsvc.TagRecord{
			{TagID: "tag-a", Name: "远端-A", Version: "r1"},
			{TagID: "tag-b", Name: "远端-B", Version: "r1"},
		},
		[]socialsvc.TagRecord{
			{TagID: "tag-a", Name: "远端-A", Version: "r1"},
			{TagID: "tag-b", Name: "远端-B", Version: "r1"},
		},
		"",
	)
	require.NoError(t, err)
	require.Equal(t, 0, noopRes.Pulled)
	require.Equal(t, 0, noopRes.Conflicts)
	require.Equal(t, "v1", noopRes.SnapshotVersion)
	require.Equal(t, "cursor-pull-1", noopRes.Cursor)
}

func openTagSyncIntegrationDB(t *testing.T, name string) *gorm.DB {
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
	require.NoError(t, db.Exec(`CREATE TABLE IF NOT EXISTS social_sync_checkpoints (
		checkpoint_uuid TEXT PRIMARY KEY,
		tenant_uuid TEXT NOT NULL,
		domain TEXT NOT NULL,
		direction TEXT NOT NULL,
		cursor TEXT,
		snapshot_version TEXT,
		last_event_time DATETIME,
		created_at DATETIME,
		updated_at DATETIME
	);`).Error)
	require.NoError(t, db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS uq_social_sync_checkpoints_scope
		ON social_sync_checkpoints (tenant_uuid, domain, direction);`).Error)
	return db
}
