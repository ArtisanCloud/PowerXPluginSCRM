package marketplace

import (
	"context"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/config"
	mrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/marketplace"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/recommendation"
	"github.com/sirupsen/logrus"
)

// MetricsProvider defines the interface to fetch signals for recommendation scoring.
type MetricsProvider interface {
	FetchSignals(ctx context.Context, tenantID string) ([]recommendation.Signal, error)
}

// SyncJob periodically refreshes listing recommendation weights.
type SyncJob struct {
	cfg      *config.Config
	repo     *mrepo.ListingRepository
	provider MetricsProvider
	interval time.Duration
	logger   *logrus.Entry
	tenants  func(context.Context) ([]string, error)
}

// NewSyncJob constructs a new recommendation synchronization job.
func NewSyncJob(cfg *config.Config, repo *mrepo.ListingRepository, provider MetricsProvider, logger *logrus.Entry, tenantResolver func(context.Context) ([]string, error)) *SyncJob {
	if logger == nil {
		logger = logrus.New().WithField("component", "marketplace_recommendation_sync")
	}
	interval := time.Hour
	if cfg != nil {
		interval = cfg.RecommendationFrequency()
	}
	if tenantResolver == nil {
		tenantResolver = func(context.Context) ([]string, error) { return []string{"default"}, nil }
	}
	return &SyncJob{
		cfg:      cfg,
		repo:     repo,
		provider: provider,
		interval: interval,
		logger:   logger,
		tenants:  tenantResolver,
	}
}

// Run starts the background synchronization loop until the context is canceled.
func (j *SyncJob) Run(ctx context.Context) {
	ticker := time.NewTicker(j.interval)
	defer ticker.Stop()

	j.execute(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			j.execute(ctx)
		}
	}
}

func (j *SyncJob) execute(ctx context.Context) {
	if j.cfg != nil && j.cfg.Marketplace != nil && !j.cfg.Marketplace.Recommendation.Enabled {
		return
	}
	tenants, err := j.tenants(ctx)
	if err != nil {
		j.logger.WithError(err).Warn("failed to enumerate tenants for recommendation sync")
		return
	}
	engine := recommendation.NewEngine(j.repo, j.provider, j.logger)
	for _, tenantID := range tenants {
		result, err := engine.RefreshRecommendations(ctx, tenantID)
		if err != nil {
			j.logger.WithError(err).WithField("tenant_uuid", tenantID).Error("recommendation sync failed")
			continue
		}
		j.logger.WithFields(logrus.Fields{
			"tenant_uuid":         tenantID,
			"updated":           result.UpdatedCount,
			"average_weight":    result.AverageWeight,
			"exploration_share": result.ExplorationShare,
		}).Info("recommendation weights refreshed")
	}
}
