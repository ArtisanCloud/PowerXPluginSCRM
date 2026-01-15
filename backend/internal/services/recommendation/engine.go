package recommendation

import (
	"context"
	"math"
	"time"

	dbm "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/marketplace"
	mrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/marketplace"
	"github.com/sirupsen/logrus"
)

// Signal captures marketplace listing metrics used for recommendation scoring.
type Signal struct {
	ListingID           string
	InstallCount        int
	RatingAverage       float64
	RatingCount         int
	LastPublishedAt     time.Time
	ReadyChecklistScore int
	AvgResponseMs       float64
	BrandCompleteness   float64
	CreatedAt           time.Time
}

// MetricsProvider fetches listing recommendation signals for a tenant.
type MetricsProvider interface {
	FetchSignals(ctx context.Context, tenantID string) ([]Signal, error)
}

// Weights controls contribution of each signal to the final score.
type Weights struct {
	Installs  float64
	Rating    float64
	Freshness float64
	Latency   float64
	Checklist float64
	Brand     float64
}

// Config holds tunable parameters for the recommendation engine.
type Config struct {
	Weights            Weights
	FreshnessHalfLife  time.Duration
	LatencySLA         float64
	ExplorationBoost   float64
	ExplorationInstall int
}

// DefaultConfig provides baseline scoring weights.
var DefaultConfig = Config{
	Weights: Weights{
		Installs:  0.35,
		Rating:    0.25,
		Freshness: 0.15,
		Latency:   0.10,
		Checklist: 0.10,
		Brand:     0.05,
	},
	FreshnessHalfLife:  30 * 24 * time.Hour,
	LatencySLA:         600, // milliseconds
	ExplorationBoost:   0.35,
	ExplorationInstall: 50,
}

// Option customizes engine configuration.
type Option func(*Config)

// WithConfig overrides the default configuration.
func WithConfig(cfg Config) Option {
	return func(c *Config) {
		*c = cfg
	}
}

// EngineResult summarises a refresh cycle.
type EngineResult struct {
	UpdatedCount     int
	AverageWeight    float64
	ExplorationShare float64
}

// Engine computes recommendation weights and persists them into listings.
type Engine struct {
	listings *mrepo.ListingRepository
	provider MetricsProvider
	logger   *logrus.Entry
	config   Config
}

// NewEngine constructs a recommendation engine with optional configuration overrides.
func NewEngine(repo *mrepo.ListingRepository, provider MetricsProvider, logger *logrus.Entry, opts ...Option) *Engine {
	cfg := DefaultConfig
	for _, opt := range opts {
		opt(&cfg)
	}
	if logger == nil {
		logger = logrus.New().WithField("component", "marketplace_recommendation_engine")
	}
	return &Engine{
		listings: repo,
		provider: provider,
		logger:   logger,
		config:   cfg,
	}
}

// RefreshRecommendations pulls metrics for the tenant and updates recommended weights.
func (e *Engine) RefreshRecommendations(ctx context.Context, tenantID string) (EngineResult, error) {
	result := EngineResult{}
	signals, err := e.provider.FetchSignals(ctx, tenantID)
	if err != nil {
		return result, err
	}
	if len(signals) == 0 {
		return result, nil
	}

	var (
		totalWeight       float64
		explorationCounts int
	)

	for _, signal := range signals {
		weight := e.calculateWeight(signal)
		if err := e.listings.UpdateRecommendedWeight(ctx, tenantID, signal.ListingID, weight); err != nil {
			e.logger.WithError(err).WithField("listing_id", signal.ListingID).Warn("failed to update recommended weight")
			continue
		}
		result.UpdatedCount++
		totalWeight += weight
		if signal.InstallCount <= e.config.ExplorationInstall {
			explorationCounts++
		}
	}

	if result.UpdatedCount > 0 {
		result.AverageWeight = totalWeight / float64(result.UpdatedCount)
		result.ExplorationShare = float64(explorationCounts) / float64(result.UpdatedCount)
	}
	return result, nil
}

func (e *Engine) calculateWeight(signal Signal) float64 {
	cfg := e.config
	weights := cfg.Weights

	installComponent := weights.Installs * math.Log1p(float64(signal.InstallCount))

	ratingBase := signal.RatingAverage / 5.0
	if ratingBase < 0 {
		ratingBase = 0
	}
	if ratingBase > 1 {
		ratingBase = 1
	}
	ratingConfidence := math.Log1p(float64(signal.RatingCount))
	ratingComponent := weights.Rating * ratingBase * (1 + ratingConfidence/5.0)

	var freshnessComponent float64
	if !signal.LastPublishedAt.IsZero() && cfg.FreshnessHalfLife > 0 {
		age := time.Since(signal.LastPublishedAt)
		decay := math.Exp(-age.Hours() / cfg.FreshnessHalfLife.Hours())
		freshnessComponent = weights.Freshness * decay
	}

	latency := signal.AvgResponseMs
	if latency <= 0 {
		latency = cfg.LatencySLA
	}
	latencyScore := 1 - (latency / cfg.LatencySLA)
	if latencyScore < 0 {
		latencyScore = 0
	}
	if latencyScore > 1 {
		latencyScore = 1
	}
	latencyComponent := weights.Latency * latencyScore

	checklistComponent := weights.Checklist * (float64(signal.ReadyChecklistScore) / 100.0)
	if checklistComponent < 0 {
		checklistComponent = 0
	}
	if checklistComponent > weights.Checklist {
		checklistComponent = weights.Checklist
	}

	brandComponent := weights.Brand * clamp01(signal.BrandCompleteness)

	score := installComponent + ratingComponent + freshnessComponent + latencyComponent + checklistComponent + brandComponent

	if signal.InstallCount <= cfg.ExplorationInstall {
		score += cfg.ExplorationBoost
	}

	if score < 0 {
		return 0
	}
	return math.Round(score*1e4) / 1e4
}

func clamp01(v float64) float64 {
	switch {
	case v < 0:
		return 0
	case v > 1:
		return 1
	default:
		return v
	}
}

// PrepareSignalFromListing builds default signals from listing entity when supplemental metrics are unavailable.
func PrepareSignalFromListing(listing *dbm.Listing) Signal {
	if listing == nil {
		return Signal{}
	}
	return Signal{
		ListingID:           listing.ID,
		LastPublishedAt:     derefTime(listing.PublishedAt),
		ReadyChecklistScore: listing.ReadyChecklistScore,
		BrandCompleteness:   1,
		CreatedAt:           listing.CreatedAt,
	}
}

func derefTime(t *time.Time) time.Time {
	if t == nil {
		return time.Time{}
	}
	return *t
}
