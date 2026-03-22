package lead_capture

import (
	"errors"
	"fmt"
	"strings"
	"sync"
)

var (
	ErrLeadSyncAdapterNotRegistered   = errors.New("lead sync adapter not registered")
	ErrTaskProviderNotRegistered      = errors.New("task provider adapter not registered")
	ErrLeadSyncChannelIdentityInvalid = errors.New("channel and app_type are required")
)

type channelFactoryEntry struct {
	leadAdapter WeComLeadAdapter
	provider    SyncTaskProviderAdapter
}

// ChannelSyncFactory resolves lead-sync adapters by channel + app_type.
type ChannelSyncFactory struct {
	mu      sync.RWMutex
	entries map[string]channelFactoryEntry
}

func NewChannelSyncFactory() *ChannelSyncFactory {
	return &ChannelSyncFactory{
		entries: map[string]channelFactoryEntry{},
	}
}

func (f *ChannelSyncFactory) Register(channel, appType string, leadAdapter WeComLeadAdapter, provider SyncTaskProviderAdapter) error {
	if leadAdapter == nil {
		return errors.New("lead adapter is required")
	}
	if provider == nil {
		return errors.New("task provider adapter is required")
	}
	key, err := normalizeChannelFactoryKey(channel, appType)
	if err != nil {
		return err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.entries[key] = channelFactoryEntry{
		leadAdapter: leadAdapter,
		provider:    provider,
	}
	return nil
}

func (f *ChannelSyncFactory) RegisterLeadAdapter(channel, appType string, leadAdapter WeComLeadAdapter) error {
	if leadAdapter == nil {
		return errors.New("lead adapter is required")
	}
	key, err := normalizeChannelFactoryKey(channel, appType)
	if err != nil {
		return err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	entry := f.entries[key]
	entry.leadAdapter = leadAdapter
	f.entries[key] = entry
	return nil
}

func (f *ChannelSyncFactory) RegisterTaskProvider(channel, appType string, provider SyncTaskProviderAdapter) error {
	if provider == nil {
		return errors.New("task provider adapter is required")
	}
	key, err := normalizeChannelFactoryKey(channel, appType)
	if err != nil {
		return err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	entry := f.entries[key]
	entry.provider = provider
	f.entries[key] = entry
	return nil
}

func (f *ChannelSyncFactory) ResolveLeadAdapter(channel, appType string) (WeComLeadAdapter, error) {
	key, err := normalizeChannelFactoryKey(channel, appType)
	if err != nil {
		return nil, err
	}
	f.mu.RLock()
	defer f.mu.RUnlock()
	entry, ok := f.entries[key]
	if !ok || entry.leadAdapter == nil {
		return nil, fmt.Errorf("%w: %s", ErrLeadSyncAdapterNotRegistered, key)
	}
	return entry.leadAdapter, nil
}

func (f *ChannelSyncFactory) ResolveTaskProvider(channel, appType string) (SyncTaskProviderAdapter, error) {
	key, err := normalizeChannelFactoryKey(channel, appType)
	if err != nil {
		return nil, err
	}
	f.mu.RLock()
	defer f.mu.RUnlock()
	entry, ok := f.entries[key]
	if !ok || entry.provider == nil {
		return nil, fmt.Errorf("%w: %s", ErrTaskProviderNotRegistered, key)
	}
	return entry.provider, nil
}

func normalizeChannelFactoryKey(channel, appType string) (string, error) {
	channel = strings.ToLower(strings.TrimSpace(channel))
	appType = strings.ToLower(strings.TrimSpace(appType))
	if channel == "" || appType == "" {
		return "", ErrLeadSyncChannelIdentityInvalid
	}
	return channel + ":" + appType, nil
}
