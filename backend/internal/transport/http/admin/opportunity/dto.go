package opportunity

type createOpportunityRequest struct {
	LeadUUID        string   `json:"lead_uuid" binding:"required"`
	Title           string   `json:"title" binding:"required"`
	OwnerUserUUID   string   `json:"owner_user_uuid"`
	OwnerMemberUUID string   `json:"owner_member_uuid"`
	Amount          *float64 `json:"amount"`
	Currency        string   `json:"currency"`
	Probability     *int     `json:"probability"`
	ExpectedCloseAt string   `json:"expected_close_at"`
}

type updateOpportunityRequest struct {
	Title           *string  `json:"title"`
	OwnerUserUUID   *string  `json:"owner_user_uuid"`
	OwnerMemberUUID *string  `json:"owner_member_uuid"`
	Amount          *float64 `json:"amount"`
	Currency        *string  `json:"currency"`
	Probability     *int     `json:"probability"`
	ExpectedCloseAt *string  `json:"expected_close_at"`
}

type lineItemRequest struct {
	Name        string  `json:"name" binding:"required"`
	Quantity    float64 `json:"quantity"`
	UnitPrice   float64 `json:"unit_price"`
	TotalAmount float64 `json:"total_amount"`
	Currency    string  `json:"currency"`
}

type quoteApprovalRequest struct {
	Action  string `json:"action" binding:"required"`
	Comment string `json:"comment"`
}

type contractRequest struct {
	Title         string   `json:"title" binding:"required"`
	ContractNo    string   `json:"contract_no"`
	CustomerUUID  string   `json:"customer_uuid"`
	QuoteItemUUID string   `json:"quote_item_uuid"`
	Amount        *float64 `json:"amount"`
	Currency      string   `json:"currency"`
	Status        string   `json:"status"`
	SignedAt      string   `json:"signed_at"`
}

type contractStatusRequest struct {
	Status   string `json:"status" binding:"required"`
	SignedAt string `json:"signed_at"`
}

type paymentRequest struct {
	ContractUUID  string  `json:"contract_uuid" binding:"required"`
	Title         string  `json:"title" binding:"required"`
	PlannedAmount float64 `json:"planned_amount"`
	DueAt         string  `json:"due_at"`
}

type paymentStatusRequest struct {
	Status        string   `json:"status" binding:"required"`
	PaidAmount    *float64 `json:"paid_amount"`
	PaidAt        string   `json:"paid_at"`
	Method        string   `json:"method"`
	TransactionNo string   `json:"transaction_no"`
	Note          string   `json:"note"`
}

type stageConfigRequest struct {
	PipelineGroupUUID string `json:"pipeline_group_uuid"`
	StageKey          string `json:"stage_key" binding:"required"`
	Label             string `json:"label" binding:"required"`
	SortOrder         int    `json:"sort_order"`
	DefaultWinRate    int    `json:"default_win_rate"`
	SLADays           int    `json:"sla_days"`
	StageType         string `json:"stage_type"`
	FixedStage        string `json:"fixed_stage" binding:"required"`
	IsActive          *bool  `json:"is_active"`
	MigrationPolicy   string `json:"migration_policy"`
}

type pipelineGroupRequest struct {
	GroupKey      string `json:"group_key"`
	Name          string `json:"name" binding:"required"`
	Description   string `json:"description"`
	IsDefault     bool   `json:"is_default"`
	CopyFromGroup string `json:"copy_from_group"`
	TemplateKey   string `json:"template_key"`
}

type mergeOpportunityRequest struct {
	SourceOpportunityUUID string `json:"source_opportunity_uuid" binding:"required"`
	Reason                string `json:"reason"`
}

type taskRequest struct {
	Title string `json:"title" binding:"required"`
	DueAt string `json:"due_at"`
}

type taskStatusRequest struct {
	Status string `json:"status" binding:"required"`
}

type listOpportunityQuery struct {
	Stage             string `form:"stage"`
	OwnerUserUUID     string `form:"owner_user_uuid"`
	OwnerMemberUUID   string `form:"owner_member_uuid"`
	LeadUUID          string `form:"lead_uuid"`
	Keyword           string `form:"keyword"`
	SourceChannel     string `form:"source_channel"`
	RiskOnly          bool   `form:"risk_only"`
	ExpectedCloseFrom string `form:"expected_close_from"`
	ExpectedCloseTo   string `form:"expected_close_to"`
	Limit             int    `form:"limit"`
}

type stageOpportunityRequest struct {
	Stage string `json:"stage" binding:"required"`
}

type closeOpportunityRequest struct {
	Result     string `json:"result" binding:"required"`
	LostReason string `json:"lost_reason"`
}

type riskOpportunityRequest struct {
	Flag    string         `json:"flag"`
	Payload map[string]any `json:"payload"`
}
