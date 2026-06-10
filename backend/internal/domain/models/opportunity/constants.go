package opportunity

const (
	StageOpen        = "open"
	StageQualified   = "qualified"
	StageProposal    = "proposal"
	StageNegotiation = "negotiation"
	StageWon         = "won"
	StageLost        = "lost"
)

const (
	ActivityCreate      = "create"
	ActivityStageChange = "stage_change"
	ActivityClose       = "close"
	ActivityReopen      = "reopen"
	ActivityNote        = "note"
	ActivityRiskFlag    = "risk_flag"
	ActivityLineItem    = "line_item"
	ActivityTask        = "task"
)

const (
	LineItemKindManual    = "manual"
	LineItemKindQuoteFile = "quote_file"
)

const (
	TaskStatusOpen = "open"
	TaskStatusDone = "done"
)
