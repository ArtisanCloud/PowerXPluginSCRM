package marketplace

import (
	"encoding/json"
	"strings"
	"time"

	dbm "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/marketplace"
	"github.com/sirupsen/logrus"
)

const (
	eventLicenseRenewalScheduled = "marketplace.license.renewal.scheduled"
	eventLicenseRenewalDue       = "marketplace.license.renewal.due"
	eventUsageSpikeDetected      = "marketplace.usage.spike.detected"
)

// EmitLicenseRenewalScheduled records when a renewal reminder is queued.
func EmitLicenseRenewalScheduled(logger *logrus.Entry, license *dbm.License, scheduledAt time.Time, channels []string) {
	emitLicenseReminderEvent(logger, eventLicenseRenewalScheduled, license, scheduledAt, channels)
}

// EmitLicenseRenewalDue emits that a renewal reminder is executing.
func EmitLicenseRenewalDue(logger *logrus.Entry, license *dbm.License, scheduledAt time.Time, channels []string) {
	emitLicenseReminderEvent(logger, eventLicenseRenewalDue, license, scheduledAt, channels)
}

// EmitUsageSpikeDetected records a usage spike event for observability.
func EmitUsageSpikeDetected(logger *logrus.Entry, tenantID, licenseID, metric string, delta float64) {
	if logger == nil {
		return
	}
	payload := map[string]any{
		"event":       eventUsageSpikeDetected,
		"tenant_uuid": tenantID,
		"license_id":  licenseID,
		"metric":      metric,
		"delta":       delta,
		"emitted_at":  time.Now().UTC().Format(time.RFC3339),
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		logger.WithError(err).Warn("failed to marshal usage spike event payload")
		return
	}
	logger.WithField("usage_event", string(raw)).Info("usage spike detected")
}

func emitLicenseReminderEvent(logger *logrus.Entry, event string, license *dbm.License, scheduledAt time.Time, channels []string) {
	if logger == nil || license == nil {
		return
	}
	if scheduledAt.IsZero() {
		scheduledAt = time.Now()
	}
	payload := map[string]interface{}{
		"event":          event,
		"license_id":     license.ID,
		"tenant_uuid":    license.TenantUuid,
		"listing_id":     license.ListingID,
		"status":         license.Status,
		"plan_id":        license.PlanID,
		"scheduled_at":   scheduledAt.UTC().Format(time.RFC3339),
		"expires_at":     license.ExpiresAt.UTC().Format(time.RFC3339),
		"channels":       strings.Join(channels, ","),
		"days_to_expiry": time.Until(license.ExpiresAt).Hours() / 24,
	}
	if license.OfflineUntil != nil && !license.OfflineUntil.IsZero() {
		payload["offline_until"] = license.OfflineUntil.UTC().Format(time.RFC3339)
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		logger.WithError(err).Warn("failed to marshal license renewal event payload")
		return
	}
	logger.WithField("license_event", string(raw)).Info("license renewal reminder event")
}
