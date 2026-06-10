package integration

import (
	"context"
	"errors"
	"testing"

	fwevent "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/event"
	fweventbridge "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/eventbridge"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/config"
	leadsvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/lead_capture"
)

type stubEmitter struct {
	err error
}

func (s stubEmitter) Emit(ctx context.Context, e fwevent.Event) error {
	_ = ctx
	_ = e
	return s.err
}

var _ fweventbridge.Emitter = (*stubEmitter)(nil)

func TestSyncTaskProvider_FrameworkAndFallbackSwitch(t *testing.T) {
	t.Setenv("POWERX_PROXY", "1")

	cfg := &config.Config{
		Gateway: &config.GatewayConfig{BaseURL: "http://127.0.0.1:8077"},
		GRPCUpstream: &config.GRPCUpstream{
			STSClientID:     "client-id",
			STSClientSecret: "client-secret",
		},
	}
	req := leadsvc.TriggerSyncRequest{
		TenantUUID:         "00000000-0000-0000-0000-000000000001",
		Channel:            "wechat",
		AppType:            "wecom",
		ChannelAccountUUID: "11111111-1111-1111-1111-111111111111",
		TraceID:            "trace-001",
		TriggerType:        "manual",
	}

	frameworkAdapter := leadsvc.NewDefaultSyncTaskProviderAdapter(cfg, stubEmitter{err: nil})
	resultFramework := frameworkAdapter.SubmitSyncTask(context.Background(), req, req.ChannelAccountUUID)
	if resultFramework.Provider != "framework" {
		t.Fatalf("期望 provider=framework，实际=%s", resultFramework.Provider)
	}
	if resultFramework.ExternalTaskID == nil || *resultFramework.ExternalTaskID == "" {
		t.Fatal("期望 framework 路径返回 external_task_id")
	}
	if resultFramework.Status != "queued" {
		t.Fatalf("期望 status=queued，实际=%s", resultFramework.Status)
	}

	fallbackAdapter := leadsvc.NewDefaultSyncTaskProviderAdapter(cfg, stubEmitter{err: errors.New("emit failed")})
	resultFallback := fallbackAdapter.SubmitSyncTask(context.Background(), req, req.ChannelAccountUUID)
	if resultFallback.Provider != "local_fallback" {
		t.Fatalf("期望 emitter 失败时回落 local_fallback，实际=%s", resultFallback.Provider)
	}
	if resultFallback.ExternalTaskID != nil {
		t.Fatal("期望 local_fallback 不返回 external_task_id")
	}
}

func TestSyncTaskProvider_ExplicitLocalFallback(t *testing.T) {
	t.Setenv("POWERX_PROXY", "1")
	cfg := &config.Config{
		Gateway: &config.GatewayConfig{BaseURL: "http://127.0.0.1:8077"},
		GRPCUpstream: &config.GRPCUpstream{
			STSClientID:     "client-id",
			STSClientSecret: "client-secret",
		},
	}
	adapter := leadsvc.NewDefaultSyncTaskProviderAdapter(cfg, stubEmitter{err: nil})

	result := adapter.SubmitSyncTask(context.Background(), leadsvc.TriggerSyncRequest{
		TenantUUID:   "00000000-0000-0000-0000-000000000001",
		Channel:      "wechat",
		AppType:      "wecom",
		TraceID:      "trace-002",
		TriggerType:  "manual",
		TaskProvider: "local_fallback",
	}, "11111111-1111-1111-1111-111111111111")

	if result.Provider != "local_fallback" {
		t.Fatalf("显式 local_fallback 时不应走 framework，实际=%s", result.Provider)
	}
}
