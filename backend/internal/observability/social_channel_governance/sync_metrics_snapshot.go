package social_channel_governance

import (
	"sort"
	"strings"
	"time"

	model "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/social_channel_governance"
)

type SyncMetricsSnapshot struct {
	Window            string                 `json:"window"`
	Summary           SyncMetricsSummary     `json:"summary"`
	Latency           SyncLatencySummary     `json:"latency"`
	OpenConflictsTopN []SyncTopNItem         `json:"open_conflicts_topn"`
	FailedTopN        []SyncTopNItem         `json:"failed_topn"`
	PerDomain         map[string]DomainStats `json:"per_domain"`
}

type SyncMetricsSummary struct {
	TotalJobs         int     `json:"total_jobs"`
	SuccessJobs       int     `json:"success_jobs"`
	FailedJobs        int     `json:"failed_jobs"`
	RunningJobs       int     `json:"running_jobs"`
	QueuedJobs        int     `json:"queued_jobs"`
	OpenConflicts     int     `json:"open_conflicts"`
	DeadLetters       int     `json:"dead_letters"`
	SuccessRate       float64 `json:"success_rate"`
	FailureRate       float64 `json:"failure_rate"`
	ConflictRate      float64 `json:"conflict_rate"`
	DeadLetterRate    float64 `json:"dead_letter_rate"`
	LastCalculatedAt  string  `json:"last_calculated_at"`
	SampleJobTotal    int     `json:"sample_job_total"`
	SampleDeadLetters int     `json:"sample_dead_letters"`
}

type SyncLatencySummary struct {
	Count int     `json:"count"`
	AvgMS float64 `json:"avg_ms"`
	P50MS float64 `json:"p50_ms"`
	P95MS float64 `json:"p95_ms"`
	MaxMS float64 `json:"max_ms"`
}

type DomainStats struct {
	TotalJobs     int `json:"total_jobs"`
	SuccessJobs   int `json:"success_jobs"`
	FailedJobs    int `json:"failed_jobs"`
	RunningJobs   int `json:"running_jobs"`
	QueuedJobs    int `json:"queued_jobs"`
	OpenConflicts int `json:"open_conflicts"`
	DeadLetters   int `json:"dead_letters"`
}

type SyncTopNItem struct {
	Key   string `json:"key"`
	Count int    `json:"count"`
}

func BuildSyncMetricsSnapshot(
	window string,
	jobs []model.SyncJob,
	conflicts []model.SyncConflict,
	deadLetters []model.SyncDeadLetterItem,
	topN int,
) SyncMetricsSnapshot {
	if topN <= 0 {
		topN = 5
	}
	if topN > 20 {
		topN = 20
	}
	snapshot := SyncMetricsSnapshot{
		Window:    strings.TrimSpace(window),
		PerDomain: map[string]DomainStats{},
	}

	latencies := make([]float64, 0, len(jobs))
	for _, job := range jobs {
		domain := strings.TrimSpace(strings.ToLower(job.Domain))
		if domain == "" {
			domain = "unknown"
		}
		stats := snapshot.PerDomain[domain]
		stats.TotalJobs++
		snapshot.Summary.TotalJobs++
		switch strings.TrimSpace(strings.ToLower(job.Status)) {
		case "success":
			stats.SuccessJobs++
			snapshot.Summary.SuccessJobs++
		case "failed":
			stats.FailedJobs++
			snapshot.Summary.FailedJobs++
		case "running":
			stats.RunningJobs++
			snapshot.Summary.RunningJobs++
		case "queued", "pending":
			stats.QueuedJobs++
			snapshot.Summary.QueuedJobs++
		}
		snapshot.PerDomain[domain] = stats
		if ms, ok := computeLatencyMS(job); ok {
			latencies = append(latencies, ms)
		}
	}

	openConflictCountByDomain := map[string]int{}
	for _, conflict := range conflicts {
		if strings.TrimSpace(strings.ToLower(conflict.Status)) != "open" {
			continue
		}
		domain := strings.TrimSpace(strings.ToLower(conflict.Domain))
		if domain == "" {
			domain = "unknown"
		}
		openConflictCountByDomain[domain]++
		snapshot.Summary.OpenConflicts++
		stats := snapshot.PerDomain[domain]
		stats.OpenConflicts++
		snapshot.PerDomain[domain] = stats
	}

	failureTop := map[string]int{}
	for _, dead := range deadLetters {
		domain := strings.TrimSpace(strings.ToLower(dead.Domain))
		if domain == "" {
			domain = "unknown"
		}
		code := strings.TrimSpace(strings.ToUpper(dead.LastErrorCode))
		if code == "" {
			code = "UNKNOWN"
		}
		failureTop[domain+":"+code]++
		snapshot.Summary.DeadLetters++
		stats := snapshot.PerDomain[domain]
		stats.DeadLetters++
		snapshot.PerDomain[domain] = stats
	}

	snapshot.OpenConflictsTopN = topNFromMap(openConflictCountByDomain, topN)
	snapshot.FailedTopN = topNFromMap(failureTop, topN)
	snapshot.Latency = latencySummary(latencies)

	totalJobs := snapshot.Summary.TotalJobs
	if totalJobs > 0 {
		snapshot.Summary.SuccessRate = float64(snapshot.Summary.SuccessJobs) / float64(totalJobs)
		snapshot.Summary.FailureRate = float64(snapshot.Summary.FailedJobs) / float64(totalJobs)
		snapshot.Summary.ConflictRate = float64(snapshot.Summary.OpenConflicts) / float64(totalJobs)
		snapshot.Summary.DeadLetterRate = float64(snapshot.Summary.DeadLetters) / float64(totalJobs)
	}
	snapshot.Summary.SampleJobTotal = len(jobs)
	snapshot.Summary.SampleDeadLetters = len(deadLetters)
	snapshot.Summary.LastCalculatedAt = time.Now().UTC().Format(time.RFC3339)
	return snapshot
}

func computeLatencyMS(job model.SyncJob) (float64, bool) {
	if job.StartedAt != nil && job.FinishedAt != nil && job.FinishedAt.After(*job.StartedAt) {
		return job.FinishedAt.Sub(*job.StartedAt).Seconds() * 1000, true
	}
	if job.CreatedAt.IsZero() || job.UpdatedAt.IsZero() || !job.UpdatedAt.After(job.CreatedAt) {
		return 0, false
	}
	return job.UpdatedAt.Sub(job.CreatedAt).Seconds() * 1000, true
}

func topNFromMap(in map[string]int, topN int) []SyncTopNItem {
	out := make([]SyncTopNItem, 0, len(in))
	for k, v := range in {
		out = append(out, SyncTopNItem{Key: k, Count: v})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count == out[j].Count {
			return out[i].Key < out[j].Key
		}
		return out[i].Count > out[j].Count
	})
	if len(out) > topN {
		out = out[:topN]
	}
	return out
}

func latencySummary(values []float64) SyncLatencySummary {
	out := SyncLatencySummary{}
	if len(values) == 0 {
		return out
	}
	sorted := append([]float64{}, values...)
	sort.Float64s(sorted)
	out.Count = len(sorted)
	sum := 0.0
	for _, v := range sorted {
		sum += v
		if v > out.MaxMS {
			out.MaxMS = v
		}
	}
	out.AvgMS = sum / float64(out.Count)
	out.P50MS = percentile(sorted, 50)
	out.P95MS = percentile(sorted, 95)
	return out
}

func percentile(sorted []float64, p int) float64 {
	if len(sorted) == 0 {
		return 0
	}
	if p <= 0 {
		return sorted[0]
	}
	if p >= 100 {
		return sorted[len(sorted)-1]
	}
	idx := int(float64(len(sorted)-1) * (float64(p) / 100.0))
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}
