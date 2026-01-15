package marketplace

import (
	"context"
	"fmt"
	"testing"

	dbm "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/marketplace"
	mrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/marketplace"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type stubVendorGuard struct {
	revoked bool
}

func (s stubVendorGuard) VendorRevoked(ctx context.Context, vendorID string) (bool, error) {
	return s.revoked, nil
}

func setupServiceDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:%s-%s?mode=memory&cache=private", t.Name(), uuid.NewString())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)

	stmts := []string{
		`CREATE TABLE IF NOT EXISTS marketplace_listings (
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
        );`,
		`CREATE TABLE IF NOT EXISTS marketplace_listing_assets (
            id TEXT PRIMARY KEY,
            listing_id TEXT NOT NULL,
            tenant_uuid TEXT NOT NULL,
            asset_type TEXT NOT NULL,
            storage_uri TEXT NOT NULL,
            checksum TEXT,
            is_primary INTEGER DEFAULT 0,
            locale TEXT,
            weight INTEGER DEFAULT 0,
            metadata TEXT,
            created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
            updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
        );`,
		`CREATE TABLE IF NOT EXISTS marketplace_listing_versions (
            id TEXT PRIMARY KEY,
            listing_id TEXT NOT NULL,
            tenant_uuid TEXT NOT NULL,
            version TEXT NOT NULL,
            changelog TEXT,
            metadata TEXT,
            submitted_by TEXT NOT NULL,
            review_state TEXT,
            reviewer_id TEXT,
            reviewed_at DATETIME,
            created_at DATETIME DEFAULT CURRENT_TIMESTAMP
        );`,
		`CREATE TABLE IF NOT EXISTS marketplace_pricing_plans (
            id TEXT PRIMARY KEY,
            listing_id TEXT NOT NULL,
            tenant_uuid TEXT NOT NULL,
            plan_code TEXT NOT NULL,
            plan_type TEXT NOT NULL,
            currency TEXT NOT NULL,
            created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
            updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
            amount REAL,
            billing_period TEXT,
            trial_period_days INTEGER,
            quota_limit REAL,
            overage_policy TEXT,
            feature_matrix TEXT,
            is_default INTEGER DEFAULT 0,
            status TEXT DEFAULT 'active'
        );`,
		`CREATE TABLE IF NOT EXISTS marketplace_plan_tiers (
            id TEXT PRIMARY KEY,
            plan_id TEXT NOT NULL,
            tenant_uuid TEXT NOT NULL,
            metric TEXT NOT NULL,
            range_from REAL NOT NULL,
            unit_amount REAL NOT NULL,
            range_to REAL,
            unit_name TEXT,
            created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
            updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
        );`,
		`CREATE TABLE IF NOT EXISTS marketplace_checklist_runs (
            id TEXT PRIMARY KEY,
            listing_id TEXT NOT NULL,
            tenant_uuid TEXT NOT NULL,
            trigger_source TEXT NOT NULL,
            run_number INTEGER NOT NULL,
            status TEXT NOT NULL,
            summary TEXT,
            started_at DATETIME,
            completed_at DATETIME,
            ci_pipeline_id TEXT,
            created_at DATETIME DEFAULT CURRENT_TIMESTAMP
        );`,
		`CREATE TABLE IF NOT EXISTS marketplace_checklist_items (
            id TEXT PRIMARY KEY,
            checklist_run_id TEXT NOT NULL,
            tenant_uuid TEXT NOT NULL,
            code TEXT NOT NULL,
            description TEXT NOT NULL,
            result TEXT NOT NULL,
            evidence_uri TEXT,
            notes TEXT,
            auto_fix_link TEXT,
            created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
            updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
        );`,
		`CREATE TABLE IF NOT EXISTS marketplace_licenses (
            id TEXT PRIMARY KEY,
            tenant_uuid TEXT NOT NULL,
            listing_id TEXT NOT NULL,
            plan_id TEXT NOT NULL,
            license_token TEXT NOT NULL,
            status TEXT NOT NULL,
            issued_at DATETIME NOT NULL,
            expires_at DATETIME NOT NULL,
            renewal_token TEXT,
            offline_until DATETIME,
            last_validated_at DATETIME,
            issued_by TEXT,
            metadata TEXT,
            created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
            updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
        );`,
		`CREATE TABLE IF NOT EXISTS marketplace_license_events (
            id TEXT PRIMARY KEY,
            tenant_uuid TEXT NOT NULL,
            license_id TEXT NOT NULL,
            event_type TEXT NOT NULL,
            event_payload TEXT,
            emitted_at DATETIME DEFAULT CURRENT_TIMESTAMP,
            actor_id TEXT,
            trace_id TEXT,
            created_at DATETIME DEFAULT CURRENT_TIMESTAMP
        );`,
	}
	for _, stmt := range stmts {
		require.NoError(t, db.Exec(stmt).Error)
	}
	return db
}

func newService(t *testing.T) (*ListingService, *mrepo.ListingRepository, *mrepo.ChecklistRepository) {
	db := setupServiceDB(t)
	listingRepo := mrepo.NewListingRepository(db)
	checklistRepo := mrepo.NewChecklistRepository(db)
	svc := NewListingService(listingRepo, checklistRepo, logrus.New().WithField("test", "listing_service"))
	return svc, listingRepo, checklistRepo
}

func TestCreateAndUpdateDraft(t *testing.T) {
	svc, repo, _ := newService(t)
	ctx := context.Background()

	input := ListingDraftInput{
		PluginID:   "com.powerx.plugins.base",
		VendorID:   "vendor-1",
		Title:      "Base Plugin",
		Slug:       "base-plugin",
		Summary:    "summary",
		Locale:     "en",
		Categories: []string{"automation"},
		Tags:       []string{"beta"},
	Assets: []ListingAssetInput{
		{
			AssetType:  "cover",
			StorageURI: "https://cdn.example/cover.png",
			IsPrimary:  true,
			Locale:     "en",
			Metadata: map[string]any{
				"width":  1920,
				"height": 1080,
			},
		},
	},
	}

	listing, err := svc.CreateDraft(ctx, "tenant-1", input)
	require.NoError(t, err)
	require.Equal(t, "Base Plugin", listing.Title)

	updated, err := svc.UpdateDraft(ctx, "tenant-1", listing.ID, ListingUpdateInput{
		Title:      ptr("Updated Title"),
		Summary:    ptr("updated summary"),
		Categories: &[]string{"automation", "analytics"},
	Assets: []ListingAssetInput{
		{
			AssetType:  "cover",
			StorageURI: "https://cdn.example/cover2.png",
			IsPrimary:  true,
			Locale:     "en",
			Metadata: map[string]any{
				"width":  1920,
				"height": 1080,
			},
		},
	},
	})
	require.NoError(t, err)
	require.Equal(t, "Updated Title", updated.Title)

	stored, err := repo.FindByID(ctx, "tenant-1", listing.ID)
	require.NoError(t, err)
	require.Len(t, stored.Assets, 1)
	require.Equal(t, "https://cdn.example/cover2.png", stored.Assets[0].StorageURI)
}

func TestSubmitReviewAndPublish(t *testing.T) {
	svc, repo, _ := newService(t)
	ctx := context.Background()

	listing, err := svc.CreateDraft(ctx, "tenant-1", ListingDraftInput{
		PluginID: "com.powerx.plugins.base",
		VendorID: "vendor-1",
		Title:    "Listing",
		Slug:     "listing",
	Assets: []ListingAssetInput{
		{
			AssetType:  "cover",
			StorageURI: "https://cdn.example/cover.png",
			IsPrimary:  true,
			Locale:     "en",
			Metadata: map[string]any{
				"width":  1920,
				"height": 1080,
			},
		},
	},
	})
	require.NoError(t, err)

	svc.SetVendorGuard(stubVendorGuard{revoked: false})

	listing, err = svc.SubmitForReview(ctx, "tenant-1", listing.ID, "vendor-1", map[string]any{"changelog": "initial"})
	require.NoError(t, err)
	require.Equal(t, dbm.ListingStatusInReview, listing.Status)

	published, err := svc.UpdateListingStatus(ctx, "tenant-1", listing.ID, "reviewer", dbm.ListingStatusPublished, "ok")
	require.NoError(t, err)
	require.Equal(t, dbm.ListingStatusPublished, published.Status)
	require.NotNil(t, published.PublishedAt)

	versions := []dbm.ListingVersion{}
	require.NoError(t, repo.BaseRepository.DB.WithContext(ctx).Find(&versions).Error)
	require.Len(t, versions, 1)
}

func TestVendorGuardBlocksSubmission(t *testing.T) {
	svc, _, _ := newService(t)
	ctx := context.Background()

	listing, err := svc.CreateDraft(ctx, "tenant-1", ListingDraftInput{
		PluginID: "com.powerx.plugins.base",
		VendorID: "vendor-1",
		Title:    "Listing",
		Slug:     "listing",
		Assets: []ListingAssetInput{
			{
				AssetType:  "cover",
				StorageURI: "https://cdn.example/cover.png",
				IsPrimary:  true,
				Locale:     "en",
				Metadata: map[string]any{
					"width":  1920,
					"height": 1080,
				},
			},
		},
	})
	require.NoError(t, err)

	svc.SetVendorGuard(stubVendorGuard{revoked: true})
	_, err = svc.SubmitForReview(ctx, "tenant-1", listing.ID, "vendor-1", nil)
	require.Error(t, err)
}

func TestChecklistRunUpdatesScore(t *testing.T) {
	svc, repo, _ := newService(t)
	ctx := context.Background()

	listing, err := svc.CreateDraft(ctx, "tenant-1", ListingDraftInput{
		PluginID: "com.powerx.plugins.base",
		VendorID: "vendor-1",
		Title:    "Listing",
		Slug:     "listing",
		Assets: []ListingAssetInput{
			{
				AssetType:  "cover",
				StorageURI: "https://cdn.example/cover.png",
				Locale:     "en",
				Metadata: map[string]any{
					"width":  1920,
					"height": 1080,
				},
			},
		},
	})
	require.NoError(t, err)

	_, err = svc.RecordChecklistRun(ctx, "tenant-1", listing.ID, ChecklistRunInput{
		TriggerSource: "ci",
		Items: []ChecklistItemInput{
			{Code: "ASSET", Description: "Asset", Result: "passed"},
			{Code: "METADATA", Description: "Metadata", Result: "failed"},
		},
	})
	require.NoError(t, err)

	refreshed, err := repo.FindByID(ctx, "tenant-1", listing.ID)
	require.NoError(t, err)
	require.Equal(t, 50, refreshed.ReadyChecklistScore)
}

func ptr[T any](value T) *T {
	return &value
}

func TestCreateDraftRejectsInvalidVideoAssets(t *testing.T) {
	svc, _, _ := newService(t)
	ctx := context.Background()

	_, err := svc.CreateDraft(ctx, "tenant-1", ListingDraftInput{
		PluginID: "com.powerx.plugins.base",
		VendorID: "vendor-1",
		Title:    "Listing",
		Slug:     "listing",
		Assets: []ListingAssetInput{
			{
				AssetType:  "video",
				StorageURI: "https://cdn.example/sample.mp4",
				Metadata: map[string]any{
					"duration_seconds": 20,
					"size_mb":          60,
				},
			},
		},
	})
	require.Error(t, err)
}
