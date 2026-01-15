package security

import (
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	"gorm.io/datatypes"
)

// AuditReport stores metadata for each security audit execution.
type AuditReport struct {
	ID               string            `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	BaselineID       string            `gorm:"column:baseline_id;type:uuid;not null" json:"baseline_id"`
	InitiatedBy      string            `gorm:"column:initiated_by;type:text;not null" json:"initiated_by"`
	Status           string            `gorm:"column:status;type:text;not null" json:"status"`
	Findings         datatypes.JSONMap `gorm:"column:findings;type:jsonb" json:"findings,omitempty"`
	ArtifactPath     string            `gorm:"column:artifact_path;type:text" json:"artifact_path,omitempty"`
	SarifPath        string            `gorm:"column:sarif_path;type:text" json:"sarif_path,omitempty"`
	ReportHash       string            `gorm:"column:report_hash;type:text" json:"report_hash,omitempty"`
	ChecklistVersion string            `gorm:"column:checklist_version;type:text;not null" json:"checklist_version"`
	CreatedAt        time.Time         `gorm:"column:created_at;type:timestamptz;autoCreateTime" json:"created_at"`
}

// TableName returns the fully qualified table name.
func (*AuditReport) TableName() string {
	return models.S(models.TableSecurityAuditReports)
}
