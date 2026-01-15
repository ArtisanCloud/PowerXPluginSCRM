package config

import "strings"

// AdminConsoleConfig defines knobs for the Dev Console experience.
type AdminConsoleConfig struct {
	AuditRetentionDays     int                         `yaml:"audit_retention_days" json:"audit_retention_days"`
	ConfigChangeRetention  int                         `yaml:"config_change_retention_days" json:"config_change_retention_days"`
	JobHistoryDays         int                         `yaml:"job_history_days" json:"job_history_days"`
	Export                 AdminConsoleExportConfig    `yaml:"export" json:"export"`
	Troubleshooting        AdminConsoleTroubleshooting `yaml:"troubleshooting" json:"troubleshooting"`
	SafeOps                AdminConsoleSafeOps         `yaml:"safe_ops" json:"safe_ops"`
	NavigationHiddenRoutes []string                    `yaml:"navigation_hidden_routes" json:"navigation_hidden_routes"`
}

// AdminConsoleExportConfig tunes audit export behaviour.
type AdminConsoleExportConfig struct {
	MaxRangeDays  int    `yaml:"max_range_days" json:"max_range_days"`
	DefaultFormat string `yaml:"default_format" json:"default_format"`
}

// AdminConsoleTroubleshooting controls dashboard refresh cadence.
type AdminConsoleTroubleshooting struct {
	RefreshIntervalSeconds int `yaml:"refresh_interval_seconds" json:"refresh_interval_seconds"`
	CacheTTLSeconds        int `yaml:"cache_ttl_seconds" json:"cache_ttl_seconds"`
}

// AdminConsoleSafeOps defines execution locks and dry-run limits.
type AdminConsoleSafeOps struct {
	LockTTLSeconds   int `yaml:"lock_ttl_seconds" json:"lock_ttl_seconds"`
	MaxConcurrentOps int `yaml:"max_concurrent_ops" json:"max_concurrent_ops"`
}

// AdminConsoleAuditRetention returns retention days for audits (default 365).
func (cfg *Config) AdminConsoleAuditRetention() int {
	if cfg == nil || cfg.AdminConsole == nil || cfg.AdminConsole.AuditRetentionDays <= 0 {
		return 365
	}
	return cfg.AdminConsole.AuditRetentionDays
}

// AdminConsoleConfigRetention returns retention for configuration change snapshots (default 365).
func (cfg *Config) AdminConsoleConfigRetention() int {
	if cfg == nil || cfg.AdminConsole == nil || cfg.AdminConsole.ConfigChangeRetention <= 0 {
		return 365
	}
	return cfg.AdminConsole.ConfigChangeRetention
}

// AdminConsoleJobHistory returns number of days to keep job histories (default 45).
func (cfg *Config) AdminConsoleJobHistory() int {
	if cfg == nil || cfg.AdminConsole == nil || cfg.AdminConsole.JobHistoryDays <= 0 {
		return 45
	}
	return cfg.AdminConsole.JobHistoryDays
}

// AdminConsoleExportFormat returns the default export format (default csv).
func (cfg *Config) AdminConsoleExportFormat() string {
	if cfg == nil || cfg.AdminConsole == nil {
		return "csv"
	}
	format := strings.ToLower(strings.TrimSpace(cfg.AdminConsole.Export.DefaultFormat))
	if format == "" {
		return "csv"
	}
	if format != "csv" && format != "json" {
		return "csv"
	}
	return format
}

// AdminConsoleExportRange returns maximum export window in days (default 31).
func (cfg *Config) AdminConsoleExportRange() int {
	if cfg == nil || cfg.AdminConsole == nil || cfg.AdminConsole.Export.MaxRangeDays <= 0 {
		return 31
	}
	return cfg.AdminConsole.Export.MaxRangeDays
}

// AdminConsoleRefreshInterval returns troubleshooting refresh cadence (default 300 seconds).
func (cfg *Config) AdminConsoleRefreshInterval() int {
	if cfg == nil || cfg.AdminConsole == nil || cfg.AdminConsole.Troubleshooting.RefreshIntervalSeconds <= 0 {
		return 300
	}
	return cfg.AdminConsole.Troubleshooting.RefreshIntervalSeconds
}

// AdminConsoleSafeOpsLockTTL returns safe-op lock ttl (default 120 seconds).
func (cfg *Config) AdminConsoleSafeOpsLockTTL() int {
	if cfg == nil || cfg.AdminConsole == nil || cfg.AdminConsole.SafeOps.LockTTLSeconds <= 0 {
		return 120
	}
	return cfg.AdminConsole.SafeOps.LockTTLSeconds
}
