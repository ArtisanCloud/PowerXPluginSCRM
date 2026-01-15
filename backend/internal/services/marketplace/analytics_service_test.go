package marketplace

import (
	"context"
	"testing"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/config"
	dbm "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/marketplace"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

type usageRepoStub struct {
	aggregates []*dbm.UsageAggregate
	lastUpsert *dbm.UsageAggregate
}

func (r *usageRepoStub) InsertEnvelopes(ctx context.Context, tenantID string, envelopes []*dbm.UsageEnvelope) (int, error) {
	return len(envelopes), nil
}

func (r *usageRepoStub) UpsertAggregate(ctx context.Context, aggregate *dbm.UsageAggregate) error {
	r.lastUpsert = aggregate
	r.aggregates = []*dbm.UsageAggregate{aggregate}
	return nil
}

func (r *usageRepoStub) ListAggregates(ctx context.Context, tenantID, licenseID string, window dbm.AggregationWindow) ([]*dbm.UsageAggregate, error) {
	return r.aggregates, nil
}

func (r *usageRepoStub) LatestAggregate(ctx context.Context, tenantID, licenseID, metric string, window dbm.AggregationWindow) (*dbm.UsageAggregate, error) {
	if len(r.aggregates) == 0 {
		return nil, nil
	}
	return r.aggregates[len(r.aggregates)-1], nil
}

func (r *usageRepoStub) GetAggregate(ctx context.Context, tenantID, licenseID, metric string, window dbm.AggregationWindow, bucket time.Time) (*dbm.UsageAggregate, error) {
	for _, agg := range r.aggregates {
		if agg.Metric == metric && agg.Window == window && agg.TimeBucket.Equal(bucket) {
			return agg, nil
		}
	}
	return nil, nil
}

func (r *usageRepoStub) DeleteEnvelopesBefore(ctx context.Context, tenantID, licenseID string, before time.Time) (int, error) {
	return 0, nil
}

func (r *usageRepoStub) DeleteAggregatesBefore(ctx context.Context, tenantID, licenseID string, before time.Time) (int, error) {
	return 0, nil
}

type revenueRepoStub struct {
	last *dbm.RevenueShareReport
	list []*dbm.RevenueShareReport
}

func (r *revenueRepoStub) UpsertReport(ctx context.Context, report *dbm.RevenueShareReport) error {
	r.last = report
	r.list = []*dbm.RevenueShareReport{report}
	return nil
}

func (r *revenueRepoStub) ListReports(ctx context.Context, tenantID, vendorID string, from, to time.Time) ([]*dbm.RevenueShareReport, error) {
	return r.list, nil
}

type notificationRepoStub struct {
	last *dbm.Notification
	list []*dbm.Notification
}

func (r *notificationRepoStub) QueueNotification(ctx context.Context, notification *dbm.Notification) error {
	r.last = notification
	r.list = append(r.list, notification)
	return nil
}

func (r *notificationRepoStub) ListByTenant(ctx context.Context, tenantID string) ([]*dbm.Notification, error) {
	return r.list, nil
}

type licenseRepoStub struct {
	license *dbm.License
}

func (r *licenseRepoStub) GetLicense(ctx context.Context, tenantID, licenseID string) (*dbm.License, error) {
	return r.license, nil
}

type pricingRepoStub struct {
	plan *dbm.PricingPlan
}

func (r *pricingRepoStub) GetPlan(ctx context.Context, tenantID, planID string) (*dbm.PricingPlan, error) {
	return r.plan, nil
}

type listingRepoStub struct {
	listing *dbm.Listing
}

func (r *listingRepoStub) FindByID(ctx context.Context, tenantID, listingID string) (*dbm.Listing, error) {
	return r.listing, nil
}

func TestAnalyticsService_BuildDashboardAlerts(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Hour)
	usageStub := &usageRepoStub{
		aggregates: []*dbm.UsageAggregate{
			{
				TenantUuid:   "tenant-1",
				LicenseID:  "license-1",
				Metric:     "calls",
				Window:     dbm.AggregationWindowDay,
				TimeBucket: now.Add(-24 * time.Hour),
				Total:      120,
				Delta:      120,
			},
			{
				TenantUuid:   "tenant-1",
				LicenseID:  "license-1",
				Metric:     "calls",
				Window:     dbm.AggregationWindowDay,
				TimeBucket: now,
				Total:      270,
				Delta:      150,
			},
		},
	}

	quota := 200.0
	cfg := &config.Config{
		Integration: &config.IntegrationConfig{
			Billing: config.IntegrationBillingConfig{
				Reconciliation: config.IntegrationRevenueSplitConfig{
					VendorShare:   0.8,
					PlatformShare: 0.15,
					FeeShare:      0.05,
					Currency:      "USD",
				},
			},
		},
	}
	listing := &dbm.Listing{ID: "listing-1", TenantUuid: "tenant-1", VendorID: "vendor-1"}
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)

	svc := NewAnalyticsService(
		cfg,
		usageStub,
		&revenueRepoStub{},
		&notificationRepoStub{},
		&licenseRepoStub{license: &dbm.License{ID: "license-1", TenantUuid: "tenant-1", ListingID: listing.ID, PlanID: "plan-1"}},
		&pricingRepoStub{plan: &dbm.PricingPlan{ID: "plan-1", TenantUuid: "tenant-1", QuotaLimit: &quota, Currency: "USD"}},
		&listingRepoStub{listing: listing},
		logrus.NewEntry(logger),
	)

	dashboard, err := svc.BuildDashboard(context.Background(), "tenant-1", "license-1", UsageDashboardQuery{
		Window: dbm.AggregationWindowDay,
		Metric: "calls",
	})
	require.NoError(t, err)
	require.Len(t, dashboard.Series, 2)
	require.NotEmpty(t, dashboard.Alerts)

	foundQuota := false
	foundSpike := false
	for _, alert := range dashboard.Alerts {
		switch alert.Code {
		case "quota_exceeded":
			foundQuota = true
		case "usage_spike":
			foundSpike = true
		}
	}
	require.True(t, foundQuota, "expected quota_exceeded alert")
	require.True(t, foundSpike, "expected usage_spike alert")
}

func TestAnalyticsService_EnsureRevenueReport(t *testing.T) {
	cfg := &config.Config{
		Integration: &config.IntegrationConfig{
			Billing: config.IntegrationBillingConfig{
				Reconciliation: config.IntegrationRevenueSplitConfig{
					VendorShare:   0.8,
					PlatformShare: 0.15,
					FeeShare:      0.05,
					Currency:      "USD",
				},
			},
		},
	}

	revenueStub := &revenueRepoStub{}
	svc := NewAnalyticsService(
		cfg,
		&usageRepoStub{},
		revenueStub,
		&notificationRepoStub{},
		&licenseRepoStub{},
		&pricingRepoStub{},
		&listingRepoStub{},
		logrus.NewEntry(logrus.New()),
	)

	start := time.Now().Add(-30 * 24 * time.Hour).UTC()
	end := time.Now().UTC()
	report, err := svc.EnsureRevenueReport(context.Background(), "tenant-1", "vendor-1", start, end, 500.0, "USD")
	require.NoError(t, err)
	require.NotNil(t, report)
	require.InDelta(t, 400.0, report.VendorShare, 0.001)
	require.InDelta(t, 75.0, report.PlatformShare, 0.001)
	require.InDelta(t, 25.0, report.Fees, 0.001)
	require.Equal(t, revenueStub.last, report)
}
