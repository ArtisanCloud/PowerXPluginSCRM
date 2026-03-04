package lead_capture

import (
	"context"
	"encoding/json"
	"strings"

	fwevent "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/event"
	fweventbridge "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/eventbridge"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/config"
	leadmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/lead_capture"
	runtimeswitch "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/runtime/switches"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
	"github.com/google/uuid"
)

const TopicLeadSyncRequestedV1 = "powerx.lead.sync.requested.v1"

type SyncTaskSubmitResult struct {
	Provider       string
	ExternalTaskID *string
	Status         string
}

type SyncTaskProviderAdapter interface {
	SubmitSyncTask(ctx context.Context, req TriggerSyncRequest, resolvedChannelAccountUUID string) SyncTaskSubmitResult
}

type DefaultSyncTaskProviderAdapter struct {
	cfg     *config.Config
	emitter fweventbridge.Emitter
}

func NewDefaultSyncTaskProviderAdapter(cfg *config.Config, emitter fweventbridge.Emitter) *DefaultSyncTaskProviderAdapter {
	return &DefaultSyncTaskProviderAdapter{cfg: cfg, emitter: emitter}
}

func (a *DefaultSyncTaskProviderAdapter) SubmitSyncTask(ctx context.Context, req TriggerSyncRequest, resolvedChannelAccountUUID string) SyncTaskSubmitResult {
	requestedProvider := strings.TrimSpace(req.TaskProvider)
	if requestedProvider == leadmodel.LeadSyncTaskProviderLocalFallback {
		return SyncTaskSubmitResult{Provider: leadmodel.LeadSyncTaskProviderLocalFallback, Status: "queued"}
	}

	if requestedProvider == leadmodel.LeadSyncTaskProviderFramework {
		return a.submitFramework(ctx, req, resolvedChannelAccountUUID)
	}

	if a.frameworkAvailable() {
		return a.submitFramework(ctx, req, resolvedChannelAccountUUID)
	}

	return SyncTaskSubmitResult{Provider: leadmodel.LeadSyncTaskProviderLocalFallback, Status: "queued"}
}

func (a *DefaultSyncTaskProviderAdapter) submitFramework(ctx context.Context, req TriggerSyncRequest, resolvedChannelAccountUUID string) SyncTaskSubmitResult {
	externalTaskID := "fw-" + uuid.NewString()
	if a == nil || a.emitter == nil {
		return SyncTaskSubmitResult{Provider: leadmodel.LeadSyncTaskProviderLocalFallback, Status: "queued"}
	}

	metaBuilder := fwevent.NewMetaBuilder(app.PluginID, "v1")
	meta, err := metaBuilder.Build(strings.TrimSpace(req.TenantUUID), strings.TrimSpace(req.TraceID), strings.TrimSpace(req.TraceID))
	if err != nil {
		return SyncTaskSubmitResult{Provider: leadmodel.LeadSyncTaskProviderLocalFallback, Status: "queued"}
	}

	payload, err := json.Marshal(map[string]any{
		"external_task_id":        externalTaskID,
		"trigger_type":            strings.TrimSpace(req.TriggerType),
		"channel":                 strings.TrimSpace(req.Channel),
		"app_type":                strings.TrimSpace(req.AppType),
		"channel_account_uuid":    strings.TrimSpace(resolvedChannelAccountUUID),
		"tenant_uuid":             strings.TrimSpace(req.TenantUUID),
		"requested_task_provider": strings.TrimSpace(req.TaskProvider),
	})
	if err != nil {
		return SyncTaskSubmitResult{Provider: leadmodel.LeadSyncTaskProviderLocalFallback, Status: "queued"}
	}

	err = a.emitter.Emit(ctx, fwevent.Event{
		Topic:   fwevent.Topic(TopicLeadSyncRequestedV1),
		Meta:    meta,
		Payload: payload,
	})
	if err != nil {
		return SyncTaskSubmitResult{Provider: leadmodel.LeadSyncTaskProviderLocalFallback, Status: "queued"}
	}

	return SyncTaskSubmitResult{Provider: leadmodel.LeadSyncTaskProviderFramework, ExternalTaskID: &externalTaskID, Status: "queued"}
}

func (a *DefaultSyncTaskProviderAdapter) frameworkAvailable() bool {
	if !runtimeswitch.IsHostDriver(runtimeswitch.Resolve(a.cfg).TaskBus) {
		return false
	}
	if a == nil || a.cfg == nil || a.cfg.Gateway == nil {
		return false
	}
	baseURL := strings.TrimSpace(a.cfg.Gateway.BaseURL)
	authScheme := strings.ToLower(strings.TrimSpace(a.cfg.Gateway.AuthScheme))
	toolToken := strings.TrimSpace(a.cfg.Gateway.ToolToken)
	apiKey := strings.TrimSpace(a.cfg.Gateway.APIKey)
	if authScheme == "apikey" || authScheme == "api_key" || authScheme == "api-key" {
		return baseURL != "" && apiKey != ""
	}
	if authScheme == "" && apiKey != "" && toolToken == "" {
		return baseURL != "" && apiKey != ""
	}
	return baseURL != "" && toolToken != ""
}
