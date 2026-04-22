package acquisition

import "gorm.io/datatypes"

type GroupLiveCodeCreateRequest struct {
	Channel            string `json:"channel" binding:"required,max=64"`
	AppType            string `json:"app_type" binding:"required,max=64"`
	ChannelAccountUUID string `json:"channel_account_uuid" binding:"required,uuid4"`
	ActivityName       string `json:"activity_name" binding:"required,max=128"`
	JoinScene          int    `json:"join_scene" binding:"omitempty,min=1,max=3"`
	SkipVerify         bool   `json:"skip_verify"`
	AutoCreateRoom     bool   `json:"auto_create_room"`
}

type GroupLiveCodeUpdateRequest struct {
	ActivityName   *string `json:"activity_name" binding:"omitempty,max=128"`
	SkipVerify     *bool   `json:"skip_verify"`
	AutoCreateRoom *bool   `json:"auto_create_room"`
	Status         *string `json:"status" binding:"omitempty,oneof=draft active disabled"`
}

type GroupLiveCodeListQuery struct {
	Limit int `form:"limit" binding:"omitempty,min=1,max=200"`
}

type GroupLiveCodeSyncRequest struct {
	ChatIDs []string `json:"chat_ids" binding:"omitempty,dive,min=1"`
}

type GroupChatSyncRequest struct {
	Mode string `json:"mode" binding:"omitempty,oneof=full incremental"`
}

type GroupChatListQuery struct {
	Limit int `form:"limit" binding:"omitempty,min=1,max=500"`
}

type GroupTagCreateRequest struct {
	TagName     string         `json:"tag_name" binding:"required,max=128"`
	Color       string         `json:"color" binding:"omitempty,max=32"`
	RuleMode    string         `json:"rule_mode" binding:"omitempty,oneof=manual rule_based"`
	RulePayload datatypes.JSON `json:"rule_payload"`
}

type GroupTagListQuery struct {
	Limit int `form:"limit" binding:"omitempty,min=1,max=200"`
}

type GroupTagBindRequest struct {
	ChatIDs    []string `json:"chat_ids" binding:"required,min=1,dive,min=1"`
	BindSource string   `json:"bind_source" binding:"omitempty,oneof=manual rule_engine"`
}

type GroupTagBindingListQuery struct {
	Limit int `form:"limit" binding:"omitempty,min=1,max=1000"`
}

type GroupTagReplayRequest struct {
	TriggerSource string `json:"trigger_source" binding:"omitempty,max=32"`
}
