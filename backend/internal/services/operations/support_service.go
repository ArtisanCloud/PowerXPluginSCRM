package operations

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/config"
	opmodels "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/operations"
	oprepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/operations"
	opmetrics "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/observability/operations"
	runtimeops "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/runtime_ops"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
	"gorm.io/datatypes"
)

// WebhookDispatcher dispatches support events to external systems (optional).
type WebhookDispatcher interface {
	DispatchSupportEvent(ctx context.Context, tenantID, eventType string, payload map[string]any) error
}

type noopDispatcher struct{}

func (noopDispatcher) DispatchSupportEvent(ctx context.Context, tenantID, eventType string, payload map[string]any) error {
	return nil
}

// SupportService orchestrates support channel configuration and ticket flows.
type SupportService struct {
	repo       *oprepo.SupportRepository
	cfg        *config.Config
	metrics    *opmetrics.Metrics
	dispatcher WebhookDispatcher
	readiness  runtimeops.ReadinessBlueprint
}

// NewSupportService constructs the service.
func NewSupportService(repo *oprepo.SupportRepository, cfg *config.Config, metrics *opmetrics.Metrics, dispatcher WebhookDispatcher) *SupportService {
	if metrics == nil {
		metrics = opmetrics.NewMetrics()
	}
	if dispatcher == nil {
		dispatcher = noopDispatcher{}
	}
	return &SupportService{
		repo:       repo,
		cfg:        cfg,
		metrics:    metrics,
		dispatcher: dispatcher,
		readiness:  runtimeops.DefaultReadinessBlueprint(),
	}
}

// SupportChannelInput captures desired channel configuration.
type SupportChannelInput struct {
	Channel       string         `json:"channel" binding:"required"`
	Address       string         `json:"address"`
	Escalates     []string       `json:"escalates"`
	ServiceWindow map[string]any `json:"service_window"`
	Metadata      map[string]any `json:"metadata"`
	Enabled       *bool          `json:"enabled"`
}

// KnowledgeBaseDoc describes documentation references.
type KnowledgeBaseDoc struct {
	Label string `json:"label" binding:"required"`
	URL   string `json:"url" binding:"required"`
}

// ConfigurePlaybookInput describes playbook update payloads.
type ConfigurePlaybookInput struct {
	TenantUuid    *string               `json:"tenant_uuid"`
	Channels      []SupportChannelInput `json:"channels"`
	KnowledgeBase []KnowledgeBaseDoc    `json:"knowledge_base"`
}

// SupportPlaybookPayload is returned to clients.
type SupportPlaybookPayload struct {
	Channels      []SupportChannelDTO `json:"channels"`
	KnowledgeBase []KnowledgeBaseDoc  `json:"knowledge_base"`
	Readiness     []ReadinessItemDTO  `json:"readiness"`
}

// SupportChannelDTO for API responses.
type SupportChannelDTO struct {
	ID            string         `json:"id"`
	Channel       string         `json:"channel"`
	Address       string         `json:"address,omitempty"`
	Escalates     []string       `json:"escalates,omitempty"`
	ServiceWindow map[string]any `json:"service_window,omitempty"`
	Metadata      map[string]any `json:"metadata,omitempty"`
	Enabled       bool           `json:"enabled"`
}

// ReadinessItemDTO summarises readiness checklist entries.
type ReadinessItemDTO struct {
	Key       string `json:"key"`
	Status    string `json:"status"`
	Blocking  bool   `json:"blocking"`
	Completed bool   `json:"completed"`
	Notes     string `json:"notes,omitempty"`
}

// SupportMetrics aggregates KPI figures.
type SupportMetrics struct {
	FirstResponseHours float64 `json:"first_response_hours"`
	ResolutionHours    float64 `json:"resolution_hours"`
	CSATAverage        float64 `json:"csat_average"`
	ResolutionRate     float64 `json:"resolution_rate"`
}

// ConfigurePlaybook updates support channels and readiness checklist.
func (s *SupportService) ConfigurePlaybook(ctx context.Context, input ConfigurePlaybookInput) (*SupportPlaybookPayload, error) {
	if s.repo == nil {
		return nil, errors.New("support repository not configured")
	}

	pluginID := app.PluginID
	var tenantID *string
	if input.TenantUuid != nil && strings.TrimSpace(*input.TenantUuid) != "" {
		clean := strings.TrimSpace(*input.TenantUuid)
		tenantID = &clean
	}

	if err := s.repo.DeleteChannels(ctx, pluginID, tenantID); err != nil {
		return nil, err
	}

	for _, ch := range input.Channels {
		metadata := datatypes.JSONMap{}
		for k, v := range ch.Metadata {
			metadata[k] = v
		}
		escalates := datatypes.JSONMap{"levels": ch.Escalates}
		serviceWindow := datatypes.JSONMap{}
		for k, v := range ch.ServiceWindow {
			serviceWindow[k] = v
		}
		sc := &opmodels.SupportChannel{
			PluginID:       pluginID,
			Channel:        strings.TrimSpace(ch.Channel),
			IsEnabled:      true,
			Metadata:       metadata,
			EscalationPath: escalates,
			ServiceWindow:  serviceWindow,
		}
		if tenantID != nil {
			sc.TenantUuid = tenantID
		}
		if ch.Enabled != nil {
			sc.IsEnabled = *ch.Enabled
		}
		if ch.Address != "" {
			sc.Metadata["address"] = ch.Address
		}
		if _, err := s.repo.UpsertChannel(ctx, sc); err != nil {
			return nil, err
		}
	}

	for _, doc := range input.KnowledgeBase {
		sc := &opmodels.SupportChannel{
			PluginID:  pluginID,
			Channel:   "knowledge_base",
			IsEnabled: true,
			Metadata: datatypes.JSONMap{
				"label": doc.Label,
				"url":   doc.URL,
			},
		}
		if tenantID != nil {
			sc.TenantUuid = tenantID
		}
		if _, err := s.repo.UpsertChannel(ctx, sc); err != nil {
			return nil, err
		}
	}

	if err := s.updateReadiness(ctx, pluginID, len(input.Channels) > 0, len(input.KnowledgeBase) > 0); err != nil {
		return nil, err
	}

	payload, err := s.GetPlaybook(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	return payload, nil
}

func (s *SupportService) updateReadiness(ctx context.Context, pluginID string, hasChannels, hasDocs bool) error {
	existing, err := s.repo.ListReadinessByType(ctx, pluginID, string(runtimeops.ChecklistSupportReady))
	if err != nil {
		return err
	}
	byKey := map[string]*opmodels.ReadinessChecklistItem{}
	for _, item := range existing {
		byKey[item.ItemKey] = item
	}

	blueprint := s.readiness[runtimeops.ChecklistSupportReady]
	for _, blueprintItem := range blueprint {
		item := &opmodels.ReadinessChecklistItem{
			PluginID:    pluginID,
			Type:        string(runtimeops.ChecklistSupportReady),
			ItemKey:     blueprintItem.Key,
			Description: blueprintItem.Description,
			OwnerRole:   blueprintItem.OwnerRole,
		}
		if existing := byKey[blueprintItem.Key]; existing != nil {
			item.ID = existing.ID
		}

		switch blueprintItem.Key {
		case "support_channels_configured":
			if hasChannels {
				item.Status = runtimeops.ChecklistStatusCompleted
				completed := time.Now().UTC()
				item.CompletedAt = &completed
			} else {
				item.Status = runtimeops.ChecklistStatusPending
				item.CompletedAt = nil
			}
		case "knowledge_base_published":
			if hasDocs {
				item.Status = runtimeops.ChecklistStatusCompleted
				completed := time.Now().UTC()
				item.CompletedAt = &completed
			} else {
				item.Status = runtimeops.ChecklistStatusPending
				item.CompletedAt = nil
			}
		default:
			item.Status = runtimeops.ChecklistStatusPending
		}

		if _, err := s.repo.UpsertReadinessItem(ctx, item); err != nil {
			return err
		}
	}
	return nil
}

// GetPlaybook returns current playbook configuration for optional tenant scope.
func (s *SupportService) GetPlaybook(ctx context.Context, tenantID *string) (*SupportPlaybookPayload, error) {
	pluginID := app.PluginID
	channels, err := s.repo.ListChannels(ctx, pluginID, tenantID)
	if err != nil {
		return nil, err
	}

	out := &SupportPlaybookPayload{}
	blockingMap := map[string]bool{}
	for _, blueprintItem := range s.readiness[runtimeops.ChecklistSupportReady] {
		blockingMap[blueprintItem.Key] = blueprintItem.Blocking
	}
	for _, ch := range channels {
		if ch.Channel == "knowledge_base" {
			out.KnowledgeBase = append(out.KnowledgeBase, KnowledgeBaseDoc{
				Label: fmt.Sprint(ch.Metadata["label"]),
				URL:   fmt.Sprint(ch.Metadata["url"]),
			})
			continue
		}

		dto := SupportChannelDTO{
			ID:      ch.ID,
			Channel: ch.Channel,
			Enabled: ch.IsEnabled,
		}
		if addr, ok := ch.Metadata["address"].(string); ok {
			dto.Address = addr
		}
		if levels, ok := ch.EscalationPath["levels"].([]any); ok {
			var escalates []string
			for _, v := range levels {
				escalates = append(escalates, fmt.Sprint(v))
			}
			dto.Escalates = escalates
		}
		if len(ch.ServiceWindow) > 0 {
			dto.ServiceWindow = map[string]any(ch.ServiceWindow)
		}
		if len(ch.Metadata) > 0 {
			dto.Metadata = map[string]any(ch.Metadata)
		}
		out.Channels = append(out.Channels, dto)
	}

	items, err := s.repo.ListReadinessByType(ctx, pluginID, string(runtimeops.ChecklistSupportReady))
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		out.Readiness = append(out.Readiness, ReadinessItemDTO{
			Key:       item.ItemKey,
			Status:    item.Status,
			Blocking:  blockingMap[item.ItemKey],
			Completed: strings.EqualFold(item.Status, runtimeops.ChecklistStatusCompleted),
			Notes:     item.Notes,
		})
	}

	return out, nil
}

// CreateTicketRequest describes ticket creation payloads.
type CreateTicketRequest struct {
	TenantUuid  string         `json:"tenant_uuid" binding:"required"`
	ChannelID   *string        `json:"channel_id"`
	Subject     string         `json:"subject" binding:"required"`
	Description string         `json:"description"`
	Priority    string         `json:"priority" binding:"required"`
	RequestedBy map[string]any `json:"requested_by"`
}

// CreateTicket creates a support ticket and emits events.
func (s *SupportService) CreateTicket(ctx context.Context, req CreateTicketRequest) (*opmodels.SupportTicket, error) {
	if strings.TrimSpace(req.TenantUuid) == "" {
		return nil, errors.New("tenant_uuid is required")
	}
	if strings.TrimSpace(req.Subject) == "" {
		return nil, errors.New("subject is required")
	}

	ticket := &opmodels.SupportTicket{
		PluginID:    app.PluginID,
		TenantUuid:  strings.TrimSpace(req.TenantUuid),
		ChannelID:   req.ChannelID,
		Subject:     req.Subject,
		Description: req.Description,
		Priority:    strings.ToUpper(req.Priority),
		Status:      "created",
		RequestedBy: datatypes.JSONMap(req.RequestedBy),
	}

	saved, err := s.repo.CreateTicket(ctx, ticket)
	if err != nil {
		return nil, err
	}

	eventPayload := map[string]any{
		"ticket_id": saved.ID,
		"status":    saved.Status,
		"priority":  saved.Priority,
	}
	if _, err := s.repo.AppendEvent(ctx, &opmodels.SupportTicketEvent{
		TicketID:  saved.ID,
		EventType: "ticket.created",
		Payload:   datatypes.JSONMap(eventPayload),
	}); err != nil {
		return nil, err
	}

	s.metrics.RecordSupportTicket(saved.Status, saved.Priority)
	_ = s.dispatcher.DispatchSupportEvent(ctx, saved.TenantUuid, "operations.support.ticket.created", eventPayload)
	return saved, nil
}

// ComputeMetrics aggregates support KPIs by scanning ticket history.
func (s *SupportService) ComputeMetrics(ctx context.Context) (*SupportMetrics, error) {
	tickets, err := s.repo.ListTickets(ctx, app.PluginID)
	if err != nil {
		return nil, err
	}

	var frtTotal, frtCount, mttrTotal, mttrCount float64
	var csatTotal, csatCount float64
	var resolved, total int
	for _, ticket := range tickets {
		total++
		if ticket.FirstResponseAt != nil {
			delta := ticket.FirstResponseAt.Sub(ticket.CreatedAt).Hours()
			frtTotal += delta
			frtCount++
		}
		if ticket.ResolvedAt != nil {
			delta := ticket.ResolvedAt.Sub(ticket.CreatedAt).Hours()
			mttrTotal += delta
			mttrCount++
			resolved++
		}
		if ticket.CSATScore != nil {
			csatTotal += *ticket.CSATScore
			csatCount++
		}
	}

	metrics := &SupportMetrics{}
	if frtCount > 0 {
		metrics.FirstResponseHours = frtTotal / frtCount
	}
	if mttrCount > 0 {
		metrics.ResolutionHours = mttrTotal / mttrCount
	}
	if csatCount > 0 {
		metrics.CSATAverage = csatTotal / csatCount
	}
	if total > 0 {
		metrics.ResolutionRate = float64(resolved) / float64(total)
	}
	return metrics, nil
}

// RegisterSupportRoutes wires HTTP handlers for support endpoints.
