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

// JobRunRepository persists admin console job run records.
type JobRunRepository struct {
	*repository.BaseRepository[model.JobRun]
}

// NewJobRunRepository constructs a job run repository backed by GORM.
func NewJobRunRepository(db *gorm.DB) *JobRunRepository {
	return &JobRunRepository{
		BaseRepository: repository.NewBaseRepository[model.JobRun](db),
	}
}

// Create inserts a new job run row.
func (r *JobRunRepository) Create(ctx context.Context, run *model.JobRun) error {
	if run == nil {
		return fmt.Errorf("job run cannot be nil")
	}
	now := time.Now().UTC()
	if run.CreatedAt.IsZero() {
		run.CreatedAt = now
	}
	if run.UpdatedAt.IsZero() {
		run.UpdatedAt = now
	}
	return r.DB.WithContext(ctx).Create(run).Error
}

// UpdateStatus applies status/message timestamps to a run.
func (r *JobRunRepository) UpdateStatus(ctx context.Context, runID string, status string, message *string, startedAt, finishedAt *time.Time) (*model.JobRun, error) {
	if runID == "" {
		return nil, fmt.Errorf("run id required")
	}
	updates := map[string]any{
		"status":     strings.ToLower(strings.TrimSpace(status)),
		"updated_at": time.Now().UTC(),
	}
	if message != nil {
		updates["message"] = *message
	}
	if startedAt != nil {
		updates["started_at"] = startedAt.UTC()
	}
	if finishedAt != nil {
		updates["finished_at"] = finishedAt.UTC()
	}
	res := r.DB.WithContext(ctx).
		Model(&model.JobRun{}).
		Where("id = ?", runID).
		Updates(updates)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	return r.GetByID(ctx, runID)
}

// GetByID fetches a job run by identifier.
func (r *JobRunRepository) GetByID(ctx context.Context, runID string) (*model.JobRun, error) {
	if runID == "" {
		return nil, fmt.Errorf("run id required")
	}
	var run model.JobRun
	if err := r.DB.WithContext(ctx).Where("id = ?", runID).First(&run).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &run, nil
}

// JobRunListOptions describes filters for listing job runs.
type JobRunListOptions struct {
	PluginID   string
	TenantUuid *string
	JobType    string
	Status     string
	Cursor     string
	Limit      int
}

// List returns job runs ordered by created_at desc with cursor pagination.
func (r *JobRunRepository) List(ctx context.Context, opts JobRunListOptions) ([]model.JobRun, string, error) {
	if opts.PluginID == "" {
		return nil, "", fmt.Errorf("plugin id required")
	}
	limit := opts.Limit
	if limit <= 0 || limit > 100 {
		limit = 25
	}
	query := r.DB.WithContext(ctx).
		Where("plugin_id = ?", opts.PluginID)
	if opts.TenantUuid != nil && strings.TrimSpace(*opts.TenantUuid) != "" {
		query = query.Where("tenant_uuid = ?", strings.TrimSpace(*opts.TenantUuid))
	}
	if strings.TrimSpace(opts.JobType) != "" {
		query = query.Where("job_type = ?", strings.TrimSpace(opts.JobType))
	}
	if strings.TrimSpace(opts.Status) != "" {
		query = query.Where("status = ?", strings.TrimSpace(opts.Status))
	}
	if opts.Cursor != "" {
		createdAt, id, err := decodeJobRunCursor(opts.Cursor)
		if err != nil {
			return nil, "", err
		}
		query = query.Where("(created_at < ?) OR (created_at = ? AND id < ?)", createdAt, createdAt, id)
	}
	var runs []model.JobRun
	if err := query.
		Order("created_at DESC").
		Order("id DESC").
		Limit(limit + 1).
		Find(&runs).Error; err != nil {
		return nil, "", err
	}
	var next string
	if len(runs) > limit {
		last := runs[limit]
		next = encodeJobRunCursor(last.CreatedAt, last.ID)
		runs = runs[:limit]
	}
	return runs, next, nil
}

// HasActiveRun checks if a safe-op with same scope/action is already executing.
func (r *JobRunRepository) HasActiveRun(ctx context.Context, pluginID string, tenantID *string, action string, scopeType string, scopeRef string) (bool, error) {
	if pluginID == "" || action == "" || scopeRef == "" {
		return false, fmt.Errorf("plugin, action, scope_ref required")
	}
	query := r.DB.WithContext(ctx).
		Model(&model.JobRun{}).
		Where("plugin_id = ? AND action = ? AND scope_ref = ? AND status IN ?", pluginID, action, scopeRef, []string{"pending", "running"})
	if scopeType != "" {
		query = query.Where("scope_type = ?", scopeType)
	}
	if tenantID != nil && strings.TrimSpace(*tenantID) != "" {
		query = query.Where("tenant_uuid = ?", strings.TrimSpace(*tenantID))
	}
	var count int64
	if err := query.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// AttachAuditEvent links a job run to an audit event.
func (r *JobRunRepository) AttachAuditEvent(ctx context.Context, runID string, auditEventID string) error {
	if runID == "" || auditEventID == "" {
		return fmt.Errorf("run id and audit event id required")
	}
	return r.DB.WithContext(ctx).
		Model(&model.JobRun{}).
		Where("id = ?", runID).
		Updates(map[string]any{
			"audit_event_id": auditEventID,
			"updated_at":     time.Now().UTC(),
		}).Error
}

func encodeJobRunCursor(ts time.Time, id string) string {
	payload := fmt.Sprintf("%d|%s", ts.UTC().UnixNano(), id)
	return base64.URLEncoding.EncodeToString([]byte(payload))
}

func decodeJobRunCursor(cursor string) (time.Time, string, error) {
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
