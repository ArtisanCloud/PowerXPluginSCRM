package marketplace

import (
	"errors"
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/contracts"
	dbm "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/marketplace"
	mrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/marketplace"
	svc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/marketplace"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
	httpmw "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/middleware"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ListingHandler exposes admin endpoints for marketplace listings.
type ListingHandler struct {
	deps    *app.Deps
	service *svc.ListingService
}

// NewListingHandler constructs the handler and its dependencies.
func NewListingHandler(deps *app.Deps) *ListingHandler {
	if deps == nil || deps.DB == nil {
		return &ListingHandler{deps: deps}
	}
	listingRepo := mrepo.NewListingRepository(deps.DB)
	checklistRepo := mrepo.NewChecklistRepository(deps.DB)
	logger := deps.RuntimeLogger(deps.Ctx, "admin_marketplace_listings", nil)
	service := svc.NewListingService(listingRepo, checklistRepo, logger)
	return &ListingHandler{
		deps:    deps,
		service: service,
	}
}

// Service exposes the underlying listing service for sibling handlers.
func (h *ListingHandler) Service() *svc.ListingService {
	if h == nil {
		return nil
	}
	return h.service
}

type listingListQuery struct {
	Status []string `form:"status"`
	Locale string   `form:"locale"`
	Search string   `form:"search"`
	Limit  int      `form:"limit,default=20"`
	Offset int      `form:"offset,default=0"`
}

type createListingRequest struct {
	ID            string            `json:"id"`
	PluginID      string            `json:"plugin_id" binding:"required"`
	VendorID      string            `json:"vendor_id" binding:"required"`
	Title         string            `json:"title" binding:"required"`
	Slug          string            `json:"slug" binding:"required"`
	Summary       string            `json:"summary"`
	Description   string            `json:"description"`
	Locale        string            `json:"locale"`
	Categories    []string          `json:"categories"`
	Tags          []string          `json:"tags"`
	BrandingTheme map[string]any    `json:"branding_theme"`
	Assets        []assetPayload    `json:"assets"`
	PricingPlans  []planPayload     `json:"pricing_plans"`
	Checklist     *checklistPayload `json:"checklist"`
}

type updateListingRequest struct {
	Title         *string         `json:"title"`
	Summary       *string         `json:"summary"`
	Description   *string         `json:"description"`
	Categories    *[]string       `json:"categories"`
	Tags          *[]string       `json:"tags"`
	BrandingTheme *map[string]any `json:"branding_theme"`
	Locale        *string         `json:"locale"`
	Assets        *[]assetPayload `json:"assets"`
	PricingPlans  *[]planPayload  `json:"pricing_plans"`
}

type assetPayload struct {
	ID         string         `json:"id"`
	AssetType  string         `json:"asset_type" binding:"required"`
	StorageURI string         `json:"storage_uri" binding:"required"`
	Checksum   string         `json:"checksum"`
	IsPrimary  bool           `json:"is_primary"`
	Locale     string         `json:"locale"`
	Weight     int            `json:"weight"`
	Metadata   map[string]any `json:"metadata"`
}

type planPayload struct {
	ID            string            `json:"id"`
	PlanCode      string            `json:"plan_code" binding:"required"`
	PlanType      string            `json:"plan_type"`
	Currency      string            `json:"currency" binding:"required"`
	Amount        *float64          `json:"amount"`
	BillingPeriod string            `json:"billing_period"`
	TrialDays     *int              `json:"trial_days"`
	QuotaLimit    *float64          `json:"quota_limit"`
	OveragePolicy string            `json:"overage_policy"`
	FeatureMatrix map[string]any    `json:"feature_matrix"`
	IsDefault     bool              `json:"is_default"`
	Tiers         []planTierPayload `json:"tiers"`
}

type planTierPayload struct {
	ID         string   `json:"id"`
	Metric     string   `json:"metric" binding:"required"`
	RangeFrom  float64  `json:"range_from" binding:"required"`
	RangeTo    *float64 `json:"range_to"`
	UnitAmount float64  `json:"unit_amount" binding:"required"`
	UnitName   string   `json:"unit_name"`
}

type checklistPayload struct {
	ID            string                 `json:"id"`
	TriggerSource string                 `json:"trigger_source"`
	Summary       string                 `json:"summary"`
	RunNumber     int                    `json:"run_number"`
	Status        string                 `json:"status"`
	Items         []checklistItemPayload `json:"items"`
}

type checklistItemPayload struct {
	ID          string `json:"id"`
	Code        string `json:"code" binding:"required"`
	Description string `json:"description"`
	Result      string `json:"result"`
	EvidenceURI string `json:"evidence_uri"`
	Notes       string `json:"notes"`
	AutoFixLink string `json:"auto_fix_link"`
}

type reviewListingRequest struct {
	SubmittedBy string         `json:"submitted_by" binding:"required"`
	Metadata    map[string]any `json:"metadata"`
}

type reviewerDecisionRequest struct {
	ReviewerID string `json:"reviewer_id" binding:"required"`
	Notes      string `json:"notes"`
}

// List returns paginated listings for the tenant.
func (h *ListingHandler) List(c *gin.Context) {
	if h.service == nil {
		contracts.ResponseServiceUnavailable(c, "listing service not available", nil)
		return
	}
	var query listingListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		contracts.ResponseBadRequest(c, "invalid query parameters: "+err.Error())
		return
	}
	statuses := normalizeListFilter(query.Status)
	tenantID, ok := httpmw.TenantUuidString(c)
	if !ok {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	listings, total, err := h.service.ListListings(c.Request.Context(), tenantID, svc.ListListingsOptions{
		Status: statuses,
		Locale: query.Locale,
		Search: query.Search,
		Limit:  query.Limit,
		Offset: query.Offset,
	})
	if err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, gin.H{
		"items": NewListingListResponse(listings),
		"total": total,
	})
}

// Get returns a single listing by identifier.
func (h *ListingHandler) Get(c *gin.Context) {
	if h.service == nil {
		contracts.ResponseServiceUnavailable(c, "listing service not available", nil)
		return
	}
	tenantID, ok := httpmw.TenantUuidString(c)
	if !ok {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	listing, err := h.service.GetListing(c.Request.Context(), tenantID, c.Param("id"))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			contracts.ResponseNotFound(c, "listing not found")
			return
		}
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, NewListingResponse(listing))
}

// Create stores a new listing draft.
func (h *ListingHandler) Create(c *gin.Context) {
	if h.service == nil {
		contracts.ResponseServiceUnavailable(c, "listing service not available", nil)
		return
	}
	var req createListingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid request body: "+err.Error())
		return
	}
	tenantID, ok := httpmw.TenantUuidString(c)
	if !ok {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	listing, err := h.service.CreateDraft(c.Request.Context(), tenantID, convertCreateRequest(req))
	if err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccessWithMessage(c, NewListingResponse(listing), "listing draft created")
}

// Update mutates a listing draft.
func (h *ListingHandler) Update(c *gin.Context) {
	if h.service == nil {
		contracts.ResponseServiceUnavailable(c, "listing service not available", nil)
		return
	}
	var req updateListingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid request body: "+err.Error())
		return
	}
	tenantID, ok := httpmw.TenantUuidString(c)
	if !ok {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	listing, err := h.service.UpdateDraft(c.Request.Context(), tenantID, c.Param("id"), convertUpdateRequest(req))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			contracts.ResponseNotFound(c, "listing not found")
			return
		}
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccessWithMessage(c, NewListingResponse(listing), "listing updated")
}

// SubmitForReview transitions a listing into review.
func (h *ListingHandler) SubmitForReview(c *gin.Context) {
	if h.service == nil {
		contracts.ResponseServiceUnavailable(c, "listing service not available", nil)
		return
	}
	var req reviewListingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid request body: "+err.Error())
		return
	}
	tenantID, ok := httpmw.TenantUuidString(c)
	if !ok {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	listing, err := h.service.SubmitForReview(c.Request.Context(), tenantID, c.Param("id"), req.SubmittedBy, req.Metadata)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			contracts.ResponseNotFound(c, "listing not found")
			return
		}
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccessWithMessage(c, NewListingResponse(listing), "listing moved to review")
}

// Publish marks a listing as published.
func (h *ListingHandler) Publish(c *gin.Context) {
	h.applyReviewerDecision(c, dbm.ListingStatusPublished, "listing published")
}

// Suspend marks a listing as suspended.
func (h *ListingHandler) Suspend(c *gin.Context) {
	h.applyReviewerDecision(c, dbm.ListingStatusSuspended, "listing suspended")
}

func (h *ListingHandler) applyReviewerDecision(c *gin.Context, status string, successMessage string) {
	if h.service == nil {
		contracts.ResponseServiceUnavailable(c, "listing service not available", nil)
		return
	}
	var req reviewerDecisionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid request body: "+err.Error())
		return
	}
	tenantID, ok := httpmw.TenantUuidString(c)
	if !ok {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	listing, err := h.service.UpdateListingStatus(c.Request.Context(), tenantID, c.Param("id"), req.ReviewerID, status, req.Notes)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			contracts.ResponseNotFound(c, "listing not found")
			return
		}
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccessWithMessage(c, NewListingResponse(listing), successMessage)
}

func convertCreateRequest(req createListingRequest) svc.ListingDraftInput {
	input := svc.ListingDraftInput{
		ID:            req.ID,
		PluginID:      req.PluginID,
		VendorID:      req.VendorID,
		Title:         req.Title,
		Slug:          req.Slug,
		Summary:       req.Summary,
		Description:   req.Description,
		Locale:        req.Locale,
		Categories:    req.Categories,
		Tags:          req.Tags,
		BrandingTheme: req.BrandingTheme,
		Assets:        convertAssetPayloads(req.Assets),
		PricingPlans:  convertPlanPayloads(req.PricingPlans),
	}
	if req.Checklist != nil {
		input.ReadyChecklist = convertChecklistPayload(*req.Checklist)
	}
	return input
}

func convertUpdateRequest(req updateListingRequest) svc.ListingUpdateInput {
	input := svc.ListingUpdateInput{}
	if req.Title != nil {
		input.Title = req.Title
	}
	if req.Summary != nil {
		input.Summary = req.Summary
	}
	if req.Description != nil {
		input.Description = req.Description
	}
	if req.Categories != nil {
		input.Categories = req.Categories
	}
	if req.Tags != nil {
		input.Tags = req.Tags
	}
	if req.BrandingTheme != nil {
		input.BrandingTheme = *req.BrandingTheme
	}
	if req.Locale != nil {
		input.Locale = req.Locale
	}
	if req.Assets != nil {
		input.Assets = convertAssetPayloads(*req.Assets)
	}
	if req.PricingPlans != nil {
		input.PricingPlans = convertPlanPayloads(*req.PricingPlans)
	}
	return input
}

func convertAssetPayloads(assets []assetPayload) []svc.ListingAssetInput {
	if len(assets) == 0 {
		return nil
	}
	out := make([]svc.ListingAssetInput, len(assets))
	for i, asset := range assets {
		out[i] = svc.ListingAssetInput{
			ID:         asset.ID,
			AssetType:  asset.AssetType,
			StorageURI: asset.StorageURI,
			Checksum:   asset.Checksum,
			IsPrimary:  asset.IsPrimary,
			Locale:     asset.Locale,
			Weight:     asset.Weight,
			Metadata:   asset.Metadata,
		}
	}
	return out
}

func convertPlanPayloads(plans []planPayload) []svc.PricingPlanInput {
	if len(plans) == 0 {
		return nil
	}
	out := make([]svc.PricingPlanInput, len(plans))
	for i, plan := range plans {
		out[i] = svc.PricingPlanInput{
			ID:            plan.ID,
			PlanCode:      plan.PlanCode,
			PlanType:      plan.PlanType,
			Currency:      plan.Currency,
			Amount:        plan.Amount,
			BillingPeriod: plan.BillingPeriod,
			TrialDays:     plan.TrialDays,
			QuotaLimit:    plan.QuotaLimit,
			OveragePolicy: plan.OveragePolicy,
			FeatureMatrix: plan.FeatureMatrix,
			IsDefault:     plan.IsDefault,
			Tiers:         convertPlanTierPayloads(plan.Tiers),
		}
	}
	return out
}

func convertPlanTierPayloads(tiers []planTierPayload) []svc.PlanTierInput {
	if len(tiers) == 0 {
		return nil
	}
	out := make([]svc.PlanTierInput, len(tiers))
	for i, tier := range tiers {
		out[i] = svc.PlanTierInput{
			ID:         tier.ID,
			Metric:     tier.Metric,
			RangeFrom:  tier.RangeFrom,
			RangeTo:    tier.RangeTo,
			UnitAmount: tier.UnitAmount,
			UnitName:   tier.UnitName,
		}
	}
	return out
}

func convertChecklistPayload(payload checklistPayload) *svc.ChecklistRunInput {
	input := &svc.ChecklistRunInput{
		ID:            payload.ID,
		TriggerSource: payload.TriggerSource,
		Summary:       payload.Summary,
		RunNumber:     payload.RunNumber,
		Status:        payload.Status,
		Items:         convertChecklistItems(payload.Items),
	}
	return input
}

func convertChecklistItems(items []checklistItemPayload) []svc.ChecklistItemInput {
	if len(items) == 0 {
		return nil
	}
	out := make([]svc.ChecklistItemInput, len(items))
	for i, item := range items {
		out[i] = svc.ChecklistItemInput{
			ID:          item.ID,
			Code:        item.Code,
			Description: item.Description,
			Result:      item.Result,
			EvidenceURI: item.EvidenceURI,
			Notes:       item.Notes,
			AutoFixLink: item.AutoFixLink,
		}
	}
	return out
}

func normalizeListFilter(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, raw := range values {
		if strings.TrimSpace(raw) == "" {
			continue
		}
		parts := strings.Split(raw, ",")
		for _, part := range parts {
			trimmed := strings.TrimSpace(part)
			if trimmed == "" {
				continue
			}
			key := strings.ToLower(trimmed)
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			out = append(out, trimmed)
		}
	}
	return out
}
