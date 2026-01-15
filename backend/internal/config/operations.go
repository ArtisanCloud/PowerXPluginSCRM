package config

import "strings"

// OperationsConfig aggregates support, incident, and SLA configuration knobs.
type OperationsConfig struct {
	Support  OperationsSupportConfig  `yaml:"support" json:"support"`
	Incident OperationsIncidentConfig `yaml:"incident" json:"incident"`
	SLA      OperationsSLAConfig      `yaml:"sla" json:"sla"`
}

// OperationsSupportConfig defines defaults for support channels and webhook signing.
type OperationsSupportConfig struct {
	WebhookSecret   string                     `yaml:"webhook_secret" json:"webhook_secret"`
	DefaultChannels []OperationsSupportChannel `yaml:"default_channels" json:"default_channels"`
}

// OperationsSupportChannel enumerates a support channel template.
type OperationsSupportChannel struct {
	Channel   string   `yaml:"channel" json:"channel"`
	Address   string   `yaml:"address" json:"address"`
	Escalates []string `yaml:"escalates" json:"escalates"`
}

// OperationsIncidentConfig captures default communication preferences for incidents.
type OperationsIncidentConfig struct {
	StatusPageURL      string   `yaml:"status_page_url" json:"status_page_url"`
	SecurityContacts   []string `yaml:"security_contacts" json:"security_contacts"`
	CommunicationModes []string `yaml:"communication_modes" json:"communication_modes"`
}

// OperationsSLAConfig defines aggregation cadences for SLA metrics.
type OperationsSLAConfig struct {
	Timezone      string `yaml:"timezone" json:"timezone"`
	DailyCron     string `yaml:"daily_cron" json:"daily_cron"`
	MonthlyCron   string `yaml:"monthly_cron" json:"monthly_cron"`
	QuarterlyCron string `yaml:"quarterly_cron" json:"quarterly_cron"`
}

// OperationsWebhookSecret returns the configured webhook signing secret.
func (cfg *Config) OperationsWebhookSecret() string {
	if cfg == nil || cfg.Operations == nil {
		return ""
	}
	return strings.TrimSpace(cfg.Operations.Support.WebhookSecret)
}

// OperationsIncidentChannels returns normalized incident communication channels.
func (cfg *Config) OperationsIncidentChannels() []string {
	defaults := []string{"support_hub", "status_page", "security_email"}
	if cfg == nil || cfg.Operations == nil {
		return defaults
	}
	raw := cfg.Operations.Incident.CommunicationModes
	if len(raw) == 0 {
		return defaults
	}
	out := make([]string, 0, len(raw))
	seen := map[string]struct{}{}
	for _, v := range raw {
		normalized := strings.ToLower(strings.TrimSpace(v))
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}
	if len(out) == 0 {
		return defaults
	}
	return out
}

// OperationsSLACron returns cron expressions for SLA aggregation windows.
func (cfg *Config) OperationsSLACron() (daily, monthly, quarterly string) {
	if cfg == nil || cfg.Operations == nil {
		return "0 5 * * *", "0 6 1 * *", "0 7 1 1,4,7,10 *"
	}
	sla := cfg.Operations.SLA
	daily = strings.TrimSpace(sla.DailyCron)
	if daily == "" {
		daily = "0 5 * * *"
	}
	monthly = strings.TrimSpace(sla.MonthlyCron)
	if monthly == "" {
		monthly = "0 6 1 * *"
	}
	quarterly = strings.TrimSpace(sla.QuarterlyCron)
	if quarterly == "" {
		quarterly = "0 7 1 1,4,7,10 *"
	}
	return
}
