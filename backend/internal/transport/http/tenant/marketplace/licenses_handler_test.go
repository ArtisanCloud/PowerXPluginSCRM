package marketplace

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/config"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/contracts"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	dbm "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/marketplace"
	mrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/marketplace"
	adminmetrics "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/observability/admin_console"
	opsmetrics "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/observability/operations"
	svc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/marketplace"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupLicenseHandlerDeps(t *testing.T) (*app.Deps, *gorm.DB) {
	t.Helper()
	models.ForceSchemaForTests("")
	db, err := gorm.Open(sqlite.Open("file:tenant_license_handler?mode=memory&cache=shared"), &gorm.Config{})
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
	for _, stmt := range statements {
		require.NoError(t, db.Exec(stmt).Error)
	}

	deps := &app.Deps{
		DB:                  db,
		Ctx:                 context.Background(),
		Config:              &config.Config{},
		MarketplaceBilling:  &handlerBillingStub{responseID: "bill-handler"},
		LicenseAuthority:    &handlerAuthorityStub{issueResp: &svc.LicenseIssueResponse{Token: "handler-token", ExpiresAt: time.Now().Add(24 * time.Hour)}},
		LicenseCache:        &handlerCacheStub{},
		OperationsMetrics:   opsmetrics.NewMetrics(),
		AdminConsoleMetrics: adminmetrics.NewMetrics(),
	}
	return deps, db
}

func TestLicenseHandler_Flow(t *testing.T) {
	gin.SetMode(gin.TestMode)
	deps, db := setupLicenseHandlerDeps(t)

	pricingRepo := mrepo.NewPricingRepository(db)
	amount := 9.99
	plan := &dbm.PricingPlan{
		TenantUuid:  "1",
		ListingID: "listing-1",
		PlanCode:  "basic",
		PlanType:  dbm.PricingPlanTypeSubscription,
		Currency:  "USD",
		Amount:    &amount,
		Status:    "active",
	}
	require.NoError(t, pricingRepo.CreatePlan(context.Background(), plan, nil))

	router := gin.New()
	RegisterRoutes(router.Group("/marketplace"), deps)

	// Issue license
	body := map[string]any{
		"listing_id":        "listing-1",
		"plan_id":           plan.ID,
		"payment_intent_id": "pi_test",
	}
	buf, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/marketplace/licenses?tenant_uuid=1", bytes.NewReader(buf))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var createResp contracts.APIResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &createResp))
	require.True(t, createResp.Success)
	dataMap := createResp.Data.(map[string]any)
	licenseID := dataMap["id"].(string)
	renewalToken := dataMap["renewal_token"].(string)
	require.NotEmpty(t, licenseID)
	require.NotEmpty(t, renewalToken)

	var dbLicense dbm.License
	require.NoError(t, db.Where("id = ?", licenseID).First(&dbLicense).Error)
	require.Equal(t, "handler-token", dbLicense.LicenseToken)

	// Renew license
	renewBody := map[string]any{
		"renewal_token": renewalToken,
	}
	renewBuf, _ := json.Marshal(renewBody)
	renewReq := httptest.NewRequest(http.MethodPost, "/marketplace/licenses/"+licenseID+"?tenant_uuid=1", bytes.NewReader(renewBuf))
	renewReq.Header.Set("Content-Type", "application/json")
	renewRec := httptest.NewRecorder()
	router.ServeHTTP(renewRec, renewReq)
	require.Equal(t, http.StatusOK, renewRec.Code)

	// Extend offline
	extendBody := map[string]any{"requested_hours": 6}
	extendBuf, _ := json.Marshal(extendBody)
	extendReq := httptest.NewRequest(http.MethodPost, "/marketplace/licenses/"+licenseID+"/offline-extend?tenant_uuid=1", bytes.NewReader(extendBuf))
	extendReq.Header.Set("Content-Type", "application/json")
	extendRec := httptest.NewRecorder()
	router.ServeHTTP(extendRec, extendReq)
	require.Equal(t, http.StatusOK, extendRec.Code)

	// Fetch license detail
	getReq := httptest.NewRequest(http.MethodGet, "/marketplace/licenses/"+licenseID+"?tenant_uuid=1", nil)
	getRec := httptest.NewRecorder()
	router.ServeHTTP(getRec, getReq)
	require.Equal(t, http.StatusOK, getRec.Code)
}

type handlerBillingStub struct {
	responseID string
	calls      int
}

func (s *handlerBillingStub) ChargeSubscription(ctx context.Context, tenantID string, plan *dbm.PricingPlan, metadata map[string]any) (string, error) {
	s.calls++
	return s.responseID, nil
}

type handlerAuthorityStub struct {
	issueResp *svc.LicenseIssueResponse
}

func (s *handlerAuthorityStub) Issue(ctx context.Context, req *svc.LicenseIssueRequest) (*svc.LicenseIssueResponse, error) {
	if s.issueResp == nil {
		return &svc.LicenseIssueResponse{}, nil
	}
	return s.issueResp, nil
}

func (s *handlerAuthorityStub) Renew(ctx context.Context, req *svc.LicenseRenewRequest) (*svc.LicenseIssueResponse, error) {
	if s.issueResp == nil {
		return &svc.LicenseIssueResponse{}, nil
	}
	return &svc.LicenseIssueResponse{Token: s.issueResp.Token + "-renewed", ExpiresAt: time.Now().Add(24 * time.Hour)}, nil
}

func (s *handlerAuthorityStub) Revoke(ctx context.Context, licenseID string, reason string) error {
	return nil
}

func (s *handlerAuthorityStub) Verify(ctx context.Context, token string) (bool, error) {
	return true, nil
}

type handlerCacheStub struct{}

func (h *handlerCacheStub) Get(ctx context.Context, tenantID, listingID string) (*dbm.License, bool) {
	return nil, false
}

func (h *handlerCacheStub) Set(ctx context.Context, tenantID, listingID string, license *dbm.License, ttl time.Duration) {
}

func (h *handlerCacheStub) Delete(ctx context.Context, tenantID, listingID string) {}
