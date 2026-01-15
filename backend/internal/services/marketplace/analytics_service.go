package marketplace

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/config"
	dbm "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/marketplace"
	marketobs "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/observability/marketplace"
	"github.com/sirupsen/logrus"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// UsageDataRepository defines the persistence operations required for usage analytics.
type UsageDataRepository interface {
	InsertEnvelopes(ctx context.Context, tenantID string, envelopes []*dbm.UsageEnvelope) (int, error)
	UpsertAggregate(ctx context.Context, aggregate *dbm.UsageAggregate) error
	ListAggregates(ctx context.Context, tenantID, licenseID string, window dbm.AggregationWindow) ([]*dbm.UsageAggregate, error)
	LatestAggregate(ctx context.Context, tenantID, licenseID, metric string, window dbm.AggregationWindow) (*dbm.UsageAggregate, error)
	GetAggregate(ctx context.Context, tenantID, licenseID, metric string, window dbm.AggregationWindow, bucket time.Time) (*dbm.UsageAggregate, error)
	DeleteEnvelopesBefore(ctx context.Context, tenantID, licenseID string, before time.Time) (int, error)
	DeleteAggregatesBefore(ctx context.Context, tenantID, licenseID string, before time.Time) (int, error)
}

// RevenueReportRepository defines required operations for revenue share reports.
type RevenueReportRepository interface {
	UpsertReport(ctx context.Context, report *dbm.RevenueShareReport) error
	ListReports(ctx context.Context, tenantID, vendorID string, from, to time.Time) ([]*dbm.RevenueShareReport, error)
}

// NotificationRepository exposes notification persistence.
type NotificationRepository interface {
	QueueNotification(ctx context.Context, notification *dbm.Notification) error
	ListByTenant(ctx context.Context, tenantID string) ([]*dbm.Notification, error)
}

// LicenseRepositoryReader fetches licenses.
type LicenseRepositoryReader interface {
	GetLicense(ctx context.Context, tenantID, licenseID string) (*dbm.License, error)
}

// PricingRepositoryReader fetches pricing plans.
type PricingRepositoryReader interface {
	GetPlan(ctx context.Context, tenantID, planID string) (*dbm.PricingPlan, error)
}

// ListingRepositoryReader fetches listings.
type ListingRepositoryReader interface {
	FindByID(ctx context.Context, tenantID, listingID string) (*dbm.Listing, error)
}

// AnalyticsService aggregates usage metrics, quota alerts and revenue share reports.
type AnalyticsService struct {
	cfg              *config.Config
	usageRepo        UsageDataRepository
	revenueRepo      RevenueReportRepository
	notificationRepo NotificationRepository
	licenseRepo      LicenseRepositoryReader
	pricingRepo      PricingRepositoryReader
	listingRepo      ListingRepositoryReader
	logger           *logrus.Entry
	spikeMultiplier  float64
	quotaThreshold   float64
}

// UsageDashboardQuery captures filters for dashboard retrieval.
type UsageDashboardQuery struct {
	Window dbm.AggregationWindow
	Metric string
	From   time.Time
	To     time.Time
}

// UsageDataPoint represents a single metric data point for the dashboard chart.
type UsageDataPoint struct {
	Timestamp      time.Time `json:"timestamp"`
	Metric         string    `json:"metric"`
	Value          float64   `json:"value"`
	Revenue        float64   `json:"revenue"`
	QuotaRemaining *float64  `json:"quota_remaining,omitempty"`
	Window         string    `json:"window"`
	Currency       string    `json:"currency,omitempty"`
}

// UsageAlert communicates anomalous usage observability signals.
type UsageAlert struct {
	Code     string `json:"code"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

// UsageDashboard contains chart data points and triggered alerts.
type UsageDashboard struct {
	Series []UsageDataPoint `json:"series"`
	Alerts []UsageAlert     `json:"alerts"`
}

// NewAnalyticsService constructs the analytics service with dependencies.
func NewAnalyticsService(
	cfg *config.Config,
	usageRepo UsageDataRepository,
	revenueRepo RevenueReportRepository,
	notificationRepo NotificationRepository,
	licenseRepo LicenseRepositoryReader,
	pricingRepo PricingRepositoryReader,
	listingRepo ListingRepositoryReader,
	logger *logrus.Entry,
) *AnalyticsService {
	if logger == nil {
		logger = logrus.New().WithField("component", "marketplace_analytics_service")
	}
	svc := &AnalyticsService{
		cfg:              cfg,
		usageRepo:        usageRepo,
		revenueRepo:      revenueRepo,
		notificationRepo: notificationRepo,
		licenseRepo:      licenseRepo,
		pricingRepo:      pricingRepo,
		listingRepo:      listingRepo,
		logger:           logger,
		spikeMultiplier:  1.2,
		quotaThreshold:   0.9,
	}
	return svc
}

// RecordEnvelope updates aggregates, alerts and revenue share for a processed envelope.
func (s *AnalyticsService) RecordEnvelope(ctx context.Context, tenantID string, license *dbm.License, plan *dbm.PricingPlan, listing *dbm.Listing, envelope *dbm.UsageEnvelope, metrics []dbm.UsageMetric) error {
	if license == nil || envelope == nil {
		return errors.New("license and envelope are required")
	}
	if len(metrics) == 0 {
		return nil
	}

	if plan == nil {
		fetched, err := s.pricingRepo.GetPlan(ctx, tenantID, license.PlanID)
		if err != nil {
			return fmt.Errorf("fetch plan: %w", err)
		}
		plan = fetched
	}
	if listing == nil && s.listingRepo != nil {
		fetched, err := s.listingRepo.FindByID(ctx, tenantID, license.ListingID)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("fetch listing: %w", err)
		}
		if err == nil {
			listing = fetched
		}
	}

	currency := strings.TrimSpace(plan.Currency)
	if currency == "" {
		currency = "USD"
	}

	var (
		totalGross float64
		dayBefore  *dbm.UsageAggregate
		dayAfter   *dbm.UsageAggregate
	)

	for _, metric := range metrics {
		if metric.Name == "" {
			continue
		}
		totalGross += metric.Value
		for _, window := range []dbm.AggregationWindow{dbm.AggregationWindowHour, dbm.AggregationWindowDay, dbm.AggregationWindowMonth} {
			bucket := bucketTimestamp(window, envelope.TimestampEnd)
			existing, err := s.usageRepo.GetAggregate(ctx, tenantID, license.ID, metric.Name, window, bucket)
			if err != nil {
				return fmt.Errorf("fetch usage aggregate: %w", err)
			}
			total := metric.Value
			revenue := metric.Value
			if existing != nil {
				total += existing.Total
				revenue += existing.Revenue
			}
			aggregate := &dbm.UsageAggregate{
				ID:         "",
				TenantUuid: tenantID,
				LicenseID:  license.ID,
				Metric:     metric.Name,
				Window:     window,
				TimeBucket: bucket,
				Total:      total,
				Delta:      metric.Value,
				Currency:   currency,
				Revenue:    revenue,
			}
			if existing != nil {
				aggregate.ID = existing.ID
			}
			if err := s.usageRepo.UpsertAggregate(ctx, aggregate); err != nil {
				return fmt.Errorf("upsert usage aggregate: %w", err)
			}
			if window == dbm.AggregationWindowDay {
				if dayBefore == nil && existing != nil {
					copy := *existing
					dayBefore = &copy
				}
				copy := *aggregate
				dayAfter = &copy
			}
		}
	}

	if dayAfter != nil {
		alerts := s.evaluateAlerts(plan, dayBefore, dayAfter)
		for _, alert := range alerts {
			if listing != nil && strings.TrimSpace(listing.VendorID) != "" && s.notificationRepo != nil {
				payload := dbm.Notification{
					TenantUuid:    tenantID,
					RecipientType: "vendor",
					RecipientID:   listing.VendorID,
					Channel:       "in_app",
					TemplateCode:  "marketplace.usage.alert",
					Payload: datatypes.JSONMap{
						"license_id": license.ID,
						"metric":     dayAfter.Metric,
						"code":       alert.Code,
						"message":    alert.Message,
						"severity":   alert.Severity,
						"value":      dayAfter.Total,
						"quota":      plan.QuotaLimit,
					},
					Status: dbm.NotificationStatusPending,
				}
				if err := s.notificationRepo.QueueNotification(ctx, &payload); err != nil {
					s.logger.WithError(err).Warn("failed to queue usage alert notification")
				}
			}
			if alert.Code == "usage_spike" {
				marketobs.EmitUsageSpikeDetected(s.logger, tenantID, license.ID, dayAfter.Metric, dayAfter.Delta)
			}
		}
	}

	if listing != nil && strings.TrimSpace(listing.VendorID) != "" && totalGross > 0 {
		periodStart, periodEnd := monthPeriod(envelope.TimestampEnd)
		if report, err := s.EnsureRevenueReport(ctx, tenantID, listing.VendorID, periodStart, periodEnd, totalGross, currency); err != nil {
			s.logger.WithError(err).Warn("failed to upsert revenue report")
		} else {
			marketobs.RecordRevenueGenerated(tenantID, report.Currency, report.GrossAmount)
		}
	}

	return nil
}

// BuildDashboard returns usage series and alerts for a license.
func (s *AnalyticsService) BuildDashboard(ctx context.Context, tenantID, licenseID string, query UsageDashboardQuery) (*UsageDashboard, error) {
	if query.Window == "" {
		query.Window = dbm.AggregationWindowDay
	}

	license, err := s.licenseRepo.GetLicense(ctx, tenantID, licenseID)
	if err != nil {
		return nil, fmt.Errorf("fetch license: %w", err)
	}
	plan, err := s.pricingRepo.GetPlan(ctx, tenantID, license.PlanID)
	if err != nil {
		return nil, fmt.Errorf("fetch plan: %w", err)
	}

	aggregates, err := s.usageRepo.ListAggregates(ctx, tenantID, licenseID, query.Window)
	if err != nil {
		return nil, fmt.Errorf("list aggregates: %w", err)
	}

	var (
		series []UsageDataPoint
		prev   *dbm.UsageAggregate
		last   *dbm.UsageAggregate
	)

	for _, agg := range aggregates {
		if query.Metric != "" && !strings.EqualFold(query.Metric, agg.Metric) {
			continue
		}
		if !query.From.IsZero() && agg.TimeBucket.Before(query.From) {
			continue
		}
		if !query.To.IsZero() && agg.TimeBucket.After(query.To) {
			continue
		}
		point := UsageDataPoint{
			Timestamp: agg.TimeBucket,
			Metric:    agg.Metric,
			Value:     agg.Total,
			Revenue:   agg.Revenue,
			Window:    string(agg.Window),
			Currency:  agg.Currency,
		}
		if plan.QuotaLimit != nil {
			remaining := *plan.QuotaLimit - agg.Total
			point.QuotaRemaining = floatPtr(math.Max(0, remaining))
		}
		series = append(series, point)
		prev = last
		copy := *agg
		last = &copy
	}

	dashboard := &UsageDashboard{
		Series: series,
		Alerts: []UsageAlert{},
	}
	if last != nil {
		alerts := s.evaluateAlerts(plan, prev, last)
		dashboard.Alerts = append(dashboard.Alerts, alerts...)
	}
	return dashboard, nil
}

// EnsureRevenueReport upserts the revenue share report for the given period and gross amount.
func (s *AnalyticsService) EnsureRevenueReport(ctx context.Context, tenantID, vendorID string, periodStart, periodEnd time.Time, gross float64, currency string) (*dbm.RevenueShareReport, error) {
	if strings.TrimSpace(tenantID) == "" || strings.TrimSpace(vendorID) == "" {
		return nil, errors.New("tenant_uuid and vendor_id are required")
	}
	if periodEnd.Before(periodStart) {
		periodStart, periodEnd = periodEnd, periodStart
	}
	split := s.cfg.IntegrationRevenueSplit()
	if strings.TrimSpace(currency) == "" {
		currency = split.Currency
	}
	report := &dbm.RevenueShareReport{
		TenantUuid:    tenantID,
		VendorID:      vendorID,
		PeriodStart:   periodStart.UTC(),
		PeriodEnd:     periodEnd.UTC(),
		GrossAmount:   gross,
		VendorShare:   gross * split.VendorShare,
		PlatformShare: gross * split.PlatformShare,
		Fees:          gross * split.FeeShare,
		Currency:      currency,
		Status:        dbm.RevenueReportStatusReady,
	}
	if err := s.revenueRepo.UpsertReport(ctx, report); err != nil {
		return nil, err
	}
	return report, nil
}

// ListReports proxies to repository to list revenue share reports.
func (s *AnalyticsService) ListReports(ctx context.Context, tenantID, vendorID string, from, to time.Time) ([]*dbm.RevenueShareReport, error) {
	return s.revenueRepo.ListReports(ctx, tenantID, vendorID, from, to)
}

func (s *AnalyticsService) evaluateAlerts(plan *dbm.PricingPlan, before, after *dbm.UsageAggregate) []UsageAlert {
	alerts := []UsageAlert{}
	if plan == nil || after == nil {
		return alerts
	}

	if plan.QuotaLimit != nil && *plan.QuotaLimit > 0 {
		if after.Total >= *plan.QuotaLimit {
			alerts = append(alerts, UsageAlert{
				Code:     "quota_exceeded",
				Severity: "critical",
				Message:  fmt.Sprintf("Usage quota exceeded for metric %s (%.2f / %.2f)", after.Metric, after.Total, *plan.QuotaLimit),
			})
		} else if after.Total >= s.quotaThreshold*(*plan.QuotaLimit) {
			alerts = append(alerts, UsageAlert{
				Code:     "quota_nearing",
				Severity: "warning",
				Message:  fmt.Sprintf("Usage quota nearing limit for metric %s (%.2f / %.2f)", after.Metric, after.Total, *plan.QuotaLimit),
			})
		}
	}

	if before != nil && before.Delta > 0 && after.Delta >= s.spikeMultiplier*before.Delta {
		alerts = append(alerts, UsageAlert{
			Code:     "usage_spike",
			Severity: "warning",
			Message:  fmt.Sprintf("Usage spike detected for metric %s (%.2f → %.2f)", after.Metric, before.Delta, after.Delta),
		})
	}
	return alerts
}

func bucketTimestamp(window dbm.AggregationWindow, ts time.Time) time.Time {
	ts = ts.UTC()
	switch window {
	case dbm.AggregationWindowHour:
		return time.Date(ts.Year(), ts.Month(), ts.Day(), ts.Hour(), 0, 0, 0, time.UTC)
	case dbm.AggregationWindowMonth:
		return time.Date(ts.Year(), ts.Month(), 1, 0, 0, 0, 0, time.UTC)
	default:
		return time.Date(ts.Year(), ts.Month(), ts.Day(), 0, 0, 0, 0, time.UTC)
	}
}

func monthPeriod(ts time.Time) (time.Time, time.Time) {
	ts = ts.UTC()
	start := time.Date(ts.Year(), ts.Month(), 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0).Add(-time.Nanosecond)
	return start, end
}

func floatPtr(v float64) *float64 {
	return &v
}
