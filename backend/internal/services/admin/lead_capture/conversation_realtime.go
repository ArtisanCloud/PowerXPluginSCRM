package lead_capture

import (
	"context"
	"time"

	fwwsbus "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/wsbus"
	leadobs "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/observability/lead_capture"
)

const TopicLeadConversationUpdatedV1 = "powerx.lead.conversation.updated.v1"

type LeadConversationUpdatedEvent struct {
	TenantUUID         string `json:"tenant_uuid"`
	LeadUUID           string `json:"lead_uuid"`
	ConversationID     string `json:"conversation_id"`
	ChannelAccountUUID string `json:"channel_account_uuid"`
	LatestMessage      string `json:"latest_message,omitempty"`
	LatestActorType    string `json:"latest_actor_type,omitempty"`
	LatestAt           string `json:"latest_at,omitempty"`
	UnreadCount        int    `json:"unread_count"`
	OccurredAt         string `json:"occurred_at"`
}

type ConversationRealtimePublisher struct {
	publisher fwwsbus.Publisher
	metrics   *leadobs.Metrics
}

func NewConversationRealtimePublisher(publisher fwwsbus.Publisher, metrics *leadobs.Metrics) *ConversationRealtimePublisher {
	return &ConversationRealtimePublisher{publisher: publisher, metrics: metrics}
}

func (p *ConversationRealtimePublisher) PublishLeadConversationUpdated(ctx context.Context, tenantUUID string, payload LeadConversationUpdatedEvent) fwwsbus.PublishResult {
	if p == nil || p.publisher == nil || tenantUUID == "" {
		return fwwsbus.PublishResult{OK: false, ErrorCode: fwwsbus.ErrorCodePublisherNotConfigured, ErrorMessage: "publisher unavailable"}
	}
	if payload.OccurredAt == "" {
		payload.OccurredAt = time.Now().UTC().Format(time.RFC3339Nano)
	}
	result := p.publisher.Publish(ctx, TopicLeadConversationUpdatedV1, payload, fwwsbus.PublishOptions{TenantUUID: tenantUUID})
	if p.metrics != nil {
		if result.OK {
			p.metrics.RecordConversationEvent("local_fallback", "published")
		} else {
			p.metrics.RecordConversationEvent("local_fallback", "publish_failed")
		}
	}
	return result
}
