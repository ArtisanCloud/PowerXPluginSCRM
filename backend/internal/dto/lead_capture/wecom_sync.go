package lead_capture

// TriggerWeComSyncRequest defines payload for manually triggering WeCom lead sync.
type TriggerWeComSyncRequest struct {
	ChannelAccountUUID string `json:"channel_account_uuid" binding:"omitempty,uuid4"`
	TraceID            string `json:"trace_id" binding:"omitempty,max=128"`
	Action             string `json:"action" binding:"omitempty,oneof=pull_external_contacts push_leads"`
	Domain             string `json:"domain" binding:"omitempty,oneof=external_contacts leads"`
	Direction          string `json:"direction" binding:"omitempty,oneof=pull push"`
	Mode               string `json:"mode" binding:"omitempty,oneof=incremental pushback bootstrap"`
	CheckpointCursor   string `json:"checkpoint_cursor" binding:"omitempty,max=255"`
	LeadWriteback      []struct {
		LeadUUID        string         `json:"lead_uuid" binding:"omitempty,uuid4"`
		ExternalUserID  string         `json:"external_userid" binding:"omitempty,max=128"`
		CorpID          string         `json:"corp_id" binding:"omitempty,max=128"`
		Phone           string         `json:"phone" binding:"omitempty,max=64"`
		OrderVersion    int64          `json:"order_version" binding:"omitempty,min=0"`
		IdempotencyHint string         `json:"idempotency_hint" binding:"omitempty,max=128"`
		Fields          map[string]any `json:"fields"`
	} `json:"lead_writeback"`
}

// ListWeComSyncTasksQuery defines query params for listing sync tasks.
type ListWeComSyncTasksQuery struct {
	ChannelAccountUUID string `form:"channel_account_uuid" binding:"omitempty,uuid4"`
	Status             string `form:"status" binding:"omitempty,oneof=queued running success failed"`
	Limit              int    `form:"limit" binding:"omitempty,min=1,max=200"`
}

type WeComWritebackPolicyResponse struct {
	Domain           string         `json:"domain"`
	CapabilityStatus string         `json:"capability_status"`
	Enabled          bool           `json:"enabled"`
	OverwriteMode    string         `json:"overwrite_mode"`
	MappingRules     map[string]any `json:"mapping_rules"`
	ProtectedFields  map[string]any `json:"protected_fields"`
}

type UpdateWeComWritebackPolicyRequest struct {
	Enabled         bool           `json:"enabled"`
	OverwriteMode   string         `json:"overwrite_mode" binding:"omitempty,oneof=safe force"`
	MappingRules    map[string]any `json:"mapping_rules"`
	ProtectedFields map[string]any `json:"protected_fields"`
}
