package lead_capture

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
)

const (
	metricLeadSyncTaskTotal      = "powerx_lead_capture_sync_task_total"
	metricConversationEventTotal = "powerx_lead_capture_conversation_event_total"
	metricConversationLatencyMs  = "powerx_lead_capture_conversation_latency_ms"
)

type Metrics struct {
	mu       sync.RWMutex
	counters map[string]map[string]float64
	gauges   map[string]map[string]float64
}

func NewMetrics() *Metrics {
	return &Metrics{
		counters: map[string]map[string]float64{},
		gauges:   map[string]map[string]float64{},
	}
}

func (m *Metrics) RecordSyncTask(provider, status string) {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	labels := labelKey(map[string]string{
		"provider": normalize(provider),
		"status":   normalize(status),
	})
	ensureCounter(m.counters, metricLeadSyncTaskTotal)[labels]++
}

func (m *Metrics) RecordConversationEvent(provider, result string) {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	labels := labelKey(map[string]string{
		"provider": normalize(provider),
		"result":   normalize(result),
	})
	ensureCounter(m.counters, metricConversationEventTotal)[labels]++
}

func (m *Metrics) ObserveConversationLatency(provider string, latencyMs float64) {
	if m == nil {
		return
	}
	if latencyMs < 0 {
		latencyMs = 0
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	labels := labelKey(map[string]string{
		"provider": normalize(provider),
	})
	ensureGauge(m.gauges, metricConversationLatencyMs)[labels] = latencyMs
}

func (m *Metrics) RenderPrometheus(w io.Writer) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for metric, series := range m.counters {
		fmt.Fprintf(w, "# TYPE %s counter\n", metric)
		for _, labels := range sortedKeys(series) {
			fmt.Fprintf(w, "%s{%s} %g\n", metric, labels, series[labels])
		}
	}
	for metric, series := range m.gauges {
		fmt.Fprintf(w, "# TYPE %s gauge\n", metric)
		for _, labels := range sortedKeys(series) {
			fmt.Fprintf(w, "%s{%s} %g\n", metric, labels, series[labels])
		}
	}
}

func ensureCounter(store map[string]map[string]float64, metric string) map[string]float64 {
	if store[metric] == nil {
		store[metric] = make(map[string]float64)
	}
	return store[metric]
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
	pairs := make([]string, len(keys))
	for i, k := range keys {
		pairs[i] = fmt.Sprintf("%s=\"%s\"", k, labels[k])
	}
	return strings.Join(pairs, ",")
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
