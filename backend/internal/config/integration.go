package config

import (
	"strings"
	"time"
)

// IntegrationConfig 聚合协议适配、Webhook 与 Secrets 管理相关配置。
type IntegrationConfig struct {
	Idempotency IntegrationIdempotencyConfig `yaml:"idempotency" json:"idempotency"`
	Envelope    IntegrationEnvelopeConfig    `yaml:"envelope" json:"envelope"`
	Webhook     IntegrationWebhookConfig     `yaml:"webhook" json:"webhook"`
	Secrets     IntegrationSecretsConfig     `yaml:"secrets" json:"secrets"`
	Billing     IntegrationBillingConfig     `yaml:"billing" json:"billing"`
}

// IntegrationIdempotencyConfig 描述幂等后端配置。
type IntegrationIdempotencyConfig struct {
	Provider string `yaml:"provider" json:"provider"`
	RedisURL string `yaml:"redis_url" json:"redis_url"`
	TTLHours int    `yaml:"ttl_hours" json:"ttl_hours"`
}

// IntegrationEnvelopeConfig 描述 Envelope 负载约束。
type IntegrationEnvelopeConfig struct {
	PayloadThresholdBytes int64 `yaml:"payload_threshold_bytes" json:"payload_threshold_bytes"`
}

// IntegrationWebhookConfig 描述 Webhook 重试策略。
type IntegrationWebhookConfig struct {
	RetryPolicy []int  `yaml:"retry_policy" json:"retry_policy"`
	DLQTopic    string `yaml:"dlq_topic" json:"dlq_topic"`
}

// IntegrationSecretsConfig 描述外部凭证默认轮换策略。
type IntegrationSecretsConfig struct {
	RotationDaysDefault int `yaml:"rotation_days_default" json:"rotation_days_default"`
}

// IntegrationBillingConfig 描述计费与税务供应商配置。
type IntegrationBillingConfig struct {
	TaxProvider         string                        `yaml:"tax_provider" json:"tax_provider"`
	StripeTax           IntegrationStripeTaxConfig    `yaml:"stripe_tax" json:"stripe_tax"`
	Avalara             IntegrationAvalaraConfig      `yaml:"avalara" json:"avalara"`
	Reconciliation      IntegrationRevenueSplitConfig `yaml:"reconciliation" json:"reconciliation"`
	AsyncQueue          string                        `yaml:"async_queue" json:"async_queue"`
	HTTPTimeoutSeconds  int                           `yaml:"http_timeout_seconds" json:"http_timeout_seconds"`
	RetryBackoffSeconds []int                         `yaml:"retry_backoff_seconds" json:"retry_backoff_seconds"`
}

// IntegrationStripeTaxConfig 保存 Stripe Tax 凭据。
type IntegrationStripeTaxConfig struct {
	APIKey         string `yaml:"api_key" json:"api_key"`
	UserID         string `yaml:"user_id" json:"user_id"`
	Location       string `yaml:"location" json:"location"`
	APIBaseURL     string `yaml:"api_base_url" json:"api_base_url"`
	WebhookSecret  string `yaml:"webhook_secret" json:"webhook_secret"`
	TimeoutSeconds int    `yaml:"timeout_seconds" json:"timeout_seconds"`
}

// IntegrationAvalaraConfig 保存 Avalara 凭据。
type IntegrationAvalaraConfig struct {
	UserID         string `yaml:"user_id" json:"user_id"`
	LicenseKey     string `yaml:"license_key" json:"license_key"`
	CompanyCode    string `yaml:"company_code" json:"company_code"`
	Environment    string `yaml:"environment" json:"environment"`
	BaseURL        string `yaml:"base_url" json:"base_url"`
	ClientID       string `yaml:"client_id" json:"client_id"`
	ClientSecret   string `yaml:"client_secret" json:"client_secret"`
	TimeoutSeconds int    `yaml:"timeout_seconds" json:"timeout_seconds"`
}

// IntegrationRevenueSplitConfig 配置分润比例。
type IntegrationRevenueSplitConfig struct {
	VendorShare   float64 `yaml:"vendor_share" json:"vendor_share"`
	PlatformShare float64 `yaml:"platform_share" json:"platform_share"`
	FeeShare      float64 `yaml:"fee_share" json:"fee_share"`
	Currency      string  `yaml:"currency" json:"currency"`
}

// IntegrationPayloadThreshold 返回 Envelope payload 阈值（字节）。
func (cfg *Config) IntegrationPayloadThreshold() int64 {
	const defaultThreshold = int64(1 << 20) // 1 MB
	if cfg == nil || cfg.Integration == nil {
		return defaultThreshold
	}
	if v := cfg.Integration.Envelope.PayloadThresholdBytes; v > 0 {
		return v
	}
	return defaultThreshold
}

// IntegrationIdempotencyTTL 返回幂等记录存活时间。
func (cfg *Config) IntegrationIdempotencyTTL() time.Duration {
	const defaultTTL = 24 * time.Hour
	if cfg == nil || cfg.Integration == nil {
		return defaultTTL
	}
	if hours := cfg.Integration.Idempotency.TTLHours; hours > 0 {
		return time.Duration(hours) * time.Hour
	}
	return defaultTTL
}

// IntegrationIdempotencyProvider 返回幂等后端类型与 Redis 地址。
func (cfg *Config) IntegrationIdempotencyProvider() (provider string, redisURL string) {
	if cfg == nil || cfg.Integration == nil {
		return "redis", "redis://localhost:6379"
	}
	prov := cfg.Integration.Idempotency.Provider
	if prov == "" {
		prov = "redis"
	}
	redisURL = cfg.Integration.Idempotency.RedisURL
	if redisURL == "" {
		redisURL = "redis://localhost:6379"
	}
	return prov, redisURL
}

// IntegrationWebhookPolicy 返回 Webhook 重试策略与 DLQ Topic。
func (cfg *Config) IntegrationWebhookPolicy() ([]int, string) {
	defaultPolicy := []int{60, 300, 900}
	dlq := "plugin.webhook.dlq"
	if cfg == nil || cfg.Integration == nil {
		return defaultPolicy, dlq
	}
	policy := cfg.Integration.Webhook.RetryPolicy
	if len(policy) == 0 {
		policy = defaultPolicy
	}
	if topic := cfg.Integration.Webhook.DLQTopic; topic != "" {
		dlq = topic
	}
	return policy, dlq
}

// IntegrationSecretRotationDays 返回默认 Secrets 轮换天数。
func (cfg *Config) IntegrationSecretRotationDays() int {
	const defaultDays = 30
	if cfg == nil || cfg.Integration == nil {
		return defaultDays
	}
	if days := cfg.Integration.Secrets.RotationDaysDefault; days > 0 {
		return days
	}
	return defaultDays
}

// IntegrationTaxProvider 返回计费使用的税务供应商标识。
func (cfg *Config) IntegrationTaxProvider() string {
	const defaultProvider = "stripe_tax"
	if cfg == nil || cfg.Integration == nil {
		return defaultProvider
	}
	if provider := strings.TrimSpace(cfg.Integration.Billing.TaxProvider); provider != "" {
		return strings.ToLower(provider)
	}
	return defaultProvider
}

// StripeTaxConfig 返回 Stripe Tax 配置。
func (cfg *Config) StripeTaxConfig() IntegrationStripeTaxConfig {
	defaultCfg := IntegrationStripeTaxConfig{
		APIBaseURL:     "https://api.stripe.com",
		TimeoutSeconds: 15,
	}
	if cfg == nil || cfg.Integration == nil {
		return defaultCfg
	}
	out := cfg.Integration.Billing.StripeTax
	if strings.TrimSpace(out.APIBaseURL) == "" {
		out.APIBaseURL = defaultCfg.APIBaseURL
	}
	if out.TimeoutSeconds <= 0 {
		out.TimeoutSeconds = defaultCfg.TimeoutSeconds
	}
	return out
}

// AvalaraConfig 返回 Avalara 配置。
func (cfg *Config) AvalaraConfig() IntegrationAvalaraConfig {
	defaultCfg := IntegrationAvalaraConfig{
		Environment:    "sandbox",
		BaseURL:        "https://sandbox-rest.avatax.com",
		TimeoutSeconds: 15,
	}
	if cfg == nil || cfg.Integration == nil {
		return defaultCfg
	}
	out := cfg.Integration.Billing.Avalara
	if strings.TrimSpace(out.Environment) == "" {
		out.Environment = defaultCfg.Environment
	}
	if strings.TrimSpace(out.BaseURL) == "" {
		out.BaseURL = defaultCfg.BaseURL
	}
	if out.TimeoutSeconds <= 0 {
		out.TimeoutSeconds = defaultCfg.TimeoutSeconds
	}
	return out
}

// IntegrationRevenueSplit 返回分润比例配置，若缺省则使用默认值。
func (cfg *Config) IntegrationRevenueSplit() IntegrationRevenueSplitConfig {
	const (
		defaultVendor   = 0.80
		defaultPlatform = 0.15
		defaultFee      = 0.05
		defaultCurrency = "USD"
	)
	if cfg == nil || cfg.Integration == nil {
		return IntegrationRevenueSplitConfig{
			VendorShare:   defaultVendor,
			PlatformShare: defaultPlatform,
			FeeShare:      defaultFee,
			Currency:      defaultCurrency,
		}
	}
	split := cfg.Integration.Billing.Reconciliation
	if split.VendorShare <= 0 {
		split.VendorShare = defaultVendor
	}
	if split.PlatformShare <= 0 {
		split.PlatformShare = defaultPlatform
	}
	if split.FeeShare <= 0 {
		split.FeeShare = defaultFee
	}
	if strings.TrimSpace(split.Currency) == "" {
		split.Currency = defaultCurrency
	}
	return split
}

// BillingHTTPTimeout returns the configured HTTP client timeout for tax providers.
func (cfg *Config) BillingHTTPTimeout() time.Duration {
	const defaultTimeout = 15 * time.Second
	if cfg == nil || cfg.Integration == nil {
		return defaultTimeout
	}
	if seconds := cfg.Integration.Billing.HTTPTimeoutSeconds; seconds > 0 {
		return time.Duration(seconds) * time.Second
	}
	return defaultTimeout
}

// BillingRetrySchedule returns the retry backoff settings for billing integrations.
func (cfg *Config) BillingRetrySchedule() []time.Duration {
	defaultSchedule := []time.Duration{5 * time.Second, 30 * time.Second, 120 * time.Second}
	if cfg == nil || cfg.Integration == nil {
		return defaultSchedule
	}
	raw := cfg.Integration.Billing.RetryBackoffSeconds
	if len(raw) == 0 {
		return defaultSchedule
	}
	schedule := make([]time.Duration, 0, len(raw))
	for _, sec := range raw {
		if sec <= 0 {
			continue
		}
		schedule = append(schedule, time.Duration(sec)*time.Second)
	}
	if len(schedule) == 0 {
		return defaultSchedule
	}
	return schedule
}
