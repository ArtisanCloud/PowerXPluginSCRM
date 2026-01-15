package runtime_ops

import (
	"sync"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/config"
)

var (
	observabilityCfg config.ObservabilityConfig
	alertThresholds  config.AlertThresholds
	cfgOnce          sync.Once
)

// ConfigureRuntimeOps stores observability defaults from configuration.
func ConfigureRuntimeOps(defaults *config.RuntimeOpsDefaults) {
	if defaults == nil {
		return
	}
	cfgOnce.Do(func() {
		observabilityCfg = defaults.Observability
		alertThresholds = defaults.Alerts
	})
}

// ObservabilityConfig returns current observability endpoints.
func ObservabilityConfig() config.ObservabilityConfig {
	return observabilityCfg
}

// AlertThresholds returns configured alert thresholds.
func AlertThresholds() config.AlertThresholds {
	return alertThresholds
}
