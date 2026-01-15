package marketplace

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/config"
	dbm "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/marketplace"
	marketobs "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/observability/marketplace"
	"github.com/sirupsen/logrus"
)

// UsageMetricInput captures a single metric within an ingest request.
type UsageMetricInput struct {
	Name  string  `json:"name"`
	Unit  string  `json:"unit"`
	Value float64 `json:"value"`
}

// UsageEnvelopeInput represents an ingest payload item.
type UsageEnvelopeInput struct {
	LicenseID      string             `json:"license_id"`
	PluginID       string             `json:"plugin_id"`
	Metrics        []UsageMetricInput `json:"metrics"`
	TimestampStart time.Time          `json:"timestamp_start"`
	TimestampEnd   time.Time          `json:"timestamp_end"`
	Signature      string             `json:"signature"`
}

// UsageIngestResult summarises ingest outcomes.
type UsageIngestResult struct {
	Accepted   int      `json:"accepted"`
	Duplicates int      `json:"duplicates"`
	Failed     int      `json:"failed"`
	Errors     []string `json:"errors,omitempty"`
}

// UsageIngestService handles raw usage ingest and dispatch to analytics.
type UsageIngestService struct {
	cfg              *config.Config
	usageRepo        UsageDataRepository
	licenseRepo      LicenseRepositoryReader
	analytics        *AnalyticsService
	logger           *logrus.Entry
	payloadThreshold int64
}

// NewUsageIngestService constructs usage ingest service.
func NewUsageIngestService(cfg *config.Config, usageRepo UsageDataRepository, licenseRepo LicenseRepositoryReader, listingRepo ListingRepositoryReader, analytics *AnalyticsService, logger *logrus.Entry) *UsageIngestService {
	_ = listingRepo
	if logger == nil {
		logger = logrus.New().WithField("component", "marketplace_usage_ingest_service")
	}
	threshold := int64(1 << 20)
	if cfg != nil {
		threshold = cfg.IntegrationPayloadThreshold()
	}
	return &UsageIngestService{
		cfg:              cfg,
		usageRepo:        usageRepo,
		licenseRepo:      licenseRepo,
		analytics:        analytics,
		logger:           logger,
		payloadThreshold: threshold,
	}
}

// IngestBatch validates and stores usage envelopes, triggering analytics updates per accepted item.
func (s *UsageIngestService) IngestBatch(ctx context.Context, tenantID string, inputs []UsageEnvelopeInput) (*UsageIngestResult, error) {
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		return nil, errors.New("tenant_uuid is required")
	}
	result := &UsageIngestResult{}
	if len(inputs) == 0 {
		return result, nil
	}

	licenseCache := map[string]*dbm.License{}
	var ingestErr error

	for _, item := range inputs {
		envelope, metrics, err := s.buildEnvelope(tenantID, item)
		if err != nil {
			result.Failed++
			result.Errors = append(result.Errors, err.Error())
			ingestErr = err
			marketobs.RecordUsageIngest("invalid", tenantID, len(item.Metrics))
			continue
		}

		inserted, err := s.usageRepo.InsertEnvelopes(ctx, tenantID, []*dbm.UsageEnvelope{envelope})
		if err != nil {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("persist envelope %s: %v", envelope.Checksum, err))
			ingestErr = err
			marketobs.RecordUsageIngest("error", tenantID, len(metrics))
			continue
		}
		if inserted == 0 {
			result.Duplicates++
			marketobs.RecordUsageIngest("duplicate", tenantID, len(metrics))
			continue
		}

		license, ok := licenseCache[envelope.LicenseID]
		if !ok {
			license, err = s.licenseRepo.GetLicense(ctx, tenantID, envelope.LicenseID)
			if err != nil {
				result.Failed++
				result.Errors = append(result.Errors, fmt.Sprintf("fetch license %s: %v", envelope.LicenseID, err))
				ingestErr = err
				marketobs.RecordUsageIngest("error", tenantID, len(metrics))
				continue
			}
			licenseCache[envelope.LicenseID] = license
		}

		if err := s.analytics.RecordEnvelope(ctx, tenantID, license, nil, nil, envelope, metrics); err != nil {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("aggregate usage %s: %v", envelope.Checksum, err))
			ingestErr = err
			marketobs.RecordUsageIngest("error", tenantID, len(metrics))
			continue
		}

		result.Accepted++
		marketobs.RecordUsageIngest("accepted", tenantID, len(metrics))
		marketobs.ObserveUsageLag(tenantID, time.Since(envelope.TimestampEnd))
	}

	if ingestErr != nil && result.Accepted == 0 {
		return result, ingestErr
	}
	return result, nil
}

func (s *UsageIngestService) buildEnvelope(tenantID string, input UsageEnvelopeInput) (*dbm.UsageEnvelope, []dbm.UsageMetric, error) {
	licenseID := strings.TrimSpace(input.LicenseID)
	pluginID := strings.TrimSpace(input.PluginID)
	signature := strings.TrimSpace(input.Signature)

	if licenseID == "" || pluginID == "" || signature == "" {
		return nil, nil, errors.New("license_id, plugin_id and signature are required")
	}
	if len(input.Metrics) == 0 {
		return nil, nil, errors.New("metrics payload required")
	}

	start := input.TimestampStart.UTC()
	end := input.TimestampEnd.UTC()
	if end.Before(start) {
		end = start
	}

	metrics := make([]dbm.UsageMetric, 0, len(input.Metrics))
	for _, metric := range input.Metrics {
		name := strings.TrimSpace(metric.Name)
		if name == "" {
			return nil, nil, errors.New("metric name required")
		}
		metrics = append(metrics, dbm.UsageMetric{
			Name:  name,
			Unit:  strings.TrimSpace(metric.Unit),
			Value: metric.Value,
		})
	}

	checksum := dbm.ComputeUsageChecksum(tenantID, licenseID, signature, metrics, start, end)
	envelope := &dbm.UsageEnvelope{
		TenantUuid:     tenantID,
		LicenseID:      licenseID,
		PluginID:       pluginID,
		TimestampStart: start,
		TimestampEnd:   end,
		Signature:      signature,
		Checksum:       checksum,
		IngestStatus:   dbm.UsageIngestStatusProcessed,
		IngestedAt:     time.Now().UTC(),
	}
	if err := envelope.EncodeMetrics(metrics); err != nil {
		return nil, nil, fmt.Errorf("encode metrics: %w", err)
	}
	if int64(len(envelope.Metrics)) > s.payloadThreshold {
		return nil, nil, fmt.Errorf("metrics payload exceeds threshold (%d bytes)", s.payloadThreshold)
	}
	return envelope, metrics, nil
}
