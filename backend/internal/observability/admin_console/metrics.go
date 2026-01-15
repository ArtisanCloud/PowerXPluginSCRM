package admin_console

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
)

const (
	metricAuditExportTotal       = "powerx_admin_console_audit_export_total"
	metricSafeOpExecutionTotal   = "powerx_admin_console_safe_op_total"
	metricDashboardFreshnessSecs = "powerx_admin_console_dashboard_refresh_seconds"
)

// Metrics provides counters and gauges for admin console activity.
type Metrics struct {
	mu       sync.RWMutex
	counters map[string]map[string]float64
	gauges   map[string]map[string]float64
}

// NewMetrics constructs a metrics collector with empty state.
func NewMetrics() *Metrics {
	return &Metrics{
		counters: map[string]map[string]float64{},
		gauges:   map[string]map[string]float64{},
	}
}

// RecordAuditExport increments the audit export counter for a given format.
func (m *Metrics) RecordAuditExport(format string) {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	key := labelKey(map[string]string{
		"format": normalize(format),
	})
	ensureCounter(m.counters, metricAuditExportTotal)[key]++
}

// RecordSafeOp increments safe operation counters grouped by action and outcome.
func (m *Metrics) RecordSafeOp(action, outcome string) {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	key := labelKey(map[string]string{
		"action":  normalize(action),
		"outcome": normalize(outcome),
	})
	ensureCounter(m.counters, metricSafeOpExecutionTotal)[key]++
}

// ObserveDashboardFreshness records the last refresh lag in seconds for a tenant scope.
func (m *Metrics) ObserveDashboardFreshness(scope string, seconds float64) {
	if m == nil {
		return
	}
	if seconds < 0 {
		seconds = 0
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	key := labelKey(map[string]string{
		"scope": normalize(scope),
	})
	ensureGauge(m.gauges, metricDashboardFreshnessSecs)[key] = seconds
}

// RenderPrometheus emits collected metrics in Prometheus exposition format.
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
