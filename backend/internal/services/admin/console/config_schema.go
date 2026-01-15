package console

import (
	"maps"
	"slices"
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/config"
)

// FieldOption represents selectable values for a field.
type FieldOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// FieldDefinition describes a single configuration input field.
type FieldDefinition struct {
	Name       string         `json:"name"`
	Type       string         `json:"type"`
	Label      string         `json:"label"`
	Required   bool           `json:"required,omitempty"`
	HelpText   string         `json:"help_text,omitempty"`
	Options    []FieldOption  `json:"options,omitempty"`
	Validation map[string]any `json:"validation,omitempty"`
}

// SectionDefinition aggregates related fields under a logical key.
type SectionDefinition struct {
	Key         string            `json:"key"`
	Title       string            `json:"title"`
	Description string            `json:"description,omitempty"`
	Fields      []FieldDefinition `json:"fields"`
}

func (s SectionDefinition) clone() SectionDefinition {
	cp := SectionDefinition{
		Key:         s.Key,
		Title:       s.Title,
		Description: s.Description,
	}
	cp.Fields = make([]FieldDefinition, len(s.Fields))
	copy(cp.Fields, s.Fields)
	return cp
}

// DefaultSections builds the static schema for the Dev Console configuration.
func DefaultSections(cfg *config.Config) []SectionDefinition {
	sections := []SectionDefinition{
		{
			Key:         "admin_console.retention",
			Title:       "Retention Policies",
			Description: "Manage retention windows for audit logs, configuration history, and job runs.",
			Fields: []FieldDefinition{
				{
					Name:       "audit_retention_days",
					Type:       "number",
					Label:      "Audit log retention (days)",
					Required:   true,
					Validation: map[string]any{"min": 1, "max": 3650},
				},
				{
					Name:       "config_change_retention_days",
					Type:       "number",
					Label:      "Configuration history retention (days)",
					Required:   true,
					Validation: map[string]any{"min": 1, "max": 3650},
				},
				{
					Name:       "job_history_days",
					Type:       "number",
					Label:      "Job history retention (days)",
					Required:   true,
					Validation: map[string]any{"min": 1, "max": 365},
				},
			},
		},
		{
			Key:         "admin_console.export",
			Title:       "Audit Export",
			Description: "Control audit export defaults and allowed ranges.",
			Fields: []FieldDefinition{
				{
					Name:     "default_format",
					Type:     "select",
					Label:    "Default export format",
					Required: true,
					Options: []FieldOption{
						{Value: "csv", Label: "CSV"},
						{Value: "json", Label: "JSON"},
					},
				},
				{
					Name:       "max_range_days",
					Type:       "number",
					Label:      "Maximum export range (days)",
					Required:   true,
					Validation: map[string]any{"min": 1, "max": 180},
				},
			},
		},
		{
			Key:         "admin_console.troubleshooting",
			Title:       "Troubleshooting Dashboard",
			Description: "Tune auto-refresh cadence for health, quota, and webhook diagnostics.",
			Fields: []FieldDefinition{
				{
					Name:       "refresh_interval_seconds",
					Type:       "number",
					Label:      "Auto-refresh interval (seconds)",
					Required:   true,
					Validation: map[string]any{"min": 60, "max": 1800},
				},
				{
					Name:       "cache_ttl_seconds",
					Type:       "number",
					Label:      "Cache TTL (seconds)",
					Required:   true,
					Validation: map[string]any{"min": 30, "max": 1800},
				},
			},
		},
		{
			Key:         "admin_console.safe_ops",
			Title:       "Safe Operations",
			Description: "Prevent duplicate safe-ops executions and manage concurrency.",
			Fields: []FieldDefinition{
				{
					Name:       "lock_ttl_seconds",
					Type:       "number",
					Label:      "Execution lock TTL (seconds)",
					Required:   true,
					Validation: map[string]any{"min": 30, "max": 1800},
				},
				{
					Name:       "max_concurrent_ops",
					Type:       "number",
					Label:      "Max concurrent operations",
					Required:   true,
					Validation: map[string]any{"min": 1, "max": 10},
				},
			},
		},
	}
	return sections
}

// DefaultValues produces a map of default values for this section given config.
func (s SectionDefinition) DefaultValues(cfg *config.Config) map[string]any {
	values := map[string]any{}
	adminCfg := cfg
	switch s.Key {
	case "admin_console.retention":
		values["audit_retention_days"] = adminCfg.AdminConsoleAuditRetention()
		values["config_change_retention_days"] = adminCfg.AdminConsoleConfigRetention()
		values["job_history_days"] = adminCfg.AdminConsoleJobHistory()
	case "admin_console.export":
		values["default_format"] = adminCfg.AdminConsoleExportFormat()
		values["max_range_days"] = adminCfg.AdminConsoleExportRange()
	case "admin_console.troubleshooting":
		values["refresh_interval_seconds"] = adminCfg.AdminConsoleRefreshInterval()
		if cfg != nil && cfg.AdminConsole != nil && cfg.AdminConsole.Troubleshooting.CacheTTLSeconds > 0 {
			values["cache_ttl_seconds"] = cfg.AdminConsole.Troubleshooting.CacheTTLSeconds
		} else {
			values["cache_ttl_seconds"] = 120
		}
	case "admin_console.safe_ops":
		values["lock_ttl_seconds"] = adminCfg.AdminConsoleSafeOpsLockTTL()
		if cfg != nil && cfg.AdminConsole != nil && cfg.AdminConsole.SafeOps.MaxConcurrentOps > 0 {
			values["max_concurrent_ops"] = cfg.AdminConsole.SafeOps.MaxConcurrentOps
		} else {
			values["max_concurrent_ops"] = 1
		}
	default:
		// no-op
	}
	return values
}

// Keys returns section keys in stable order.
func Keys(defs []SectionDefinition) []string {
	keys := make([]string, 0, len(defs))
	for _, def := range defs {
		keys = append(keys, def.Key)
	}
	return keys
}

// ToMap returns map keyed by section key.
func ToMap(defs []SectionDefinition) map[string]SectionDefinition {
	out := make(map[string]SectionDefinition, len(defs))
	for _, def := range defs {
		cloned := def.clone()
		if _, exists := out[def.Key]; exists {
			continue
		}
		out[def.Key] = cloned
	}
	return out
}

// SortedSections returns clone sorted by key for deterministic mapping.
func SortedSections(defs []SectionDefinition) []SectionDefinition {
	cp := make([]SectionDefinition, len(defs))
	copy(cp, defs)
	slices.SortFunc(cp, func(a, b SectionDefinition) int {
		return strings.Compare(a.Key, b.Key)
	})
	return cp
}

// MergeValues merges source onto target returning copy.
func MergeValues(target, source map[string]any) map[string]any {
	if len(source) == 0 {
		return maps.Clone(target)
	}
	merged := maps.Clone(target)
	for k, v := range source {
		merged[k] = v
	}
	return merged
}
