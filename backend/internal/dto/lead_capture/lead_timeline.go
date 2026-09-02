package lead_capture

type LeadTimelineQuery struct {
	StageKey  string `form:"stage_key"`
	EventType string `form:"event_type"`
	Page      int    `form:"page" binding:"omitempty,min=1"`
	PageSize  int    `form:"page_size" binding:"omitempty,min=1,max=100"`
}
