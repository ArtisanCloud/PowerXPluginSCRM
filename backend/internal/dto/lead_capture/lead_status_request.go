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

// LeadActivityCreateRequest defines payload for recording a lead lifecycle activity.
type LeadActivityCreateRequest struct {
	Method         string `json:"method" binding:"required" validate:"required"`
	Subject        string `json:"subject" binding:"omitempty" validate:"omitempty"`
	Content        string `json:"content" binding:"required" validate:"required"`
	Result         string `json:"result" binding:"omitempty" validate:"omitempty"`
	NextStep       string `json:"next_step" binding:"omitempty" validate:"omitempty"`
	NextFollowUpAt string `json:"next_follow_up_at" binding:"omitempty" validate:"omitempty"`
	StageKey       string `json:"stage_key" binding:"omitempty" validate:"omitempty"`
	ActionKey      string `json:"action_key" binding:"omitempty" validate:"omitempty"`
}
