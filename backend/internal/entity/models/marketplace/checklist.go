package marketplace

import (
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
)

const (
	ChecklistTriggerVendor = "vendor"
	ChecklistTriggerCI     = "ci"
	ChecklistTriggerAuto   = "auto"

	ChecklistStatusPending = "pending"
	ChecklistStatusPassed  = "passed"
	ChecklistStatusFailed  = "failed"
)

// ChecklistRun records an execution of the ready checklist for a listing.
type ChecklistRun struct {
	ID            string          `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	ListingID     string          `gorm:"column:listing_id;type:uuid;not null;index" json:"listing_id"`
	TenantUuid    string          `gorm:"column:tenant_uuid;type:uuid;not null;index" json:"tenant_uuid"`
	TriggerSource string          `gorm:"column:trigger_source;type:text;not null" json:"trigger_source"`
	RunNumber     int             `gorm:"column:run_number;type:int;not null;default:1" json:"run_number"`
	Status        string          `gorm:"column:status;type:text;not null;default:'pending'" json:"status"`
	StartedAt     time.Time       `gorm:"column:started_at;type:timestamptz;not null;default:now()" json:"started_at"`
	CompletedAt   *time.Time      `gorm:"column:completed_at;type:timestamptz" json:"completed_at,omitempty"`
	Summary       string          `gorm:"column:summary;type:text" json:"summary,omitempty"`
	CIPipelineID  string          `gorm:"column:ci_pipeline_id;type:text" json:"ci_pipeline_id,omitempty"`
	CreatedAt     time.Time       `gorm:"column:created_at;type:timestamptz;autoCreateTime" json:"created_at"`
	Items         []ChecklistItem `gorm:"foreignKey:ChecklistRunID" json:"items,omitempty"`
}

func (*ChecklistRun) TableName() string {
	return models.S(models.TableMarketplaceChecklistRuns)
}

// ChecklistItem stores per-item checklist evaluation.
type ChecklistItem struct {
	ID             string    `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	ChecklistRunID string    `gorm:"column:checklist_run_id;type:uuid;not null;index" json:"checklist_run_id"`
	TenantUuid     string    `gorm:"column:tenant_uuid;type:uuid;not null;index" json:"tenant_uuid"`
	Code           string    `gorm:"column:code;type:text;not null" json:"code"`
	Description    string    `gorm:"column:description;type:text;not null" json:"description"`
	Result         string    `gorm:"column:result;type:text;not null" json:"result"`
	EvidenceURI    string    `gorm:"column:evidence_uri;type:text" json:"evidence_uri,omitempty"`
	Notes          string    `gorm:"column:notes;type:text" json:"notes,omitempty"`
	AutoFixLink    string    `gorm:"column:auto_fix_link;type:text" json:"auto_fix_link,omitempty"`
	CreatedAt      time.Time `gorm:"column:created_at;type:timestamptz;autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time `gorm:"column:updated_at;type:timestamptz;autoUpdateTime" json:"updated_at"`
}

func (*ChecklistItem) TableName() string {
	return models.S(models.TableMarketplaceChecklistItems)
}
