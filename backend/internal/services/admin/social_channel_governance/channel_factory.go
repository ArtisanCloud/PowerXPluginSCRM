package social_channel_governance

import (
	"context"
	"fmt"
	"strings"
)

type ChannelSyncAdapter interface {
	CapabilityMatrix(ctx context.Context, tenantUUID string) map[string]string
}

type ChannelFactory struct {
	adapters map[string]ChannelSyncAdapter
}

func NewChannelFactory() *ChannelFactory {
	return &ChannelFactory{
		adapters: map[string]ChannelSyncAdapter{},
	}
}

func (f *ChannelFactory) Register(channel, appType string, adapter ChannelSyncAdapter) error {
	if f == nil {
		return fmt.Errorf("channel factory is nil")
	}
	if adapter == nil {
		return fmt.Errorf("adapter is nil")
	}
	key := buildChannelAdapterKey(channel, appType)
	if key == ":" {
		return fmt.Errorf("channel and app_type are required")
	}
	f.adapters[key] = adapter
	return nil
}

func (f *ChannelFactory) Resolve(channel, appType string) (ChannelSyncAdapter, bool) {
	if f == nil {
		return nil, false
	}
	adapter, ok := f.adapters[buildChannelAdapterKey(channel, appType)]
	return adapter, ok
}

func buildChannelAdapterKey(channel, appType string) string {
	return strings.ToLower(strings.TrimSpace(channel)) + ":" + strings.ToLower(strings.TrimSpace(appType))
}
