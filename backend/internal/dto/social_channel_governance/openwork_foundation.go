package social_channel_governance

// OpenWorkEventIngestRequest defines payload for OpenWork callback/event ingestion.
type OpenWorkEventIngestRequest struct {
	SuiteID     string         `json:"suite_id" binding:"required"`
	EventType   string         `json:"event_type" binding:"required"`
	SuiteTicket string         `json:"suite_ticket"`
	CorpID      string         `json:"corp_id"`
	AgentID     string         `json:"agent_id"`
	EventTime   int64          `json:"event_time"`
	Payload     map[string]any `json:"payload"`
}

// OpenWorkAuthorizeStartRequest starts delegated authorization.
type OpenWorkAuthorizeStartRequest struct {
	SuiteID     string `json:"suite_id" binding:"required"`
	SuiteSecret string `json:"suite_secret" binding:"required"`
	SuiteTicket string `json:"suite_ticket"`
	RedirectURI string `json:"redirect_uri"`
	State       string `json:"state"`
}

// OpenWorkAuthorizeCompleteRequest finishes delegated authorization.
type OpenWorkAuthorizeCompleteRequest struct {
	SuiteID            string `json:"suite_id" binding:"required"`
	SuiteSecret        string `json:"suite_secret" binding:"required"`
	SuiteTicket        string `json:"suite_ticket"`
	AuthCode           string `json:"auth_code" binding:"required"`
	ChannelAccountUUID string `json:"channel_account_uuid"`
	SetDefault         bool   `json:"set_default"`
}

// OpenWorkSetDefaultRequest switches default corp authorization binding.
type OpenWorkSetDefaultRequest struct {
	ChannelAccountUUID string `json:"channel_account_uuid"`
}

// SyncBaselineJobCreateRequest triggers baseline sync jobs for tags/org/external contacts.
type SyncBaselineJobCreateRequest struct {
	BindingUUID     string         `json:"binding_uuid"`
	Domain          string         `json:"domain" binding:"required"`
	Mode            string         `json:"mode" binding:"required"`
	IdempotencyKey  string         `json:"idempotency_key"`
	MaxRetries      int            `json:"max_retries"`
	WriteBackFields map[string]any `json:"write_back_fields"`
	Context         map[string]any `json:"context"`
}

// SyncConflictReplayRequest replays an open conflict.
type SyncConflictReplayRequest struct {
	Note string `json:"note"`
}
