package lead_capture

// WeComBotCommandRequest defines webhook payload for bot command lead creation.
type WeComBotCommandRequest struct {
	ChannelAccountUUID string         `json:"channel_account_uuid" binding:"required,uuid4"`
	ExternalEventID    string         `json:"external_event_id" binding:"required"`
	ConversationID     string         `json:"conversation_id" binding:"required"`
	OperatorID         string         `json:"operator_id" binding:"required"`
	CommandText        string         `json:"command_text" binding:"required"`
	Permissions        []string       `json:"permissions" binding:"required"`
	OccurredAt         string         `json:"occurred_at" binding:"required"`
	RawPayload         map[string]any `json:"raw_payload"`
}
