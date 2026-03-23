package lead_capture

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	metricLeadSyncTaskTotal      = "powerx_lead_capture_sync_task_total"
	metricConversationEventTotal = "powerx_lead_capture_conversation_event_total"
	metricConversationLatencyMs  = "powerx_lead_capture_conversation_latency_ms"
	metricConversationLatencyP95 = "powerx_lead_capture_conversation_latency_p95_ms"
	metricConversationDupRate    = "powerx_lead_capture_conversation_duplicate_rate"
	latencySampleWindow          = 200
)

type Metrics struct {
	mu       sync.RWMutex
	counters map[string]map[string]float64
	gauges   map[string]map[string]float64
	latency  map[string][]float64
	events   map[string]map[string]float64
}

func NewMetrics() *Metrics {
	return &Metrics{
		counters: map[string]map[string]float64{},
		gauges:   map[string]map[string]float64{},
		latency:  map[string][]float64{},
		events:   map[string]map[string]float64{},
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
	p := normalize(provider)
	if p == "" {
		p = "unknown"
	}
	if m.events[p] == nil {
		m.events[p] = map[string]float64{}
	}
	r := normalize(result)
	if r == "" {
		r = "unknown"
	}
	m.events[p][r]++
	total := m.events[p]["ingest_created"] + m.events[p]["ingest_duplicate"]
	if total > 0 {
		dupRate := m.events[p]["ingest_duplicate"] / total
		ensureGauge(m.gauges, metricConversationDupRate)[labelKey(map[string]string{"provider": p})] = dupRate
	}
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
	p := normalize(provider)
	if p == "" {
		p = "unknown"
	}
	m.latency[p] = append(m.latency[p], latencyMs)
	if len(m.latency[p]) > latencySampleWindow {
		m.latency[p] = m.latency[p][len(m.latency[p])-latencySampleWindow:]
	}
	p95 := percentile95(m.latency[p])
	ensureGauge(m.gauges, metricConversationLatencyP95)[labelKey(map[string]string{"provider": p})] = p95
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

func percentile95(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	cp := make([]float64, len(values))
	copy(cp, values)
	sort.Float64s(cp)
	idx := int(float64(len(cp)-1) * 0.95)
	if idx < 0 {
		idx = 0
	}
	if idx >= len(cp) {
		idx = len(cp) - 1
	}
	return cp[idx]
}

func SinceMs(start time.Time) float64 {
	if start.IsZero() {
		return 0
	}
	return float64(time.Since(start).Milliseconds())
}
