package marketplace

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// PrivacyService handles GDPR deletion workflows for marketplace usage data.
type PrivacyService struct {
	usageRepo UsageDataRepository
	logger    *logrus.Entry
}

// PurgeResult summarises deletion counts for usage data purge.
type PurgeResult struct {
	EnvelopesDeleted  int `json:"envelopes_deleted"`
	AggregatesDeleted int `json:"aggregates_deleted"`
}

// NewPrivacyService constructs the service.
func NewPrivacyService(usageRepo UsageDataRepository, logger *logrus.Entry) *PrivacyService {
	if logger == nil {
		logger = logrus.New().WithField("component", "marketplace_privacy_service")
	}
	return &PrivacyService{
		usageRepo: usageRepo,
		logger:    logger,
	}
}

// PurgeUsageData removes envelopes and aggregates for a license up to the cutoff timestamp.
func (s *PrivacyService) PurgeUsageData(ctx context.Context, tenantID, licenseID string, cutoff time.Time) (*PurgeResult, error) {
	tenantID = strings.TrimSpace(tenantID)
	licenseID = strings.TrimSpace(licenseID)
	if tenantID == "" || licenseID == "" {
		return nil, errors.New("tenant_uuid and license_id are required")
	}
	if cutoff.IsZero() {
		cutoff = time.Now().UTC()
	}

	envDeleted, err := s.usageRepo.DeleteEnvelopesBefore(ctx, tenantID, licenseID, cutoff)
	if err != nil {
		return nil, err
	}
	aggDeleted, err := s.usageRepo.DeleteAggregatesBefore(ctx, tenantID, licenseID, cutoff)
	if err != nil {
		return nil, err
	}
	if s.logger != nil {
		s.logger.WithFields(logrus.Fields{
			"tenant_uuid":          tenantID,
			"license_id":         licenseID,
			"cutoff":             cutoff.Format(time.RFC3339),
			"envelopes_deleted":  envDeleted,
			"aggregates_deleted": aggDeleted,
		}).Info("purged marketplace usage data")
	}
	return &PurgeResult{EnvelopesDeleted: envDeleted, AggregatesDeleted: aggDeleted}, nil
}
