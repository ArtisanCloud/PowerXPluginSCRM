package social_channel_governance

// WeComOpenWorkPlatformConfigRequest updates platform-level OpenWork settings.
type WeComOpenWorkPlatformConfigRequest struct {
	Enabled                 bool                           `json:"enabled"`
	TemplateID              string                         `json:"template_id"`
	TemplateSecret          string                         `json:"template_secret"`
	TemplateTicket          string                         `json:"template_ticket"`
	TemplateTicketUpdatedAt string                         `json:"template_ticket_updated_at"`
	TemplateTicketSource    string                         `json:"template_ticket_source"`
	ProviderCorpID          string                         `json:"provider_corpid"`
	ProviderSecret          string                         `json:"provider_secret"`
	Token                   string                         `json:"token"`
	AESKey                  string                         `json:"aes_key"`
	HTTPDebug               bool                           `json:"http_debug"`
	CallbackHost            string                         `json:"callback_host"`
	RedirectURI             string                         `json:"redirect_uri"`
	DefaultTemplateID       string                         `json:"default_template_id"`
	Templates               []WeComOpenWorkTemplateRequest `json:"templates"`
}

type WeComOpenWorkTemplateRequest struct {
	TemplateID              string `json:"template_id"`
	TemplateSecret          string `json:"template_secret"`
	TemplateTicket          string `json:"template_ticket"`
	TemplateTicketUpdatedAt string `json:"template_ticket_updated_at"`
	TemplateTicketSource    string `json:"template_ticket_source"`
	ProviderCorpID          string `json:"provider_corpid"`
	ProviderSecret          string `json:"provider_secret"`
}
