package recommendation

import (
	"context"
	"testing"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	dbm "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/marketplace"
	mrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/marketplace"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type stubMetricsProvider struct {
	signals []Signal
}

func (s stubMetricsProvider) FetchSignals(ctx context.Context, tenantID string) ([]Signal, error) {
	return s.signals, nil
}

func setupListingRepo(t *testing.T) *mrepo.ListingRepository {
	t.Helper()
	models.ForceSchemaForTests("")
	db, err := gorm.Open(sqlite.Open("file:recommendation_engine?mode=memory&cache=shared"), &gorm.Config{})
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
        created_at DATETIME,
        updated_at DATETIME,
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
	listing := &dbm.Listing{
		ID:        "listing-1",
		TenantUuid:  "tenant-1",
		PluginID:  "com.example.plugin",
		VendorID:  "vendor-1",
		Status:    dbm.ListingStatusPublished,
		Title:     "Example",
		Slug:      "example",
		CreatedAt: time.Now().Add(-48 * time.Hour),
		UpdatedAt: time.Now().Add(-12 * time.Hour),
	}
	require.NoError(t, repo.Create(context.Background(), listing))
	return repo
}

func TestRefreshRecommendations(t *testing.T) {
	repo := setupListingRepo(t)
	provider := stubMetricsProvider{signals: []Signal{
		{
			ListingID:           "listing-1",
			InstallCount:        1200,
			RatingAverage:       4.6,
			RatingCount:         230,
			LastPublishedAt:     time.Now().Add(-72 * time.Hour),
			ReadyChecklistScore: 92,
			AvgResponseMs:       450,
			BrandCompleteness:   0.9,
			CreatedAt:           time.Now().Add(-15 * 24 * time.Hour),
		},
	}}

	engine := NewEngine(repo, provider, logrus.New().WithField("test", "recommendation"))
	result, err := engine.RefreshRecommendations(context.Background(), "tenant-1")
	require.NoError(t, err)
	require.Equal(t, 1, result.UpdatedCount)
	require.Greater(t, result.AverageWeight, 0.0)

	listing, err := repo.FindByID(context.Background(), "tenant-1", "listing-1")
	require.NoError(t, err)
	require.InDelta(t, result.AverageWeight, listing.RecommendedWeight, 1e-4)
}

func TestExplorationShare(t *testing.T) {
	repo := setupListingRepo(t)
	provider := stubMetricsProvider{signals: []Signal{
		{
			ListingID:           "listing-1",
			InstallCount:        10,
			RatingAverage:       4.0,
			RatingCount:         5,
			LastPublishedAt:     time.Now(),
			ReadyChecklistScore: 80,
			AvgResponseMs:       520,
			BrandCompleteness:   0.75,
			CreatedAt:           time.Now().Add(-48 * time.Hour),
		},
	}}

	engine := NewEngine(repo, provider, logrus.New().WithField("test", "exploration"))
	result, err := engine.RefreshRecommendations(context.Background(), "tenant-1")
	require.NoError(t, err)
	require.Equal(t, 1, result.UpdatedCount)
	require.InDelta(t, 1.0, result.ExplorationShare, 1e-6)
}
