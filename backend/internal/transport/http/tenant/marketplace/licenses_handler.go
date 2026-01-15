package marketplace

import (
	"errors"
	"strings"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/contracts"
	mrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/marketplace"
	svc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/marketplace"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
	httpmw "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/middleware"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// LicenseHandler exposes tenant-facing license endpoints.
type LicenseHandler struct {
	deps    *app.Deps
	service *svc.LicenseService
}

// NewLicenseHandler constructs a tenant license handler.
func NewLicenseHandler(deps *app.Deps) *LicenseHandler {
	if deps == nil || deps.DB == nil {
		return &LicenseHandler{deps: deps}
	}
	pricingRepo := mrepo.NewPricingRepository(deps.DB)
	licenseRepo := mrepo.NewLicenseRepository(deps.DB)
	logger := deps.RuntimeLogger(deps.Ctx, "tenant_marketplace_license", nil)
	service := svc.NewLicenseService(
		deps.Config,
		pricingRepo,
		licenseRepo,
		deps.TaxProviderClient,
		deps.MarketplaceBilling,
		deps.LicenseAuthority,
		deps.LicenseCache,
		logger,
	)

	return &LicenseHandler{
		deps:    deps,
		service: service,
	}
}

// Create issues a new license for the tenant.
func (h *LicenseHandler) Create(c *gin.Context) {
	if h.service == nil {
		contracts.ResponseServiceUnavailable(c, "license service not available", nil)
		return
	}

	var req createLicenseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid request body: "+err.Error())
		return
	}

	tenantID, ok := httpmw.TenantUuidString(c)
	if !ok {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	if strings.TrimSpace(req.TenantUuid) != "" && req.TenantUuid != tenantID {
		contracts.ResponseUnauthorized(c, "tenant mismatch")
		return
	}

	metadata := map[string]any{}
	if req.PaymentIntentID != "" {
		metadata["payment_intent_id"] = req.PaymentIntentID
	}
	metadata["channel"] = "marketplace_api"

	params := svc.IssueLicenseParams{
		TenantUuid: tenantID,
		ListingID:  req.ListingID,
		PlanID:     req.PlanID,
		IssuedBy:   tenantID,
		Metadata:   metadata,
	}

	if req.TrialOverrideDays != nil && *req.TrialOverrideDays > 0 {
		params.Trial = true
		params.ExpiresAt = time.Now().Add(time.Duration(*req.TrialOverrideDays) * 24 * time.Hour)
	}

	license, err := h.service.IssueLicense(c.Request.Context(), params)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			contracts.ResponseNotFound(c, "plan or listing not found")
			return
		}
		if strings.Contains(err.Error(), "billing client not configured") {
			contracts.ResponseServiceUnavailable(c, "billing provider unavailable", nil)
			return
		}
		contracts.ResponseInternalError(c, err)
		return
	}

	contracts.ResponseSuccessWithMessage(c, newLicenseResponse(license), "license issued")
}

// Get returns a license by identifier.
func (h *LicenseHandler) Get(c *gin.Context) {
	if h.service == nil {
		contracts.ResponseServiceUnavailable(c, "license service not available", nil)
		return
	}
	tenantID, ok := httpmw.TenantUuidString(c)
	if !ok {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	license, err := h.service.GetLicense(c.Request.Context(), tenantID, c.Param("id"))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			contracts.ResponseNotFound(c, "license not found")
			return
		}
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, newLicenseResponse(license))
}

// Renew renews an existing license optionally applying a new plan.
func (h *LicenseHandler) Renew(c *gin.Context) {
	if h.service == nil {
		contracts.ResponseServiceUnavailable(c, "license service not available", nil)
		return
	}

	var req renewLicenseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid request body: "+err.Error())
		return
	}

	tenantID, ok := httpmw.TenantUuidString(c)
	if !ok {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}

	params := svc.RenewLicenseParams{
		LicenseID:    c.Param("id"),
		TenantUuid:   tenantID,
		IssuedBy:     tenantID,
		PlanID:       strings.TrimSpace(req.PlanID),
		RenewalToken: strings.TrimSpace(req.RenewalToken),
	}

	license, err := h.service.RenewLicense(c.Request.Context(), params)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			contracts.ResponseNotFound(c, "license not found")
			return
		}
		if strings.Contains(err.Error(), "invalid renewal token") {
			contracts.ResponseUnauthorized(c, "invalid renewal token")
			return
		}
		contracts.ResponseInternalError(c, err)
		return
	}

	contracts.ResponseSuccessWithMessage(c, newLicenseResponse(license), "license renewed")
}

// ExtendOffline extends a license offline window up to 72 hours.
func (h *LicenseHandler) ExtendOffline(c *gin.Context) {
	if h.service == nil {
		contracts.ResponseServiceUnavailable(c, "license service not available", nil)
		return
	}

	var req offlineExtendRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid request body: "+err.Error())
		return
	}

	tenantID, ok := httpmw.TenantUuidString(c)
	if !ok {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}

	target := time.Now().Add(time.Duration(req.RequestedHours) * time.Hour)
	if err := h.service.ExtendOffline(c.Request.Context(), tenantID, c.Param("id"), target); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			contracts.ResponseNotFound(c, "license not found")
			return
		}
		contracts.ResponseInternalError(c, err)
		return
	}

	license, err := h.service.GetLicense(c.Request.Context(), tenantID, c.Param("id"))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			contracts.ResponseNotFound(c, "license not found")
			return
		}
		contracts.ResponseInternalError(c, err)
		return
	}

	contracts.ResponseSuccessWithMessage(c, newLicenseResponse(license), "offline window extended")
}
