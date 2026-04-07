package lead_capture

import "gorm.io/datatypes"

// ChannelCodeCreateRequest defines payload for creating a channel code.
type ChannelCodeCreateRequest struct {
	Channel            string `json:"channel" binding:"required,max=64"`
	AppType            string `json:"app_type" binding:"required,max=64"`
	ChannelAccountUUID string `json:"channel_account_uuid" binding:"required,uuid4"`
	CodeKey            string `json:"code_key" binding:"required,max=128"`
	DisplayName        string `json:"display_name" binding:"required,max=128"`
	TargetType         string `json:"target_type" binding:"required,oneof=group dm entry"`
	TargetID           string `json:"target_id" binding:"required,max=128"`
}

// ChannelCodeListQuery defines query conditions for listing channel codes.
type ChannelCodeListQuery struct {
	Channel            string `form:"channel" binding:"omitempty,max=64"`
	AppType            string `form:"app_type" binding:"omitempty,max=64"`
	ChannelAccountUUID string `form:"channel_account_uuid" binding:"omitempty,uuid4"`
	Status             string `form:"status" binding:"omitempty,oneof=draft active disabled"`
	Limit              int    `form:"limit" binding:"omitempty,min=1,max=200"`
}

// ChannelCodeStatusUpdateRequest defines payload for status updates.
type ChannelCodeStatusUpdateRequest struct {
	Status string `json:"status" binding:"required,oneof=active disabled"`
}

// WelcomeConfigSaveRequest defines payload for save-only welcome config.
type WelcomeConfigSaveRequest struct {
	WelcomeEnabled bool           `json:"welcome_enabled"`
	MessageContent datatypes.JSON `json:"message_content" binding:"required"`
}

// WelcomeConfigHistoryQuery defines query parameters for change log listing.
type WelcomeConfigHistoryQuery struct {
	Limit int `form:"limit" binding:"omitempty,min=1,max=200"`
}
