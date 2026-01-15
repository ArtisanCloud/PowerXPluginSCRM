package integration

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	metricEnvelopesTotal     = "powerx_integration_envelopes_total"
	metricIdempotencyTotal   = "powerx_integration_idempotency_events_total"
	metricWebhookAttempts    = "powerx_integration_webhook_attempts_total"
	metricWebhookLatency     = "powerx_integration_webhook_delivery_seconds"
	metricSecretsRotationDue = "powerx_integration_secrets_rotations_due"
)

var (
	mtx sync.RWMutex

	counters = map[string]map[string]float64{
		metricEnvelopesTotal:   {},
		metricIdempotencyTotal: {},
		metricWebhookAttempts:  {},
	}

	gauges = map[string]map[string]float64{
		metricSecretsRotationDue: {},
	}

	histBuckets = []float64{0.05, 0.1, 0.25, 0.5, 1, 2, 5, 10}
	histograms  = map[string]map[string][]float64{
		metricWebhookLatency: {},
	}

	metricHelp = map[string]string{
		metricEnvelopesTotal:     "Total envelopes processed grouped by channel and result",
		metricIdempotencyTotal:   "Idempotency outcomes grouped by result",
		metricWebhookAttempts:    "Webhook delivery attempts grouped by status and tenant",
		metricWebhookLatency:     "Webhook delivery latency seconds histogram grouped by status",
		metricSecretsRotationDue: "Secrets pending rotation grouped by urgency window",
	}
)

// RecordEnvelope increments the envelope counter.
func RecordEnvelope(channel, result string) {
	labels := labelKey(map[string]string{
		"channel": normalize(channel),
		"result":  normalize(result),
	})
	increment(metricEnvelopesTotal, labels, 1)
}

// RecordIdempotency increments idempotency outcomes.
func RecordIdempotency(outcome string) {
	labels := labelKey(map[string]string{
		"outcome": normalize(outcome),
	})
	increment(metricIdempotencyTotal, labels, 1)
}

// RecordWebhookAttempt tracks webhook attempts.
func RecordWebhookAttempt(status, tenantID string) {
	labels := labelKey(map[string]string{
		"status":    normalize(status),
		"tenant_uuid": normalize(tenantID),
	})
	increment(metricWebhookAttempts, labels, 1)
}

// ObserveWebhookLatency records webhook latency samples.
func ObserveWebhookLatency(status string, duration time.Duration) {
	if duration < 0 {
		duration = 0
	}
	labels := labelKey(map[string]string{
		"status": normalize(status),
	})
	recordHistogram(metricWebhookLatency, labels, duration.Seconds())
}

// SetSecretsDue updates gauge value for pending rotations.
func SetSecretsDue(window string, count float64) {
	mtx.Lock()
	defer mtx.Unlock()
	key := labelKey(map[string]string{
		"window": normalize(window),
	})
	gaugeFor(metricSecretsRotationDue)[key] = count
}

// RenderMetrics outputs metrics using Prometheus exposition format.
func RenderMetrics(w io.Writer) {
	mtx.RLock()
	defer mtx.RUnlock()

	renderCounters(w)
	renderGauges(w)
	renderHistograms(w)
}

func renderCounters(w io.Writer) {
	for metric, values := range counters {
		fmt.Fprintf(w, "# HELP %s %s\n", metric, metricHelp[metric])
		fmt.Fprintf(w, "# TYPE %s counter\n", metric)
		for _, key := range sortedKeys(values) {
			if key == "" {
				fmt.Fprintf(w, "%s %g\n", metric, values[key])
				continue
			}
			fmt.Fprintf(w, "%s{%s} %g\n", metric, key, values[key])
		}
	}
}

func renderGauges(w io.Writer) {
	for metric, values := range gauges {
		fmt.Fprintf(w, "# HELP %s %s\n", metric, metricHelp[metric])
		fmt.Fprintf(w, "# TYPE %s gauge\n", metric)
		for _, key := range sortedKeys(values) {
			if key == "" {
				fmt.Fprintf(w, "%s %g\n", metric, values[key])
				continue
			}
			fmt.Fprintf(w, "%s{%s} %g\n", metric, key, values[key])
		}
	}
}

func renderHistograms(w io.Writer) {
	for metric, buckets := range histograms {
		fmt.Fprintf(w, "# HELP %s %s\n", metric, metricHelp[metric])
		fmt.Fprintf(w, "# TYPE %s histogram\n", metric)
		for _, labelKey := range sortedHistKeys(buckets) {
			counts := buckets[labelKey]
			var cumulative float64
			for i, upper := range histBuckets {
				cumulative += counts[i]
				fmt.Fprintf(w, "%s_bucket{%s,le=\"%.2f\"} %g\n", metric, labelKey, upper, cumulative)
			}
			// +Inf bucket
			cumulative += counts[len(histBuckets)]
			fmt.Fprintf(w, "%s_bucket{%s,le=\"+Inf\"} %g\n", metric, labelKey, cumulative)
			fmt.Fprintf(w, "%s_sum{%s} %g\n", metric, labelKey, counts[len(histBuckets)+1])
			fmt.Fprintf(w, "%s_count{%s} %g\n", metric, labelKey, cumulative)
		}
	}
}

func increment(metric, labels string, delta float64) {
	mtx.Lock()
	defer mtx.Unlock()
	counterFor(metric)[labels] += delta
}

func recordHistogram(metric, labels string, value float64) {
	mtx.Lock()
	defer mtx.Unlock()
	hist := histogramFor(metric)
	counts := hist[labels]
	if counts == nil {
		counts = make([]float64, len(histBuckets)+2) // buckets + inf + sum
		hist[labels] = counts
	}
	bucketIndex := len(histBuckets) // +Inf
	for i, upper := range histBuckets {
		if value <= upper {
			bucketIndex = i
			break
		}
	}
	counts[bucketIndex]++
	counts[len(histBuckets)]++          // +Inf bucket
	counts[len(histBuckets)+1] += value // sum
}

func normalize(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return "unknown"
	}
	return value
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

func counterFor(metric string) map[string]float64 {
	c := counters[metric]
	if c == nil {
		c = map[string]float64{}
		counters[metric] = c
	}
	return c
}

func gaugeFor(metric string) map[string]float64 {
	g := gauges[metric]
	if g == nil {
		g = map[string]float64{}
		gauges[metric] = g
	}
	return g
}

func histogramFor(metric string) map[string][]float64 {
	h := histograms[metric]
	if h == nil {
		h = map[string][]float64{}
		histograms[metric] = h
	}
	return h
}

func sortedKeys(m map[string]float64) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func sortedHistKeys(m map[string][]float64) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
