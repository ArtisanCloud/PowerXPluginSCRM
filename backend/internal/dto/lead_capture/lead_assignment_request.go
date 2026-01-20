package lead_capture

// LeadAssignRequest defines payload for assigning a lead owner.
type LeadAssignRequest struct {
	OwnerUserUUID string `json:"owner_user_uuid" binding:"required" validate:"required"`
	Reason        string `json:"reason" binding:"omitempty" validate:"omitempty"`
}
