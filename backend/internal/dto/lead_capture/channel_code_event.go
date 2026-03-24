package lead_capture

// ChannelCodeEventWebhookRequest defines external webhook payload for channel-code events.
type ChannelCodeEventWebhookRequest struct {
	ChannelAccountUUID string         `json:"channel_account_uuid" binding:"required,uuid4"`
	CodeKey            string         `json:"code_key" binding:"required,max=128"`
	ExternalEventID    string         `json:"external_event_id" binding:"required,max=128"`
	EventType          string         `json:"event_type" binding:"required,oneof=scan join message other"`
	OccurredAt         string         `json:"occurred_at" binding:"required,datetime=2006-01-02T15:04:05Z07:00"`
	Payload            map[string]any `json:"payload"`
}

// ChannelCodeEventListQuery defines query params for event list endpoint.
type ChannelCodeEventListQuery struct {
	Limit int `form:"limit" binding:"omitempty,min=1,max=200"`
}

type ChannelCodeEventStats struct {
	TouchTotal  int64 `json:"touch_total"`
	IntakeTotal int64 `json:"intake_total"`
	DedupTotal  int64 `json:"dedup_total"`
}
