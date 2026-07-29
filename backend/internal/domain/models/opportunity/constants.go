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
	DefaultPipelineGroupKey = "default"

	StageTypeActive = "active"
	StageTypeWon    = "won"
	StageTypeLost   = "lost"
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
	ActivityQuote       = "quote"
	ActivityContract    = "contract"
	ActivityPayment     = "payment"
	ActivityMerge       = "merge"
)

const (
	LineItemKindManual    = "manual"
	LineItemKindQuoteFile = "quote_file"
)

const (
	QuoteApprovalDraft     = "draft"
	QuoteApprovalSubmitted = "submitted"
	QuoteApprovalApproved  = "approved"
	QuoteApprovalRejected  = "rejected"
	QuoteApprovalWithdrawn = "withdrawn"
	QuoteApprovalEffective = "effective"
)

const (
	TaskStatusOpen = "open"
	TaskStatusDone = "done"
)

const (
	ContractStatusDraft     = "draft"
	ContractStatusPending   = "pending_signature"
	ContractStatusSigned    = "signed"
	ContractStatusCancelled = "cancelled"
)

const (
	PaymentStatusPlanned = "planned"
	PaymentStatusPaid    = "paid"
	PaymentStatusOverdue = "overdue"
	PaymentStatusVoided  = "voided"
)
