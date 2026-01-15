package operations

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
)

const (
	metricSupportTicketTotal = "powerx_operations_support_ticket_total"
	metricIncidentEventTotal = "powerx_operations_incident_event_total"
	metricSLAHealthGauge     = "powerx_operations_sla_score"
)

// Metrics captures counters and gauges for Operations workflows.
type Metrics struct {
	mu       sync.RWMutex
	counters map[string]map[string]float64
	gauges   map[string]map[string]float64
}

// NewMetrics constructs an empty Operations metrics collector.
func NewMetrics() *Metrics {
	return &Metrics{
		counters: map[string]map[string]float64{},
		gauges:   map[string]map[string]float64{},
	}
}

// RecordSupportTicket increments ticket counters grouped by status and priority.
func (m *Metrics) RecordSupportTicket(status, priority string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	labels := labelKey(map[string]string{
		"status":   normalize(status),
		"priority": normalize(priority),
	})
	ensureCounter(m.counters, metricSupportTicketTotal)[labels]++
}

// RecordIncidentEvent tracks incident lifecycle events grouped by severity and action.
func (m *Metrics) RecordIncidentEvent(severity, action string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	labels := labelKey(map[string]string{
		"severity": normalize(severity),
		"action":   normalize(action),
	})
	ensureCounter(m.counters, metricIncidentEventTotal)[labels]++
}

// ObserveSLAScore records the latest SLA score per plan type.
func (m *Metrics) ObserveSLAScore(plan string, score float64) {
	if score < 0 {
		score = 0
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	labels := labelKey(map[string]string{
		"plan": normalize(plan),
	})
	ensureGauge(m.gauges, metricSLAHealthGauge)[labels] = score
}

// RenderPrometheus emits metrics in Prometheus exposition format.
func (m *Metrics) RenderPrometheus(w io.Writer) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for metric, series := range m.counters {
		fmt.Fprintf(w, "# TYPE %s counter\n", metric)
		keys := sortedKeys(series)
		for _, labels := range keys {
			fmt.Fprintf(w, "%s{%s} %g\n", metric, labels, series[labels])
		}
	}

	for metric, series := range m.gauges {
		fmt.Fprintf(w, "# TYPE %s gauge\n", metric)
		keys := sortedKeys(series)
		for _, labels := range keys {
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
