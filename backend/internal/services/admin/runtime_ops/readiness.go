package runtime_ops

import "time"

// ChecklistType identifies a readiness checklist.
type ChecklistType string

const (
	// ChecklistSupportReady ensures support channels are prepared.
	ChecklistSupportReady ChecklistType = "support_ready"
	// ChecklistIncidentReady ensures incident response rituals are established.
	ChecklistIncidentReady ChecklistType = "incident_ready"
	// ChecklistSLAReady ensures SLA aggregation and disclosure are configured.
	ChecklistSLAReady ChecklistType = "sla_ready"
)

// ReadinessItem represents a checklist entry and its blocking status.
type ReadinessItem struct {
	Key         string    `json:"key"`
	Description string    `json:"description"`
	Blocking    bool      `json:"blocking"`
	Completed   bool      `json:"completed"`
	OwnerRole   string    `json:"owner_role"`
	UpdatedAt   time.Time `json:"updated_at"`
}

const (
	ChecklistStatusPending   = "pending"
	ChecklistStatusCompleted = "completed"
)

// ReadinessBlueprint enumerates checklist templates keyed by readiness type.
type ReadinessBlueprint map[ChecklistType][]ReadinessItem

// DefaultReadinessBlueprint returns the scaffold for the three Operations checklists.
func DefaultReadinessBlueprint() ReadinessBlueprint {
	now := time.Now().UTC()
	return ReadinessBlueprint{
		ChecklistSupportReady: {
			{
				Key:         "support_channels_configured",
				Description: "Support channels (Marketplace ticket, vendor email, emergency hotline) configured and verified",
				Blocking:    true,
				Completed:   false,
				OwnerRole:   "agent",
				UpdatedAt:   now,
			},
			{
				Key:         "knowledge_base_published",
				Description: "README/FAQ/Troubleshooting/Support Policy published to documentation hub",
				Blocking:    false,
				Completed:   false,
				OwnerRole:   "operations",
				UpdatedAt:   now,
			},
		},
		ChecklistIncidentReady: {
			{
				Key:         "sev_matrix_defined",
				Description: "SEV-0~SEV-4 matrix and response windows approved",
				Blocking:    true,
				Completed:   false,
				OwnerRole:   "manager",
				UpdatedAt:   now,
			},
			{
				Key:         "communication_channels_tested",
				Description: "Support Hub, Hotline, security@powerx.io, status page notifications tested end-to-end",
				Blocking:    true,
				Completed:   false,
				OwnerRole:   "liaison",
				UpdatedAt:   now,
			},
		},
		ChecklistSLAReady: {
			{
				Key:         "sla_targets_committed",
				Description: "Plan-level SLA/SLO/SLI targets documented and accepted by stakeholders",
				Blocking:    true,
				Completed:   false,
				OwnerRole:   "manager",
				UpdatedAt:   now,
			},
			{
				Key:         "sla_sampling_cron_configured",
				Description: "Daily/Monthly/Quarterly SLA aggregation jobs scheduled",
				Blocking:    true,
				Completed:   false,
				OwnerRole:   "operations",
				UpdatedAt:   now,
			},
		},
	}
}

// ListReadinessTypes returns the canonical readiness checklist types.
func ListReadinessTypes() []ChecklistType {
	return []ChecklistType{
		ChecklistSupportReady,
		ChecklistIncidentReady,
		ChecklistSLAReady,
	}
}
