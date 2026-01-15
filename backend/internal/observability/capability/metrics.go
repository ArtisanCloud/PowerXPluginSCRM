package capability

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	metricCatalogSyncStatus    = "powerx_capability_catalog_sync_status"
	metricWorkflowAsyncSeconds = "powerx_capability_workflow_async_duration_seconds"
)

// Metrics records capability catalog sync状态与异步 Workflow 时长。
type Metrics struct {
	mu     sync.RWMutex
	gauges map[string]map[string]float64
}

// NewMetrics constructs an empty capability metrics collector.
func NewMetrics() *Metrics {
	return &Metrics{
		gauges: map[string]map[string]float64{},
	}
}

// ObserveCatalogSync records the latest catalog sync status per plugin and mode.
func (m *Metrics) ObserveCatalogSync(pluginID, mode, status string) {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	value := 0.0
	if strings.EqualFold(strings.TrimSpace(status), "success") {
		value = 1.0
	}
	labels := labelKey(map[string]string{
		"plugin_id": normalize(pluginID),
		"mode":      normalize(mode),
		"status":    normalize(status),
	})
	ensureGauge(m.gauges, metricCatalogSyncStatus)[labels] = value
}

// ObserveAsyncWorkflowDuration tracks async capability workflow durations in seconds.
func (m *Metrics) ObserveAsyncWorkflowDuration(capabilityID, workflow, status string, duration time.Duration) {
	if m == nil {
		return
	}
	if duration < 0 {
		duration = 0
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	labels := labelKey(map[string]string{
		"capability_id": normalize(capabilityID),
		"workflow":      normalize(workflow),
		"status":        normalize(status),
	})
	ensureGauge(m.gauges, metricWorkflowAsyncSeconds)[labels] = duration.Seconds()
}

// RenderPrometheus emits metrics in Prometheus exposition format.
func (m *Metrics) RenderPrometheus(w io.Writer) {
	if m == nil {
		return
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	for metric, series := range m.gauges {
		fmt.Fprintf(w, "# TYPE %s gauge\n", metric)
		keys := sortedKeys(series)
		for _, labels := range keys {
			fmt.Fprintf(w, "%s{%s} %g\n", metric, labels, series[labels])
		}
	}
}

func ensureGauge(store map[string]map[string]float64, metric string) map[string]float64 {
	if store[metric] == nil {
		store[metric] = make(map[string]float64)
	}
	return store[metric]
}

func labelKey(labels map[string]string) string {
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, len(keys))
	for i, k := range keys {
		parts[i] = fmt.Sprintf("%s=\"%s\"", k, labels[k])
	}
	return strings.Join(parts, ",")
}

func sortedKeys(m map[string]float64) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func normalize(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}
