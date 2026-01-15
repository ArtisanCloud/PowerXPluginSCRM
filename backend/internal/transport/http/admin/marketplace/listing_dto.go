package marketplace

import (
	"time"

	dbm "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/marketplace"
	"gorm.io/datatypes"
)

// ListingResponse represents the admin listing payload.
type ListingResponse struct {
	ID                  string                 `json:"id"`
	PluginID            string                 `json:"plugin_id"`
	VendorID            string                 `json:"vendor_id"`
	Status              string                 `json:"status"`
	Title               string                 `json:"title"`
	Slug                string                 `json:"slug"`
	Summary             string                 `json:"summary,omitempty"`
	Description         string                 `json:"description,omitempty"`
	CoverAssetID        *string                `json:"cover_asset_id,omitempty"`
	HeroVideoAssetID    *string                `json:"hero_video_asset_id,omitempty"`
	Categories          []string               `json:"categories,omitempty"`
	Tags                []string               `json:"tags,omitempty"`
	Locale              string                 `json:"locale"`
	Version             string                 `json:"version,omitempty"`
	ReadyChecklistScore int                    `json:"ready_checklist_score"`
	RecommendedWeight   float64                `json:"recommended_weight"`
	PublishedAt         *time.Time             `json:"published_at,omitempty"`
	ReviewedAt          *time.Time             `json:"reviewed_at,omitempty"`
	ReviewerID          *string                `json:"reviewer_id,omitempty"`
	AuditNotes          string                 `json:"audit_notes,omitempty"`
	BrandingTheme       map[string]any         `json:"branding_theme"`
	CreatedAt           time.Time              `json:"created_at"`
	UpdatedAt           time.Time              `json:"updated_at"`
	Assets              []ListingAssetResponse `json:"assets,omitempty"`
	PricingPlans        []PricingPlanResponse  `json:"pricing_plans,omitempty"`
}

// ListingAssetResponse encodes listing asset information.
type ListingAssetResponse struct {
	ID         string         `json:"id"`
	ListingID  string         `json:"listing_id"`
	AssetType  string         `json:"asset_type"`
	StorageURI string         `json:"storage_uri"`
	Checksum   string         `json:"checksum,omitempty"`
	IsPrimary  bool           `json:"is_primary"`
	Locale     string         `json:"locale"`
	Weight     int            `json:"weight"`
	Metadata   map[string]any `json:"metadata"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
}

// PricingPlanResponse encodes pricing plan and tiers.
type PricingPlanResponse struct {
	ID              string                `json:"id"`
	ListingID       string                `json:"listing_id"`
	PlanCode        string                `json:"plan_code"`
	PlanType        string                `json:"plan_type"`
	Currency        string                `json:"currency"`
	Amount          *float64              `json:"amount,omitempty"`
	BillingPeriod   string                `json:"billing_period,omitempty"`
	TrialPeriodDays *int                  `json:"trial_period_days,omitempty"`
	QuotaLimit      *float64              `json:"quota_limit,omitempty"`
	OveragePolicy   string                `json:"overage_policy,omitempty"`
	FeatureMatrix   map[string]any        `json:"feature_matrix"`
	IsDefault       bool                  `json:"is_default"`
	CreatedAt       time.Time             `json:"created_at"`
	UpdatedAt       time.Time             `json:"updated_at"`
	Tiers           []PricingTierResponse `json:"tiers,omitempty"`
}

// PricingTierResponse encodes a usage tier.
type PricingTierResponse struct {
	ID         string    `json:"id"`
	PlanID     string    `json:"plan_id"`
	Metric     string    `json:"metric"`
	RangeFrom  float64   `json:"range_from"`
	RangeTo    *float64  `json:"range_to,omitempty"`
	UnitAmount float64   `json:"unit_amount"`
	UnitName   string    `json:"unit_name,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// ChecklistRunResponse encodes run details for GraphQL responses.
type ChecklistRunResponse struct {
	ID            string                  `json:"id"`
	ListingID     string                  `json:"listing_id"`
	TriggerSource string                  `json:"trigger_source"`
	RunNumber     int                     `json:"run_number"`
	Status        string                  `json:"status"`
	StartedAt     time.Time               `json:"started_at"`
	CompletedAt   *time.Time              `json:"completed_at,omitempty"`
	Summary       string                  `json:"summary,omitempty"`
	CiPipelineID  string                  `json:"ci_pipeline_id,omitempty"`
	Items         []ChecklistItemResponse `json:"items,omitempty"`
}

// ChecklistItemResponse encodes checklist item data.
type ChecklistItemResponse struct {
	ID          string `json:"id"`
	Code        string `json:"code"`
	Description string `json:"description"`
	Result      string `json:"result"`
	EvidenceURI string `json:"evidence_uri,omitempty"`
	Notes       string `json:"notes,omitempty"`
	AutoFixLink string `json:"auto_fix_link,omitempty"`
}

// NewListingResponse converts the domain listing to response payload.
func NewListingResponse(listing *dbm.Listing) ListingResponse {
	if listing == nil {
		return ListingResponse{}
	}
	resp := ListingResponse{
		ID:                  listing.ID,
		PluginID:            listing.PluginID,
		VendorID:            listing.VendorID,
		Status:              listing.Status,
		Title:               listing.Title,
		Slug:                listing.Slug,
		Summary:             listing.Summary,
		Description:         listing.Description,
		CoverAssetID:        listing.CoverAssetID,
		HeroVideoAssetID:    listing.HeroVideoAssetID,
		Categories:          listing.Categories,
		Tags:                listing.Tags,
		Locale:              listing.Locale,
		Version:             listing.Version,
		ReadyChecklistScore: listing.ReadyChecklistScore,
		RecommendedWeight:   listing.RecommendedWeight,
		PublishedAt:         listing.PublishedAt,
		ReviewedAt:          listing.ReviewedAt,
		ReviewerID:          listing.ReviewerID,
		AuditNotes:          listing.AuditNotes,
		BrandingTheme:       jsonMapToGeneric(listing.BrandingTheme),
		CreatedAt:           listing.CreatedAt,
		UpdatedAt:           listing.UpdatedAt,
	}
	if len(listing.Assets) > 0 {
		resp.Assets = make([]ListingAssetResponse, len(listing.Assets))
		for i, asset := range listing.Assets {
			resp.Assets[i] = ListingAssetResponse{
				ID:         asset.ID,
				ListingID:  asset.ListingID,
				AssetType:  asset.AssetType,
				StorageURI: asset.StorageURI,
				Checksum:   asset.Checksum,
				IsPrimary:  asset.IsPrimary,
				Locale:     asset.Locale,
				Weight:     asset.Weight,
				Metadata:   jsonMapToGeneric(asset.Metadata),
				CreatedAt:  asset.CreatedAt,
				UpdatedAt:  asset.UpdatedAt,
			}
		}
	}
	if len(listing.PricingPlans) > 0 {
		resp.PricingPlans = make([]PricingPlanResponse, len(listing.PricingPlans))
		for i, plan := range listing.PricingPlans {
			pp := PricingPlanResponse{
				ID:              plan.ID,
				ListingID:       plan.ListingID,
				PlanCode:        plan.PlanCode,
				PlanType:        plan.PlanType,
				Currency:        plan.Currency,
				Amount:          plan.Amount,
				BillingPeriod:   plan.BillingPeriod,
				TrialPeriodDays: plan.TrialPeriodDays,
				QuotaLimit:      plan.QuotaLimit,
				OveragePolicy:   plan.OveragePolicy,
				FeatureMatrix:   jsonMapToGeneric(plan.FeatureMatrix),
				IsDefault:       plan.IsDefault,
				CreatedAt:       plan.CreatedAt,
				UpdatedAt:       plan.UpdatedAt,
			}
			if len(plan.Tiers) > 0 {
				pp.Tiers = make([]PricingTierResponse, len(plan.Tiers))
				for j, tier := range plan.Tiers {
					pp.Tiers[j] = PricingTierResponse{
						ID:         tier.ID,
						PlanID:     tier.PlanID,
						Metric:     tier.Metric,
						RangeFrom:  tier.RangeFrom,
						RangeTo:    tier.RangeTo,
						UnitAmount: tier.UnitAmount,
						UnitName:   tier.UnitName,
						CreatedAt:  tier.CreatedAt,
						UpdatedAt:  tier.UpdatedAt,
					}
				}
			}
			resp.PricingPlans[i] = pp
		}
	}
	return resp
}

// NewListingListResponse converts listing slice to DTOs.
func NewListingListResponse(listings []*dbm.Listing) []ListingResponse {
	if len(listings) == 0 {
		return []ListingResponse{}
	}
	resp := make([]ListingResponse, len(listings))
	for i, listing := range listings {
		resp[i] = NewListingResponse(listing)
	}
	return resp
}

// NewChecklistRunResponse converts checklist run to DTO.
func NewChecklistRunResponse(run *dbm.ChecklistRun) ChecklistRunResponse {
	if run == nil {
		return ChecklistRunResponse{}
	}
	resp := ChecklistRunResponse{
		ID:            run.ID,
		ListingID:     run.ListingID,
		TriggerSource: run.TriggerSource,
		RunNumber:     run.RunNumber,
		Status:        run.Status,
		StartedAt:     run.StartedAt,
		CompletedAt:   run.CompletedAt,
		Summary:       run.Summary,
		CiPipelineID:  run.CIPipelineID,
	}
	if len(run.Items) > 0 {
		resp.Items = make([]ChecklistItemResponse, len(run.Items))
		for i, item := range run.Items {
			resp.Items[i] = ChecklistItemResponse{
				ID:          item.ID,
				Code:        item.Code,
				Description: item.Description,
				Result:      item.Result,
				EvidenceURI: item.EvidenceURI,
				Notes:       item.Notes,
				AutoFixLink: item.AutoFixLink,
			}
		}
	}
	return resp
}

// NewChecklistRunListResponse converts runs into DTOs.
func NewChecklistRunListResponse(runs []*dbm.ChecklistRun) []ChecklistRunResponse {
	if len(runs) == 0 {
		return []ChecklistRunResponse{}
	}
	out := make([]ChecklistRunResponse, len(runs))
	for i, run := range runs {
		out[i] = NewChecklistRunResponse(run)
	}
	return out
}

func jsonMapToGeneric(m datatypes.JSONMap) map[string]any {
	if len(m) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}
