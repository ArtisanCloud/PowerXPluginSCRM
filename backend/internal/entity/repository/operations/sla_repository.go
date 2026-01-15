package operations

import (
	"context"
	"errors"
	"time"

	opmodels "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/operations"
	repository "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SLARepository manages persistence for SLA profiles and adjustments.
type SLARepository struct {
	db          *gorm.DB
	profileRepo *repository.BaseRepository[opmodels.SLAProfile]
	adjustRepo  *repository.BaseRepository[opmodels.SLAAdjustment]
}

// NewSLARepository constructs a new SLARepository.
func NewSLARepository(db *gorm.DB) *SLARepository {
	return &SLARepository{
		db:          db,
		profileRepo: repository.NewBaseRepository[opmodels.SLAProfile](db),
		adjustRepo:  repository.NewBaseRepository[opmodels.SLAAdjustment](db),
	}
}

// UpsertProfile creates or updates an SLA profile keyed by plugin and plan type.
func (r *SLARepository) UpsertProfile(ctx context.Context, profile *opmodels.SLAProfile) (*opmodels.SLAProfile, error) {
	if profile == nil {
		return nil, errors.New("sla profile is required")
	}
	now := time.Now().UTC()
	existing, err := r.GetProfile(ctx, profile.PluginID, profile.PlanType)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		if profile.ID == "" {
			profile.ID = uuid.NewString()
		}
		if profile.CreatedAt.IsZero() {
			profile.CreatedAt = now
		}
		if profile.ComputedAt.IsZero() {
			profile.ComputedAt = now
		}
		profile.UpdatedAt = now
		if _, err := r.profileRepo.Create(ctx, profile); err != nil {
			return nil, err
		}
		return profile, nil
	}

	profile.ID = existing.ID
	profile.CreatedAt = existing.CreatedAt
	if profile.ComputedAt.IsZero() {
		profile.ComputedAt = now
	}
	profile.UpdatedAt = now

	if err := r.db.WithContext(ctx).Model(&opmodels.SLAProfile{}).
		Where("plugin_id = ? AND plan_type = ?", profile.PluginID, profile.PlanType).
		Updates(map[string]any{
			"uptime_target":            profile.UptimeTarget,
			"uptime_actual":            profile.UptimeActual,
			"response_target_ms":       profile.ResponseTargetMs,
			"response_actual_ms":       profile.ResponseActualMs,
			"success_target_pct":       profile.SuccessTargetPct,
			"success_actual_pct":       profile.SuccessActualPct,
			"support_frt_target_hours": profile.SupportFrtTargetHours,
			"support_frt_actual_hours": profile.SupportFrtActualHours,
			"sla_score":                profile.SLAScore,
			"incentive_applied_at":     profile.IncentiveAppliedAt,
			"penalty_applied_at":       profile.PenaltyAppliedAt,
			"notes":                    profile.Notes,
			"computed_at":              profile.ComputedAt,
			"updated_at":               profile.UpdatedAt,
		}).Error; err != nil {
		return nil, err
	}
	return profile, nil
}

// GetProfile fetches an SLA profile by plugin and plan type.
func (r *SLARepository) GetProfile(ctx context.Context, pluginID, planType string) (*opmodels.SLAProfile, error) {
	var profile opmodels.SLAProfile
	if err := r.db.WithContext(ctx).
		Where("plugin_id = ? AND plan_type = ?", pluginID, planType).
		First(&profile).Error; err != nil {
		return nil, err
	}
	return &profile, nil
}

// ListProfiles lists all SLA profiles for a plugin.
func (r *SLARepository) ListProfiles(ctx context.Context, pluginID string) ([]*opmodels.SLAProfile, error) {
	var profiles []*opmodels.SLAProfile
	if err := r.db.WithContext(ctx).
		Where("plugin_id = ?", pluginID).
		Order("plan_type").
		Find(&profiles).Error; err != nil {
		return nil, err
	}
	return profiles, nil
}

// RecordAdjustment inserts an SLA adjustment history entry.
func (r *SLARepository) RecordAdjustment(ctx context.Context, adjustment *opmodels.SLAAdjustment) (*opmodels.SLAAdjustment, error) {
	if adjustment == nil {
		return nil, errors.New("adjustment is required")
	}
	adjustment.CreatedAt = time.Now().UTC()
	if _, err := r.adjustRepo.Create(ctx, adjustment); err != nil {
		return nil, err
	}
	return adjustment, nil
}

// ListAdjustments returns adjustment history ordered by period descending.

// UpsertReadinessItem updates or creates SLA readiness checklist entries.
func (r *SLARepository) UpsertReadinessItem(ctx context.Context, item *opmodels.ReadinessChecklistItem) (*opmodels.ReadinessChecklistItem, error) {
	if item == nil {
		return nil, errors.New("readiness item is required")
	}
	now := time.Now().UTC()
	if item.ID == "" {
		item.ID = uuid.NewString()
	}
	if item.CreatedAt.IsZero() {
		item.CreatedAt = now
	}
	item.UpdatedAt = now
	if err := r.db.WithContext(ctx).Save(item).Error; err != nil {
		return nil, err
	}
	return item, nil
}

// ListReadinessByType returns readiness items by checklist type.
func (r *SLARepository) ListReadinessByType(ctx context.Context, pluginID, checklistType string) ([]*opmodels.ReadinessChecklistItem, error) {
	var items []*opmodels.ReadinessChecklistItem
	if err := r.db.WithContext(ctx).
		Where("plugin_id = ? AND type = ?", pluginID, checklistType).
		Order("item_key").
		Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (r *SLARepository) ListAdjustments(ctx context.Context, pluginID, planType string, limit int) ([]*opmodels.SLAAdjustment, error) {
	query := r.db.WithContext(ctx).
		Where("plugin_id = ? AND plan_type = ?", pluginID, planType).
		Order("period_start DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	var adjustments []*opmodels.SLAAdjustment
	if err := query.Find(&adjustments).Error; err != nil {
		return nil, err
	}
	return adjustments, nil
}
