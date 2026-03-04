package lead_capture

// TriggerWeComSyncRequest defines payload for manually triggering WeCom lead sync.
type TriggerWeComSyncRequest struct {
	ChannelAccountUUID string `json:"channel_account_uuid" binding:"omitempty,uuid4"`
	TraceID            string `json:"trace_id" binding:"omitempty,max=128"`
}

// ListWeComSyncTasksQuery defines query params for listing sync tasks.
type ListWeComSyncTasksQuery struct {
	ChannelAccountUUID string `form:"channel_account_uuid" binding:"omitempty,uuid4"`
	Status             string `form:"status" binding:"omitempty,oneof=queued running success failed"`
	Limit              int    `form:"limit" binding:"omitempty,min=1,max=200"`
}
