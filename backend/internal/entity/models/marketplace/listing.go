package marketplace

import (
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	ListingStatusDraft     = "draft"
	ListingStatusInReview  = "in_review"
	ListingStatusPublished = "published"
	ListingStatusSuspended = "suspended"
)

// Listing represents a marketplace listing submitted by a vendor.
type Listing struct {
	ID                  string            `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	TenantUuid          string            `gorm:"column:tenant_uuid;type:uuid;not null;index" json:"tenant_uuid"`
	PluginID            string            `gorm:"column:plugin_id;type:text;not null;index" json:"plugin_id"`
	VendorID            string            `gorm:"column:vendor_id;type:text;not null;index" json:"vendor_id"`
	Status              string            `gorm:"column:status;type:text;not null;default:'draft';index" json:"status"`
	Title               string            `gorm:"column:title;type:text;not null" json:"title"`
	Slug                string            `gorm:"column:slug;type:text;not null" json:"slug"`
	Summary             string            `gorm:"column:summary;type:text" json:"summary,omitempty"`
	Description         string            `gorm:"column:description;type:text" json:"description,omitempty"`
	CoverAssetID        *string           `gorm:"column:cover_asset_id;type:uuid" json:"cover_asset_id,omitempty"`
	HeroVideoAssetID    *string           `gorm:"column:hero_video_asset_id;type:uuid" json:"hero_video_asset_id,omitempty"`
	Categories          []string          `gorm:"column:categories;type:jsonb;serializer:json" json:"categories,omitempty"`
	Tags                []string          `gorm:"column:tags;type:jsonb;serializer:json" json:"tags,omitempty"`
	Locale              string            `gorm:"column:locale;type:text;not null;default:'en'" json:"locale"`
	Version             string            `gorm:"column:version;type:text" json:"version,omitempty"`
	ReadyChecklistScore int               `gorm:"column:ready_checklist_score;type:int;not null;default:0" json:"ready_checklist_score"`
	RecommendedWeight   float64           `gorm:"column:recommended_weight;type:numeric(10,4);not null;default:0" json:"recommended_weight"`
	PublishedAt         *time.Time        `gorm:"column:published_at;type:timestamptz" json:"published_at,omitempty"`
	ReviewedAt          *time.Time        `gorm:"column:reviewed_at;type:timestamptz" json:"reviewed_at,omitempty"`
	ReviewerID          *string           `gorm:"column:reviewer_id;type:text" json:"reviewer_id,omitempty"`
	AuditNotes          string            `gorm:"column:audit_notes;type:text" json:"audit_notes,omitempty"`
	BrandingTheme       datatypes.JSONMap `gorm:"column:branding_theme;type:jsonb" json:"branding_theme"`
	CreatedAt           time.Time         `gorm:"column:created_at;type:timestamptz;autoCreateTime" json:"created_at"`
	UpdatedAt           time.Time         `gorm:"column:updated_at;type:timestamptz;autoUpdateTime" json:"updated_at"`
	DeletedAt           gorm.DeletedAt    `gorm:"column:deleted_at;type:timestamptz" json:"deleted_at,omitempty"`
	Assets              []ListingAsset    `gorm:"foreignKey:ListingID" json:"assets,omitempty"`
	PricingPlans        []PricingPlan     `gorm:"foreignKey:ListingID" json:"pricing_plans,omitempty"`
}

func (*Listing) TableName() string {
	return models.S(models.TableMarketplaceListings)
}

// ListingVersion stores audit history for listing submissions.
type ListingVersion struct {
	ID          string            `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	ListingID   string            `gorm:"column:listing_id;type:uuid;not null;index" json:"listing_id"`
	TenantUuid  string            `gorm:"column:tenant_uuid;type:uuid;not null;index" json:"tenant_uuid"`
	Version     string            `gorm:"column:version;type:text;not null" json:"version"`
	Changelog   string            `gorm:"column:changelog;type:text" json:"changelog,omitempty"`
	Metadata    datatypes.JSONMap `gorm:"column:metadata;type:jsonb" json:"metadata"`
	SubmittedBy string            `gorm:"column:submitted_by;type:text;not null" json:"submitted_by"`
	ReviewState string            `gorm:"column:review_state;type:text;not null;default:'draft'" json:"review_state"`
	ReviewerID  *string           `gorm:"column:reviewer_id;type:text" json:"reviewer_id,omitempty"`
	ReviewedAt  *time.Time        `gorm:"column:reviewed_at;type:timestamptz" json:"reviewed_at,omitempty"`
	CreatedAt   time.Time         `gorm:"column:created_at;type:timestamptz;autoCreateTime" json:"created_at"`
}

func (*ListingVersion) TableName() string {
	return models.S(models.TableMarketplaceListingVersions)
}
