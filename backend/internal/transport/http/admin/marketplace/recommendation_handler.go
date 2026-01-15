package marketplace

import (
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/config"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/contracts"
	mrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/marketplace"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/recommendation"
	httpmw "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/middleware"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// RecommendationHandler exposes recommendation configuration and manual controls for admins.
type RecommendationHandler struct {
	cfg      *config.Config
	repo     *mrepo.ListingRepository
	provider recommendation.MetricsProvider
	logger   *logrus.Entry
}

// NewRecommendationHandler constructs a handler instance.
func NewRecommendationHandler(cfg *config.Config, repo *mrepo.ListingRepository, provider recommendation.MetricsProvider, logger *logrus.Entry) *RecommendationHandler {
	if logger == nil {
		logger = logrus.New().WithField("component", "admin_marketplace_recommendation")
	}
	return &RecommendationHandler{
		cfg:      cfg,
		repo:     repo,
		provider: provider,
		logger:   logger,
	}
}

// GetConfig returns recommendation configuration and current recommendations for the tenant.
func (h *RecommendationHandler) GetConfig(c *gin.Context) {
	tenantID, ok := httpmw.TenantUuidString(c)
	if !ok {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}

	topListings, err := h.repo.TopRecommended(c.Request.Context(), tenantID, 10)
	if err != nil {
		h.logger.WithError(err).WithField("tenant_uuid", tenantID).Error("failed to load recommended listings")
		contracts.ResponseInternalError(c, err)
		return
	}

	enabled := true
	defaultWeight := 0.0
	experimentTopic := ""
	frequency := 60
	if h.cfg != nil && h.cfg.Marketplace != nil {
		rec := h.cfg.Marketplace.Recommendation
		enabled = rec.Enabled
		defaultWeight = rec.DefaultWeight
		experimentTopic = rec.ExperimentTopic
		if rec.FrequencyMinutes > 0 {
			frequency = rec.FrequencyMinutes
		}
	}

	contracts.ResponseSuccess(c, gin.H{
		"config": gin.H{
			"enabled":           enabled,
			"default_weight":    defaultWeight,
			"experiment_topic":  experimentTopic,
			"frequency_minutes": frequency,
		},
		"top_listings": NewListingListResponse(topListings),
	})
}

// TriggerSync recomputes recommendation weights immediately for the tenant.
func (h *RecommendationHandler) TriggerSync(c *gin.Context) {
	tenantID, ok := httpmw.TenantUuidString(c)
	if !ok {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}

	engine := recommendation.NewEngine(h.repo, h.provider, h.logger)
	result, err := engine.RefreshRecommendations(c.Request.Context(), tenantID)
	if err != nil {
		h.logger.WithError(err).WithField("tenant_uuid", tenantID).Error("manual recommendation sync failed")
		contracts.ResponseInternalError(c, err)
		return
	}

	contracts.ResponseSuccess(c, gin.H{
		"tenant_uuid":       tenantID,
		"updated":           result.UpdatedCount,
		"average_weight":    result.AverageWeight,
		"exploration_share": result.ExplorationShare,
	})
}

// ManualExperimentRequest represents payload for adjusting default weight.
type ManualExperimentRequest struct {
	DefaultWeight float64 `json:"default_weight"`
}

// UpdateExperiment adjusts the in-memory default recommendation weight for quick experiments.
func (h *RecommendationHandler) UpdateExperiment(c *gin.Context) {
	if h.cfg == nil || h.cfg.Marketplace == nil {
		contracts.ResponseBadRequest(c, "marketplace configuration missing")
		return
	}
	tenantID, ok := httpmw.TenantUuidString(c)
	if !ok {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	var req ManualExperimentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid payload")
		return
	}
	if req.DefaultWeight < 0 {
		contracts.ResponseBadRequest(c, "default_weight must be >= 0")
		return
	}
	h.cfg.Marketplace.Recommendation.DefaultWeight = req.DefaultWeight
	contracts.ResponseSuccess(c, gin.H{
		"tenant_uuid":    tenantID,
		"default_weight": req.DefaultWeight,
	})
}
