package marketplace

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	metricLicenseVerifyLatency     = "powerx_marketplace_license_verify_seconds"
	metricUsageIngestTotal         = "powerx_marketplace_usage_ingest_total"
	metricUsageIngestLagSeconds    = "powerx_marketplace_usage_ingest_lag_seconds"
	metricRevenueGeneratedTotal    = "powerx_marketplace_revenue_generated_total"
	metricTaxProviderErrors        = "powerx_marketplace_tax_provider_errors_total"
	metricListingSubmissionLatency = "powerx_marketplace_listing_submission_seconds"
	metricListingSubmissionTotal   = "powerx_marketplace_listing_submission_total"
)

var (
	mtx sync.RWMutex

	histBuckets = []float64{0.05, 0.1, 0.25, 0.5, 1, 2, 5, 10}
	histograms  = map[string]map[string][]float64{
		metricLicenseVerifyLatency:     {},
		metricUsageIngestLagSeconds:    {},
		metricListingSubmissionLatency: {},
	}

	counters = map[string]map[string]float64{
		metricUsageIngestTotal:       {},
		metricRevenueGeneratedTotal:  {},
		metricTaxProviderErrors:      {},
		metricListingSubmissionTotal: {},
	}

	metricHelp = map[string]string{
		metricLicenseVerifyLatency:     "License verification latency histogram grouped by result, provider, tenant",
		metricUsageIngestTotal:         "Usage ingest batch outcomes grouped by result and tenant",
		metricUsageIngestLagSeconds:    "Usage ingest lag histogram grouped by tenant",
		metricRevenueGeneratedTotal:    "Revenue generated totals grouped by tenant and currency",
		metricTaxProviderErrors:        "Tax provider error totals grouped by provider and code",
		metricListingSubmissionLatency: "Listing submission latency histogram grouped by result and tenant",
		metricListingSubmissionTotal:   "Listing submission totals grouped by status and tenant",
	}
)

// ObserveLicenseVerification records license verification latency samples.
func ObserveLicenseVerification(result, provider, tenant string, duration time.Duration) {
	if duration < 0 {
		duration = 0
	}
	labels := labelKey(map[string]string{
		"result":   normalize(result),
		"provider": normalize(provider),
		"tenant":   normalize(tenant),
	})
	recordHistogram(metricLicenseVerifyLatency, labels, duration.Seconds())
}

// RecordUsageIngest increments usage ingest counters.
func RecordUsageIngest(result, tenant string, batchSize int) {
	if batchSize <= 0 {
		batchSize = 1
	}
	labels := labelKey(map[string]string{
		"result": normalize(result),
		"tenant": normalize(tenant),
	})
	increment(metricUsageIngestTotal, labels, float64(batchSize))
}

// ObserveUsageLag records lag between envelope timestamp and ingestion completion.
func ObserveUsageLag(tenant string, lag time.Duration) {
	if lag < 0 {
		lag = 0
	}
	labels := labelKey(map[string]string{
		"tenant": normalize(tenant),
	})
	recordHistogram(metricUsageIngestLagSeconds, labels, lag.Seconds())
}

// RecordRevenueGenerated increments revenue counters in settlement currency.
func RecordRevenueGenerated(tenant, currency string, amount float64) {
	if amount <= 0 {
		return
	}
	labels := labelKey(map[string]string{
		"tenant":   normalize(tenant),
		"currency": strings.ToUpper(normalize(currency)),
	})
	increment(metricRevenueGeneratedTotal, labels, amount)
}

// IncrementTaxProviderError increments tax provider error counters.
func IncrementTaxProviderError(provider, code string) {
	labels := labelKey(map[string]string{
		"provider": normalize(provider),
		"code":     normalize(code),
	})
	increment(metricTaxProviderErrors, labels, 1)
}

// ObserveListingSubmission records listing submission durations and increments SLA counters.
func ObserveListingSubmission(result, tenant string, duration time.Duration) {
	if duration < 0 {
		duration = 0
	}
	labels := labelKey(map[string]string{
		"result": normalize(result),
		"tenant": normalize(tenant),
	})
	recordHistogram(metricListingSubmissionLatency, labels, duration.Seconds())

	status := normalize(result)
	if duration > 3*time.Minute {
		status = "timeout"
	}
	counterLabels := labelKey(map[string]string{
		"status": status,
		"tenant": normalize(tenant),
	})
	increment(metricListingSubmissionTotal, counterLabels, 1)
}

// RenderMetrics outputs metrics using Prometheus exposition format.
func RenderMetrics(w io.Writer) {
	mtx.RLock()
	defer mtx.RUnlock()

	renderHistograms(w)
	renderCounters(w)
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
		counts = make([]float64, len(histBuckets)+2) // buckets + +Inf + sum
		hist[labels] = counts
	}
	bucketIndex := len(histBuckets) // default +Inf
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
			cumulative += counts[len(histBuckets)]
			fmt.Fprintf(w, "%s_bucket{%s,le=\"+Inf\"} %g\n", metric, labelKey, cumulative)
			fmt.Fprintf(w, "%s_sum{%s} %g\n", metric, labelKey, counts[len(histBuckets)+1])
			fmt.Fprintf(w, "%s_count{%s} %g\n", metric, labelKey, cumulative)
		}
	}
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

func histogramFor(metric string) map[string][]float64 {
	hist := histograms[metric]
	if hist == nil {
		hist = map[string][]float64{}
		histograms[metric] = hist
	}
	return hist
}

func counterFor(metric string) map[string]float64 {
	counter := counters[metric]
	if counter == nil {
		counter = map[string]float64{}
		counters[metric] = counter
	}
	return counter
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

// ResetMetrics clears recorded metrics; intended for tests.
func ResetMetrics() {
	mtx.Lock()
	defer mtx.Unlock()
	for k := range histograms {
		histograms[k] = map[string][]float64{}
	}
	for k := range counters {
		counters[k] = map[string]float64{}
	}
}
