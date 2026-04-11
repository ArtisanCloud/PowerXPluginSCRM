package lead_capture

import (
	"context"
	"time"

	fwwsbus "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/wsbus"
	leadobs "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/observability/lead_capture"
)

const (
	TopicLeadSyncProgress   = "lead_sync.progress"
	TopicLeadSyncProgressV1 = "powerx.lead_sync.progress.v1"
)

type LeadSyncProgressEvent struct {
	TenantUUID         string `json:"tenant_uuid"`
	TaskUUID           string `json:"task_uuid"`
	Channel            string `json:"channel"`
	AppType            string `json:"app_type"`
	ChannelAccountUUID string `json:"channel_account_uuid"`
	Status             string `json:"status"`
	ProgressTotal      int    `json:"progress_total"`
	ProgressCurrent    int    `json:"progress_current"`
	ProgressPercent    int    `json:"progress_percent"`
	StatsTotal         int    `json:"stats_total"`
	StatsCreated       int    `json:"stats_created"`
	StatsUpdated       int    `json:"stats_updated"`
	StatsMerged        int    `json:"stats_merged"`
	ErrorMessage       string `json:"error_message,omitempty"`
	UpdatedAt          string `json:"updated_at"`
}

type LeadSyncRealtimePublisher struct {
	publisher fwwsbus.Publisher
	metrics   *leadobs.Metrics
}

func NewLeadSyncRealtimePublisher(publisher fwwsbus.Publisher, metrics *leadobs.Metrics) *LeadSyncRealtimePublisher {
	return &LeadSyncRealtimePublisher{publisher: publisher, metrics: metrics}
}

func (p *LeadSyncRealtimePublisher) PublishLeadSyncProgress(ctx context.Context, tenantUUID string, payload LeadSyncProgressEvent) fwwsbus.PublishResult {
	if p == nil || p.publisher == nil || tenantUUID == "" {
		return fwwsbus.PublishResult{OK: false, ErrorCode: fwwsbus.ErrorCodePublisherNotConfigured, ErrorMessage: "publisher unavailable"}
	}
	if payload.UpdatedAt == "" {
		payload.UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
	}
	result := p.publisher.Publish(ctx, TopicLeadSyncProgress, payload, fwwsbus.PublishOptions{TenantUUID: tenantUUID})
	if !result.OK {
		result = p.publisher.Publish(ctx, TopicLeadSyncProgressV1, payload, fwwsbus.PublishOptions{TenantUUID: tenantUUID})
	}
	if p.metrics != nil {
		if result.OK {
			p.metrics.RecordSyncTask("wsbus", "published")
		} else {
			p.metrics.RecordSyncTask("wsbus", "publish_failed")
		}
	}
	return result
}
