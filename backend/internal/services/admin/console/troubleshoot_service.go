package console

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/config"
	adminmetrics "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/observability/admin_console"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
	"gorm.io/gorm"
)

// HealthStatus represents a health indicator in the troubleshooting dashboard.
type HealthStatus struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// QuotaUsage captures quota consumption data.
type QuotaUsage struct {
	Capability       string  `json:"capability"`
	UsagePercent     float64 `json:"usage_percent"`
	ThresholdPercent float64 `json:"threshold_percent"`
	Window           string  `json:"window,omitempty"`
}

// WebhookAttemptSummary aggregates webhook attempt diagnostics.
type WebhookAttemptSummary struct {
	ID           string `json:"id"`
	Status       string `json:"status"`
	ResponseCode int    `json:"response_code,omitempty"`
	PayloadID    string `json:"payload_id,omitempty"`
	LastError    string `json:"last_error,omitempty"`
}

// WebhookDeliverySummary describes delivery performance metrics.
type WebhookDeliverySummary struct {
	SuccessRate    float64                 `json:"success_rate"`
	RetryRate      float64                 `json:"retry_rate"`
	DLQRate        float64                 `json:"dlq_rate"`
	RecentFailures []WebhookAttemptSummary `json:"recent_failures,omitempty"`
}

// GuidanceItem includes contextual remediation info.
type GuidanceItem struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

// WebhookAttempt summarises a webhook delivery attempt for UI consumption.
type WebhookAttempt struct {
	ID              string     `json:"id"`
	SubscriptionID  string     `json:"subscription_id"`
	TenantUuid      string     `json:"tenant_uuid"`
	Status          string     `json:"status"`
	DeliveryCount   int        `json:"delivery_count"`
	RetryCount      int        `json:"retry_count"`
	LastError       string     `json:"last_error,omitempty"`
	ResponseCode    int        `json:"response_code,omitempty"`
	PayloadID       string     `json:"payload_id,omitempty"`
	LastAttemptedAt time.Time  `json:"last_attempted_at"`
	NextRetryAt     *time.Time `json:"next_retry_at,omitempty"`
	DLQReason       string     `json:"dlq_reason,omitempty"`
}

// WebhookAttemptList wraps attempts with pagination cursor.
type WebhookAttemptList struct {
	Attempts   []WebhookAttempt `json:"attempts"`
	NextCursor string           `json:"next_cursor,omitempty"`
}

// WebhookAttemptListInput defines filters for attempts listing.
type WebhookAttemptListInput struct {
	TenantUuid     string
	Status         string
	SubscriptionID string
	Since          *time.Time
	Cursor         string
	Limit          int
}

// TroubleshootingSummary aggregates health, quota, and webhook diagnostics.
type TroubleshootingSummary struct {
	RefreshedAt            time.Time              `json:"refreshed_at"`
	RefreshIntervalSeconds int                    `json:"refresh_interval_seconds"`
	Health                 []HealthStatus         `json:"health"`
	Quota                  []QuotaUsage           `json:"quota"`
	WebhookDelivery        WebhookDeliverySummary `json:"webhook_delivery"`
	Guidance               []GuidanceItem         `json:"guidance,omitempty"`
}

// TroubleshootSummaryInput controls summary fetching.
type TroubleshootSummaryInput struct {
	TenantUuid   *string
	ForceRefresh bool
}

// HealthSource fetches health metrics.
type HealthSource interface {
	FetchHealth(ctx context.Context, tenantID *string) ([]HealthStatus, error)
}

// QuotaSource fetches quota usage.
type QuotaSource interface {
	FetchQuota(ctx context.Context, tenantID *string) ([]QuotaUsage, error)
}

// WebhookSource fetches webhook delivery data.
type WebhookSource interface {
	FetchWebhookSummary(ctx context.Context, tenantID *string) (WebhookDeliverySummary, error)
}

// GuidanceSource returns contextual remediation items.
type GuidanceSource interface {
	FetchGuidance(ctx context.Context, tenantID *string) ([]GuidanceItem, error)
}

// TroubleshootService aggregates troubleshooting data with caching.
type TroubleshootService struct {
	cfg     *config.Config
	metrics *adminmetrics.Metrics
	db      *gorm.DB

	health   HealthSource
	quota    QuotaSource
	webhooks WebhookSource
	guidance GuidanceSource

	now      func() time.Time
	cache    map[string]cachedSummary
	cacheM   sync.Mutex
	cacheTTL time.Duration
}

type cachedSummary struct {
	summary   *TroubleshootingSummary
	expiresAt time.Time
}

// TroubleshootServiceOption customises service construction.
type TroubleshootServiceOption func(*TroubleshootService)

// WithHealthSource overrides the health provider.
func WithHealthSource(source HealthSource) TroubleshootServiceOption {
	return func(s *TroubleshootService) {
		if source != nil {
			s.health = source
		}
	}
}

// WithQuotaSource overrides the quota provider.
func WithQuotaSource(source QuotaSource) TroubleshootServiceOption {
	return func(s *TroubleshootService) {
		if source != nil {
			s.quota = source
		}
	}
}

// WithWebhookSource overrides the webhook provider.
func WithWebhookSource(source WebhookSource) TroubleshootServiceOption {
	return func(s *TroubleshootService) {
		if source != nil {
			s.webhooks = source
		}
	}
}

// WithGuidanceSource overrides the guidance provider.
func WithGuidanceSource(source GuidanceSource) TroubleshootServiceOption {
	return func(s *TroubleshootService) {
		if source != nil {
			s.guidance = source
		}
	}
}

// WithTroubleshootClock injects a deterministic clock (primarily for tests).
func WithTroubleshootClock(now func() time.Time) TroubleshootServiceOption {
	return func(s *TroubleshootService) {
		if now != nil {
			s.now = now
		}
	}
}

// NewTroubleshootService constructs a troubleshooting aggregator.
func NewTroubleshootService(deps *app.Deps, opts ...TroubleshootServiceOption) *TroubleshootService {
	var metrics *adminmetrics.Metrics
	var cfg *config.Config
	var db *gorm.DB
	if deps != nil {
		metrics = deps.AdminConsoleMetrics
		cfg = deps.Config
		db = deps.DB
	}
	if metrics == nil {
		metrics = adminmetrics.NewMetrics()
	}
	cacheTTL := time.Duration(120) * time.Second
	if cfg != nil && cfg.AdminConsole != nil {
		ttl := cfg.AdminConsole.Troubleshooting.CacheTTLSeconds
		if ttl > 0 {
			cacheTTL = time.Duration(ttl) * time.Second
		}
	}
	service := &TroubleshootService{
		cfg:      cfg,
		metrics:  metrics,
		db:       db,
		health:   noopHealth{},
		quota:    noopQuota{},
		webhooks: noopWebhook{},
		guidance: noopGuidance{},
		now:      time.Now,
		cache:    make(map[string]cachedSummary),
		cacheTTL: cacheTTL,
	}
	for _, opt := range opts {
		opt(service)
	}
	return service
}

// Summary returns troubleshooting data, leveraging cache where possible.
func (s *TroubleshootService) Summary(ctx context.Context, input TroubleshootSummaryInput) (*TroubleshootingSummary, error) {
	if s == nil {
		return nil, ErrJobServiceUnavailable
	}
	key := cacheKey(input.TenantUuid)
	if !input.ForceRefresh {
		if summary := s.cached(key); summary != nil {
			return summary, nil
		}
	}
	now := s.now()
	refreshInterval := s.refreshInterval()

	health, err := s.health.FetchHealth(ctx, input.TenantUuid)
	if err != nil {
		return nil, err
	}
	quota, err := s.quota.FetchQuota(ctx, input.TenantUuid)
	if err != nil {
		return nil, err
	}
	webhookSummary, err := s.webhooks.FetchWebhookSummary(ctx, input.TenantUuid)
	if err != nil {
		return nil, err
	}
	guidance, err := s.guidance.FetchGuidance(ctx, input.TenantUuid)
	if err != nil {
		return nil, err
	}

	summary := &TroubleshootingSummary{
		RefreshedAt:            now,
		RefreshIntervalSeconds: refreshInterval,
		Health:                 health,
		Quota:                  quota,
		WebhookDelivery:        webhookSummary,
		Guidance:               guidance,
	}

	s.remember(key, summary)
	scope := cacheScope(key)
	if s.metrics != nil {
		s.metrics.ObserveDashboardFreshness(scope, 0)
	}
	return summary, nil
}

// ListWebhookAttempts returns recent webhook delivery attempts for a tenant.
func (s *TroubleshootService) ListWebhookAttempts(ctx context.Context, input WebhookAttemptListInput) (*WebhookAttemptList, error) {
	if s == nil || s.db == nil {
		return nil, ErrJobServiceUnavailable
	}
	tenant := strings.TrimSpace(input.TenantUuid)
	if tenant == "" {
		return nil, validationError{Field: "tenant_uuid", Message: "tenant id is required"}
	}
	limit := input.Limit
	if limit <= 0 || limit > 100 {
		limit = 25
	}
	query := s.db.WithContext(ctx).
		Table("integration_webhook_attempts AS a").
		Select(`a.id, a.subscription_id, s.tenant_uuid, a.status, a.retry_count, a.last_error, a.next_delivery_at, a.payload_snapshot, a.created_at, a.updated_at, dlq.failure_reason AS dlq_reason`).
		Joins("JOIN integration_webhook_subscriptions AS s ON s.id = a.subscription_id").
		Joins("LEFT JOIN integration_webhook_dlq AS dlq ON dlq.attempt_id = a.id").
		Where("s.tenant_uuid = ?", tenant)
	if status := strings.TrimSpace(input.Status); status != "" {
		query = query.Where("LOWER(a.status) = ?", strings.ToLower(status))
	}
	if sub := strings.TrimSpace(input.SubscriptionID); sub != "" {
		query = query.Where("a.subscription_id = ?", sub)
	}
	if input.Since != nil {
		query = query.Where("a.created_at >= ?", input.Since.UTC())
	}
	if input.Cursor != "" {
		ts, id, err := decodeAttemptCursor(input.Cursor)
		if err != nil {
			return nil, err
		}
		query = query.Where("(a.created_at < ?) OR (a.created_at = ? AND a.id < ?)", ts, ts, id)
	}
	var rows []attemptRow
	if err := query.
		Order("a.created_at DESC").
		Order("a.id DESC").
		Limit(limit + 1).
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	var next string
	if len(rows) > limit {
		last := rows[limit]
		next = encodeAttemptCursor(last.CreatedAt, last.ID)
		rows = rows[:limit]
	}
	attempts := make([]WebhookAttempt, len(rows))
	for i, row := range rows {
		attempts[i] = mapAttemptRow(row)
	}
	return &WebhookAttemptList{Attempts: attempts, NextCursor: next}, nil
}

// GetWebhookAttempt returns detailed attempt information scoped by tenant.
func (s *TroubleshootService) GetWebhookAttempt(ctx context.Context, attemptID string, tenantID string) (*WebhookAttempt, error) {
	if s == nil || s.db == nil {
		return nil, ErrJobServiceUnavailable
	}
	cleanAttempt := strings.TrimSpace(attemptID)
	if cleanAttempt == "" {
		return nil, validationError{Field: "attempt_id", Message: "attempt id is required"}
	}
	tenant := strings.TrimSpace(tenantID)
	if tenant == "" {
		return nil, validationError{Field: "tenant_uuid", Message: "tenant id is required"}
	}
	query := s.db.WithContext(ctx).
		Table("integration_webhook_attempts AS a").
		Select(`a.id, a.subscription_id, s.tenant_uuid, a.status, a.retry_count, a.last_error, a.next_delivery_at, a.payload_snapshot, a.created_at, a.updated_at, dlq.failure_reason AS dlq_reason`).
		Joins("JOIN integration_webhook_subscriptions AS s ON s.id = a.subscription_id").
		Joins("LEFT JOIN integration_webhook_dlq AS dlq ON dlq.attempt_id = a.id").
		Where("a.id = ? AND s.tenant_uuid = ?", cleanAttempt, tenant)
	var row attemptRow
	res := query.Take(&row)
	if errors.Is(res.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if res.Error != nil {
		return nil, res.Error
	}
	attempt := mapAttemptRow(row)
	return &attempt, nil
}

func (s *TroubleshootService) cached(key string) *TroubleshootingSummary {
	s.cacheM.Lock()
	defer s.cacheM.Unlock()
	entry, ok := s.cache[key]
	if !ok {
		return nil
	}
	if s.now().After(entry.expiresAt) {
		delete(s.cache, key)
		return nil
	}
	return entry.summary
}

func (s *TroubleshootService) remember(key string, summary *TroubleshootingSummary) {
	s.cacheM.Lock()
	defer s.cacheM.Unlock()
	s.cache[key] = cachedSummary{
		summary:   summary,
		expiresAt: s.now().Add(s.cacheTTL),
	}
}

func (s *TroubleshootService) refreshInterval() int {
	if s.cfg == nil {
		return 300
	}
	return s.cfg.AdminConsoleRefreshInterval()
}

type attemptRow struct {
	ID              string
	SubscriptionID  string
	TenantUuid      string
	Status          string
	RetryCount      int
	LastError       string
	NextDeliveryAt  *time.Time
	PayloadSnapshot []byte
	CreatedAt       time.Time
	UpdatedAt       time.Time
	DLQReason       *string
}

func mapAttemptRow(row attemptRow) WebhookAttempt {
	attempt := WebhookAttempt{
		ID:              row.ID,
		SubscriptionID:  row.SubscriptionID,
		TenantUuid:      row.TenantUuid,
		Status:          strings.ToLower(strings.TrimSpace(row.Status)),
		RetryCount:      row.RetryCount,
		DeliveryCount:   row.RetryCount + 1,
		LastError:       strings.TrimSpace(row.LastError),
		LastAttemptedAt: row.UpdatedAt.UTC(),
		NextRetryAt:     toUTCTime(row.NextDeliveryAt),
	}
	if attempt.LastAttemptedAt.IsZero() {
		attempt.LastAttemptedAt = row.CreatedAt.UTC()
	}
	if row.DLQReason != nil {
		attempt.DLQReason = strings.TrimSpace(*row.DLQReason)
		if attempt.LastError == "" {
			attempt.LastError = attempt.DLQReason
		}
	}
	payloadID, responseCode := parsePayloadSnapshot(row.PayloadSnapshot)
	if payloadID != "" {
		attempt.PayloadID = payloadID
	}
	if responseCode > 0 {
		attempt.ResponseCode = responseCode
	}
	return attempt
}

func parsePayloadSnapshot(data []byte) (string, int) {
	if len(data) == 0 {
		return "", 0
	}
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		return "", 0
	}
	var payloadID string
	if v, ok := payload["payload_id"].(string); ok {
		payloadID = v
	} else if inner, ok := payload["payload"].(map[string]any); ok {
		if id, ok := inner["id"].(string); ok {
			payloadID = id
		}
	}
	responseCode := extractResponseCode(payload)
	return payloadID, responseCode
}

func extractResponseCode(payload map[string]any) int {
	candidates := []string{"response_code", "last_response_code", "status"}
	for _, key := range candidates {
		if v, ok := payload[key]; ok {
			if code, ok := toInt(v); ok {
				return code
			}
		}
	}
	return 0
}

func toInt(value any) (int, bool) {
	switch v := value.(type) {
	case float64:
		return int(v), true
	case float32:
		return int(v), true
	case int:
		return v, true
	case int64:
		return int(v), true
	case int32:
		return int(v), true
	case json.Number:
		i, err := v.Int64()
		if err != nil {
			return 0, false
		}
		return int(i), true
	case string:
		i, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil {
			return 0, false
		}
		return i, true
	default:
		return 0, false
	}
}

func toUTCTime(ts *time.Time) *time.Time {
	if ts == nil || ts.IsZero() {
		return nil
	}
	value := ts.UTC()
	return &value
}

func encodeAttemptCursor(ts time.Time, id string) string {
	payload := fmt.Sprintf("%d|%s", ts.UTC().UnixNano(), id)
	return base64.URLEncoding.EncodeToString([]byte(payload))
}

func decodeAttemptCursor(cursor string) (time.Time, string, error) {
	if strings.TrimSpace(cursor) == "" {
		return time.Time{}, "", fmt.Errorf("cursor required")
	}
	bytes, err := base64.URLEncoding.DecodeString(cursor)
	if err != nil {
		return time.Time{}, "", fmt.Errorf("invalid cursor")
	}
	parts := strings.SplitN(string(bytes), "|", 2)
	if len(parts) != 2 {
		return time.Time{}, "", fmt.Errorf("invalid cursor format")
	}
	nanos, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return time.Time{}, "", fmt.Errorf("invalid cursor timestamp")
	}
	return time.Unix(0, nanos).UTC(), parts[1], nil
}

type noopHealth struct{}

func (noopHealth) FetchHealth(context.Context, *string) ([]HealthStatus, error) {
	return []HealthStatus{}, nil
}

type noopQuota struct{}

func (noopQuota) FetchQuota(context.Context, *string) ([]QuotaUsage, error) {
	return []QuotaUsage{}, nil
}

type noopWebhook struct{}

func (noopWebhook) FetchWebhookSummary(context.Context, *string) (WebhookDeliverySummary, error) {
	return WebhookDeliverySummary{}, nil
}

type noopGuidance struct{}

func (noopGuidance) FetchGuidance(context.Context, *string) ([]GuidanceItem, error) {
	return []GuidanceItem{}, nil
}

func cacheKey(tenantID *string) string {
	if tenantID == nil || *tenantID == "" {
		return "global"
	}
	return "tenant:" + *tenantID
}

func cacheScope(key string) string {
	if key == "global" {
		return "global"
	}
	return key
}
