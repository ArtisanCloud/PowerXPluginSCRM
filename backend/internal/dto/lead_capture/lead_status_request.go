package lead_capture

// LeadStatusUpdateRequest defines payload for updating lead status.
type LeadStatusUpdateRequest struct {
	Status string `json:"status" binding:"required" validate:"required"`
}

// LeadQualificationRequest defines payload for MQL/SQL qualification actions.
type LeadQualificationRequest struct {
	TargetStatus string `json:"target_status" binding:"omitempty" validate:"omitempty"`
	Status       string `json:"status" binding:"omitempty" validate:"omitempty"`
}
