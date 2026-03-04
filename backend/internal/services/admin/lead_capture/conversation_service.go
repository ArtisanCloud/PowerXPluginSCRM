package lead_capture

import (
	"context"
	"errors"
	"fmt"
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

func (s *ConversationService) IngestWebhook(ctx context.Context, input ConversationWebhookInput) (string, bool, error) {
	if s == nil || s.eventRepo == nil {
		return "", false, errors.New("conversation service unavailable")
	}
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
		return stored.EventUUID, created, nil
	}
	return "", created, nil
}

func (s *ConversationService) ListLeadConversations(ctx context.Context, tenantUUID, leadUUID string) ([]ConversationSummary, error) {
	if s == nil || s.bindingRepo == nil {
		return nil, errors.New("conversation service unavailable")
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
	_, err := s.bindingRepo.UpsertActive(ctx, &leadmodel.LeadConversationBinding{
		TenantUUID:         strings.ToLower(strings.TrimSpace(tenantUUID)),
		LeadUUID:           strings.ToLower(strings.TrimSpace(leadUUID)),
		ConversationID:     strings.TrimSpace(conversationID),
		ChannelAccountUUID: strings.ToLower(strings.TrimSpace(channelAccountUUID)),
		BindSource:         "manual",
		Status:             "active",
		CreatedBy:          strings.TrimSpace(actorUserUUID),
	})
	return err
}
