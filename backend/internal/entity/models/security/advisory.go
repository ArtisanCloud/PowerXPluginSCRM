package security

import (
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	"gorm.io/datatypes"
)

// Advisory severity levels.
const (
	AdvisorySeverityCritical = "CRITICAL"
	AdvisorySeverityHigh     = "HIGH"
	AdvisorySeverityMedium   = "MEDIUM"
	AdvisorySeverityLow      = "LOW"
)

// Advisory lifecycle statuses.
const (
	AdvisoryStatusOpen      = "OPEN"
	AdvisoryStatusPatched   = "PATCHED"
	AdvisoryStatusPublished = "PUBLISHED"
	AdvisoryStatusClosed    = "CLOSED"
)

// Advisory represents a vulnerability advisory record.
type Advisory struct {
	ID               string                      `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	Reference        string                      `gorm:"column:reference;type:text;not null" json:"reference"`
	Severity         string                      `gorm:"column:severity;type:text;not null" json:"severity"`
	Status           string                      `gorm:"column:status;type:text;not null" json:"status"`
	AffectedVersions datatypes.JSONSlice[string] `gorm:"column:affected_versions;type:jsonb;not null" json:"affected_versions"`
	PatchedInVersion string                      `gorm:"column:patched_in_version;type:text" json:"patched_in_version,omitempty"`
	Summary          string                      `gorm:"column:summary;type:text;not null" json:"summary"`
	DetailsMarkdown  string                      `gorm:"column:details_markdown;type:text" json:"details_markdown,omitempty"`
	PublishedAt      *time.Time                  `gorm:"column:published_at;type:timestamptz" json:"published_at,omitempty"`
	PatchedAt        *time.Time                  `gorm:"column:patched_at;type:timestamptz" json:"patched_at,omitempty"`
	ClosedAt         *time.Time                  `gorm:"column:closed_at;type:timestamptz" json:"closed_at,omitempty"`
	SlaDeadline      *time.Time                  `gorm:"column:sla_deadline;type:timestamptz" json:"sla_deadline,omitempty"`
	CreatedAt        time.Time                   `gorm:"column:created_at;type:timestamptz;autoCreateTime" json:"created_at"`
	UpdatedAt        time.Time                   `gorm:"column:updated_at;type:timestamptz;autoUpdateTime" json:"updated_at"`
}

// TableName returns the fully qualified table name.
func (*Advisory) TableName() string {
	return models.S(models.TableSecurityVulnerabilityAdvisory)
}

// AffectedVersionList returns the affected versions as a slice of strings.
func (a *Advisory) AffectedVersionList() []string {
	if a == nil || len(a.AffectedVersions) == 0 {
		return nil
	}
	return append([]string(nil), a.AffectedVersions...)
}

// SetAffectedVersions hydrates the advisory with the provided version list.
func (a *Advisory) SetAffectedVersions(versions []string) {
	if a == nil {
		return
	}
	if len(versions) == 0 {
		a.AffectedVersions = datatypes.NewJSONSlice([]string{})
		return
	}
	a.AffectedVersions = datatypes.NewJSONSlice(versions)
}
