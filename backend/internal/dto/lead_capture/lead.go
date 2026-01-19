package lead_capture

// LeadCreateRequest defines payload for creating a lead.
type LeadCreateRequest struct {
	DisplayName       string `json:"display_name" binding:"omitempty" validate:"omitempty"`
	Phone             string `json:"phone" binding:"omitempty" validate:"omitempty"`
	Email             string `json:"email" binding:"omitempty" validate:"omitempty"`
	SourceChannel     string `json:"source_channel" binding:"omitempty" validate:"omitempty"`
	SourceAppType     string `json:"source_app_type" binding:"omitempty" validate:"omitempty"`
	SourceAccountUUID string `json:"source_account_uuid" binding:"omitempty" validate:"omitempty"`
	OwnerUserUUID     string `json:"owner_user_uuid" binding:"omitempty" validate:"omitempty"`
}

// LeadUpdateRequest defines payload for updating a lead.
type LeadUpdateRequest struct {
	DisplayName   string `json:"display_name" binding:"omitempty" validate:"omitempty"`
	Phone         string `json:"phone" binding:"omitempty" validate:"omitempty"`
	Email         string `json:"email" binding:"omitempty" validate:"omitempty"`
	Status        string `json:"status" binding:"omitempty" validate:"omitempty"`
	OwnerUserUUID string `json:"owner_user_uuid" binding:"omitempty" validate:"omitempty"`
}

// LeadListRequest defines list query for leads.
type LeadListRequest struct {
	Page          int    `form:"page" json:"page" binding:"omitempty" validate:"omitempty,min=1"`
	PageSize      int    `form:"page_size" json:"page_size" binding:"omitempty" validate:"omitempty,min=1,max=200"`
	Query         string `form:"q" json:"q" binding:"omitempty" validate:"omitempty"`
	Status        string `form:"status" json:"status" binding:"omitempty" validate:"omitempty"`
	OwnerUserUUID string `form:"owner_user_uuid" json:"owner_user_uuid" binding:"omitempty" validate:"omitempty"`
	ChannelCode   string `form:"channel_code" json:"channel_code" binding:"omitempty" validate:"omitempty"`
}
