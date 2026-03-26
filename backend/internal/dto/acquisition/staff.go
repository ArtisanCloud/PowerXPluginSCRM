package acquisition

import "gorm.io/datatypes"

type StaffLiveCodeCreateRequest struct {
	Channel                 string   `json:"channel" binding:"required,max=64"`
	AppType                 string   `json:"app_type" binding:"required,max=64"`
	ChannelAccountUUID      string   `json:"channel_account_uuid" binding:"omitempty,uuid4"`
	ActivityName            string   `json:"activity_name" binding:"required,max=128"`
	CodeKey                 string   `json:"code_key" binding:"omitempty,max=128"`
	MemberUUIDs             []string `json:"member_uuids" binding:"required,min=1,dive,uuid4"`
	CorpTagIDs              []string `json:"corp_tag_ids"`
	NewCustomerRemarkEnable bool     `json:"new_customer_remark_enabled"`
}

type StaffLiveCodeListQuery struct {
	ActivityName string `form:"activity_name" binding:"omitempty,max=128"`
	Status       string `form:"status" binding:"omitempty,oneof=draft active disabled"`
	Limit        int    `form:"limit" binding:"omitempty,min=1,max=200"`
}

type StaffLiveCodeStatusUpdateRequest struct {
	Status string `json:"status" binding:"required,oneof=active disabled"`
}

type StaffWelcomeSaveRequest struct {
	WelcomeMode   string         `json:"welcome_mode" binding:"required,oneof=send silent"`
	ContentBlocks datatypes.JSON `json:"content_blocks" binding:"required"`
}
