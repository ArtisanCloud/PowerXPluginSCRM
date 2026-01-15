package config

import (
	"strings"
	"time"
)

// MarketplaceConfig 聚合 Marketplace 模块相关配置。
type MarketplaceConfig struct {
	Checklist      MarketplaceChecklistConfig `yaml:"checklist" json:"checklist"`
	Documentation  MarketplaceDocsConfig      `yaml:"documentation" json:"documentation"`
	Recommendation MarketplaceRecommendation  `yaml:"recommendation" json:"recommendation"`
	License        MarketplaceLicenseConfig   `yaml:"license" json:"license"`
}

// MarketplaceChecklistConfig 描述 Ready Checklist GraphQL 入口与文档。
type MarketplaceChecklistConfig struct {
	GraphQLPath string `yaml:"graphql_path" json:"graphql_path"`
	DocsURL     string `yaml:"docs_url" json:"docs_url"`
}

// MarketplaceDocsConfig 引用 Marketplace 相关说明文档。
type MarketplaceDocsConfig struct {
	Overview        string `yaml:"overview" json:"overview"`
	ReadyChecklist  string `yaml:"ready_checklist" json:"ready_checklist"`
	ListingPlaybook string `yaml:"listing_playbook" json:"listing_playbook"`
}

// MarketplaceRecommendation 描述推荐服务的默认配置。
type MarketplaceRecommendation struct {
	Enabled          bool    `yaml:"enabled" json:"enabled"`
	DefaultWeight    float64 `yaml:"default_weight" json:"default_weight"`
	ExperimentTopic  string  `yaml:"experiment_topic" json:"experiment_topic"`
	FrequencyMinutes int     `yaml:"frequency_minutes" json:"frequency_minutes"`
}

// MarketplaceLicenseConfig 描述 License 离线策略与缓存配置。
type MarketplaceLicenseConfig struct {
	OfflineAllowanceHours int                              `yaml:"offline_allowance_hours" json:"offline_allowance_hours"`
	Cache                 MarketplaceLicenseCacheConfig    `yaml:"cache" json:"cache"`
	Reminder              MarketplaceLicenseReminderConfig `yaml:"reminder" json:"reminder"`
}

// MarketplaceLicenseCacheConfig 定义 License 缓存后端。
type MarketplaceLicenseCacheConfig struct {
	Provider  string `yaml:"provider" json:"provider"`
	RedisURL  string `yaml:"redis_url" json:"redis_url"`
	KeyPrefix string `yaml:"key_prefix" json:"key_prefix"`
}

// MarketplaceLicenseReminderConfig 控制续费提醒策略。
type MarketplaceLicenseReminderConfig struct {
	LeadHours int      `yaml:"lead_hours" json:"lead_hours"`
	Channels  []string `yaml:"channels" json:"channels"`
}

// MarketplaceChecklistGraphQLPath 返回 checklist GraphQL 路径。
func (cfg *Config) MarketplaceChecklistGraphQLPath() string {
	const defaultPath = "/api/v1/admin/marketplace/checklist/graphql"
	if cfg == nil || cfg.Marketplace == nil {
		return defaultPath
	}
	path := strings.TrimSpace(cfg.Marketplace.Checklist.GraphQLPath)
	if path == "" {
		return defaultPath
	}
	return path
}

// MarketplaceChecklistDocs 提供 checklist 文档地址。
func (cfg *Config) MarketplaceChecklistDocs() string {
	if cfg == nil || cfg.Marketplace == nil {
		return ""
	}
	return strings.TrimSpace(cfg.Marketplace.Checklist.DocsURL)
}

// RecommendationFrequency returns refresh frequency for recommendation sync job.
func (cfg *Config) RecommendationFrequency() time.Duration {
	if cfg == nil || cfg.Marketplace == nil {
		return time.Hour
	}
	minutes := cfg.Marketplace.Recommendation.FrequencyMinutes
	if minutes <= 0 {
		return time.Hour
	}
	return time.Duration(minutes) * time.Minute
}

// LicenseCacheConfig 返回 License 缓存配置。
func (cfg *Config) LicenseCacheConfig() MarketplaceLicenseCacheConfig {
	const defaultPrefix = "powerx:marketplace:licenses"
	defaultCfg := MarketplaceLicenseCacheConfig{
		Provider:  "redis",
		RedisURL:  "redis://localhost:6379",
		KeyPrefix: defaultPrefix,
	}
	if cfg == nil || cfg.Marketplace == nil {
		return defaultCfg
	}
	out := cfg.Marketplace.License.Cache
	if strings.TrimSpace(out.Provider) == "" {
		out.Provider = defaultCfg.Provider
	}
	if strings.TrimSpace(out.RedisURL) == "" {
		out.RedisURL = defaultCfg.RedisURL
	}
	if strings.TrimSpace(out.KeyPrefix) == "" {
		out.KeyPrefix = defaultCfg.KeyPrefix
	}
	return out
}

// LicenseOfflineAllowance 返回离线许可最大时长。
func (cfg *Config) LicenseOfflineAllowance() time.Duration {
	const defaultHours = 72
	if cfg == nil || cfg.Marketplace == nil {
		return defaultHours * time.Hour
	}
	hours := cfg.Marketplace.License.OfflineAllowanceHours
	if hours <= 0 {
		hours = defaultHours
	}
	return time.Duration(hours) * time.Hour
}

// LicenseReminderLead 返回续费提醒提前时间。
func (cfg *Config) LicenseReminderLead() time.Duration {
	const defaultLead = 72 * time.Hour
	if cfg == nil || cfg.Marketplace == nil {
		return defaultLead
	}
	hours := cfg.Marketplace.License.Reminder.LeadHours
	if hours < 0 {
		return defaultLead
	}
	if hours == 0 {
		return 0
	}
	return time.Duration(hours) * time.Hour
}

// LicenseReminderChannels 返回续费提醒渠道。
func (cfg *Config) LicenseReminderChannels() []string {
	defaultChannels := []string{"email", "in_app"}
	if cfg == nil || cfg.Marketplace == nil {
		return defaultChannels
	}
	raw := cfg.Marketplace.License.Reminder.Channels
	clean := make([]string, 0, len(raw))
	seen := map[string]struct{}{}
	for _, ch := range raw {
		normalized := strings.ToLower(strings.TrimSpace(ch))
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		clean = append(clean, normalized)
	}
	if len(clean) == 0 {
		return defaultChannels
	}
	return clean
}
