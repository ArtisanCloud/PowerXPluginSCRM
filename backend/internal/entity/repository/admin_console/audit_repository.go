package admin_console

import (
	"context"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"

	model "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/admin_console"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository"
	"gorm.io/gorm"
)

// AuditRepository persists admin console audit events.
type AuditRepository struct {
	*repository.BaseRepository[model.AuditEvent]
}

// NewAuditRepository constructs the audit repository.
func NewAuditRepository(db *gorm.DB) *AuditRepository {
	return &AuditRepository{BaseRepository: repository.NewBaseRepository[model.AuditEvent](db)}
}

// Create inserts a new audit event record.
func (r *AuditRepository) Create(ctx context.Context, evt *model.AuditEvent) error {
	return r.DB.WithContext(ctx).Create(evt).Error
}

// ListOptions encapsulates filters for audit event queries.
type ListOptions struct {
	PluginID       string
	TenantUuid     *string
	ActorID        string
	Action         string
	PermissionCode string
	OccurredAfter  *time.Time
	OccurredBefore *time.Time
	Cursor         string
	Limit          int
}

// ExportOptions defines export query boundaries.
type ExportOptions struct {
	PluginID       string
	TenantUuid     *string
	ActorID        string
	Action         string
	PermissionCode string
	OccurredAfter  *time.Time
	OccurredBefore *time.Time
	Format         string
}

// ListEvents returns audit events with cursor pagination.
func (r *AuditRepository) ListEvents(ctx context.Context, opts ListOptions) ([]model.AuditEvent, string, error) {
	if r.DB == nil {
		return nil, "", gorm.ErrInvalidDB
	}
	if opts.PluginID == "" {
		return nil, "", fmt.Errorf("plugin id is required")
	}
	limit := opts.Limit
	if limit <= 0 || limit > 100 {
		limit = 25
	}

	query := r.baseQuery(ctx, opts.PluginID, opts.TenantUuid, opts.ActorID, opts.Action, opts.PermissionCode, opts.OccurredAfter, opts.OccurredBefore)
	if opts.Cursor != "" {
		ts, id, err := decodeCursor(opts.Cursor)
		if err != nil {
			return nil, "", err
		}
		query = query.Where("(occurred_at < ?) OR (occurred_at = ? AND id < ?)", ts, ts, id)
	}

	var events []model.AuditEvent
	if err := query.Order("occurred_at DESC").Order("id DESC").Limit(limit + 1).Find(&events).Error; err != nil {
		return nil, "", err
	}

	var nextCursor string
	if len(events) > limit {
		last := events[limit]
		nextCursor = encodeCursor(last.OccurredAt, last.ID)
		events = events[:limit]
	}
	return events, nextCursor, nil
}

// ExportEvents returns audit events satisfying export filters ordered ascending by time.
func (r *AuditRepository) ExportEvents(ctx context.Context, opts ExportOptions) ([]model.AuditEvent, error) {
	if r.DB == nil {
		return nil, gorm.ErrInvalidDB
	}
	if opts.PluginID == "" {
		return nil, fmt.Errorf("plugin id is required")
	}
	query := r.baseQuery(ctx, opts.PluginID, opts.TenantUuid, opts.ActorID, opts.Action, opts.PermissionCode, opts.OccurredAfter, opts.OccurredBefore)
	query = query.Order("occurred_at ASC").Order("id ASC")
	var events []model.AuditEvent
	if err := query.Find(&events).Error; err != nil {
		return nil, err
	}
	return events, nil
}

// LatestForAction returns the latest audit entry for a resource reference.
func (r *AuditRepository) LatestForAction(ctx context.Context, pluginID string, tenantID *string, resourceType, resourceRef string) (*model.AuditEvent, error) {
	if r.DB == nil {
		return nil, gorm.ErrInvalidDB
	}
	query := r.DB.WithContext(ctx).
		Where("plugin_id = ? AND resource_type = ?", pluginID, resourceType).
		Order("occurred_at DESC")
	if resourceRef != "" {
		query = query.Where("resource_ref = ?", resourceRef)
	}
	if tenantID == nil {
		query = query.Where("tenant_uuid IS NULL")
	} else {
		query = query.Where("tenant_uuid = ?", *tenantID)
	}
	var evt model.AuditEvent
	if err := query.First(&evt).Error; err != nil {
		return nil, err
	}
	return &evt, nil
}

func (r *AuditRepository) baseQuery(ctx context.Context, pluginID string, tenantID *string, actorID, action, permission string, after, before *time.Time) *gorm.DB {
	query := r.DB.WithContext(ctx).Where("plugin_id = ?", pluginID)
	if tenantID == nil {
		query = query.Where("tenant_uuid IS NULL")
	} else {
		query = query.Where("tenant_uuid = ?", *tenantID)
	}
	if strings.TrimSpace(actorID) != "" {
		query = query.Where("actor_id = ?", strings.TrimSpace(actorID))
	}
	if strings.TrimSpace(action) != "" {
		query = query.Where("action = ?", strings.TrimSpace(action))
	}
	if strings.TrimSpace(permission) != "" {
		query = query.Where("permission_code = ?", strings.TrimSpace(permission))
	}
	if after != nil {
		query = query.Where("occurred_at >= ?", after.UTC())
	}
	if before != nil {
		query = query.Where("occurred_at <= ?", before.UTC())
	}
	return query
}

func encodeCursor(ts time.Time, id string) string {
	payload := fmt.Sprintf("%d|%s", ts.UTC().UnixNano(), id)
	return base64.URLEncoding.EncodeToString([]byte(payload))
}

func decodeCursor(cursor string) (time.Time, string, error) {
	raw, err := base64.URLEncoding.DecodeString(cursor)
	if err != nil {
		return time.Time{}, "", fmt.Errorf("invalid cursor")
	}
	parts := strings.SplitN(string(raw), "|", 2)
	if len(parts) != 2 {
		return time.Time{}, "", fmt.Errorf("invalid cursor format")
	}
	nanos, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return time.Time{}, "", fmt.Errorf("invalid cursor timestamp")
	}
	return time.Unix(0, nanos).UTC(), parts[1], nil
}
