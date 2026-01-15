package operations

import (
	"context"
	"errors"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/config"
	opmodels "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/operations"
	oprepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/operations"
	opmetrics "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/observability/operations"
	runtimeops "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/runtime_ops"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
)

const (
	slaScoreIncentiveThreshold = 85.0
	slaScorePenaltyThreshold   = 70.0
)

// SLAService encapsulates SLA profile management and incentive logic.
type SLAService struct {
	cfg       *config.Config
	repo      *oprepo.SLARepository
	metrics   *opmetrics.Metrics
	readiness runtimeops.ReadinessBlueprint
}

// NewSLAService constructs an SLAService.
func NewSLAService(repo *oprepo.SLARepository, cfg *config.Config, metrics *opmetrics.Metrics) *SLAService {
	if metrics == nil {
		metrics = opmetrics.NewMetrics()
	}
	return &SLAService{
		cfg:       cfg,
		repo:      repo,
		metrics:   metrics,
		readiness: runtimeops.DefaultReadinessBlueprint(),
	}
}

// ProfileTargets represents desired SLA target updates.
type ProfileTargets struct {
	PlanType              string  `json:"planType"`
	UptimeTarget          float64 `json:"uptimeTarget"`
	ResponseTargetMs      int32   `json:"responseTargetMs"`
	SuccessTargetPct      float64 `json:"successTargetPct"`
	SupportFrtTargetHours float64 `json:"supportFrtTargetHours"`
}

// ActualMetrics represents measured SLA performance numbers.
type ActualMetrics struct {
	UptimeActual          float64 `json:"uptimeActual"`
	ResponseActualMs      int32   `json:"responseActualMs"`
	SuccessActualPct      float64 `json:"successActualPct"`
	SupportFrtActualHours float64 `json:"supportFrtActualHours"`
}

// UpsertTargets updates SLA targets for a plan type.
func (s *SLAService) UpsertTargets(ctx context.Context, targets ProfileTargets) (*opmodels.SLAProfile, error) {
	if s.repo == nil {
		return nil, errors.New("sla repository not configured")
	}
	plan := normalizePlan(targets.PlanType)
	if plan == "" {
		return nil, errors.New("planType is required")
	}
	profile := &opmodels.SLAProfile{
		PluginID:              app.PluginID,
		PlanType:              plan,
		UptimeTarget:          clampPercentage(targets.UptimeTarget),
		ResponseTargetMs:      targets.ResponseTargetMs,
		SuccessTargetPct:      clampPercentage(targets.SuccessTargetPct),
		SupportFrtTargetHours: clampHours(targets.SupportFrtTargetHours),
	}
	existing, err := s.repo.GetProfile(ctx, profile.PluginID, plan)
	if err == nil {
		profile.ID = existing.ID
		profile.CreatedAt = existing.CreatedAt
		profile.UptimeActual = existing.UptimeActual
		profile.ResponseActualMs = existing.ResponseActualMs
		profile.SuccessActualPct = existing.SuccessActualPct
		profile.SupportFrtActualHours = existing.SupportFrtActualHours
		profile.SLAScore = existing.SLAScore
		profile.IncentiveAppliedAt = existing.IncentiveAppliedAt
		profile.PenaltyAppliedAt = existing.PenaltyAppliedAt
		profile.Notes = existing.Notes
		profile.ComputedAt = existing.ComputedAt
	}
	saved, err := s.repo.UpsertProfile(ctx, profile)
	if err != nil {
		return nil, err
	}
	if err := s.ensureSLAReadiness(ctx, saved); err != nil {
		return nil, err
	}
	return saved, nil
}

// UpdateActuals applies measured metrics and recalculates score, emitting adjustments if thresholds are crossed.
func (s *SLAService) UpdateActuals(ctx context.Context, planType string, actual ActualMetrics) (*opmodels.SLAProfile, error) {
	if s.repo == nil {
		return nil, errors.New("sla repository not configured")
	}
	plan := normalizePlan(planType)
	profile, err := s.repo.GetProfile(ctx, app.PluginID, plan)
	if err != nil {
		return nil, err
	}
	previous := profile.SLAScore
	profile.UptimeActual = clampPercentage(actual.UptimeActual)
	profile.ResponseActualMs = actual.ResponseActualMs
	profile.SuccessActualPct = clampPercentage(actual.SuccessActualPct)
	profile.SupportFrtActualHours = clampHours(actual.SupportFrtActualHours)
	profile.SLAScore = computeScore(profile)
	profile.ComputedAt = time.Now().UTC()

	current := profile.SLAScore
	saved, err := s.repo.UpsertProfile(ctx, profile)
	if err != nil {
		return nil, err
	}
	profile = saved

	if err := s.applyThresholds(ctx, profile, previous, current); err != nil {
		return nil, err
	}
	return profile, nil
}

// ListProfiles returns SLA profiles for the plugin.
func (s *SLAService) ListProfiles(ctx context.Context) ([]*opmodels.SLAProfile, error) {
	if s.repo == nil {
		return nil, errors.New("sla repository not configured")
	}
	return s.repo.ListProfiles(ctx, app.PluginID)
}

// RecomputeScores recalculates SLA scores based on stored actuals.
func (s *SLAService) RecomputeScores(ctx context.Context) ([]*opmodels.SLAProfile, error) {
	profiles, err := s.ListProfiles(ctx)
	if err != nil {
		return nil, err
	}
	for i, profile := range profiles {
		previous := profile.SLAScore
		profile.SLAScore = computeScore(profile)
		profile.ComputedAt = time.Now().UTC()
		saved, err := s.repo.UpsertProfile(ctx, profile)
		if err != nil {
			return nil, err
		}
		profiles[i] = saved
		if err := s.applyThresholds(ctx, saved, previous, saved.SLAScore); err != nil {
			return nil, err
		}
	}
	return profiles, nil
}

// ChecklistSummary returns readiness status for SLA.
func (s *SLAService) ChecklistSummary(ctx context.Context) ([]*opmodels.ReadinessChecklistItem, error) {
	if s.repo == nil {
		return nil, errors.New("sla repository not configured")
	}
	return s.repo.ListReadinessByType(ctx, app.PluginID, string(runtimeops.ChecklistSLAReady))
}

// GetPublicSLA builds response for public transparency endpoint.
func (s *SLAService) GetPublicSLA(ctx context.Context, pluginID string) ([]map[string]any, error) {
	if s.repo == nil {
		return nil, errors.New("sla repository not configured")
	}
	profiles, err := s.repo.ListProfiles(ctx, pluginID)
	if err != nil {
		return nil, err
	}
	sort.Slice(profiles, func(i, j int) bool { return profiles[i].PlanType < profiles[j].PlanType })
	var public []map[string]any
	for _, profile := range profiles {
		public = append(public, map[string]any{
			"pluginId":        profile.PluginID,
			"planType":        profile.PlanType,
			"uptime":          profile.UptimeActual,
			"responseMs":      profile.ResponseActualMs,
			"successRate":     profile.SuccessActualPct,
			"supportFrtHours": profile.SupportFrtActualHours,
			"slaScore":        profile.SLAScore,
			"lastUpdated":     profile.ComputedAt,
		})
	}
	return public, nil
}

func (s *SLAService) applyThresholds(ctx context.Context, profile *opmodels.SLAProfile, previous, current float64) error {
	action := ""
	now := time.Now().UTC()
	if current >= slaScoreIncentiveThreshold && previous < slaScoreIncentiveThreshold {
		profile.IncentiveAppliedAt = &now
		action = "incentive"
	}
	if current < slaScorePenaltyThreshold && previous >= slaScorePenaltyThreshold {
		profile.PenaltyAppliedAt = &now
		action = "penalty"
	}
	if action == "" {
		return nil
	}
	adj := &opmodels.SLAAdjustment{
		PluginID:    profile.PluginID,
		PlanType:    profile.PlanType,
		PeriodStart: time.Now().UTC().AddDate(0, 0, -7),
		PeriodEnd:   time.Now().UTC(),
		ScoreBefore: previous,
		ScoreAfter:  current,
		Action:      action,
	}
	if _, err := s.repo.RecordAdjustment(ctx, adj); err != nil {
		return err
	}
	_, err := s.repo.UpsertProfile(ctx, profile)
	return err
}

func (s *SLAService) ensureSLAReadiness(ctx context.Context, profile *opmodels.SLAProfile) error {
	items, err := s.repo.ListReadinessByType(ctx, profile.PluginID, string(runtimeops.ChecklistSLAReady))
	if err != nil {
		return err
	}
	existing := map[string]*opmodels.ReadinessChecklistItem{}
	for _, item := range items {
		existing[item.ItemKey] = item
	}
	for _, blueprint := range s.readiness[runtimeops.ChecklistSLAReady] {
		item := existing[blueprint.Key]
		if item == nil {
			item = &opmodels.ReadinessChecklistItem{
				PluginID:    profile.PluginID,
				Type:        string(runtimeops.ChecklistSLAReady),
				ItemKey:     blueprint.Key,
				Description: blueprint.Description,
				Status:      runtimeops.ChecklistStatusPending,
				OwnerRole:   blueprint.OwnerRole,
			}
		}
		switch blueprint.Key {
		case "sla_targets_committed":
			if profile.UptimeTarget > 0 && profile.ResponseTargetMs > 0 && profile.SuccessTargetPct > 0 {
				markCompleted(item)
			} else {
				resetPending(item)
			}
		case "sla_sampling_cron_configured":
			if s.cfg != nil && s.cfg.Operations != nil && strings.TrimSpace(s.cfg.Operations.SLA.DailyCron) != "" {
				markCompleted(item)
			} else {
				resetPending(item)
			}
		}
		if _, err := s.repo.UpsertReadinessItem(ctx, item); err != nil {
			return err
		}
	}
	return nil
}

func markCompleted(item *opmodels.ReadinessChecklistItem) {
	now := time.Now().UTC()
	item.Status = runtimeops.ChecklistStatusCompleted
	item.CompletedAt = &now
}

func resetPending(item *opmodels.ReadinessChecklistItem) {
	item.Status = runtimeops.ChecklistStatusPending
	item.CompletedAt = nil
}

func computeScore(profile *opmodels.SLAProfile) float64 {
	supportComponent := 0.0
	if profile.SupportFrtActualHours > 0 {
		ratio := profile.SupportFrtTargetHours / profile.SupportFrtActualHours * 100
		supportComponent = math.Min(100, ratio)
	}
	uptime := clampPercentage(profile.UptimeActual)
	reliability := clampPercentage(profile.SuccessActualPct)
	score := 0.4*uptime + 0.3*supportComponent + 0.3*reliability
	return math.Round(score*100) / 100
}

func normalizePlan(plan string) string {
	return strings.TrimSpace(strings.ToLower(plan))
}

func clampPercentage(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return math.Round(value*100) / 100
}

func clampHours(value float64) float64 {
	if value < 0 {
		return 0
	}
	return math.Round(value*100) / 100
}
