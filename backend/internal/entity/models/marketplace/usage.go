package marketplace

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	"gorm.io/datatypes"
)

// AggregationWindow defines supported aggregation granularities.
type AggregationWindow string

const (
	AggregationWindowHour  AggregationWindow = "hour"
	AggregationWindowDay   AggregationWindow = "day"
	AggregationWindowMonth AggregationWindow = "month"
)

const (
	UsageIngestStatusPending   = "pending"
	UsageIngestStatusProcessed = "processed"
	UsageIngestStatusReplayed  = "replayed"
)

// UsageMetric represents a single metric sample within an envelope.
type UsageMetric struct {
	Name  string  `json:"name"`
	Unit  string  `json:"unit"`
	Value float64 `json:"value"`
}

// UsageEnvelope captures raw usage data batches submitted by SDK or integrations.
type UsageEnvelope struct {
	ID             string         `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	TenantUuid     string         `gorm:"column:tenant_uuid;type:uuid;not null;index" json:"tenant_uuid"`
	LicenseID      string         `gorm:"column:license_id;type:uuid;not null;index" json:"license_id"`
	PluginID       string         `gorm:"column:plugin_id;type:text;not null;index" json:"plugin_id"`
	Metrics        datatypes.JSON `gorm:"column:metrics;type:jsonb;not null" json:"metrics"`
	TimestampStart time.Time      `gorm:"column:timestamp_start;type:timestamptz;not null" json:"timestamp_start"`
	TimestampEnd   time.Time      `gorm:"column:timestamp_end;type:timestamptz;not null" json:"timestamp_end"`
	Signature      string         `gorm:"column:signature;type:text;not null" json:"signature"`
	Checksum       string         `gorm:"column:checksum;type:text;not null;uniqueIndex:uq_usage_checksum" json:"checksum"`
	IngestStatus   string         `gorm:"column:ingest_status;type:text;not null;default:'processed'" json:"ingest_status"`
	IngestedAt     time.Time      `gorm:"column:ingested_at;type:timestamptz;autoCreateTime" json:"ingested_at"`
	CreatedAt      time.Time      `gorm:"column:created_at;type:timestamptz;autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time      `gorm:"column:updated_at;type:timestamptz;autoUpdateTime" json:"updated_at"`
}

// TableName implements gorm tabler.
func (*UsageEnvelope) TableName() string {
	return models.S(models.TableMarketplaceUsageEnvelopes)
}

// UsageAggregate stores aggregated usage values per window/metric for dashboards.
type UsageAggregate struct {
	ID         string            `gorm:"column:id;type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	TenantUuid string            `gorm:"column:tenant_uuid;type:uuid;not null;index" json:"tenant_uuid"`
	LicenseID  string            `gorm:"column:license_id;type:uuid;not null;index" json:"license_id"`
	Metric     string            `gorm:"column:metric;type:text;not null;index:idx_usage_agg_metric" json:"metric"`
	Window     AggregationWindow `gorm:"column:window;type:text;not null;index:idx_usage_agg_metric" json:"window"`
	TimeBucket time.Time         `gorm:"column:time_bucket;type:timestamptz;not null;index:idx_usage_agg_metric" json:"time_bucket"`
	Total      float64           `gorm:"column:total;type:numeric(20,4);not null" json:"total"`
	Delta      float64           `gorm:"column:delta;type:numeric(20,4);not null" json:"delta"`
	Currency   string            `gorm:"column:currency;type:text" json:"currency,omitempty"`
	Revenue    float64           `gorm:"column:revenue;type:numeric(18,4);not null;default:0" json:"revenue"`
	CreatedAt  time.Time         `gorm:"column:created_at;type:timestamptz;autoCreateTime" json:"created_at"`
	UpdatedAt  time.Time         `gorm:"column:updated_at;type:timestamptz;autoUpdateTime" json:"updated_at"`
}

// TableName implements gorm tabler.
func (*UsageAggregate) TableName() string {
	return models.S(models.TableMarketplaceUsageAggregates)
}

// DecodeMetrics returns metrics slice representation.
func (e *UsageEnvelope) DecodeMetrics() ([]UsageMetric, error) {
	if len(e.Metrics) == 0 {
		return nil, nil
	}
	var metrics []UsageMetric
	if err := json.Unmarshal(e.Metrics, &metrics); err != nil {
		return nil, err
	}
	return metrics, nil
}

// EncodeMetrics serializes metrics into JSON payload.
func (e *UsageEnvelope) EncodeMetrics(metrics []UsageMetric) error {
	if len(metrics) == 0 {
		e.Metrics = datatypes.JSON([]byte("[]"))
		return nil
	}
	data, err := json.Marshal(metrics)
	if err != nil {
		return err
	}
	e.Metrics = datatypes.JSON(data)
	return nil
}

// ComputeUsageChecksum returns a deterministic checksum for idempotency.
func ComputeUsageChecksum(tenantID, licenseID, signature string, metrics []UsageMetric, tsStart, tsEnd time.Time) string {
	h := sha256.New()
	parts := []string{
		strings.TrimSpace(strings.ToLower(tenantID)),
		strings.TrimSpace(strings.ToLower(licenseID)),
		strings.TrimSpace(signature),
		tsStart.UTC().Format(time.RFC3339Nano),
		tsEnd.UTC().Format(time.RFC3339Nano),
	}
	h.Write([]byte(strings.Join(parts, "|")))
	if len(metrics) > 0 {
		sorted := make([]UsageMetric, len(metrics))
		copy(sorted, metrics)
		sort.Slice(sorted, func(i, j int) bool {
			if sorted[i].Name == sorted[j].Name {
				return sorted[i].Unit < sorted[j].Unit
			}
			return sorted[i].Name < sorted[j].Name
		})
		raw, _ := json.Marshal(sorted)
		h.Write(raw)
	}
	return hex.EncodeToString(h.Sum(nil))
}
