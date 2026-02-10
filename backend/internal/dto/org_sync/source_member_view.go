package org_sync

import "time"

// SourceMemberProfileView represents authorized profile data.
type SourceMemberProfileView struct {
	Name             string     `json:"name"`
	Phone            string     `json:"phone,omitempty"`
	Email            string     `json:"email,omitempty"`
	BizMail          string     `json:"biz_mail,omitempty"`
	Position         string     `json:"position,omitempty"`
	MainDepartmentID string     `json:"main_department_id,omitempty"`
	Address          string     `json:"address,omitempty"`
	AvatarURL        string     `json:"avatar_url,omitempty"`
	UpdatedAt        *time.Time `json:"updated_at,omitempty"`
}

// SourceMemberView merges base ID layer with authorized profile layer.
type SourceMemberView struct {
	SourceMemberUUID   string                   `json:"source_member_uuid"`
	TenantUUID         string                   `json:"tenant_uuid"`
	SourceAccountUUID  string                   `json:"source_account_uuid"`
	ChannelAccountUUID string                   `json:"channel_account_uuid"`
	ExternalMemberID   string                   `json:"external_member_id"`
	Name               string                   `json:"name"`
	Phone              string                   `json:"phone,omitempty"`
	Email              string                   `json:"email,omitempty"`
	ProfileStatus      string                   `json:"profile_status"`
	Status             string                   `json:"status"`
	Profile            *SourceMemberProfileView `json:"profile,omitempty"`
	CreatedAt          *time.Time               `json:"created_at,omitempty"`
	UpdatedAt          *time.Time               `json:"updated_at,omitempty"`
}
