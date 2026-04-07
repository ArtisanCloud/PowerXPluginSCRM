package social_channel_governance

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
)

const (
	metricOpenWorkCallbackTotal         = "plugin_openwork_callback_total"
	metricOpenWorkIdempotentHitTotal    = "plugin_openwork_callback_idempotent_hits_total"
	metricOpenWorkAuthCompleteTotal     = "plugin_openwork_auth_complete_total"
	metricOpenWorkCallbackAckLatencyMS  = "plugin_openwork_callback_ack_latency_ms"
	metricOpenWorkAuthCompleteLatencyMS = "plugin_openwork_auth_complete_latency_ms"
)

var (
	openWorkMetricsMu sync.RWMutex
	openWorkCounters  = map[string]map[string]float64{}
	openWorkHists     = map[string]*histSeries{
		metricOpenWorkCallbackAckLatencyMS: {
			Buckets: []float64{100, 300, 500, 1000, 2000, 5000},
			Labels:  map[string][]float64{},
		},
		metricOpenWorkAuthCompleteLatencyMS: {
			Buckets: []float64{50, 100, 300, 500, 1000, 2000, 5000, 10000},
			Labels:  map[string][]float64{},
		},
	}
)

type histSeries struct {
	Buckets []float64
	Labels  map[string][]float64
	Count   map[string]float64
	Sum     map[string]float64
}

func normalizeLabelValue(v string) string {
	v = strings.ToLower(strings.TrimSpace(v))
	if v == "" {
		return "unknown"
	}
	return v
}

func labelsKey(labels map[string]string) string {
	if len(labels) == 0 {
		return ""
	}
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	pairs := make([]string, 0, len(keys))
	for _, k := range keys {
		pairs = append(pairs, fmt.Sprintf(`%s="%s"`, k, labels[k]))
	}
	return strings.Join(pairs, ",")
}

func ensureOpenWorkCounter(metric string) map[string]float64 {
	if openWorkCounters[metric] == nil {
		openWorkCounters[metric] = map[string]float64{}
	}
	return openWorkCounters[metric]
}

func ensureHistCount(hist *histSeries) {
	if hist.Count == nil {
		hist.Count = map[string]float64{}
	}
	if hist.Sum == nil {
		hist.Sum = map[string]float64{}
	}
}

func RecordOpenWorkCallback(result, eventType string) {
	openWorkMetricsMu.Lock()
	defer openWorkMetricsMu.Unlock()
	labels := labelsKey(map[string]string{
		"result":     normalizeLabelValue(result),
		"event_type": normalizeLabelValue(eventType),
	})
	ensureOpenWorkCounter(metricOpenWorkCallbackTotal)[labels]++
}

func RecordOpenWorkIdempotentHit(reason, eventType string) {
	openWorkMetricsMu.Lock()
	defer openWorkMetricsMu.Unlock()
	labels := labelsKey(map[string]string{
		"reason":     normalizeLabelValue(reason),
		"event_type": normalizeLabelValue(eventType),
	})
	ensureOpenWorkCounter(metricOpenWorkIdempotentHitTotal)[labels]++
}

func RecordOpenWorkAuthComplete(result, reason string) {
	openWorkMetricsMu.Lock()
	defer openWorkMetricsMu.Unlock()
	labels := labelsKey(map[string]string{
		"result": normalizeLabelValue(result),
		"reason": normalizeLabelValue(reason),
	})
	ensureOpenWorkCounter(metricOpenWorkAuthCompleteTotal)[labels]++
}

func observeOpenWorkHistogram(metric string, valueMs float64, labels map[string]string) {
	hist, ok := openWorkHists[metric]
	if !ok {
		return
	}
	ensureHistCount(hist)
	key := labelsKey(labels)
	if hist.Labels[key] == nil {
		hist.Labels[key] = make([]float64, len(hist.Buckets))
	}
	for i, b := range hist.Buckets {
		if valueMs <= b {
			hist.Labels[key][i]++
		}
	}
	hist.Count[key]++
	hist.Sum[key] += valueMs
}

func ObserveOpenWorkCallbackAckLatency(ms float64, eventType string) {
	openWorkMetricsMu.Lock()
	defer openWorkMetricsMu.Unlock()
	observeOpenWorkHistogram(metricOpenWorkCallbackAckLatencyMS, ms, map[string]string{
		"event_type": normalizeLabelValue(eventType),
	})
}

func ObserveOpenWorkAuthCompleteLatency(ms float64, result string) {
	openWorkMetricsMu.Lock()
	defer openWorkMetricsMu.Unlock()
	observeOpenWorkHistogram(metricOpenWorkAuthCompleteLatencyMS, ms, map[string]string{
		"result": normalizeLabelValue(result),
	})
}

func RenderMetrics(w io.Writer) {
	openWorkMetricsMu.RLock()
	defer openWorkMetricsMu.RUnlock()

	for metric, series := range openWorkCounters {
		fmt.Fprintf(w, "# TYPE %s counter\n", metric)
		keys := make([]string, 0, len(series))
		for k := range series {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Fprintf(w, "%s{%s} %g\n", metric, k, series[k])
		}
	}

	for metric, hist := range openWorkHists {
		fmt.Fprintf(w, "# TYPE %s histogram\n", metric)
		labelKeys := make([]string, 0, len(hist.Labels))
		for key := range hist.Labels {
			labelKeys = append(labelKeys, key)
		}
		sort.Strings(labelKeys)
		for _, labelKey := range labelKeys {
			for i, bucket := range hist.Buckets {
				fmt.Fprintf(w, "%s_bucket{%s,le=\"%g\"} %g\n", metric, labelKey, bucket, hist.Labels[labelKey][i])
			}
			fmt.Fprintf(w, "%s_bucket{%s,le=\"+Inf\"} %g\n", metric, labelKey, hist.Count[labelKey])
			fmt.Fprintf(w, "%s_sum{%s} %g\n", metric, labelKey, hist.Sum[labelKey])
			fmt.Fprintf(w, "%s_count{%s} %g\n", metric, labelKey, hist.Count[labelKey])
		}
	}
}
