package lead_capture

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	leadmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/lead_capture"
	leadrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/lead_capture"
	leadobs "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/observability/lead_capture"
)

type ConversationWebhookInput struct {
	TenantUUID         string
	ChannelAccountUUID string
	ExternalEventID    string
	ConversationID     string
	ActorType          string
	ActorID            string
	Direction          string
	MessageType        string
	ContentText        string
	OccurredAt         time.Time
	RawPayload         map[string]any
}

type ConversationSummary struct {
	ConversationID  string `json:"conversation_id"`
	LatestMessage   string `json:"latest_message"`
	LatestActorType string `json:"latest_actor_type"`
	LatestAt        string `json:"latest_at"`
	UnreadCount     int    `json:"unread_count"`
}

type ConversationService struct {
	eventRepo      *leadrepo.ConversationEventRepository
	bindingRepo    *leadrepo.LeadConversationBindingRepository
	pendingRepo    *leadrepo.LeadConversationPendingRepository
	projectionRepo *leadrepo.LeadRealtimeProjectionRepository
	leadRepo       *leadrepo.LeadRepository
	realtime       *ConversationRealtimePublisher
	metrics        *leadobs.Metrics
}

func NewConversationService(
	eventRepo *leadrepo.ConversationEventRepository,
	bindingRepo *leadrepo.LeadConversationBindingRepository,
	pendingRepo *leadrepo.LeadConversationPendingRepository,
	projectionRepo *leadrepo.LeadRealtimeProjectionRepository,
	realtime *ConversationRealtimePublisher,
	metrics *leadobs.Metrics,
) *ConversationService {
	return &ConversationService{
		eventRepo:      eventRepo,
		bindingRepo:    bindingRepo,
		pendingRepo:    pendingRepo,
		projectionRepo: projectionRepo,
		realtime:       realtime,
		metrics:        metrics,
	}
}

func (s *ConversationService) WithLeadRepository(leadRepo *leadrepo.LeadRepository) *ConversationService {
	if s == nil {
		return s
	}
	s.leadRepo = leadRepo
	return s
}

func (s *ConversationService) IngestWebhook(ctx context.Context, input ConversationWebhookInput) (string, bool, error) {
	if s == nil || s.eventRepo == nil {
		return "", false, errors.New("conversation service unavailable")
	}
	started := time.Now()
	defer func() {
		if s.metrics != nil {
			s.metrics.ObserveConversationLatency("local_fallback", leadobs.SinceMs(started))
		}
	}()
	input.TenantUUID = strings.ToLower(strings.TrimSpace(input.TenantUUID))
	input.ChannelAccountUUID = strings.ToLower(strings.TrimSpace(input.ChannelAccountUUID))
	input.ExternalEventID = strings.TrimSpace(input.ExternalEventID)
	input.ConversationID = strings.TrimSpace(input.ConversationID)
	if input.TenantUUID == "" || input.ChannelAccountUUID == "" || input.ExternalEventID == "" || input.ConversationID == "" {
		return "", false, errors.New("invalid webhook payload")
	}
	if input.OccurredAt.IsZero() {
		input.OccurredAt = time.Now().UTC()
	}
	idempotencyKey := fmt.Sprintf("%s:%s:%s", input.TenantUUID, input.ChannelAccountUUID, input.ExternalEventID)
	event := &leadmodel.ConversationEvent{
		TenantUUID:         input.TenantUUID,
		Channel:            "wechat",
		AppType:            "wecom",
		ChannelAccountUUID: input.ChannelAccountUUID,
		ExternalEventID:    input.ExternalEventID,
		IdempotencyKey:     idempotencyKey,
		ConversationID:     input.ConversationID,
		ActorType:          strings.TrimSpace(input.ActorType),
		ActorID:            strings.TrimSpace(input.ActorID),
		Direction:          strings.TrimSpace(input.Direction),
		MessageType:        strings.TrimSpace(input.MessageType),
		ContentText:        strings.TrimSpace(input.ContentText),
		RawPayload:         input.RawPayload,
		OccurredAt:         input.OccurredAt,
	}
	stored, created, err := s.eventRepo.CreateIdempotent(ctx, event)
	if err != nil {
		if s.metrics != nil {
			s.metrics.RecordConversationEvent("local_fallback", "ingest_failed")
		}
		return "", false, err
	}
	if s.metrics != nil {
		if created {
			s.metrics.RecordConversationEvent("local_fallback", "ingest_created")
		} else {
			s.metrics.RecordConversationEvent("local_fallback", "ingest_duplicate")
		}
	}
	if stored != nil {
		if created {
			if err := s.routeConversationEvent(ctx, stored); err != nil {
				return stored.EventUUID, created, err
			}
		}
		return stored.EventUUID, created, nil
	}
	return "", created, nil
}

func (s *ConversationService) ListLeadConversations(ctx context.Context, tenantUUID, leadUUID string) ([]ConversationSummary, error) {
	if s == nil || s.bindingRepo == nil {
		return nil, errors.New("conversation service unavailable")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	leadUUID = strings.ToLower(strings.TrimSpace(leadUUID))
	if s.projectionRepo != nil {
		projections, err := s.projectionRepo.ListByLead(ctx, tenantUUID, leadUUID, 100)
		if err != nil {
			return nil, err
		}
		if len(projections) > 0 {
			result := make([]ConversationSummary, 0, len(projections))
			for _, item := range projections {
				result = append(result, ConversationSummary{
					ConversationID:  item.ConversationID,
					LatestMessage:   item.LatestMessage,
					LatestActorType: item.LatestActorType,
					LatestAt:        item.LatestAt.UTC().Format(time.RFC3339Nano),
					UnreadCount:     item.UnreadCount,
				})
			}
			return result, nil
		}
	}
	bindings, err := s.bindingRepo.ListByLead(ctx, tenantUUID, leadUUID)
	if err != nil {
		return nil, err
	}
	result := make([]ConversationSummary, 0, len(bindings))
	for _, binding := range bindings {
		result = append(result, ConversationSummary{ConversationID: binding.ConversationID})
	}
	return result, nil
}

func (s *ConversationService) ListConversationEvents(ctx context.Context, tenantUUID, conversationID string, limit int) ([]*leadmodel.ConversationEvent, error) {
	if s == nil || s.eventRepo == nil {
		return nil, errors.New("conversation service unavailable")
	}
	return s.eventRepo.ListByConversation(ctx, tenantUUID, conversationID, limit)
}

func (s *ConversationService) BindConversation(ctx context.Context, tenantUUID, leadUUID, conversationID, channelAccountUUID, actorUserUUID string) error {
	if s == nil || s.bindingRepo == nil {
		return errors.New("conversation service unavailable")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	leadUUID = strings.ToLower(strings.TrimSpace(leadUUID))
	conversationID = strings.TrimSpace(conversationID)
	channelAccountUUID = strings.ToLower(strings.TrimSpace(channelAccountUUID))
	_, err := s.bindingRepo.UpsertActive(ctx, &leadmodel.LeadConversationBinding{
		TenantUUID:         strings.ToLower(strings.TrimSpace(tenantUUID)),
		LeadUUID:           strings.ToLower(strings.TrimSpace(leadUUID)),
		ConversationID:     strings.TrimSpace(conversationID),
		ChannelAccountUUID: strings.ToLower(strings.TrimSpace(channelAccountUUID)),
		BindSource:         "manual",
		Status:             "active",
		CreatedBy:          strings.TrimSpace(actorUserUUID),
	})
	if err != nil {
		return err
	}
	if s.pendingRepo != nil && s.pendingRepo.DB != nil {
		now := time.Now().UTC()
		_ = s.pendingRepo.DB.WithContext(ctx).Model(&leadmodel.LeadConversationPending{}).
			Where("tenant_uuid = ? AND conversation_id = ? AND status = ?", tenantUUID, conversationID, "pending").
			Updates(map[string]any{"status": "resolved", "resolved_at": &now}).Error
	}
	if s.eventRepo != nil {
		events, _ := s.eventRepo.ListByConversation(ctx, tenantUUID, conversationID, 1)
		if len(events) > 0 {
			_ = s.upsertProjectionAndPublish(ctx, leadUUID, events[0], false)
		}
	}
	return nil
}

func (s *ConversationService) routeConversationEvent(ctx context.Context, event *leadmodel.ConversationEvent) error {
	if event == nil {
		return nil
	}
	leadUUID, reason := s.resolveLeadUUID(ctx, event)
	if leadUUID == "" {
		if s.pendingRepo != nil {
			_, _ = s.pendingRepo.Create(ctx, &leadmodel.LeadConversationPending{
				TenantUUID:         event.TenantUUID,
				EventUUID:          event.EventUUID,
				ChannelAccountUUID: event.ChannelAccountUUID,
				ConversationID:     event.ConversationID,
				Reason:             reason,
				Status:             "pending",
			})
		}
		return nil
	}
	_, err := s.bindingRepo.UpsertActive(ctx, &leadmodel.LeadConversationBinding{
		TenantUUID:         event.TenantUUID,
		LeadUUID:           leadUUID,
		ConversationID:     event.ConversationID,
		ChannelAccountUUID: event.ChannelAccountUUID,
		BindSource:         "auto",
		Status:             "active",
	})
	if err != nil {
		return err
	}
	return s.upsertProjectionAndPublish(ctx, leadUUID, event, true)
}

func (s *ConversationService) upsertProjectionAndPublish(ctx context.Context, leadUUID string, event *leadmodel.ConversationEvent, fromWebhook bool) error {
	if event == nil || strings.TrimSpace(leadUUID) == "" {
		return nil
	}
	unread := 0
	if strings.EqualFold(strings.TrimSpace(event.Direction), "inbound") {
		unread = 1
	}
	if s.projectionRepo != nil && s.projectionRepo.DB != nil {
		var existing leadmodel.LeadRealtimeProjection
		if err := s.projectionRepo.DB.WithContext(ctx).
			Where("tenant_uuid = ? AND lead_uuid = ? AND conversation_id = ?", event.TenantUUID, leadUUID, event.ConversationID).
			First(&existing).Error; err == nil {
			if unread > 0 {
				unread = existing.UnreadCount + unread
			} else {
				unread = existing.UnreadCount
			}
		}
		_, err := s.projectionRepo.Upsert(ctx, &leadmodel.LeadRealtimeProjection{
			TenantUUID:      event.TenantUUID,
			LeadUUID:        leadUUID,
			ConversationID:  event.ConversationID,
			LatestMessage:   event.ContentText,
			LatestActorType: event.ActorType,
			LatestAt:        event.OccurredAt,
			UnreadCount:     unread,
			UpdatedAt:       time.Now().UTC(),
		})
		if err != nil {
			return err
		}
	}
	if s.realtime != nil {
		s.realtime.PublishLeadConversationUpdated(ctx, event.TenantUUID, LeadConversationUpdatedEvent{
			TenantUUID:         event.TenantUUID,
			LeadUUID:           leadUUID,
			ConversationID:     event.ConversationID,
			ChannelAccountUUID: event.ChannelAccountUUID,
			LatestMessage:      event.ContentText,
			LatestActorType:    event.ActorType,
			LatestAt:           event.OccurredAt.UTC().Format(time.RFC3339Nano),
			UnreadCount:        unread,
			OccurredAt:         time.Now().UTC().Format(time.RFC3339Nano),
		})
	}
	if s.metrics != nil && fromWebhook {
		s.metrics.RecordConversationEvent("local_fallback", "routed")
	}
	return nil
}

func (s *ConversationService) resolveLeadUUID(ctx context.Context, event *leadmodel.ConversationEvent) (string, string) {
	if event == nil {
		return "", "no_match"
	}
	if s.bindingRepo != nil {
		if binding, err := s.bindingRepo.FindActiveByConversation(ctx, event.TenantUUID, event.ConversationID); err == nil && binding != nil {
			return strings.ToLower(strings.TrimSpace(binding.LeadUUID)), "existing_binding"
		}
	}
	if s.leadRepo == nil {
		return "", "no_match"
	}
	if rawLeadUUID := rawString(event.RawPayload, "lead_uuid", "lead_id"); rawLeadUUID != "" {
		if lead, err := s.leadRepo.GetByUUID(ctx, event.TenantUUID, rawLeadUUID); err == nil && lead != nil {
			return strings.ToLower(strings.TrimSpace(lead.LeadUUID)), "external_id"
		}
	}
	if phone := rawString(event.RawPayload, "phone", "customer_phone", "mobile"); phone != "" {
		if lead, err := s.leadRepo.FindFirstByPhone(ctx, event.TenantUUID, phone); err == nil && lead != nil {
			return strings.ToLower(strings.TrimSpace(lead.LeadUUID)), "phone"
		}
	}
	if email := rawString(event.RawPayload, "email", "customer_email"); email != "" {
		if lead, err := s.leadRepo.FindFirstByEmail(ctx, event.TenantUUID, email); err == nil && lead != nil {
			return strings.ToLower(strings.TrimSpace(lead.LeadUUID)), "email"
		}
	}
	return "", "no_match"
}

func rawString(m map[string]any, keys ...string) string {
	for _, key := range keys {
		if m == nil {
			return ""
		}
		v, ok := m[key]
		if !ok || v == nil {
			continue
		}
		switch vv := v.(type) {
		case string:
			if strings.TrimSpace(vv) != "" {
				return strings.TrimSpace(vv)
			}
		case fmt.Stringer:
			if strings.TrimSpace(vv.String()) != "" {
				return strings.TrimSpace(vv.String())
			}
		case int, int64, uint64, float64:
			out := fmt.Sprintf("%v", vv)
			if strings.TrimSpace(out) != "" {
				return strings.TrimSpace(out)
			}
		case jsonNumber:
			if strings.TrimSpace(string(vv)) != "" {
				return strings.TrimSpace(string(vv))
			}
		}
	}
	return ""
}

type jsonNumber string

func parseOccurredAt(raw string) time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Now().UTC()
	}
	if ts, err := strconv.ParseInt(raw, 10, 64); err == nil {
		return time.Unix(ts, 0).UTC()
	}
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t.UTC()
	}
	return time.Now().UTC()
}
