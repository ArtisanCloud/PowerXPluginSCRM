package opportunity

type createOpportunityRequest struct {
	LeadUUID        string   `json:"lead_uuid" binding:"required"`
	Title           string   `json:"title" binding:"required"`
	OwnerUserUUID   string   `json:"owner_user_uuid" binding:"required"`
	Amount          *float64 `json:"amount"`
	Currency        string   `json:"currency"`
	Probability     *int     `json:"probability"`
	ExpectedCloseAt string   `json:"expected_close_at"`
}

type updateOpportunityRequest struct {
	Title           *string  `json:"title"`
	OwnerUserUUID   *string  `json:"owner_user_uuid"`
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
