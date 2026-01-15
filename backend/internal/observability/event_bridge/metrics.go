package event_bridge

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
)

const (
	metricEmitTotal      = "plugin_event_bridge_emit_total"
	metricConsumeTotal   = "plugin_event_bridge_consume_total"
	metricLatencyGaugeMs = "plugin_event_bridge_latency_ms"
)

var (
	metricsMu sync.RWMutex
	counters  = map[string]map[string]float64{}
	gauges    = map[string]map[string]float64{}
)

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

func labelKey(labels map[string]string) string {
	if len(labels) == 0 {
		return ""
	}
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	pairs := make([]string, len(keys))
	for i, k := range keys {
		pairs[i] = fmt.Sprintf(`%s="%s"`, k, labels[k])
	}
	return strings.Join(pairs, ",")
}

func normalizedValue(value string) string {
	v := strings.ToLower(strings.TrimSpace(value))
	if v == "" {
		return "unknown"
	}
	return v
}

func normalizedTenantUUID(id string) string {
	id = strings.ToLower(strings.TrimSpace(id))
	if id == "" {
		return "unknown"
	}
	return id
}

func normalizedPluginID(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return "unknown"
	}
	return id
}

func RecordEmit(pluginID, tenantUUID, topic, result string) {
	metricsMu.Lock()
	defer metricsMu.Unlock()
	labels := map[string]string{
		"plugin_id":   normalizedPluginID(pluginID),
		"tenant_uuid": normalizedTenantUUID(tenantUUID),
		"topic":       strings.TrimSpace(topic),
		"result":      normalizedValue(result),
	}
	ensureCounter(metricEmitTotal)[labelKey(labels)]++
}

func RecordConsume(pluginID, tenantUUID, topic, result string) {
	metricsMu.Lock()
	defer metricsMu.Unlock()
	labels := map[string]string{
		"plugin_id":   normalizedPluginID(pluginID),
		"tenant_uuid": normalizedTenantUUID(tenantUUID),
		"topic":       strings.TrimSpace(topic),
		"result":      normalizedValue(result),
	}
	ensureCounter(metricConsumeTotal)[labelKey(labels)]++
}

func ObserveLatencyMs(pluginID, tenantUUID, topic, op string, ms float64) {
	if ms < 0 {
		ms = 0
	}
	metricsMu.Lock()
	defer metricsMu.Unlock()
	labels := map[string]string{
		"plugin_id":   normalizedPluginID(pluginID),
		"tenant_uuid": normalizedTenantUUID(tenantUUID),
		"topic":       strings.TrimSpace(topic),
		"op":          normalizedValue(op),
	}
	ensureGauge(metricLatencyGaugeMs)[labelKey(labels)] = ms
}

func RenderPrometheus(w io.Writer) {
	metricsMu.RLock()
	defer metricsMu.RUnlock()

	for metric, series := range counters {
		fmt.Fprintf(w, "# TYPE %s counter\n", metric)
		for _, labels := range sortedKeys(series) {
			fmt.Fprintf(w, "%s{%s} %g\n", metric, labels, series[labels])
		}
	}

	for metric, series := range gauges {
		fmt.Fprintf(w, "# TYPE %s gauge\n", metric)
		for _, labels := range sortedKeys(series) {
			fmt.Fprintf(w, "%s{%s} %g\n", metric, labels, series[labels])
		}
	}
}

func sortedKeys(m map[string]float64) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func Reset() {
	metricsMu.Lock()
	defer metricsMu.Unlock()
	counters = map[string]map[string]float64{}
	gauges = map[string]map[string]float64{}
}
