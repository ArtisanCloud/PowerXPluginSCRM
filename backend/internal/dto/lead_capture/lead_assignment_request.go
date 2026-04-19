package lead_capture

// LeadAssignRequest defines payload for assigning a lead owner.
type LeadAssignRequest struct {
	OwnerUserUUID string `json:"owner_user_uuid" binding:"required" validate:"required"`
	Reason        string `json:"reason" binding:"omitempty" validate:"omitempty"`
}

// LeadBatchAssignRequest defines payload for assigning owner to multiple leads.
type LeadBatchAssignRequest struct {
	LeadUUIDs     []string `json:"lead_uuids" binding:"required" validate:"required"`
	OwnerUserUUID string   `json:"owner_user_uuid" binding:"required" validate:"required"`
	Reason        string   `json:"reason" binding:"omitempty" validate:"omitempty"`
}
