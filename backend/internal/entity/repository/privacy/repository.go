package privacy

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	model "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/privacy"
	repository "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Repository aggregates persistence helpers for privacy-related models.
type Repository struct {
	db              *gorm.DB
	classifications *repository.BaseRepository[model.DataClassification]
	tokens          *repository.BaseRepository[model.ConsentToken]
	events          *repository.BaseRepository[model.LifecycleEvent]
}

// NewRepository constructs a privacy repository backed by the provided DB handle.
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{
		db:              db,
		classifications: repository.NewBaseRepository[model.DataClassification](db),
		tokens:          repository.NewBaseRepository[model.ConsentToken](db),
		events:          repository.NewBaseRepository[model.LifecycleEvent](db),
	}
}

// WithTx clones the repository with a transactional DB handle.
func (r *Repository) WithTx(tx *gorm.DB) *Repository {
	if tx == nil {
		return r
	}
	return &Repository{
		db:              tx,
		classifications: repository.NewBaseRepository[model.DataClassification](tx),
		tokens:          repository.NewBaseRepository[model.ConsentToken](tx),
		events:          repository.NewBaseRepository[model.LifecycleEvent](tx),
	}
}

// UpsertClassification creates or updates a data classification keyed by tenant + asset.
func (r *Repository) UpsertClassification(ctx context.Context, record *model.DataClassification) (*model.DataClassification, error) {
	if record == nil {
		return nil, gorm.ErrInvalidData
	}
	if record.TenantUuid == "" || record.AssetKey == "" {
		return nil, errors.New("tenant_uuid and asset_key are required")
	}
	return r.classifications.Upsert(ctx, record, []clause.Column{
		{Name: "tenant_uuid"},
		{Name: "asset_key"},
	})
}

// ListClassifications returns all data classifications for a tenant ordered by asset key.
func (r *Repository) ListClassifications(ctx context.Context, tenantID string) ([]*model.DataClassification, error) {
	var results []*model.DataClassification
	if err := r.db.WithContext(ctx).
		Model(&model.DataClassification{}).
		Where("tenant_uuid = ?", tenantID).
		Order("asset_key ASC").
		Find(&results).Error; err != nil {
		return nil, err
	}
	return results, nil
}

// DeleteClassification removes a classification mapping for the given tenant asset.
func (r *Repository) DeleteClassification(ctx context.Context, tenantID, assetKey string) error {
	return r.db.WithContext(ctx).
		Where("tenant_uuid = ? AND asset_key = ?", tenantID, assetKey).
		Delete(&model.DataClassification{}).Error
}

// IssueConsentToken stores a consent token with the provided scope payload.
func (r *Repository) IssueConsentToken(ctx context.Context, token *model.ConsentToken, scope []string) (*model.ConsentToken, error) {
	if token == nil {
		return nil, gorm.ErrInvalidData
	}
	if token.TenantUuid == "" || token.Token == "" {
		return nil, errors.New("tenant_uuid and consent_token are required")
	}
	if len(scope) > 0 {
		blob, err := json.Marshal(scope)
		if err != nil {
			return nil, err
		}
		token.Scope = datatypes.JSON(blob)
	}
	if token.Status == "" {
		token.Status = model.ConsentStatusActive
	}
	if token.IssuedAt.IsZero() {
		token.IssuedAt = time.Now().UTC()
	}
	return r.tokens.Create(ctx, token)
}

// GetConsentToken retrieves a consent token by ID scoped to a tenant.
func (r *Repository) GetConsentToken(ctx context.Context, tenantID, tokenID string) (*model.ConsentToken, error) {
	var record model.ConsentToken
	err := r.db.WithContext(ctx).
		Where("tenant_uuid = ? AND id = ?", tenantID, tokenID).
		Take(&record).Error
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// ListConsentTokensByStatus returns consent tokens for a tenant filtered by status list.
func (r *Repository) ListConsentTokensByStatus(ctx context.Context, tenantID string, statuses ...string) ([]*model.ConsentToken, error) {
	query := r.db.WithContext(ctx).
		Model(&model.ConsentToken{}).
		Where("tenant_uuid = ?", tenantID)
	if len(statuses) > 0 {
		query = query.Where("status IN ?", statuses)
	}
	query = query.Order("issued_at DESC")

	var tokens []*model.ConsentToken
	if err := query.Find(&tokens).Error; err != nil {
		return nil, err
	}
	return tokens, nil
}

// ActiveConsentTokens returns non-expired active consent tokens for a tenant.
func (r *Repository) ActiveConsentTokens(ctx context.Context, tenantID string, now time.Time) ([]*model.ConsentToken, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	var tokens []*model.ConsentToken
	if err := r.db.WithContext(ctx).
		Where("tenant_uuid = ? AND status = ? AND (expires_at IS NULL OR expires_at > ?)", tenantID, model.ConsentStatusActive, now).
		Order("issued_at DESC").
		Find(&tokens).Error; err != nil {
		return nil, err
	}
	return tokens, nil
}

// RevokeConsentToken marks a consent token as revoked with reason metadata.
func (r *Repository) RevokeConsentToken(ctx context.Context, tenantID, tokenID, reason string) error {
	updates := map[string]interface{}{
		"status":         model.ConsentStatusRevoked,
		"revoked_at":     time.Now().UTC(),
		"revoked_reason": reason,
	}
	return r.db.WithContext(ctx).
		Model(&model.ConsentToken{}).
		Where("tenant_uuid = ? AND id = ?", tenantID, tokenID).
		Updates(updates).Error
}

// CreateLifecycleEvent appends a lifecycle event row and returns it.
func (r *Repository) CreateLifecycleEvent(ctx context.Context, event *model.LifecycleEvent) (*model.LifecycleEvent, error) {
	if event == nil {
		return nil, gorm.ErrInvalidData
	}
	if event.TenantUuid == "" || event.AssetKey == "" {
		return nil, errors.New("tenant_uuid and asset_key are required")
	}
	if event.EventType == "" {
		return nil, errors.New("event_type is required")
	}
	if event.OccurredAt.IsZero() {
		event.OccurredAt = time.Now().UTC()
	}
	return r.events.Create(ctx, event)
}

// ListLifecycleEvents fetches lifecycle events filtered by type (optional) and limited by count.
func (r *Repository) ListLifecycleEvents(ctx context.Context, tenantID string, eventTypes []string, limit int) ([]*model.LifecycleEvent, error) {
	query := r.db.WithContext(ctx).
		Model(&model.LifecycleEvent{}).
		Where("tenant_uuid = ?", tenantID)
	if len(eventTypes) > 0 {
		query = query.Where("event_type IN ?", eventTypes)
	}
	if limit > 0 {
		query = query.Limit(limit)
	}
	query = query.Order("occurred_at DESC")

	var events []*model.LifecycleEvent
	if err := query.Find(&events).Error; err != nil {
		return nil, err
	}
	return events, nil
}

// UpdateLifecycleEventStatus updates the status (and optional payload) of a lifecycle event.
func (r *Repository) UpdateLifecycleEventStatus(ctx context.Context, tenantID, eventID, status string, payload datatypes.JSON) error {
	if status == "" {
		return errors.New("status is required")
	}
	updates := map[string]interface{}{
		"status": status,
	}
	if len(payload) > 0 {
		updates["payload"] = payload
	}
	return r.db.WithContext(ctx).
		Model(&model.LifecycleEvent{}).
		Where("tenant_uuid = ? AND id = ?", tenantID, eventID).
		Updates(updates).Error
}
