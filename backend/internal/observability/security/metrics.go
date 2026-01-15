package security

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
)

const (
	metricConsentEventsTotal     = "powerx_security_consent_events_total"
	metricToolGrantEventsTotal   = "powerx_security_toolgrant_events_total"
	metricAdvisoryLifecycleTotal = "powerx_security_advisory_events_total"
	metricAuditReportStatusTotal = "powerx_security_audit_reports_total"
	metricOpenAdvisoriesGauge    = "powerx_security_advisories_open"
)

var (
	metricsLock sync.RWMutex
	secCounters = map[string]map[string]float64{}
	secGauges   = map[string]map[string]float64{}

	metricDescriptions = map[string]string{
		metricConsentEventsTotal:     "Total consent lifecycle events grouped by type and result",
		metricToolGrantEventsTotal:   "Total ToolGrant lifecycle events grouped by event and tenant scope",
		metricAdvisoryLifecycleTotal: "Total advisory lifecycle transitions grouped by severity and status",
		metricAuditReportStatusTotal: "Total security audit reports grouped by outcome",
		metricOpenAdvisoriesGauge:    "Current count of open advisories grouped by severity",
	}
)

func counterFor(metric string) map[string]float64 {
	if secCounters[metric] == nil {
		secCounters[metric] = make(map[string]float64)
	}
	return secCounters[metric]
}

func gaugeFor(metric string) map[string]float64 {
	if secGauges[metric] == nil {
		secGauges[metric] = make(map[string]float64)
	}
	return secGauges[metric]
}

func labelsKey(labels map[string]string) (string, []string) {
	if len(labels) == 0 {
		return "", nil
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
	return strings.Join(pairs, ","), keys
}

// RecordConsentEvent increments the consent lifecycle counter.
func RecordConsentEvent(eventType, result string) {
	if eventType == "" {
		return
	}
	if result == "" {
		result = "unknown"
	}
	metricsLock.Lock()
	defer metricsLock.Unlock()
	key, _ := labelsKey(map[string]string{
		"event_type": eventType,
		"result":     strings.ToLower(result),
	})
	counterFor(metricConsentEventsTotal)[key]++
}

// RecordToolGrantEvent tracks issuance and revocation metrics.
func RecordToolGrantEvent(event, tenantID string) {
	if event == "" {
		return
	}
	metricsLock.Lock()
	defer metricsLock.Unlock()
	labels := map[string]string{
		"event": strings.ToLower(event),
	}
	if tenantID != "" {
		labels["tenant_uuid"] = tenantID
	}
	key, _ := labelsKey(labels)
	counterFor(metricToolGrantEventsTotal)[key]++
}

// RecordAdvisoryLifecycle increments advisory lifecycle counters grouped by severity/status.
func RecordAdvisoryLifecycle(severity, status string) {
	if status == "" {
		return
	}
	metricsLock.Lock()
	defer metricsLock.Unlock()
	labels := map[string]string{
		"status": strings.ToLower(status),
	}
	if severity != "" {
		labels["severity"] = strings.ToLower(severity)
	}
	key, _ := labelsKey(labels)
	counterFor(metricAdvisoryLifecycleTotal)[key]++
}

// RecordAuditReport tracks audit report outcomes.
func RecordAuditReport(status string) {
	if status == "" {
		status = "unknown"
	}
	metricsLock.Lock()
	defer metricsLock.Unlock()
	key, _ := labelsKey(map[string]string{
		"status": strings.ToLower(status),
	})
	counterFor(metricAuditReportStatusTotal)[key]++
}

// UpdateOpenAdvisories resets the advisory gauge using the supplied counts.
func UpdateOpenAdvisories(counts map[string]int) {
	metricsLock.Lock()
	defer metricsLock.Unlock()
	gauge := gaugeFor(metricOpenAdvisoriesGauge)
	for key := range gauge {
		delete(gauge, key)
	}
	for severity, count := range counts {
		key, _ := labelsKey(map[string]string{
			"severity": strings.ToLower(severity),
		})
		gauge[key] = float64(count)
	}
}

// RenderMetrics writes security metrics in the Prometheus exposition format.
func RenderMetrics(w io.Writer) {
	metricsLock.RLock()
	defer metricsLock.RUnlock()
	for metric, values := range secCounters {
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
	for metric, values := range secGauges {
		fmt.Fprintf(w, "# HELP %s %s\n", metric, metricDescriptions[metric])
		fmt.Fprintf(w, "# TYPE %s gauge\n", metric)
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

// sortedKeys returns sorted map keys for stable output.
func sortedKeys(m map[string]float64) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
