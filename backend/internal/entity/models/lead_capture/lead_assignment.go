package lead_capture

import (
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
)

// LeadAssignment records owner assignments for a lead.
type LeadAssignment struct {
	AssignmentUUID string    `gorm:"column:assignment_uuid;type:uuid;default:gen_random_uuid();primaryKey" json:"assignment_uuid"`
	TenantUUID     string    `gorm:"column:tenant_uuid;type:uuid;not null;index:idx_lead_capture_assignments_tenant" json:"tenant_uuid"`
	LeadUUID       string    `gorm:"column:lead_uuid;type:uuid;not null;index:idx_lead_capture_assignments_lead" json:"lead_uuid"`
	OwnerUserUUID  string    `gorm:"column:owner_user_uuid;type:text;not null;index:idx_lead_capture_assignments_owner" json:"owner_user_uuid"`
	Reason         string    `gorm:"column:reason;type:text" json:"reason"`
	CreatedAt      time.Time `gorm:"column:created_at;type:timestamptz;autoCreateTime" json:"created_at"`
}

func (LeadAssignment) TableName() string {
	return models.S(models.TableLeadCaptureAssignments)
}
