package marketplace

import (
	"net/http"
	"strings"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/contracts"
	dbm "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/marketplace"
	mrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/marketplace"
	svc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/marketplace"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
	httpmw "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/middleware"
	"github.com/gin-gonic/gin"
)

// AnalyticsHandler exposes usage ingest and analytics endpoints.
type AnalyticsHandler struct {
	deps         *app.Deps
	usageService *svc.UsageIngestService
	analytics    *svc.AnalyticsService
}

// NewAnalyticsHandler constructs handler dependencies.
func NewAnalyticsHandler(deps *app.Deps) *AnalyticsHandler {
	if deps == nil || deps.DB == nil {
		return &AnalyticsHandler{deps: deps}
	}
	usageRepo := mrepo.NewUsageRepository(deps.DB)
	revenueRepo := mrepo.NewRevenueRepository(deps.DB)
	notificationRepo := mrepo.NewNotificationRepository(deps.DB)
	licenseRepo := mrepo.NewLicenseRepository(deps.DB)
	pricingRepo := mrepo.NewPricingRepository(deps.DB)
	listingRepo := mrepo.NewListingRepository(deps.DB)
	logger := deps.RuntimeLogger(deps.Ctx, "admin_marketplace_analytics", nil)

	analytics := svc.NewAnalyticsService(deps.Config, usageRepo, revenueRepo, notificationRepo, licenseRepo, pricingRepo, listingRepo, logger)
	usageService := svc.NewUsageIngestService(deps.Config, usageRepo, licenseRepo, listingRepo, analytics, logger)

	return &AnalyticsHandler{
		deps:         deps,
		usageService: usageService,
		analytics:    analytics,
	}
}

type usageBatchPayload struct {
	Envelopes []svc.UsageEnvelopeInput `json:"envelopes" binding:"required"`
}

type usageMetricsQuery struct {
	Window string `form:"window,default=day"`
	Metric string `form:"metric"`
	From   string `form:"from"`
	To     string `form:"to"`
}

type revenueReportQuery struct {
	VendorID    string `form:"vendor_id"`
	PeriodStart string `form:"period_start"`
	PeriodEnd   string `form:"period_end"`
}

// Ingest handles POST /marketplace/usage ingest calls.
func (h *AnalyticsHandler) Ingest(c *gin.Context) {
	if h.usageService == nil {
		contracts.ResponseServiceUnavailable(c, "usage service not available", nil)
		return
	}
	var payload usageBatchPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		contracts.ResponseBadRequest(c, "invalid usage payload: "+err.Error())
		return
	}
	tenantID, ok := httpmw.TenantUuidString(c)
	if !ok {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	ctx := c.Request.Context()
	result, err := h.usageService.IngestBatch(ctx, tenantID, payload.Envelopes)
	if err != nil && (result == nil || result.Accepted == 0) {
		contracts.ResponseInternalError(c, err)
		return
	}
	c.JSON(http.StatusAccepted, gin.H{
		"result": result,
	})
}

// GetMetrics handles GET /marketplace/usage/tenants/:tenantId/licenses/:licenseId/metrics.
func (h *AnalyticsHandler) GetMetrics(c *gin.Context) {
	if h.analytics == nil {
		contracts.ResponseServiceUnavailable(c, "analytics service not available", nil)
		return
	}
	var query usageMetricsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		contracts.ResponseBadRequest(c, "invalid query parameters: "+err.Error())
		return
	}
	pathTenant := strings.TrimSpace(c.Param("tenantId"))
	tenantID, ok := httpmw.TenantUuidString(c)
	if !ok {
		tenantID = pathTenant
	}
	if pathTenant != "" && tenantID != "" && !strings.EqualFold(pathTenant, tenantID) {
		contracts.ResponseUnauthorized(c, "tenant mismatch")
		return
	}
	if tenantID == "" {
		tenantID = pathTenant
	}
	licenseID := strings.TrimSpace(c.Param("licenseId"))
	if tenantID == "" || licenseID == "" {
		contracts.ResponseBadRequest(c, "tenant_uuid and license_id required")
		return
	}
	window := svcWindow(query.Window)
	var from, to time.Time
	var err error
	if query.From != "" {
		from, err = time.Parse(time.RFC3339, query.From)
		if err != nil {
			contracts.ResponseBadRequest(c, "invalid from timestamp: "+err.Error())
			return
		}
	}
	if query.To != "" {
		to, err = time.Parse(time.RFC3339, query.To)
		if err != nil {
			contracts.ResponseBadRequest(c, "invalid to timestamp: "+err.Error())
			return
		}
	}
	ctx := c.Request.Context()
	dashboard, err := h.analytics.BuildDashboard(ctx, tenantID, licenseID, svc.UsageDashboardQuery{
		Window: window,
		Metric: strings.TrimSpace(query.Metric),
		From:   from,
		To:     to,
	})
	if err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": dashboard})
}

// ListRevenueReports handles GET /marketplace/revenue-share/reports.
func (h *AnalyticsHandler) ListRevenueReports(c *gin.Context) {
	if h.analytics == nil {
		contracts.ResponseServiceUnavailable(c, "analytics service not available", nil)
		return
	}
	var query revenueReportQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		contracts.ResponseBadRequest(c, "invalid query parameters: "+err.Error())
		return
	}
	tenantID, ok := httpmw.TenantUuidString(c)
	if !ok || tenantID == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	var (
		from time.Time
		to   time.Time
		err  error
	)
	if strings.TrimSpace(query.PeriodStart) != "" {
		from, err = time.Parse(time.DateOnly, query.PeriodStart)
		if err != nil {
			contracts.ResponseBadRequest(c, "invalid period_start: "+err.Error())
			return
		}
	}
	if strings.TrimSpace(query.PeriodEnd) != "" {
		to, err = time.Parse(time.DateOnly, query.PeriodEnd)
		if err != nil {
			contracts.ResponseBadRequest(c, "invalid period_end: "+err.Error())
			return
		}
	}
	ctx := c.Request.Context()
	reports, err := h.analytics.ListReports(ctx, tenantID, strings.TrimSpace(query.VendorID), from, to)
	if err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": reports})
}

func svcWindow(raw string) dbm.AggregationWindow {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case string(dbm.AggregationWindowHour):
		return dbm.AggregationWindowHour
	case string(dbm.AggregationWindowMonth):
		return dbm.AggregationWindowMonth
	default:
		return dbm.AggregationWindowDay
	}
}
