package driver

import (
	"fmt"
	"strings"
)

// Registry stores drivers by channel + app type.
type Registry struct {
	drivers map[string]OrgSyncDriver
}

func NewRegistry() *Registry {
	return &Registry{drivers: make(map[string]OrgSyncDriver)}
}

func (r *Registry) Register(channel, appType string, drv OrgSyncDriver) {
	if r == nil || drv == nil {
		return
	}
	key := normalizeKey(channel, appType)
	r.drivers[key] = drv
}

func (r *Registry) Resolve(channel, appType string) (OrgSyncDriver, error) {
	if r == nil {
		return nil, fmt.Errorf("driver registry not configured")
	}
	key := normalizeKey(channel, appType)
	drv, ok := r.drivers[key]
	if !ok {
		return nil, fmt.Errorf("driver not found for %s/%s", channel, appType)
	}
	return drv, nil
}

func normalizeKey(channel, appType string) string {
	return strings.ToLower(strings.TrimSpace(channel)) + "/" + strings.ToLower(strings.TrimSpace(appType))
}
