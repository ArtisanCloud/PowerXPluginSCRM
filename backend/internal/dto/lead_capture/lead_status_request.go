package lead_capture

// LeadStatusUpdateRequest defines payload for updating lead status.
type LeadStatusUpdateRequest struct {
	Status string `json:"status" binding:"required" validate:"required"`
}
