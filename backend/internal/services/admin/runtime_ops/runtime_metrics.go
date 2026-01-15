package runtime_ops

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
	"time"

	authmetrics "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/observability/auth"
	customermetrics "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/observability/customer"
	ebmetrics "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/observability/event_bridge"
	secmetrics "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/observability/security"
)

const (
	metricRequestTotal   = "powerx_plugin_request_total"
	metricErrorTotal     = "powerx_plugin_error_total"
	metricLatencySeconds = "powerx_plugin_latency_seconds"
	metricCPUSeconds     = "powerx_plugin_cpu_seconds_total"
	metricMemoryBytes    = "powerx_plugin_memory_bytes"
	metricMCPSessions    = "powerx_mcp_sessions_total"
	metricQuotaUsage     = "powerx_plugin_quota_usage"
	metricCostTotal      = "powerx_plugin_cost_total"
	metricRestartTotal   = "powerx_plugin_restart_total"
	metricHealthStatus   = "powerx_plugin_health_status"
)

var (
	metricsMu sync.RWMutex

	counters   = map[string]map[string]float64{}
	gauges     = map[string]map[string]float64{}
	histograms = map[string]*histogramMetric{}

	metricDescriptions = map[string]string{
		metricRequestTotal:   "Total requests handled by runtime ops capabilities",
		metricErrorTotal:     "Total failed requests handled by runtime ops",
		metricLatencySeconds: "Latency distribution for runtime ops",
		metricCPUSeconds:     "CPU seconds consumed by plugin instances",
		metricMemoryBytes:    "Memory usage of plugin instances",
		metricMCPSessions:    "Active MCP sessions per plugin",
		metricQuotaUsage:     "Quota usage ratios by scope",
		metricCostTotal:      "Aggregated plugin cost by tenant",
		metricRestartTotal:   "Restart attempts per plugin instance",
		metricHealthStatus:   "Health status of plugin instances (1=healthy,0=unhealthy)",
	}
)

func resetMetrics() {
	metricsMu.Lock()
	defer metricsMu.Unlock()
	counters = map[string]map[string]float64{}
	gauges = map[string]map[string]float64{}
	histograms = map[string]*histogramMetric{}
}

var defaultBuckets = []float64{0.1, 0.5, 1, 2, 5, 10}

type histogramMetric struct {
	Buckets []float64
	Counts  []float64
	Sum     float64
	Count   float64
	Labels  map[string][]float64 // key -> counts copy
}

func ensureCounter(metric string) map[string]float64 {
	if counters[metric] == nil {
		counters[metric] = make(map[string]float64)
	}
	return counters[metric]
}

func ensureGauge(metric string) map[string]float64 {
	if gauges[metric] == nil {
		gauges[metric] = make(map[string]float64)
	}
	return gauges[metric]
}

func ensureHistogram(metric string) *histogramMetric {
	h, ok := histograms[metric]
	if !ok {
		h = &histogramMetric{
			Buckets: append([]float64{}, defaultBuckets...),
			Counts:  make([]float64, len(defaultBuckets)),
			Labels:  map[string][]float64{},
		}
		histograms[metric] = h
	}
	return h
}

func labelsKey(labels map[string]string) (string, []string) {
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	pairs := make([]string, len(keys))
	for i, k := range keys {
		pairs[i] = fmt.Sprintf("%s=\"%s\"", k, labels[k])
	}
	return strings.Join(pairs, ","), keys
}

func IncRequest(pluginID, capability string, duration time.Duration, err error) {
	metricsMu.Lock()
	defer metricsMu.Unlock()
	key, _ := labelsKey(map[string]string{"plugin_id": pluginID, "capability": capability})
	ensureCounter(metricRequestTotal)[key]++
	ensureHistogram(metricLatencySeconds).observe(key, duration.Seconds())
	if err != nil {
		ensureCounter(metricErrorTotal)[key]++
	}
}

func (h *histogramMetric) observe(labelKey string, value float64) {
	counts, ok := h.Labels[labelKey]
	if !ok {
		counts = make([]float64, len(h.Buckets))
		h.Labels[labelKey] = counts
	}
	for i, b := range h.Buckets {
		if value <= b {
			counts[i]++
			h.Counts[i]++
			break
		}
	}
	h.Sum += value
	h.Count++
	h.Labels[labelKey] = counts
}

func ObserveCPU(pluginID, instanceID string, seconds float64) {
	metricsMu.Lock()
	defer metricsMu.Unlock()
	key, _ := labelsKey(map[string]string{"plugin_id": pluginID, "instance_id": instanceID})
	ensureCounter(metricCPUSeconds)[key] += seconds
}

func ObserveMemory(pluginID, instanceID string, bytes float64) {
	metricsMu.Lock()
	defer metricsMu.Unlock()
	key, _ := labelsKey(map[string]string{"plugin_id": pluginID, "instance_id": instanceID})
	ensureGauge(metricMemoryBytes)[key] = bytes
}

func SetMCPSessions(pluginID string, count float64) {
	metricsMu.Lock()
	defer metricsMu.Unlock()
	key, _ := labelsKey(map[string]string{"plugin_id": pluginID})
	ensureGauge(metricMCPSessions)[key] = count
}

func SetQuotaUsage(pluginID, scope, scopeRef string, usage float64) {
	metricsMu.Lock()
	defer metricsMu.Unlock()
	key, _ := labelsKey(map[string]string{"plugin_id": pluginID, "scope": scope, "scope_ref": scopeRef})
	ensureGauge(metricQuotaUsage)[key] = usage
}

func AddCost(pluginID, tenantID string, amount float64) {
	metricsMu.Lock()
	defer metricsMu.Unlock()
	key, _ := labelsKey(map[string]string{"plugin_id": pluginID, "tenant_uuid": tenantID})
	ensureCounter(metricCostTotal)[key] += amount
}

func IncRestart(pluginID, instanceID string) {
	metricsMu.Lock()
	defer metricsMu.Unlock()
	key, _ := labelsKey(map[string]string{"plugin_id": pluginID, "instance_id": instanceID})
	ensureCounter(metricRestartTotal)[key]++
}

func SetHealthStatus(pluginID, instanceID string, healthy bool) {
	metricsMu.Lock()
	defer metricsMu.Unlock()
	value := 0.0
	if healthy {
		value = 1
	}
	key, _ := labelsKey(map[string]string{"plugin_id": pluginID, "instance_id": instanceID})
	ensureGauge(metricHealthStatus)[key] = value
}

func renderMetrics(w io.Writer) {
	metricsMu.RLock()
	defer metricsMu.RUnlock()

	for name, values := range counters {
		fmt.Fprintf(w, "# HELP %s %s\n", name, metricDescriptions[name])
		fmt.Fprintf(w, "# TYPE %s counter\n", name)
		keys := sortedKeys(values)
		for _, key := range keys {
			fmt.Fprintf(w, "%s{%s} %g\n", name, key, values[key])
		}
	}

	for name, values := range gauges {
		fmt.Fprintf(w, "# HELP %s %s\n", name, metricDescriptions[name])
		fmt.Fprintf(w, "# TYPE %s gauge\n", name)
		keys := sortedKeys(values)
		for _, key := range keys {
			fmt.Fprintf(w, "%s{%s} %g\n", name, key, values[key])
		}
	}

	for name, hist := range histograms {
		fmt.Fprintf(w, "# HELP %s %s\n", name, metricDescriptions[name])
		fmt.Fprintf(w, "# TYPE %s histogram\n", name)
		labelKeys := make([]string, 0, len(hist.Labels))
		for key := range hist.Labels {
			labelKeys = append(labelKeys, key)
		}
		sort.Strings(labelKeys)
		for _, labelKey := range labelKeys {
			counts := hist.Labels[labelKey]
			cumulative := 0.0
			for i, bucket := range hist.Buckets {
				cumulative += counts[i]
				fmt.Fprintf(w, "%s_bucket{%s,le=\"%g\"} %g\n", name, labelKey, bucket, cumulative)
			}
			fmt.Fprintf(w, "%s_bucket{%s,le=\"+Inf\"} %g\n", name, labelKey, hist.Count)
			fmt.Fprintf(w, "%s_sum{%s} %g\n", name, labelKey, hist.Sum)
			fmt.Fprintf(w, "%s_count{%s} %g\n", name, labelKey, hist.Count)
		}
	}

	authmetrics.RenderMetrics(w)
	secmetrics.RenderMetrics(w)
	customermetrics.RenderMetrics(w)
	ebmetrics.RenderPrometheus(w)
}

func sortedKeys(m map[string]float64) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
