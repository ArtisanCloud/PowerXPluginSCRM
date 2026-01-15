package marketplace

import (
	"context"
	"testing"
	"time"

	dbm "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/marketplace"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestLicenseRepository_CreateLicenseAndEvent(t *testing.T) {
	db := setupMarketplaceTestDB(t)
	repo := NewLicenseRepository(db)
	ctx := context.Background()

	license := &dbm.License{
		TenantUuid:     "tenant-1",
		ListingID:    "listing-1",
		PlanID:       "plan-1",
		LicenseToken: "token-abc",
		Status:       dbm.LicenseStatusActive,
		IssuedAt:     time.Now().Add(-time.Hour),
		ExpiresAt:    time.Now().Add(24 * time.Hour),
	}
	event := &dbm.LicenseEvent{
		TenantUuid:  "tenant-1",
		EventType: dbm.LicenseEventIssued,
	}

	require.NoError(t, repo.CreateLicense(ctx, license, event))
	require.NotEmpty(t, license.ID)

	fetched, err := repo.GetLicense(ctx, "tenant-1", license.ID)
	require.NoError(t, err)
	require.Equal(t, "token-abc", fetched.LicenseToken)

	events, err := repo.ListEvents(ctx, "tenant-1", license.ID, 10)
	require.NoError(t, err)
	require.Len(t, events, 1)
	require.Equal(t, dbm.LicenseEventIssued, events[0].EventType)
	require.NotEmpty(t, events[0].ID)
}

func TestLicenseRepository_UpdateAndOfflineWindow(t *testing.T) {
	db := setupMarketplaceTestDB(t)
	repo := NewLicenseRepository(db)
	ctx := context.Background()

	license := &dbm.License{
		TenantUuid:     "tenant-1",
		ListingID:    "listing-1",
		PlanID:       "plan-1",
		LicenseToken: "initial",
		Status:       dbm.LicenseStatusActive,
		IssuedAt:     time.Now(),
		ExpiresAt:    time.Now().Add(2 * time.Hour),
	}
	require.NoError(t, repo.CreateLicense(ctx, license, nil))

	newExpiry := time.Now().Add(10 * time.Hour)
	fields := map[string]any{
		"expires_at": newExpiry,
		"status":     dbm.LicenseStatusActive,
	}
	require.NoError(t, repo.UpdateLicenseToken(ctx, "tenant-1", license.ID, fields))

	stored, err := repo.GetLicense(ctx, "tenant-1", license.ID)
	require.NoError(t, err)
	require.WithinDuration(t, newExpiry, stored.ExpiresAt, time.Second)

	offlineUntil := time.Now().Add(3 * time.Hour)
	require.NoError(t, repo.UpdateOfflineWindow(ctx, "tenant-1", license.ID, &offlineUntil))

	stored, err = repo.GetLicense(ctx, "tenant-1", license.ID)
	require.NoError(t, err)
	require.NotNil(t, stored.OfflineUntil)
	require.WithinDuration(t, offlineUntil, *stored.OfflineUntil, time.Second)
}

func TestLicenseRepository_RecordTaxTransaction(t *testing.T) {
	db := setupMarketplaceTestDB(t)
	repo := NewLicenseRepository(db)
	ctx := context.Background()

	exRate := 1.25
	settled := 15.625
	txn := &dbm.TaxTransaction{
		TenantUuid:            "tenant-1",
		BillingID:           "billing-123",
		ExternalProvider:    "stripe_tax",
		TaxAmount:           12.50,
		Currency:            "USD",
		SettlementCurrency:  "EUR",
		ExchangeRate:        &exRate,
		TaxAmountSettlement: &settled,
		Status:              "completed",
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}

	require.NoError(t, repo.RecordTaxTransaction(ctx, txn))
	require.NotEmpty(t, txn.ID)

	var count int64
	require.NoError(t, db.Model(&dbm.TaxTransaction{}).
		Where("tenant_uuid = ?", "tenant-1").
		Count(&count).Error)
	require.Equal(t, int64(1), count)

	var stored dbm.TaxTransaction
	require.NoError(t, db.First(&stored).Error)
	require.Equal(t, "EUR", stored.SettlementCurrency)
	require.NotNil(t, stored.ExchangeRate)
	require.InDelta(t, exRate, *stored.ExchangeRate, 1e-9)
	require.NotNil(t, stored.TaxAmountSettlement)
	require.InDelta(t, settled, *stored.TaxAmountSettlement, 1e-9)
}

func TestLicenseRepository_FindByBillingID(t *testing.T) {
	db := setupMarketplaceTestDB(t)
	repo := NewLicenseRepository(db)
	ctx := context.Background()

	first := &dbm.License{
		TenantUuid:     "tenant-1",
		ListingID:    "listing-1",
		PlanID:       "plan-1",
		LicenseToken: "token-a",
		Status:       dbm.LicenseStatusActive,
		IssuedAt:     time.Now(),
		ExpiresAt:    time.Now().Add(24 * time.Hour),
		Metadata: map[string]any{
			"billing_id": "billing-001",
		},
	}
	require.NoError(t, repo.CreateLicense(ctx, first, nil))

	second := &dbm.License{
		TenantUuid:     "tenant-1",
		ListingID:    "listing-2",
		PlanID:       "plan-2",
		LicenseToken: "token-b",
		Status:       dbm.LicenseStatusActive,
		IssuedAt:     time.Now(),
		ExpiresAt:    time.Now().Add(48 * time.Hour),
		Metadata: map[string]any{
			"billing_id": "billing-002",
		},
	}
	require.NoError(t, repo.CreateLicense(ctx, second, nil))

	found, err := repo.FindByBillingID(ctx, "tenant-1", "billing-002")
	require.NoError(t, err)
	require.Equal(t, second.ID, found.ID)

	_, err = repo.FindByBillingID(ctx, "tenant-1", "missing")
	require.Error(t, err)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func TestLicenseRepository_ListExpiringWithin(t *testing.T) {
	db := setupMarketplaceTestDB(t)
	repo := NewLicenseRepository(db)
	ctx := context.Background()
	require.NoError(t, db.Exec("DELETE FROM marketplace_licenses").Error)
	require.NoError(t, db.Exec("DELETE FROM marketplace_license_events").Error)

	now := time.Now()
	soon := &dbm.License{
		TenantUuid:     "tenant-1",
		ListingID:    "listing-1",
		PlanID:       "plan-1",
		LicenseToken: "soon",
		Status:       dbm.LicenseStatusActive,
		IssuedAt:     now.Add(-24 * time.Hour),
		ExpiresAt:    now.Add(12 * time.Hour),
	}
	require.NoError(t, repo.CreateLicense(ctx, soon, nil))

	later := &dbm.License{
		TenantUuid:     "tenant-1",
		ListingID:    "listing-2",
		PlanID:       "plan-2",
		LicenseToken: "later",
		Status:       dbm.LicenseStatusActive,
		IssuedAt:     now,
		ExpiresAt:    now.Add(7 * 24 * time.Hour),
	}
	require.NoError(t, repo.CreateLicense(ctx, later, nil))

	offline := &dbm.License{
		TenantUuid:     "tenant-1",
		ListingID:    "listing-3",
		PlanID:       "plan-3",
		LicenseToken: "offline",
		Status:       dbm.LicenseStatusTrial,
		IssuedAt:     now,
		ExpiresAt:    now.Add(5 * 24 * time.Hour),
		OfflineUntil: func() *time.Time {
			target := now.Add(10 * time.Hour)
			return &target
		}(),
	}
	require.NoError(t, repo.CreateLicense(ctx, offline, nil))

	results, err := repo.ListExpiringWithin(ctx, "tenant-1", 36*time.Hour)
	require.NoError(t, err)
	require.Len(t, results, 2)

	ids := map[string]bool{}
	for _, lic := range results {
		ids[lic.LicenseToken] = true
	}
	require.Contains(t, ids, "soon")
	require.Contains(t, ids, "offline")
	require.NotContains(t, ids, "later")
}
