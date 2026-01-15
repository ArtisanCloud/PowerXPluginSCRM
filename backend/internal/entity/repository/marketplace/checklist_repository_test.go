package marketplace

import (
	"context"
	"fmt"
	"testing"
	"time"

	dbm "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/marketplace"
	"github.com/stretchr/testify/require"
)

func TestChecklistRepository_CreateAndFetch(t *testing.T) {
	db := setupMarketplaceTestDB(t)
	repo := NewChecklistRepository(db)
	ctx := context.Background()

	run := &dbm.ChecklistRun{
		ID:            "run-1",
		ListingID:     "listing-1",
		TenantUuid:      "1",
		RunNumber:     1,
		TriggerSource: dbm.ChecklistTriggerVendor,
	}
	items := []dbm.ChecklistItem{
		{ID: "item-1", Code: "PKG_STRUCTURE", Description: "Package contains required files", Result: dbm.ChecklistStatusPassed},
		{ID: "item-2", Code: "DOCS_COMPLETE", Description: "Documentation present", Result: dbm.ChecklistStatusPassed},
	}

	require.NoError(t, repo.CreateRun(ctx, run, items))
	require.NotEmpty(t, run.ID)

	latest, err := repo.LatestRun(ctx, "1", "listing-1")
	require.NoError(t, err)
	require.Equal(t, 2, len(latest.Items))
	require.Equal(t, dbm.ChecklistStatusPending, latest.Status)
}

func TestChecklistRepository_UpdateRunResult(t *testing.T) {
	db := setupMarketplaceTestDB(t)
	repo := NewChecklistRepository(db)
	ctx := context.Background()

	run := &dbm.ChecklistRun{
		ID:            "run-2",
		ListingID:     "listing-2",
		TenantUuid:      "1",
		RunNumber:     1,
		TriggerSource: dbm.ChecklistTriggerCI,
	}
	require.NoError(t, repo.CreateRun(ctx, run, nil))

	completed := time.Now().UTC()
	require.NoError(t, repo.UpdateRunResult(ctx, run.ID, "1", dbm.ChecklistStatusPassed, "All good", &completed))

	latest, err := repo.LatestRun(ctx, "1", "listing-2")
	require.NoError(t, err)
	require.Equal(t, dbm.ChecklistStatusPassed, latest.Status)
	require.NotNil(t, latest.CompletedAt)
}

func TestChecklistRepository_ListRuns(t *testing.T) {
	db := setupMarketplaceTestDB(t)
	repo := NewChecklistRepository(db)
	ctx := context.Background()

	for i := 1; i <= 3; i++ {
		run := &dbm.ChecklistRun{
			ID:            fmt.Sprintf("run-3-%d", i),
			ListingID:     "listing-3",
			TenantUuid:      "1",
			RunNumber:     i,
			TriggerSource: dbm.ChecklistTriggerAuto,
		}
		require.NoError(t, repo.CreateRun(ctx, run, nil))
	}

	runs, err := repo.ListRuns(ctx, "1", "listing-3", 2)
	require.NoError(t, err)
	require.Len(t, runs, 2)
	require.Equal(t, 3, runs[0].RunNumber)
}
