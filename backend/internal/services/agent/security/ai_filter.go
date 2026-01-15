package security

import (
	"strings"
	"sync"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/config"
)

type AIFilter struct {
	placeholder string
	piiKeys     map[string]struct{}
	mu          sync.Mutex
	store       map[string]interface{}
}

func NewAIFilter(cfg *config.Config) *AIFilter {
	baseline := cfg.SecurityBaselineConfig()
	keys := make(map[string]struct{})
	for _, key := range baseline.MaskingRules.PIIFields {
		if key = strings.TrimSpace(strings.ToLower(key)); key != "" {
			keys[key] = struct{}{}
		}
	}
	placeholder := baseline.MaskingRules.LogRedaction.Placeholder
	if placeholder == "" {
		placeholder = "[REDACTED]"
	}
	return &AIFilter{
		placeholder: placeholder,
		piiKeys:     keys,
		store:       make(map[string]interface{}),
	}
}

func (f *AIFilter) FilterMap(data map[string]interface{}) map[string]interface{} {
	if len(f.piiKeys) == 0 || len(data) == 0 {
		return data
	}
	out := make(map[string]interface{}, len(data))
	for k, v := range data {
		if _, ok := f.piiKeys[strings.ToLower(k)]; ok {
			out[k] = f.placeholder
		} else {
			out[k] = v
		}
	}
	return out
}

func (f *AIFilter) StoreEphemeral(key string, value interface{}) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.store[key] = value
}

func (f *AIFilter) RetrieveEphemeral(key string) (interface{}, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	val, ok := f.store[key]
	return val, ok
}

func (f *AIFilter) PurgeEphemeral(keys ...string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(keys) == 0 {
		f.store = make(map[string]interface{})
		return
	}
	for _, key := range keys {
		delete(f.store, key)
	}
}
