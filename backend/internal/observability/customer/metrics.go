package customer

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	metricCustomerValidationTotal          = "powerx_customer_auth_validation_total"
	metricCustomerValidationLatencyMSSum   = "powerx_customer_auth_validation_latency_ms_sum"
	metricCustomerValidationLatencyMSCount = "powerx_customer_auth_validation_latency_ms_count"

	metricCustomerLoginTotal        = "powerx_customer_auth_login_total"
	metricCustomerLoginLatencyMSSum = "powerx_customer_auth_login_latency_ms_sum"
	metricCustomerLoginLatencyCount = "powerx_customer_auth_login_latency_ms_count"
)

var (
	metricsMu sync.RWMutex
	counters  = map[string]map[string]float64{}

	metricDescriptions = map[string]string{
		metricCustomerValidationTotal:          "Total customer token validations grouped by mode and result",
		metricCustomerValidationLatencyMSSum:   "Total customer token validation latency in milliseconds (sum)",
		metricCustomerValidationLatencyMSCount: "Total customer token validation observations (count)",
		metricCustomerLoginTotal:               "Total customer logins grouped by result",
		metricCustomerLoginLatencyMSSum:        "Total customer login latency in milliseconds (sum)",
		metricCustomerLoginLatencyCount:        "Total customer login observations (count)",
	}
)

func normalize(value string) string {
	v := strings.ToLower(strings.TrimSpace(value))
	if v == "" {
		return "unknown"
	}
	return v
}

func resolvePluginID(pluginID string) string {
	if strings.TrimSpace(pluginID) != "" {
		return pluginID
	}
	return strings.TrimSpace(os.Getenv("POWERX_PLUGIN_ID"))
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
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf(`%s="%s"`, k, labels[k]))
	}
	return strings.Join(parts, ",")
}

func counter(metric string) map[string]float64 {
	if counters[metric] == nil {
		counters[metric] = make(map[string]float64)
	}
	return counters[metric]
}

// RecordValidation tracks customer token validation outcomes and latency.
func RecordValidation(pluginID, mode, result string, latency time.Duration) {
	metricsMu.Lock()
	defer metricsMu.Unlock()
	labels := labelKey(map[string]string{
		"plugin_id": normalize(resolvePluginID(pluginID)),
		"mode":      normalize(mode),
		"result":    normalize(result),
	})
	counter(metricCustomerValidationTotal)[labels]++
	counter(metricCustomerValidationLatencyMSSum)[labels] += float64(latency.Milliseconds())
	counter(metricCustomerValidationLatencyMSCount)[labels]++
}

// RecordLogin tracks Skeleton customer login outcomes and latency.
func RecordLogin(pluginID, result string, latency time.Duration) {
	metricsMu.Lock()
	defer metricsMu.Unlock()
	labels := labelKey(map[string]string{
		"plugin_id": normalize(resolvePluginID(pluginID)),
		"result":    normalize(result),
	})
	counter(metricCustomerLoginTotal)[labels]++
	counter(metricCustomerLoginLatencyMSSum)[labels] += float64(latency.Milliseconds())
	counter(metricCustomerLoginLatencyCount)[labels]++
}

// RenderMetrics writes customer metrics in Prometheus exposition format.
func RenderMetrics(w io.Writer) {
	metricsMu.RLock()
	defer metricsMu.RUnlock()
	for metric, values := range counters {
		fmt.Fprintf(w, "# HELP %s %s\n", metric, metricDescriptions[metric])
		fmt.Fprintf(w, "# TYPE %s counter\n", metric)
		keys := sortedKeys(values)
		for _, key := range keys {
			if key == "" {
				fmt.Fprintf(w, "%s %g\n", metric, values[key])
				continue
			}
			fmt.Fprintf(w, "%s{%s} %g\n", metric, key, values[key])
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
}
