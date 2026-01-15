package console

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/config"
	model "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/admin_console"
	consolerepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/admin_console"
	adminmetrics "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/observability/admin_console"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
)

// AuditService coordinates audit history queries and exports.
type AuditService struct {
	cfg     *config.Config
	repo    *consolerepo.AuditRepository
	metrics *adminmetrics.Metrics
}

// NewAuditService constructs AuditService with shared dependencies.
func NewAuditService(deps *app.Deps) *AuditService {
	if deps == nil || deps.DB == nil {
		return &AuditService{}
	}
	metrics := deps.AdminConsoleMetrics
	if metrics == nil {
		metrics = adminmetrics.NewMetrics()
	}
	return &AuditService{
		cfg:     deps.Config,
		repo:    consolerepo.NewAuditRepository(deps.DB),
		metrics: metrics,
	}
}

// AuditActor captures actor metadata.
type AuditActor struct {
	ID    string `json:"id"`
	Name  string `json:"name,omitempty"`
	Email string `json:"email,omitempty"`
}

// AuditEventDTO represents a normalized audit event.
type AuditEventDTO struct {
	ID             string         `json:"id"`
	Action         string         `json:"action"`
	PermissionCode string         `json:"permission_code"`
	ResourceType   string         `json:"resource_type"`
	ResourceRef    string         `json:"resource_ref,omitempty"`
	Summary        string         `json:"summary,omitempty"`
	Diff           map[string]any `json:"diff,omitempty"`
	OccurredAt     time.Time      `json:"occurred_at"`
	Actor          AuditActor     `json:"actor"`
}

// ListAuditInput defines filters for audit history queries.
type ListAuditInput struct {
	TenantUuid     *string
	ActorID        string
	Action         string
	PermissionCode string
	OccurredAfter  *time.Time
	OccurredBefore *time.Time
	Cursor         string
	Limit          int
}

// ListAuditResult groups event payload with pagination cursor.
type ListAuditResult struct {
	Events     []AuditEventDTO `json:"events"`
	NextCursor string          `json:"next_cursor,omitempty"`
}

// ExportAuditInput defines audit export filters.
type ExportAuditInput struct {
	TenantUuid     *string
	ActorID        string
	Action         string
	PermissionCode string
	OccurredAfter  *time.Time
	OccurredBefore *time.Time
	Format         string
}

// ExportAuditResult contains exported content metadata.
type ExportAuditResult struct {
	Filename    string
	ContentType string
	Content     []byte
}

// ListEvents returns paginated audit entries.
func (s *AuditService) ListEvents(ctx context.Context, input ListAuditInput) (*ListAuditResult, error) {
	if s == nil || s.repo == nil {
		return nil, ErrServiceUnavailable
	}
	opts := consolerepo.ListOptions{
		PluginID:       app.PluginID,
		TenantUuid:     input.TenantUuid,
		ActorID:        input.ActorID,
		Action:         input.Action,
		PermissionCode: input.PermissionCode,
		OccurredAfter:  input.OccurredAfter,
		OccurredBefore: input.OccurredBefore,
		Cursor:         input.Cursor,
		Limit:          input.Limit,
	}
	events, nextCursor, err := s.repo.ListEvents(ctx, opts)
	if err != nil {
		return nil, err
	}
	dto := make([]AuditEventDTO, len(events))
	for i, evt := range events {
		dto[i] = mapAuditEvent(evt)
	}
	return &ListAuditResult{Events: dto, NextCursor: nextCursor}, nil
}

// ExportEvents returns audit entries in requested format.
func (s *AuditService) ExportEvents(ctx context.Context, input ExportAuditInput) (*ExportAuditResult, error) {
	if s == nil || s.repo == nil {
		return nil, ErrServiceUnavailable
	}
	format := strings.ToLower(strings.TrimSpace(input.Format))
	if format == "" {
		format = s.cfg.AdminConsoleExportFormat()
	}
	if format != "csv" && format != "json" {
		return nil, validationError{Field: "format", Message: "must be csv or json"}
	}
	opts := consolerepo.ExportOptions{
		PluginID:       app.PluginID,
		TenantUuid:     input.TenantUuid,
		ActorID:        input.ActorID,
		Action:         input.Action,
		PermissionCode: input.PermissionCode,
		OccurredAfter:  input.OccurredAfter,
		OccurredBefore: input.OccurredBefore,
		Format:         format,
	}
	events, err := s.repo.ExportEvents(ctx, opts)
	if err != nil {
		return nil, err
	}
	payload := make([]AuditEventDTO, len(events))
	for i, evt := range events {
		payload[i] = mapAuditEvent(evt)
	}

	var content []byte
	var filename string
	var contentType string
	switch format {
	case "json":
		content, err = json.MarshalIndent(payload, "", "  ")
		if err != nil {
			return nil, err
		}
		filename = fmt.Sprintf("audit-events-%s.json", time.Now().UTC().Format("20060102-150405"))
		contentType = "application/json"
	case "csv":
		buf := &bytes.Buffer{}
		writer := csv.NewWriter(buf)
		_ = writer.Write([]string{"occurred_at", "actor_id", "actor_name", "actor_email", "permission_code", "action", "resource_type", "resource_ref", "summary"})
		for _, evt := range payload {
			data := []string{
				evt.OccurredAt.UTC().Format(time.RFC3339),
				evt.Actor.ID,
				evt.Actor.Name,
				evt.Actor.Email,
				evt.PermissionCode,
				evt.Action,
				evt.ResourceType,
				evt.ResourceRef,
				evt.Summary,
			}
			_ = writer.Write(data)
		}
		writer.Flush()
		if err := writer.Error(); err != nil {
			return nil, err
		}
		content = buf.Bytes()
		filename = fmt.Sprintf("audit-events-%s.csv", time.Now().UTC().Format("20060102-150405"))
		contentType = "text/csv"
	default:
		return nil, validationError{Field: "format", Message: "unsupported format"}
	}
	if s.metrics != nil {
		s.metrics.RecordAuditExport(format)
	}
	return &ExportAuditResult{
		Filename:    filename,
		ContentType: contentType,
		Content:     content,
	}, nil
}

func mapAuditEvent(evt model.AuditEvent) AuditEventDTO {
	dto := AuditEventDTO{
		ID:             evt.ID,
		Action:         evt.Action,
		PermissionCode: evt.PermissionCode,
		ResourceType:   evt.ResourceType,
		OccurredAt:     evt.OccurredAt,
		Actor: AuditActor{
			ID:    evt.ActorID,
			Name:  deref(evt.ActorName),
			Email: deref(evt.ActorEmail),
		},
	}
	if evt.ResourceRef != nil {
		dto.ResourceRef = *evt.ResourceRef
	}
	if evt.Summary != nil {
		dto.Summary = *evt.Summary
	}
	if len(evt.Diff) > 0 {
		if diff := jsonToMap(evt.Diff); len(diff) > 0 {
			dto.Diff = diff
		}
	}
	return dto
}

func deref(ptr *string) string {
	if ptr == nil {
		return ""
	}
	return *ptr
}

func jsonToMap(data []byte) map[string]any {
	if len(data) == 0 {
		return nil
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		return nil
	}
	return out
}
