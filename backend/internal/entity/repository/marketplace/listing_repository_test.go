package marketplace

import (
	"context"
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	dbm "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/marketplace"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupMarketplaceTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	models.ForceSchemaForTests("")
	db, err := gorm.Open(sqlite.Open("file:marketplace_tests?mode=memory&cache=shared"), &gorm.Config{})
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
            created_at DATETIME,
            updated_at DATETIME,
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
            created_at DATETIME,
            updated_at DATETIME
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
            created_at DATETIME
        );`,
		`CREATE TABLE IF NOT EXISTS marketplace_pricing_plans (
            id TEXT PRIMARY KEY,
            listing_id TEXT NOT NULL,
            tenant_uuid TEXT NOT NULL,
            plan_code TEXT NOT NULL,
            plan_type TEXT NOT NULL,
            currency TEXT NOT NULL,
            amount REAL,
            billing_period TEXT,
            trial_period_days INTEGER,
            quota_limit REAL,
            overage_policy TEXT,
            feature_matrix TEXT,
            is_default INTEGER DEFAULT 0,
            status TEXT DEFAULT 'active',
            created_at DATETIME,
            updated_at DATETIME
        );`,
		`CREATE TABLE IF NOT EXISTS marketplace_plan_tiers (
            id TEXT PRIMARY KEY,
            plan_id TEXT NOT NULL,
            tenant_uuid TEXT NOT NULL,
            metric TEXT NOT NULL,
            range_from REAL NOT NULL,
            range_to REAL,
            unit_amount REAL NOT NULL,
            unit_name TEXT,
            created_at DATETIME,
            updated_at DATETIME
        );`,
		`CREATE TABLE IF NOT EXISTS marketplace_checklist_runs (
            id TEXT PRIMARY KEY,
            listing_id TEXT NOT NULL,
            tenant_uuid TEXT NOT NULL,
            trigger_source TEXT NOT NULL,
            run_number INTEGER NOT NULL,
            status TEXT NOT NULL,
            started_at DATETIME,
            completed_at DATETIME,
            summary TEXT,
            ci_pipeline_id TEXT,
            created_at DATETIME
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
            created_at DATETIME,
            updated_at DATETIME
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
            created_at DATETIME,
            updated_at DATETIME
        );`,
		`CREATE TABLE IF NOT EXISTS marketplace_license_events (
            id TEXT PRIMARY KEY,
            tenant_uuid TEXT NOT NULL,
            license_id TEXT NOT NULL,
            event_type TEXT NOT NULL,
            event_payload TEXT,
            emitted_at DATETIME,
            actor_id TEXT,
            trace_id TEXT,
            created_at DATETIME
        );`,
		`CREATE TABLE IF NOT EXISTS marketplace_tax_transactions (
            id TEXT PRIMARY KEY,
            tenant_uuid TEXT NOT NULL,
            billing_id TEXT NOT NULL,
            external_provider TEXT NOT NULL,
            external_transaction_id TEXT,
            jurisdiction TEXT,
            tax_amount REAL NOT NULL,
            currency TEXT NOT NULL,
            settlement_currency TEXT,
            exchange_rate REAL,
            tax_amount_settlement REAL,
            raw_payload TEXT,
            status TEXT NOT NULL,
            synced_at DATETIME,
            created_at DATETIME,
            updated_at DATETIME
        );`,
	}
	for _, stmt := range stmts {
		require.NoError(t, db.Exec(stmt).Error)
	}
	return db
}

func TestListingRepository_CreateAndFind(t *testing.T) {
	db := setupMarketplaceTestDB(t)
	repo := NewListingRepository(db)
	ctx := context.Background()

	listing := &dbm.Listing{
		ID:         "listing-create",
		TenantUuid:   "1",
		PluginID:   "com.example.plugin",
		VendorID:   "vendor-1",
		Title:      "Test Listing",
		Slug:       "test-listing",
		Status:     dbm.ListingStatusDraft,
		Categories: []string{"ai", "automation"},
		Tags:       []string{"beta"},
	}

	require.NoError(t, repo.Create(ctx, listing))
	require.NotEmpty(t, listing.ID)

	fetched, err := repo.FindByID(ctx, "1", listing.ID)
	require.NoError(t, err)
	require.Equal(t, "Test Listing", fetched.Title)
	require.ElementsMatch(t, []string{"ai", "automation"}, fetched.Categories)
}

func TestListingRepository_ReplaceAssets(t *testing.T) {
	db := setupMarketplaceTestDB(t)
	repo := NewListingRepository(db)
	ctx := context.Background()

	listing := &dbm.Listing{
		ID:       "listing-asset",
		TenantUuid: "1",
		PluginID: "com.example.plugin",
		VendorID: "vendor-1",
		Title:    "Listing",
		Slug:     "listing",
	}
	require.NoError(t, repo.Create(ctx, listing))

	assets := []dbm.ListingAsset{
		{ID: "asset-1", AssetType: dbm.AssetTypeLogo, StorageURI: "s3://logo.png", IsPrimary: true},
		{ID: "asset-2", AssetType: dbm.AssetTypeScreenshot, StorageURI: "s3://shot.png"},
	}
	require.NoError(t, repo.ReplaceAssets(ctx, "1", listing.ID, assets))

	fetched, err := repo.FindByID(ctx, "1", listing.ID)
	require.NoError(t, err)
	require.Len(t, fetched.Assets, 2)

	newAssets := []dbm.ListingAsset{
		{ID: "asset-3", AssetType: dbm.AssetTypeCover, StorageURI: "s3://cover.png", IsPrimary: true},
	}
	require.NoError(t, repo.ReplaceAssets(ctx, "1", listing.ID, newAssets))

	fetched, err = repo.FindByID(ctx, "1", listing.ID)
	require.NoError(t, err)
	require.Len(t, fetched.Assets, 1)
	require.Equal(t, dbm.AssetTypeCover, fetched.Assets[0].AssetType)
}

func TestListingRepository_ListFilters(t *testing.T) {
	db := setupMarketplaceTestDB(t)
	repo := NewListingRepository(db)
	ctx := context.Background()

	listings := []*dbm.Listing{
		{ID: "listing-alpha", TenantUuid: "1", PluginID: "p1", VendorID: "v1", Title: "Alpha", Slug: "alpha", Status: dbm.ListingStatusDraft},
		{ID: "listing-beta", TenantUuid: "1", PluginID: "p2", VendorID: "v1", Title: "Beta", Slug: "beta", Status: dbm.ListingStatusPublished},
		{ID: "listing-gamma", TenantUuid: "1", PluginID: "p3", VendorID: "v1", Title: "Gamma", Slug: "gamma", Status: dbm.ListingStatusInReview},
	}
	for _, l := range listings {
		require.NoError(t, repo.Create(ctx, l))
	}

	result, total, err := repo.List(ctx, "1", ListingQuery{
		Status: []string{dbm.ListingStatusPublished},
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, result, 1)
	require.Equal(t, "Beta", result[0].Title)
}

func TestListingRepository_ReplacePricingPlans(t *testing.T) {
	db := setupMarketplaceTestDB(t)
	repo := NewListingRepository(db)
	ctx := context.Background()

	listing := &dbm.Listing{
		ID:       "listing-pricing",
		TenantUuid: "1",
		PluginID: "plugin",
		VendorID: "vendor",
		Title:    "Pricing Test",
		Slug:     "pricing-test",
	}
	require.NoError(t, repo.Create(ctx, listing))

	amount := 199.0
	plan := dbm.PricingPlan{
		ID:       "plan-1",
		PlanCode: "pro",
		PlanType: dbm.PricingPlanTypeSubscription,
		Currency: "USD",
		Amount:   &amount,
		Tiers: []dbm.PlanTier{
			{ID: "tier-1", Metric: "requests", RangeFrom: 0, RangeTo: floatPtr(1000), UnitAmount: 0.0, UnitName: "req"},
			{ID: "tier-2", Metric: "requests", RangeFrom: 1000, RangeTo: nil, UnitAmount: 0.05, UnitName: "req"},
		},
	}

	require.NoError(t, repo.ReplacePricingPlans(ctx, "1", listing.ID, []dbm.PricingPlan{plan}))

	fetched, err := repo.FindByID(ctx, "1", listing.ID)
	require.NoError(t, err)
	require.Len(t, fetched.PricingPlans, 1)
	require.Len(t, fetched.PricingPlans[0].Tiers, 2)
}

func floatPtr(v float64) *float64 {
	return &v
}
