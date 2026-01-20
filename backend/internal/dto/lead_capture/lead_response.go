package lead_capture

// LeadResponse defines output payload for lead queries.
type LeadResponse struct {
	LeadUUID          string `json:"lead_uuid"`
	TenantUUID        string `json:"tenant_uuid"`
	DisplayName       string `json:"display_name"`
	Phone             string `json:"phone"`
	Email             string `json:"email"`
	Status            string `json:"status"`
	OwnerUserUUID     string `json:"owner_user_uuid"`
	SourceChannel     string `json:"source_channel"`
	SourceAppType     string `json:"source_app_type"`
	SourceAccountUUID string `json:"source_account_uuid"`
	CreatedAt         string `json:"created_at"`
	UpdatedAt         string `json:"updated_at"`
}

// LeadListResponse defines list payload for leads.
type LeadListResponse struct {
	Items []*LeadResponse `json:"items"`
}
