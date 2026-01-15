package runtime_ops

import (
	"context"
	"sync"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/config"
	model "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/runtime_ops"
	repo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository"
	runtimeRepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/runtime_ops"
	authx "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/middleware"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// QuotaService manages quota tracking and enforcement.
type QuotaService struct {
	repo     *runtimeRepo.QuotaRepository
	audits   *repo.BaseRepository[model.RuntimeAuditEvent]
	bucketMu sync.Mutex
	buckets  map[string]*tokenBucket
	defaults *config.RuntimeOpsDefaults
}

// QuotaCost represents resource consumption for a request.
type QuotaCost struct {
	Tokens      float64
	CPUSeconds  float64
	BandwidthMB float64
	Cost        float64
}

// NewQuotaService constructs QuotaService with repositories and defaults.
func NewQuotaService(db *gorm.DB, defaults *config.RuntimeOpsDefaults) *QuotaService {
	if defaults != nil {
		ConfigureRuntimeOps(defaults)
	}
	return &QuotaService{
		repo:     runtimeRepo.NewQuotaRepository(db),
		audits:   repo.NewBaseRepository[model.RuntimeAuditEvent](db),
		buckets:  make(map[string]*tokenBucket),
		defaults: defaults,
	}
}

// Defaults returns configured runtime ops defaults (may be nil).
func (s *QuotaService) Defaults() *config.RuntimeOpsDefaults {
	return s.defaults
}

// AllowRequest evaluates quota for given scope and capability.
func (s *QuotaService) AllowRequest(ctx context.Context, scopeType, scopeRef, pluginID, capability string, cost QuotaCost) (bool, error) {
	if s.defaults == nil {
		s.defaults = &config.RuntimeOpsDefaults{QuotaWindowMinutes: 5}
	}

	bucketKey := scopeType + ":" + scopeRef + ":" + capability
	s.bucketMu.Lock()
	bucket := s.buckets[bucketKey]
	if bucket == nil {
		capacity := 100.0
		refillRate := capacity / float64(max(1, s.defaults.QuotaWindowMinutes*60))
		bucket = newTokenBucket(capacity, refillRate)
		s.buckets[bucketKey] = bucket
	}
	s.bucketMu.Unlock()

	if !bucket.Allow(cost.Tokens) {
		// Emit audit event & metric
		SetQuotaUsage(pluginID, scopeType, scopeRef, 1.0)
		_ = s.RecordBreach(ctx, pluginID, scopeRef, capability, ActionThrottle)
		s.HandleBreach(ctx, pluginID, scopeRef, capability, ActionThrottle)
		return false, nil
	}

	// Update metrics for successful request
	SetQuotaUsage(pluginID, scopeType, scopeRef, bucket.Utilization())
	ObserveCPU(pluginID, scopeRef, cost.CPUSeconds)
	AddCost(pluginID, scopeRef, cost.Cost)
	return true, nil
}

// RecordUsage writes ledger entry for reporting.
func (s *QuotaService) RecordUsage(ctx context.Context, entry *model.QuotaLedger) (*model.QuotaLedger, error) {
	if entry != nil && entry.ID == "" {
		entry.ID = uuid.NewString()
	}
	return s.repo.RecordUsage(ctx, entry)
}

// ListUsage fetches ledger entries within the specified window.
func (s *QuotaService) ListUsage(ctx context.Context, scopeType, scopeRef string, start, end time.Time) ([]*model.QuotaLedger, error) {
	return s.repo.ListWindow(ctx, scopeType, scopeRef, start, end)
}

// RecordBreach stores audit event for quota breach.
func (s *QuotaService) RecordBreach(ctx context.Context, pluginID, scopeRef, capability, action string) error {
	tenantUUID, err := authx.RequireTenantUUID(ctx)
	if err != nil {
		return err
	}
	event := &model.RuntimeAuditEvent{
		ID:         uuid.NewString(),
		PluginID:   pluginID,
		TenantUUID: tenantUUID,
		EventType:  "quota_breach",
		Payload:    capability + ":" + action,
		OccurredAt: time.Now(),
	}
	_, err = s.audits.Create(ctx, event)
	return err
}

// ScheduleMarketplaceSummary persists overage summary for Marketplace reporting.
func (s *QuotaService) ScheduleMarketplaceSummary(ctx context.Context, summary *model.MarketplaceOverage) (*model.MarketplaceOverage, error) {
	if summary != nil && summary.ID == "" {
		summary.ID = uuid.NewString()
	}
	return s.repo.CreateOverage(ctx, summary)
}

// tokenBucket implements per-scope token bucket rate limiting.
type tokenBucket struct {
	capacity   float64
	tokens     float64
	refillRate float64
	lastRefill time.Time
	mu         sync.Mutex
}

func newTokenBucket(capacity, refillRate float64) *tokenBucket {
	return &tokenBucket{
		capacity:   capacity,
		tokens:     capacity,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

func (b *tokenBucket) refill() {
	now := time.Now()
	delta := now.Sub(b.lastRefill).Seconds()
	if delta <= 0 {
		return
	}
	b.tokens += delta * b.refillRate
	if b.tokens > b.capacity {
		b.tokens = b.capacity
	}
	b.lastRefill = now
}

func (b *tokenBucket) Allow(tokens float64) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.refill()
	if b.tokens < tokens {
		return false
	}
	b.tokens -= tokens
	return true
}

func (b *tokenBucket) Utilization() float64 {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.capacity == 0 {
		return 0
	}
	return 1 - b.tokens/b.capacity
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
