package social_channel_governance

// ChannelAccountCreateRequest defines payload for channel account onboarding.
type ChannelAccountCreateRequest struct {
	Channel       string            `json:"channel" binding:"required" validate:"required"`
	AppType       string            `json:"app_type" binding:"required" validate:"required"`
	AccountID     string            `json:"account_id" validate:"omitempty"`
	DisplayName   string            `json:"display_name" binding:"required" validate:"required"`
	OwnerMemberUUID string          `json:"owner_member_uuid" binding:"required" validate:"required"`
	Credentials   map[string]string `json:"credentials" validate:"omitempty"`
}

// ChannelAccountUpdateRequest defines payload for updating channel account metadata.
type ChannelAccountUpdateRequest struct {
	AccountID     string            `json:"account_id" validate:"omitempty"`
	DisplayName   string            `json:"display_name" binding:"required" validate:"required"`
	OwnerMemberUUID string          `json:"owner_member_uuid" binding:"required" validate:"required"`
	Status        string            `json:"status" binding:"required" validate:"required"`
	Credentials   map[string]string `json:"credentials" validate:"omitempty"`
}

// ChannelAccountListRequest defines list params for accounts.
type ChannelAccountListRequest struct {
	Page     int `form:"page" json:"page" validate:"omitempty,min=1"`
	PageSize int `form:"page_size" json:"page_size" validate:"omitempty,min=1,max=200"`
}
