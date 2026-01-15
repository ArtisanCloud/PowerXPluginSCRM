package marketplace

import (
	"context"
	"testing"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	dbm "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/marketplace"
	mrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/marketplace"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/recommendation"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type stubProvider struct {
	signals map[string][]recommendation.Signal
}

func (s stubProvider) FetchSignals(ctx context.Context, tenantID string) ([]recommendation.Signal, error) {
	return s.signals[tenantID], nil
}

func setupListingRepoForJob(t *testing.T) *mrepo.ListingRepository {
	t.Helper()
	models.ForceSchemaForTests("")
	db, err := gorm.Open(sqlite.Open("file:recommendation_job?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`DROP TABLE IF EXISTS marketplace_listings`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE marketplace_listings (
        id TEXT PRIMARY KEY,
        tenant_uuid TEXT NOT NULL,
        plugin_id TEXT NOT NULL,
        vendor_id TEXT NOT NULL,
        status TEXT NOT NULL,
        title TEXT NOT NULL,
        slug TEXT NOT NULL,
        summary TEXT,
        description TEXT,
        cover_asset_id TEXT,
        hero_video_asset_id TEXT,
        categories TEXT,
        tags TEXT,
        locale TEXT,
        version TEXT,
        ready_checklist_score INTEGER DEFAULT 0,
        recommended_weight REAL DEFAULT 0,
        published_at DATETIME,
        reviewed_at DATETIME,
        reviewer_id TEXT,
        audit_notes TEXT,
        branding_theme TEXT,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
        updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
        deleted_at DATETIME
    )`).Error)
	require.NoError(t, db.Exec(`DROP TABLE IF EXISTS marketplace_listing_assets`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE marketplace_listing_assets (
        id TEXT PRIMARY KEY,
        listing_id TEXT NOT NULL,
        tenant_uuid TEXT NOT NULL,
        asset_type TEXT,
        storage_uri TEXT,
        created_at DATETIME,
        updated_at DATETIME
    )`).Error)
	require.NoError(t, db.Exec(`DROP TABLE IF EXISTS marketplace_pricing_plans`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE marketplace_pricing_plans (
        id TEXT PRIMARY KEY,
        tenant_uuid TEXT NOT NULL,
        listing_id TEXT NOT NULL,
        plan_code TEXT,
        plan_type TEXT,
        currency TEXT,
        amount REAL,
        created_at DATETIME,
        updated_at DATETIME
    )`).Error)
	repo := mrepo.NewListingRepository(db)
	seed := []*dbm.Listing{
		{ID: "l1", TenantUuid: "tenant-a", PluginID: "p", VendorID: "v", Status: dbm.ListingStatusPublished, Title: "A", Slug: "a"},
		{ID: "l2", TenantUuid: "tenant-b", PluginID: "p", VendorID: "v", Status: dbm.ListingStatusDraft, Title: "B", Slug: "b"},
	}
	for _, listing := range seed {
		require.NoError(t, repo.Create(context.Background(), listing))
	}
	return repo
}

func TestSyncJobRefreshesWeights(t *testing.T) {
	repo := setupListingRepoForJob(t)
	provider := stubProvider{signals: map[string][]recommendation.Signal{
		"tenant-a": {
			{ListingID: "l1", ReadyChecklistScore: 90},
		},
	}}
	job := NewSyncJob(nil, repo, provider, logrus.New().WithField("test", "sync"), repo.ListTenantUuids)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	go job.Run(ctx)
	<-ctx.Done()

	listing, err := repo.FindByID(context.Background(), "tenant-a", "l1")
	require.NoError(t, err)
	require.NotZero(t, listing.RecommendedWeight)
}
