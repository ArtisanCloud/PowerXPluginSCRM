package auth

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
)

const (
	metricLoginTotal          = "plugin_auth_login_total"
	metricRefreshTotal        = "plugin_auth_refresh_total"
	metricLogoutTotal         = "plugin_auth_logout_total"
	metricDelegateErrorsTotal = "plugin_iam_delegate_errors_total"
	metricIAMModeGauge        = "plugin_iam_mode"
	metricIAMRoleChangeTotal  = "plugin_iam_role_change_total"
	metricRBACDeniedTotal     = "plugin_rbac_denied_total"
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

func normalizedMode(mode string) string {
	mode = strings.ToLower(strings.TrimSpace(mode))
	if mode == "" {
		return "unknown"
	}
	return mode
}

func normalizedPluginID(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return "unknown"
	}
	return id
}

func normalizedTenantUUID(id string) string {
	id = strings.ToLower(strings.TrimSpace(id))
	if id == "" {
		return "unknown"
	}
	return id
}

func normalizedValue(value string) string {
	v := strings.ToLower(strings.TrimSpace(value))
	if v == "" {
		return "unknown"
	}
	return v
}

// RecordLogin increments login counters grouped by IAM mode and result.
func RecordLogin(pluginID, mode, result string) {
	metricsMu.Lock()
	defer metricsMu.Unlock()
	labels := map[string]string{
		"mode":      normalizedMode(mode),
		"result":    strings.ToLower(strings.TrimSpace(result)),
		"plugin_id": normalizedPluginID(pluginID),
	}
	ensureCounter(metricLoginTotal)[labelKey(labels)]++
}

// RecordRefresh increments refresh counters grouped by IAM mode and result.
func RecordRefresh(pluginID, mode, result string) {
	metricsMu.Lock()
	defer metricsMu.Unlock()
	labels := map[string]string{
		"mode":      normalizedMode(mode),
		"result":    strings.ToLower(strings.TrimSpace(result)),
		"plugin_id": normalizedPluginID(pluginID),
	}
	ensureCounter(metricRefreshTotal)[labelKey(labels)]++
}

// RecordLogout increments logout counters grouped by IAM mode.
func RecordLogout(pluginID, mode string) {
	metricsMu.Lock()
	defer metricsMu.Unlock()
	labels := map[string]string{
		"mode":      normalizedMode(mode),
		"plugin_id": normalizedPluginID(pluginID),
	}
	ensureCounter(metricLogoutTotal)[labelKey(labels)]++
}

// RecordDelegateError tracks delegated auth errors grouped by category.
func RecordDelegateError(pluginID, category string) {
	metricsMu.Lock()
	defer metricsMu.Unlock()
	labels := map[string]string{
		"type":      strings.ToLower(strings.TrimSpace(category)),
		"plugin_id": normalizedPluginID(pluginID),
	}
	ensureCounter(metricDelegateErrorsTotal)[labelKey(labels)]++
}

// RecordRoleChange increments counters for IAM role mutations.
func RecordRoleChange(pluginID, tenantUUID, event, scope string) {
	metricsMu.Lock()
	defer metricsMu.Unlock()
	labels := map[string]string{
		"plugin_id":   normalizedPluginID(pluginID),
		"tenant_uuid": normalizedTenantUUID(tenantUUID),
		"event":       normalizedValue(event),
		"scope":       normalizedValue(scope),
	}
	ensureCounter(metricIAMRoleChangeTotal)[labelKey(labels)]++
}

// RecordRBACDenied tracks RBAC deny events grouped by resource/action.
func RecordRBACDenied(pluginID, resource, action string) {
	metricsMu.Lock()
	defer metricsMu.Unlock()
	labels := map[string]string{
		"plugin_id": normalizedPluginID(pluginID),
		"resource":  normalizedValue(resource),
		"action":    normalizedValue(action),
	}
	ensureCounter(metricRBACDeniedTotal)[labelKey(labels)]++
}

// ObserveMode sets the IAM mode gauge (1 for selected mode, 0 for the opposite).
func ObserveMode(mode string) {
	metricsMu.Lock()
	defer metricsMu.Unlock()
	mode = normalizedMode(mode)
	gauge := ensureGauge(metricIAMModeGauge)
	gauge[labelKey(map[string]string{"mode": mode})] = 1
	other := "local"
	if mode == "local" {
		other = "delegated"
	}
	gauge[labelKey(map[string]string{"mode": other})] = 0
}

// RenderMetrics emits auth metrics in Prometheus exposition format.
func RenderMetrics(w io.Writer) {
	metricsMu.RLock()
	defer metricsMu.RUnlock()

	for metric, series := range counters {
		fmt.Fprintf(w, "# TYPE %s counter\n", metric)
		for _, labels := range sortedKeys(series) {
			fmt.Fprintf(w, "%s{%s} %g\n", metric, labels, series[labels])
		}
	}

	if gaugeSeries, ok := gauges[metricIAMModeGauge]; ok {
		fmt.Fprintf(w, "# TYPE %s gauge\n", metricIAMModeGauge)
		for _, labels := range sortedKeys(gaugeSeries) {
			fmt.Fprintf(w, "%s{%s} %g\n", metricIAMModeGauge, labels, gaugeSeries[labels])
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
