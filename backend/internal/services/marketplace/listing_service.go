package marketplace

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	dbm "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/marketplace"
	mrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/marketplace"
	obs "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/observability/marketplace"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// ListingService coordinates marketplace listing workflows.
type ListingService struct {
	listings    *mrepo.ListingRepository
	checklists  *mrepo.ChecklistRepository
	vendorGuard VendorGuard
	logger      *logrus.Entry
}

func NewListingService(listingRepo *mrepo.ListingRepository, checklistRepo *mrepo.ChecklistRepository, logger *logrus.Entry) *ListingService {
	return &ListingService{
		listings:   listingRepo,
		checklists: checklistRepo,
		logger:     logger,
	}
}

// VendorGuard checks vendor KYC status for gating submissions.
type VendorGuard interface {
	VendorRevoked(ctx context.Context, vendorID string) (bool, error)
}

// SetVendorGuard injects vendor KYC guard implementation.
func (s *ListingService) SetVendorGuard(guard VendorGuard) {
	s.vendorGuard = guard
}

// ListingDraftInput captures initial listing information.
type ListingDraftInput struct {
	ID             string
	PluginID       string
	VendorID       string
	Title          string
	Slug           string
	Summary        string
	Description    string
	Locale         string
	Categories     []string
	Tags           []string
	BrandingTheme  map[string]any
	Assets         []ListingAssetInput
	PricingPlans   []PricingPlanInput
	ReadyChecklist *ChecklistRunInput
}

// ListingUpdateInput allows partial updates to listing metadata.
type ListingUpdateInput struct {
	Title         *string
	Summary       *string
	Description   *string
	Categories    *[]string
	Tags          *[]string
	BrandingTheme map[string]any
	Locale        *string
	Assets        []ListingAssetInput
	PricingPlans  []PricingPlanInput
}

// ListingAssetInput describes an asset mutation.
type ListingAssetInput struct {
	ID         string
	AssetType  string
	StorageURI string
	Checksum   string
	IsPrimary  bool
	Locale     string
	Weight     int
	Metadata   map[string]any
}

// PricingPlanInput describes plan state.
type PricingPlanInput struct {
	ID            string
	PlanCode      string
	PlanType      string
	Currency      string
	Amount        *float64
	BillingPeriod string
	TrialDays     *int
	QuotaLimit    *float64
	OveragePolicy string
	FeatureMatrix map[string]any
	IsDefault     bool
	Tiers         []PlanTierInput
}

// PlanTierInput describes a usage tier.
type PlanTierInput struct {
	ID         string
	Metric     string
	RangeFrom  float64
	RangeTo    *float64
	UnitAmount float64
	UnitName   string
}

// ChecklistRunInput captures checklist details.
type ChecklistRunInput struct {
	ID            string
	TriggerSource string
	Summary       string
	RunNumber     int
	Items         []ChecklistItemInput
	Status        string
}

// ChecklistItemInput captures evaluation result.
type ChecklistItemInput struct {
	ID          string
	Code        string
	Description string
	Result      string
	EvidenceURI string
	Notes       string
	AutoFixLink string
}

// ListListingsOptions defines filters applied to listing queries.
type ListListingsOptions struct {
	Status []string
	Locale string
	Search string
	Limit  int
	Offset int
}

// ListListings returns paginated listings for a tenant.
func (s *ListingService) ListListings(ctx context.Context, tenantID string, opts ListListingsOptions) ([]*dbm.Listing, int64, error) {
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		return nil, 0, errors.New("tenant_uuid is required")
	}
	query := mrepo.ListingQuery{
		Status: dedupeStrings(opts.Status),
		Locale: strings.TrimSpace(opts.Locale),
		Search: opts.Search,
		Limit:  opts.Limit,
		Offset: opts.Offset,
	}
	return s.listings.List(ctx, tenantID, query)
}

// GetListing fetches a listing with related aggregates.
func (s *ListingService) GetListing(ctx context.Context, tenantID, listingID string) (*dbm.Listing, error) {
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		return nil, errors.New("tenant_uuid is required")
	}
	listingID = strings.TrimSpace(listingID)
	if listingID == "" {
		return nil, errors.New("listing_id is required")
	}
	return s.listings.FindByID(ctx, tenantID, listingID)
}

// ListChecklistRuns returns recent checklist runs for the listing.
func (s *ListingService) ListChecklistRuns(ctx context.Context, tenantID, listingID string, limit int) ([]*dbm.ChecklistRun, error) {
	if s.checklists == nil {
		return nil, errors.New("checklist repository not configured")
	}
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		return nil, errors.New("tenant_uuid is required")
	}
	listingID = strings.TrimSpace(listingID)
	if listingID == "" {
		return nil, errors.New("listing_id is required")
	}
	return s.checklists.ListRuns(ctx, tenantID, listingID, limit)
}

// LatestChecklistRun fetches the most recent run for a listing.
func (s *ListingService) LatestChecklistRun(ctx context.Context, tenantID, listingID string) (*dbm.ChecklistRun, error) {
	if s.checklists == nil {
		return nil, errors.New("checklist repository not configured")
	}
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		return nil, errors.New("tenant_uuid is required")
	}
	listingID = strings.TrimSpace(listingID)
	if listingID == "" {
		return nil, errors.New("listing_id is required")
	}
	return s.checklists.LatestRun(ctx, tenantID, listingID)
}

// CreateDraft stores a listing draft together with assets and pricing.
func (s *ListingService) CreateDraft(ctx context.Context, tenantID string, input ListingDraftInput) (*dbm.Listing, error) {
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		return nil, errors.New("tenant_uuid is required")
	}
	if err := validateDraftInput(input); err != nil {
		return nil, err
	}
	if err := validateBrandAssets(input.Assets); err != nil {
		return nil, err
	}

	listing := &dbm.Listing{
		ID:          coalesceID(input.ID),
		TenantUuid:  tenantID,
		PluginID:    strings.TrimSpace(input.PluginID),
		VendorID:    strings.TrimSpace(input.VendorID),
		Title:       strings.TrimSpace(input.Title),
		Slug:        strings.TrimSpace(input.Slug),
		Summary:     input.Summary,
		Description: input.Description,
		Locale:      fallbackLocale(input.Locale),
		Status:      dbm.ListingStatusDraft,
		Categories:  dedupeStrings(input.Categories),
		Tags:        dedupeStrings(input.Tags),
	}

	if len(input.BrandingTheme) > 0 {
		listing.BrandingTheme = toJSONMap(input.BrandingTheme)
	}

	if err := s.listings.Create(ctx, listing); err != nil {
		return nil, err
	}

	if err := s.listings.ReplaceAssets(ctx, tenantID, listing.ID, convertAssets(input.Assets)); err != nil {
		return nil, err
	}
	if err := s.listings.ReplacePricingPlans(ctx, tenantID, listing.ID, convertPricingPlans(input.PricingPlans, tenantID)); err != nil {
		return nil, err
	}

	if input.ReadyChecklist != nil && s.checklists != nil {
		if err := s.createChecklistRun(ctx, tenantID, listing.ID, *input.ReadyChecklist); err != nil {
			return nil, err
		}
	}

	return s.listings.FindByID(ctx, tenantID, listing.ID)
}

// UpdateDraft mutates a draft listing metadata and optionally assets/pricing.
func (s *ListingService) UpdateDraft(ctx context.Context, tenantID, listingID string, input ListingUpdateInput) (*dbm.Listing, error) {
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		return nil, errors.New("tenant_uuid is required")
	}
	listingID = strings.TrimSpace(listingID)
	if listingID == "" {
		return nil, errors.New("listing_id is required")
	}
	listing, err := s.listings.FindByID(ctx, tenantID, listingID)
	if err != nil {
		return nil, err
	}
	if listing.Status != dbm.ListingStatusDraft && listing.Status != dbm.ListingStatusInReview {
		return nil, fmt.Errorf("listing %s cannot be updated from status %s", listingID, listing.Status)
	}

	if input.Title != nil {
		listing.Title = strings.TrimSpace(*input.Title)
	}
	if input.Summary != nil {
		listing.Summary = *input.Summary
	}
	if input.Description != nil {
		listing.Description = *input.Description
	}
	if input.Locale != nil && strings.TrimSpace(*input.Locale) != "" {
		listing.Locale = strings.TrimSpace(*input.Locale)
	}
	if input.Categories != nil {
		listing.Categories = dedupeStrings(*input.Categories)
	}
	if input.Tags != nil {
		listing.Tags = dedupeStrings(*input.Tags)
	}
	if input.BrandingTheme != nil {
		listing.BrandingTheme = toJSONMap(input.BrandingTheme)
	}

	if err := s.listings.Update(ctx, listing); err != nil {
		return nil, err
	}
	if input.Assets != nil {
		if err := validateBrandAssets(input.Assets); err != nil {
			return nil, err
		}
		if err := s.listings.ReplaceAssets(ctx, tenantID, listing.ID, convertAssets(input.Assets)); err != nil {
			return nil, err
		}
	}
	if input.PricingPlans != nil {
		if err := s.listings.ReplacePricingPlans(ctx, tenantID, listing.ID, convertPricingPlans(input.PricingPlans, tenantID)); err != nil {
			return nil, err
		}
	}
	return s.listings.FindByID(ctx, tenantID, listing.ID)
}

// SubmitForReview transitions a listing into review and records a version snapshot.
func (s *ListingService) SubmitForReview(ctx context.Context, tenantID, listingID, submittedBy string, metadata map[string]any) (listing *dbm.Listing, err error) {
	tenantID = strings.TrimSpace(tenantID)
	listingID = strings.TrimSpace(listingID)
	start := time.Now()
	defer func() {
		status := "success"
		if err != nil {
			status = "failed"
		}
		obs.ObserveListingSubmission(status, tenantID, time.Since(start))
	}()

	if tenantID == "" {
		err = errors.New("tenant_uuid is required")
		return
	}
	if listingID == "" {
		err = errors.New("listing_id is required")
		return
	}

	listing, err = s.listings.FindByID(ctx, tenantID, listingID)
	if err != nil {
		return
	}
	if len(listing.Assets) == 0 {
		err = fmt.Errorf("listing %s has no assets and cannot be submitted", listingID)
		return
	}
	if vendorErr := s.ensureVendorActive(ctx, listing.VendorID); vendorErr != nil {
		err = vendorErr
		return
	}
	if listing.Status != dbm.ListingStatusDraft && listing.Status != dbm.ListingStatusSuspended {
		err = fmt.Errorf("listing %s cannot enter review from %s", listingID, listing.Status)
		return
	}

	listing.Status = dbm.ListingStatusInReview
	if err = s.listings.Update(ctx, listing); err != nil {
		return
	}

	version := &dbm.ListingVersion{
		ID:          uuid.NewString(),
		ListingID:   listing.ID,
		TenantUuid:  tenantID,
		Version:     coalesce(listing.Version, "draft"),
		Changelog:   metadataString(metadata, "changelog"),
		Metadata:    toJSONMap(metadata),
		SubmittedBy: submittedBy,
		ReviewState: dbm.ListingStatusInReview,
	}
	if err = s.listings.CreateVersion(ctx, version); err != nil {
		return
	}
	return
}

// UpdateListingStatus updates reviewer decision.
func (s *ListingService) UpdateListingStatus(ctx context.Context, tenantID, listingID, reviewerID, status, notes string) (*dbm.Listing, error) {
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		return nil, errors.New("tenant_uuid is required")
	}
	listingID = strings.TrimSpace(listingID)
	if listingID == "" {
		return nil, errors.New("listing_id is required")
	}
	if !isValidStatusTransition(status) {
		return nil, fmt.Errorf("invalid target status %s", status)
	}
	listing, err := s.listings.FindByID(ctx, tenantID, listingID)
	if err != nil {
		return nil, err
	}
	if status == dbm.ListingStatusPublished {
		if len(listing.Assets) == 0 {
			return nil, fmt.Errorf("listing %s cannot be published without assets", listingID)
		}
		if err := s.ensureVendorActive(ctx, listing.VendorID); err != nil {
			return nil, err
		}
	}
	listing.Status = status
	listing.ReviewerID = &reviewerID
	now := time.Now().UTC()
	listing.ReviewedAt = &now
	listing.AuditNotes = notes
	if status == dbm.ListingStatusPublished {
		listing.PublishedAt = &now
	}
	if err := s.listings.Update(ctx, listing); err != nil {
		return nil, err
	}
	return listing, nil
}

// RecordChecklistRun persists checklist result and updates listing score.
func (s *ListingService) RecordChecklistRun(ctx context.Context, tenantID, listingID string, input ChecklistRunInput) (*dbm.ChecklistRun, error) {
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		return nil, errors.New("tenant_uuid is required")
	}
	listingID = strings.TrimSpace(listingID)
	if listingID == "" {
		return nil, errors.New("listing_id is required")
	}
	if s.checklists == nil {
		return nil, errors.New("checklist repository not configured")
	}
	run := dbm.ChecklistRun{
		ID:            coalesceID(input.ID),
		ListingID:     listingID,
		TenantUuid:    tenantID,
		TriggerSource: fallbackTrigger(input.TriggerSource),
		RunNumber:     nextRunNumber(input.RunNumber),
		Status:        fallbackStatus(input.Status),
		Summary:       input.Summary,
		StartedAt:     time.Now().UTC(),
	}
	items := make([]dbm.ChecklistItem, len(input.Items))
	for i, item := range input.Items {
		items[i] = dbm.ChecklistItem{
			ID:          coalesceID(item.ID),
			Code:        item.Code,
			Description: item.Description,
			Result:      fallbackChecklistResult(item.Result),
			EvidenceURI: item.EvidenceURI,
			Notes:       item.Notes,
			AutoFixLink: item.AutoFixLink,
		}
	}
	run.Items = items
	if err := s.checklists.CreateRun(ctx, &run, items); err != nil {
		return nil, err
	}

	if err := s.updateReadyScoreFromRun(ctx, tenantID, listingID, &run); err != nil && s.logger != nil {
		s.logger.WithError(err).Warn("failed to update listing checklist score")
	}
	return &run, nil
}

func (s *ListingService) createChecklistRun(ctx context.Context, tenantID, listingID string, input ChecklistRunInput) error {
	if s.checklists == nil {
		return nil
	}
	_, err := s.RecordChecklistRun(ctx, tenantID, listingID, input)
	return err
}

func (s *ListingService) updateReadyScoreFromRun(ctx context.Context, tenantID, listingID string, run *dbm.ChecklistRun) error {
	if run == nil {
		return nil
	}
	listing, err := s.listings.FindByID(ctx, tenantID, listingID)
	if err != nil {
		return err
	}
	passed, total := 0, len(run.Items)
	if total == 0 {
		total = 1
	}
	for _, item := range run.Items {
		if strings.EqualFold(item.Result, dbm.ChecklistStatusPassed) {
			passed++
		}
	}
	score := int(float64(passed) / float64(total) * 100)
	listing.ReadyChecklistScore = score
	return s.listings.Update(ctx, listing)
}

func (s *ListingService) ensureVendorActive(ctx context.Context, vendorID string) error {
	if s.vendorGuard == nil {
		return nil
	}
	vendorID = strings.TrimSpace(vendorID)
	if vendorID == "" {
		return errors.New("vendor_id is required")
	}
	revoked, err := s.vendorGuard.VendorRevoked(ctx, vendorID)
	if err != nil {
		return err
	}
	if revoked {
		return fmt.Errorf("vendor %s is not permitted to submit listings", vendorID)
	}
	return nil
}

func convertAssets(inputs []ListingAssetInput) []dbm.ListingAsset {
	if len(inputs) == 0 {
		return nil
	}
	res := make([]dbm.ListingAsset, len(inputs))
	for i, input := range inputs {
		res[i] = dbm.ListingAsset{
			ID:         coalesceID(input.ID),
			AssetType:  input.AssetType,
			StorageURI: input.StorageURI,
			Checksum:   input.Checksum,
			IsPrimary:  input.IsPrimary,
			Locale:     fallbackLocale(input.Locale),
			Weight:     input.Weight,
			Metadata:   toJSONMap(input.Metadata),
		}
	}
	return res
}

func convertPricingPlans(inputs []PricingPlanInput, tenantID string) []dbm.PricingPlan {
	if len(inputs) == 0 {
		return nil
	}
	out := make([]dbm.PricingPlan, len(inputs))
	for i, in := range inputs {
		plan := dbm.PricingPlan{
			ID:              coalesceID(in.ID),
			TenantUuid:      tenantID,
			PlanCode:        in.PlanCode,
			PlanType:        fallbackPlanType(in.PlanType),
			Currency:        strings.ToUpper(strings.TrimSpace(in.Currency)),
			Amount:          in.Amount,
			BillingPeriod:   strings.TrimSpace(in.BillingPeriod),
			TrialPeriodDays: in.TrialDays,
			QuotaLimit:      in.QuotaLimit,
			OveragePolicy:   in.OveragePolicy,
			FeatureMatrix:   toJSONMap(in.FeatureMatrix),
			IsDefault:       in.IsDefault,
		}
		if len(in.Tiers) > 0 {
			plan.Tiers = make([]dbm.PlanTier, len(in.Tiers))
			for j, tier := range in.Tiers {
				plan.Tiers[j] = dbm.PlanTier{
					ID:         coalesceID(tier.ID),
					TenantUuid: tenantID,
					Metric:     tier.Metric,
					RangeFrom:  tier.RangeFrom,
					RangeTo:    tier.RangeTo,
					UnitAmount: tier.UnitAmount,
					UnitName:   tier.UnitName,
				}
			}
		}
		out[i] = plan
	}
	return out
}

func validateDraftInput(input ListingDraftInput) error {
	switch {
	case strings.TrimSpace(input.PluginID) == "":
		return errors.New("plugin_id is required")
	case strings.TrimSpace(input.VendorID) == "":
		return errors.New("vendor_id is required")
	case strings.TrimSpace(input.Title) == "":
		return errors.New("title is required")
	case strings.TrimSpace(input.Slug) == "":
		return errors.New("slug is required")
	}
	return nil
}

func dedupeStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, v := range values {
		trim := strings.TrimSpace(v)
		if trim == "" {
			continue
		}
		lower := strings.ToLower(trim)
		if _, ok := seen[lower]; ok {
			continue
		}
		seen[lower] = struct{}{}
		out = append(out, trim)
	}
	return out
}

func fallbackLocale(locale string) string {
	if strings.TrimSpace(locale) == "" {
		return "en"
	}
	return locale
}

func fallbackTrigger(trigger string) string {
	if strings.TrimSpace(trigger) == "" {
		return dbm.ChecklistTriggerVendor
	}
	return trigger
}

func fallbackStatus(status string) string {
	if strings.TrimSpace(status) == "" {
		return dbm.ChecklistStatusPending
	}
	return status
}

func fallbackChecklistResult(result string) string {
	if strings.TrimSpace(result) == "" {
		return dbm.ChecklistStatusPending
	}
	return result
}

func fallbackPlanType(planType string) string {
	switch strings.TrimSpace(planType) {
	case dbm.PricingPlanTypeFree,
		dbm.PricingPlanTypeOneTime,
		dbm.PricingPlanTypeSubscription,
		dbm.PricingPlanTypeUsage:
		return strings.TrimSpace(planType)
	default:
		return dbm.PricingPlanTypeFree
	}
}

func coalesceID(id string) string {
	if strings.TrimSpace(id) != "" {
		return id
	}
	return uuid.NewString()
}

func coalesce(value, fallback string) string {
	if strings.TrimSpace(value) != "" {
		return value
	}
	return fallback
}

func nextRunNumber(num int) int {
	if num > 0 {
		return num
	}
	return 1
}

func metadataString(metadata map[string]any, key string) string {
	if metadata == nil {
		return ""
	}
	if v, ok := metadata[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func isValidStatusTransition(status string) bool {
	switch status {
	case dbm.ListingStatusInReview, dbm.ListingStatusPublished, dbm.ListingStatusSuspended:
		return true
	default:
		return false
	}
}

func validateBrandAssets(assets []ListingAssetInput) error {
	for _, asset := range assets {
		atype := strings.ToLower(strings.TrimSpace(asset.AssetType))
		uri := strings.TrimSpace(asset.StorageURI)
		if uri == "" {
			return errors.New("listing asset storage_uri is required")
		}

		switch atype {
		case "logo":
			if !strings.HasSuffix(uri, ".svg") && !strings.HasSuffix(uri, ".png") {
				return fmt.Errorf("logo asset must be SVG or PNG: %s", uri)
			}
		case "cover":
			width, okW := metadataFloat(asset.Metadata, "width")
			height, okH := metadataFloat(asset.Metadata, "height")
			if okW && okH && width > 0 && height > 0 {
				ratio := width / height
				if math.Abs(ratio-(16.0/9.0)) > 0.1 {
					return fmt.Errorf("cover asset must approximate 16:9 aspect ratio, got %.2f", ratio)
				}
			} else {
				return errors.New("cover asset requires width and height metadata")
			}
		case "video":
			if !strings.HasSuffix(uri, ".mp4") && !strings.HasSuffix(uri, ".webm") {
				return fmt.Errorf("video asset must be MP4 or WebM: %s", uri)
			}
			duration, okDuration := metadataFloat(asset.Metadata, "duration_seconds")
			sizeMB, okSize := metadataFloat(asset.Metadata, "size_mb")
			if !okDuration || duration <= 0 {
				return errors.New("video asset requires duration_seconds metadata")
			}
			if duration > 15 {
				return fmt.Errorf("video duration must be <= 15s, got %.0fs", duration)
			}
			if !okSize || sizeMB <= 0 {
				return errors.New("video asset requires size_mb metadata")
			}
			if sizeMB > 50 {
				return fmt.Errorf("video size must be <= 50MB, got %.1fMB", sizeMB)
			}
		}
	}
	return nil
}

func metadataFloat(meta map[string]any, key string) (float64, bool) {
	if meta == nil {
		return 0, false
	}
	value, ok := meta[key]
	if !ok {
		return 0, false
	}
	switch v := value.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case uint:
		return float64(v), true
	case uint64:
		return float64(v), true
	case string:
		if val, err := strconv.ParseFloat(v, 64); err == nil {
			return val, true
		}
	}
	return 0, false
}
