package marketplace

import (
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	"gorm.io/datatypes"
)

const (
	AssetTypeLogo       = "logo"
	AssetTypeCover      = "cover"
	AssetTypeScreenshot = "screenshot"
	AssetTypeVideo      = "video"
)

// ListingAsset stores media assets associated with a listing.
type ListingAsset struct {
	ID         string            `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	ListingID  string            `gorm:"column:listing_id;type:uuid;not null;index" json:"listing_id"`
	TenantUuid string            `gorm:"column:tenant_uuid;type:uuid;not null;index" json:"tenant_uuid"`
	AssetType  string            `gorm:"column:asset_type;type:text;not null" json:"asset_type"`
	StorageURI string            `gorm:"column:storage_uri;type:text;not null" json:"storage_uri"`
	Checksum   string            `gorm:"column:checksum;type:text" json:"checksum,omitempty"`
	IsPrimary  bool              `gorm:"column:is_primary;type:boolean;not null;default:false" json:"is_primary"`
	Locale     string            `gorm:"column:locale;type:text;not null;default:'en'" json:"locale"`
	Weight     int               `gorm:"column:weight;type:int;not null;default:0" json:"weight"`
	Metadata   datatypes.JSONMap `gorm:"column:metadata;type:jsonb" json:"metadata"`
	CreatedAt  time.Time         `gorm:"column:created_at;type:timestamptz;autoCreateTime" json:"created_at"`
	UpdatedAt  time.Time         `gorm:"column:updated_at;type:timestamptz;autoUpdateTime" json:"updated_at"`
}

func (*ListingAsset) TableName() string {
	return models.S(models.TableMarketplaceListingAssets)
}
