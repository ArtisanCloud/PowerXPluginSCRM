package marketplace

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/config"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	dbm "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/marketplace"
	mrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/marketplace"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupLicenseServiceDB(t *testing.T, name string) *gorm.DB {
	t.Helper()
	models.ForceSchemaForTests("")
	db, err := gorm.Open(sqlite.Open("file:"+name+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)

	statements := []string{
		`CREATE TABLE IF NOT EXISTS marketplace_pricing_plans (
            id TEXT PRIMARY KEY,
            tenant_uuid TEXT NOT NULL,
            listing_id TEXT NOT NULL,
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
            emitted_at DATETIME DEFAULT CURRENT_TIMESTAMP,
            actor_id TEXT,
            trace_id TEXT,
            created_at DATETIME DEFAULT CURRENT_TIMESTAMP
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
            created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
            updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
        );`,
	}
	for _, stmt := range statements {
		require.NoError(t, db.Exec(stmt).Error)
	}
	return db
}

type stubBillingClient struct {
	responseID string
	err        error
	lastTenant string
	lastPlan   string
	calls      int
}

func (s *stubBillingClient) ChargeSubscription(ctx context.Context, tenantID string, plan *dbm.PricingPlan, metadata map[string]any) (string, error) {
	s.calls++
	s.lastTenant = tenantID
	if plan != nil {
		s.lastPlan = plan.ID
	}
	if s.err != nil {
		return "", s.err
	}
	if s.responseID == "" {
		s.responseID = "billing-txn"
	}
	return s.responseID, nil
}

type stubAuthority struct {
	issueResp  *LicenseIssueResponse
	issueErr   error
	renewResp  *LicenseIssueResponse
	renewErr   error
	revokeErr  error
	verifyResp bool
	verifyErr  error
}

func (s *stubAuthority) Issue(ctx context.Context, req *LicenseIssueRequest) (*LicenseIssueResponse, error) {
	if s.issueResp == nil {
		s.issueResp = &LicenseIssueResponse{}
	}
	return s.issueResp, s.issueErr
}

func (s *stubAuthority) Renew(ctx context.Context, req *LicenseRenewRequest) (*LicenseIssueResponse, error) {
	if s.renewResp == nil {
		s.renewResp = &LicenseIssueResponse{}
	}
	return s.renewResp, s.renewErr
}

func (s *stubAuthority) Revoke(ctx context.Context, licenseID string, reason string) error {
	return s.revokeErr
}

func (s *stubAuthority) Verify(ctx context.Context, token string) (bool, error) {
	return s.verifyResp, s.verifyErr
}

type stubLicenseCache struct {
	setCalls []cacheEntry
}

type cacheEntry struct {
	tenantID  string
	listingID string
	ttl       time.Duration
	license   *dbm.License
}

func (s *stubLicenseCache) Get(ctx context.Context, tenantID, listingID string) (*dbm.License, bool) {
	return nil, false
}

func (s *stubLicenseCache) Set(ctx context.Context, tenantID, listingID string, license *dbm.License, ttl time.Duration) {
	s.setCalls = append(s.setCalls, cacheEntry{
		tenantID:  tenantID,
		listingID: listingID,
		ttl:       ttl,
		license:   license,
	})
}

func (s *stubLicenseCache) Delete(ctx context.Context, tenantID, listingID string) {}

type stubTaxAdapter struct {
	request     *TaxChargeRequest
	result      *TaxChargeResult
	dispatchErr error
	replayErr   error
}

func (s *stubTaxAdapter) Name() string {
	return "stub"
}

func (s *stubTaxAdapter) Dispatch(ctx context.Context, req *TaxChargeRequest) (*TaxChargeResult, error) {
	s.request = req
	return s.result, s.dispatchErr
}

func (s *stubTaxAdapter) Replay(ctx context.Context, externalTransactionID string) error {
	return s.replayErr
}

func TestLicenseService_IssueLicenseSuccess(t *testing.T) {
	db := setupLicenseServiceDB(t, "license_issue")
	pricingRepo := mrepo.NewPricingRepository(db)
	licenseRepo := mrepo.NewLicenseRepository(db)
	ctx := context.Background()

	amount := 29.99
	plan := &dbm.PricingPlan{
		TenantUuid:  "tenant-1",
		ListingID: "listing-1",
		PlanCode:  "pro",
		PlanType:  dbm.PricingPlanTypeSubscription,
		Currency:  "USD",
		Amount:    &amount,
	}
	require.NoError(t, pricingRepo.CreatePlan(ctx, plan, nil))
	require.NotEmpty(t, plan.ID)

	billing := &stubBillingClient{responseID: "bill-123"}
	authority := &stubAuthority{
		issueResp: &LicenseIssueResponse{
			Token:     "license-token-abc",
			ExpiresAt: time.Now().Add(48 * time.Hour),
			Metadata:  map[string]any{"authority": "central"},
		},
	}
	cache := &stubLicenseCache{}
	taxAdapter := &stubTaxAdapter{
		result: &TaxChargeResult{
			ExternalTransactionID: "tax-456",
			TaxAmountCents:        120,
			TaxAmountMinorUnits:   120,
			Currency:              "USD",
			SettlementCurrency:    "USD",
			ExchangeRate:          1,
			Jurisdiction:          "US-CA",
			RawPayload:            []byte(`{"id":"tax-456"}`),
		},
	}
	taxClient := &TaxProviderClient{
		provider: "stub",
		retries:  nil,
		adapter:  taxAdapter,
	}

	svc := NewLicenseService(&config.Config{}, pricingRepo, licenseRepo, taxClient, billing, authority, cache, testLogger())

	expiresAt := time.Now().Add(24 * time.Hour)
	license, err := svc.IssueLicense(ctx, IssueLicenseParams{
		TenantUuid:  "tenant-1",
		ListingID: "listing-1",
		PlanID:    plan.ID,
		IssuedBy:  "tester",
		Metadata:  map[string]any{"source": "unit-test"},
		ExpiresAt: expiresAt,
	})
	require.NoError(t, err)
	require.NotNil(t, license)
	require.Equal(t, "license-token-abc", license.LicenseToken)
	require.Equal(t, dbm.LicenseStatusActive, license.Status)
	require.WithinDuration(t, authority.issueResp.ExpiresAt, license.ExpiresAt, time.Second)
	require.NotNil(t, license.OfflineUntil)
	require.True(t, license.OfflineUntil.After(time.Now()))
	require.True(t, license.OfflineUntil.Before(time.Now().Add(73*time.Hour)))
	require.NotNil(t, license.LastValidatedAt)
	require.NotNil(t, license.RenewalToken)
	require.Equal(t, 1, billing.calls)

	require.Equal(t, "tenant-1", billing.lastTenant)
	require.Equal(t, plan.ID, billing.lastPlan)
	require.NotNil(t, taxAdapter.request)
	require.Equal(t, "tenant-1", taxAdapter.request.TenantUuid)

	events, err := licenseRepo.ListEvents(ctx, "tenant-1", license.ID, 5)
	require.NoError(t, err)
	require.Len(t, events, 1)
	require.Equal(t, dbm.LicenseEventIssued, events[0].EventType)
	require.Equal(t, "bill-123", events[0].EventPayload["billing_id"])
	require.Equal(t, "tax-456", events[0].EventPayload["tax_transaction_id"])

	var txnCount int64
	require.NoError(t, db.Model(&dbm.TaxTransaction{}).Count(&txnCount).Error)
	require.Equal(t, int64(1), txnCount)

	require.Len(t, cache.setCalls, 1)
	require.Equal(t, "tenant-1", cache.setCalls[0].tenantID)
	require.Equal(t, "listing-1", cache.setCalls[0].listingID)
	require.True(t, cache.setCalls[0].ttl > 0)
	settle, ok := license.Metadata["settlement_currency"].(string)
	require.True(t, ok)
	require.Equal(t, "USD", settle)
	rateVal, ok := license.Metadata["exchange_rate"].(float64)
	require.True(t, ok)
	require.InDelta(t, 1.0, rateVal, 1e-9)

	var txnRecord dbm.TaxTransaction
	require.NoError(t, db.First(&txnRecord).Error)
	require.Equal(t, "USD", txnRecord.Currency)
	require.Equal(t, "USD", txnRecord.SettlementCurrency)
	require.NotNil(t, txnRecord.ExchangeRate)
	require.InDelta(t, 1.2, txnRecord.TaxAmount, 1e-6)
	require.NotNil(t, txnRecord.TaxAmountSettlement)
	require.InDelta(t, 1.2, *txnRecord.TaxAmountSettlement, 1e-6)
	require.Equal(t, "bill-123", license.Metadata["billing_id"])
	require.Equal(t, "tax-456", license.Metadata["tax_transaction_id"])
}

func TestLicenseService_IssueLicenseSkipBilling(t *testing.T) {
	db := setupLicenseServiceDB(t, "license_skip")
	pricingRepo := mrepo.NewPricingRepository(db)
	licenseRepo := mrepo.NewLicenseRepository(db)
	ctx := context.Background()

	amount := 49.0
	plan := &dbm.PricingPlan{
		TenantUuid:  "tenant-1",
		ListingID: "listing-1",
		PlanCode:  "enterprise",
		PlanType:  dbm.PricingPlanTypeSubscription,
		Currency:  "USD",
		Amount:    &amount,
		Status:    "active",
	}
	require.NoError(t, pricingRepo.CreatePlan(ctx, plan, nil))

	billing := &stubBillingClient{}
	authority := &stubAuthority{
		issueResp: &LicenseIssueResponse{
			Token:     "skip-token",
			ExpiresAt: time.Now().Add(72 * time.Hour),
		},
	}
	cache := &stubLicenseCache{}
	logger := testLogger()

	service := NewLicenseService(&config.Config{}, pricingRepo, licenseRepo, nil, billing, authority, cache, logger)
	meta := map[string]any{"billing_id": "existing-billing-1"}

	license, err := service.IssueLicense(ctx, IssueLicenseParams{
		TenantUuid:    "tenant-1",
		ListingID:   "listing-1",
		PlanID:      plan.ID,
		IssuedBy:    "recovery-bot",
		Metadata:    meta,
		SkipBilling: true,
	})
	require.NoError(t, err)
	require.NotNil(t, license)
	require.Equal(t, "skip-token", license.LicenseToken)
	require.Equal(t, dbm.LicenseStatusActive, license.Status)
	require.Equal(t, "existing-billing-1", license.Metadata["billing_id"])
	require.Equal(t, 0, billing.calls, "billing client should not be invoked when SkipBilling is true")

	events, err := licenseRepo.ListEvents(ctx, "tenant-1", license.ID, 5)
	require.NoError(t, err)
	require.NotEmpty(t, events)
	payload := events[0].EventPayload
	require.Equal(t, "existing-billing-1", payload["billing_id"])

	require.Len(t, cache.setCalls, 1)
	require.Equal(t, "tenant-1", cache.setCalls[0].tenantID)
	require.Equal(t, "listing-1", cache.setCalls[0].listingID)
}

func TestLicenseService_RenewLicenseUpdatesExpiry(t *testing.T) {
	db := setupLicenseServiceDB(t, "license_renew")
	pricingRepo := mrepo.NewPricingRepository(db)
	licenseRepo := mrepo.NewLicenseRepository(db)
	ctx := context.Background()

	plan := &dbm.PricingPlan{
		TenantUuid:  "tenant-1",
		ListingID: "listing-1",
		PlanCode:  "pro",
		PlanType:  dbm.PricingPlanTypeSubscription,
		Currency:  "USD",
	}
	require.NoError(t, pricingRepo.CreatePlan(ctx, plan, nil))

	license := &dbm.License{
		TenantUuid:     "tenant-1",
		ListingID:    "listing-1",
		PlanID:       plan.ID,
		LicenseToken: "initial-token",
		Status:       dbm.LicenseStatusActive,
		IssuedAt:     time.Now().Add(-48 * time.Hour),
		ExpiresAt:    time.Now().Add(12 * time.Hour),
	}
	require.NoError(t, licenseRepo.CreateLicense(ctx, license, nil))

	cache := &stubLicenseCache{}
	authority := &stubAuthority{
		renewResp: &LicenseIssueResponse{
			Token:     "renewed-token",
			ExpiresAt: time.Now().Add(96 * time.Hour),
		},
	}

	svc := NewLicenseService(&config.Config{}, pricingRepo, licenseRepo, nil, nil, authority, cache, testLogger())

	updated, err := svc.RenewLicense(ctx, RenewLicenseParams{
		LicenseID: license.ID,
		TenantUuid:  "tenant-1",
		IssuedBy:  "tester",
		Metadata:  map[string]any{"initiator": "unit-test"},
	})
	require.NoError(t, err)
	require.Equal(t, "renewed-token", updated.LicenseToken)
	require.WithinDuration(t, authority.renewResp.ExpiresAt, updated.ExpiresAt, time.Second)

	events, err := licenseRepo.ListEvents(ctx, "tenant-1", license.ID, 5)
	require.NoError(t, err)
	require.NotEmpty(t, events)
	require.Equal(t, dbm.LicenseEventRenewed, events[0].EventType)

	require.Len(t, cache.setCalls, 1)
	require.Equal(t, license.ID, cache.setCalls[0].license.ID)
}

func TestLicenseService_ExtendOfflineClampsTo72Hours(t *testing.T) {
	db := setupLicenseServiceDB(t, "license_offline")
	pricingRepo := mrepo.NewPricingRepository(db)
	licenseRepo := mrepo.NewLicenseRepository(db)
	ctx := context.Background()

	plan := &dbm.PricingPlan{
		TenantUuid:  "tenant-1",
		ListingID: "listing-1",
		PlanCode:  "pro",
		PlanType:  dbm.PricingPlanTypeSubscription,
		Currency:  "USD",
	}
	require.NoError(t, pricingRepo.CreatePlan(ctx, plan, nil))

	license := &dbm.License{
		TenantUuid:     "tenant-1",
		ListingID:    "listing-1",
		PlanID:       plan.ID,
		LicenseToken: "token",
		Status:       dbm.LicenseStatusActive,
		IssuedAt:     time.Now().Add(-24 * time.Hour),
		ExpiresAt:    time.Now().Add(14 * 24 * time.Hour),
	}
	require.NoError(t, licenseRepo.CreateLicense(ctx, license, nil))

	svc := NewLicenseService(&config.Config{}, pricingRepo, licenseRepo, nil, nil, nil, nil, testLogger())

	target := time.Now().Add(120 * time.Hour)
	require.NoError(t, svc.ExtendOffline(ctx, "tenant-1", license.ID, target))

	stored, err := licenseRepo.GetLicense(ctx, "tenant-1", license.ID)
	require.NoError(t, err)
	require.NotNil(t, stored.OfflineUntil)
	max := time.Now().Add(72 * time.Hour)
	require.True(t, stored.OfflineUntil.Before(max) || stored.OfflineUntil.Equal(max))

	events, err := licenseRepo.ListEvents(ctx, "tenant-1", license.ID, 5)
	require.NoError(t, err)
	require.NotEmpty(t, events)
	require.Equal(t, dbm.LicenseEventOfflineExtend, events[0].EventType)
}

func TestLicenseService_IssueLicensePlanMismatch(t *testing.T) {
	db := setupLicenseServiceDB(t, "license_plan_mismatch")
	pricingRepo := mrepo.NewPricingRepository(db)
	licenseRepo := mrepo.NewLicenseRepository(db)
	ctx := context.Background()

	amount := 9.99
	plan := &dbm.PricingPlan{
		TenantUuid:  "tenant-1",
		ListingID: "listing-other",
		PlanCode:  "basic",
		PlanType:  dbm.PricingPlanTypeSubscription,
		Currency:  "USD",
		Amount:    &amount,
	}
	require.NoError(t, pricingRepo.CreatePlan(ctx, plan, nil))

	billing := &stubBillingClient{}
	svc := NewLicenseService(&config.Config{}, pricingRepo, licenseRepo, nil, billing, nil, nil, testLogger())

	_, err := svc.IssueLicense(ctx, IssueLicenseParams{
		TenantUuid:  "tenant-1",
		ListingID: "listing-1",
		PlanID:    plan.ID,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "does not belong")
	require.Equal(t, 0, billing.calls)
}

func TestLicenseService_IssueLicenseInactivePlan(t *testing.T) {
	db := setupLicenseServiceDB(t, "license_plan_inactive")
	pricingRepo := mrepo.NewPricingRepository(db)
	licenseRepo := mrepo.NewLicenseRepository(db)
	ctx := context.Background()

	amount := 9.99
	plan := &dbm.PricingPlan{
		TenantUuid:  "tenant-1",
		ListingID: "listing-1",
		PlanCode:  "basic",
		PlanType:  dbm.PricingPlanTypeSubscription,
		Currency:  "USD",
		Amount:    &amount,
		Status:    "inactive",
	}
	require.NoError(t, pricingRepo.CreatePlan(ctx, plan, nil))

	billing := &stubBillingClient{}
	svc := NewLicenseService(&config.Config{}, pricingRepo, licenseRepo, nil, billing, nil, nil, testLogger())

	_, err := svc.IssueLicense(ctx, IssueLicenseParams{
		TenantUuid:  "tenant-1",
		ListingID: "listing-1",
		PlanID:    plan.ID,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "not active")
	require.Equal(t, 0, billing.calls)
}

func TestLicenseService_IssueLicenseSkipsBillingForFreePlan(t *testing.T) {
	db := setupLicenseServiceDB(t, "license_free_plan")
	pricingRepo := mrepo.NewPricingRepository(db)
	licenseRepo := mrepo.NewLicenseRepository(db)
	ctx := context.Background()

	plan := &dbm.PricingPlan{
		TenantUuid:  "tenant-1",
		ListingID: "listing-1",
		PlanCode:  "free",
		PlanType:  dbm.PricingPlanTypeFree,
		Currency:  "USD",
	}
	require.NoError(t, pricingRepo.CreatePlan(ctx, plan, nil))

	billing := &stubBillingClient{}
	cache := &stubLicenseCache{}
	svc := NewLicenseService(&config.Config{}, pricingRepo, licenseRepo, nil, billing, nil, cache, testLogger())

	license, err := svc.IssueLicense(ctx, IssueLicenseParams{
		TenantUuid:  "tenant-1",
		ListingID: "listing-1",
		PlanID:    plan.ID,
		IssuedBy:  "tester",
		Trial:     true,
	})
	require.NoError(t, err)
	require.NotNil(t, license)
	require.Equal(t, 0, billing.calls)
	require.Nil(t, license.Metadata["billing_id"])
	require.Len(t, cache.setCalls, 1)
}

func TestLicenseService_TaxFailureRecordsTransaction(t *testing.T) {
	db := setupLicenseServiceDB(t, "license_tax_failure")
	pricingRepo := mrepo.NewPricingRepository(db)
	licenseRepo := mrepo.NewLicenseRepository(db)
	ctx := context.Background()

	amount := 15.0
	plan := &dbm.PricingPlan{
		TenantUuid:  "tenant-1",
		ListingID: "listing-1",
		PlanCode:  "std",
		PlanType:  dbm.PricingPlanTypeSubscription,
		Currency:  "USD",
		Amount:    &amount,
	}
	require.NoError(t, pricingRepo.CreatePlan(ctx, plan, nil))

	billing := &stubBillingClient{responseID: "billing-456"}
	cache := &stubLicenseCache{}
	taxAdapter := &stubTaxAdapter{dispatchErr: ErrNotImplemented}
	taxClient := &TaxProviderClient{
		provider: "stub",
		retries:  nil,
		adapter:  taxAdapter,
	}

	svc := NewLicenseService(&config.Config{}, pricingRepo, licenseRepo, taxClient, billing, nil, cache, testLogger())

	license, err := svc.IssueLicense(ctx, IssueLicenseParams{
		TenantUuid:  "tenant-1",
		ListingID: "listing-1",
		PlanID:    plan.ID,
		IssuedBy:  "tester",
	})
	require.NoError(t, err)
	require.NotNil(t, license)
	require.Equal(t, 1, billing.calls)

	var txn dbm.TaxTransaction
	require.NoError(t, db.Where("billing_id = ?", "billing-456").First(&txn).Error)
	require.Equal(t, "failed", txn.Status)
	require.Equal(t, "USD", txn.Currency)
	require.Equal(t, "USD", txn.SettlementCurrency)
	if txn.RawPayload != nil {
		require.Contains(t, fmt.Sprint(txn.RawPayload["error"]), "not implemented")
	}
}

func testLogger() *logrus.Entry {
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)
	return logger.WithField("suite", "license_service_test")
}
