package integration

import (
	"context"
	"time"

	repo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/integration"
	obs "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/observability/integration"
	service "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/integration"
	"github.com/sirupsen/logrus"
)

// SecretRotationWorker scans for secrets nearing rotation and schedules reminders.
type SecretRotationWorker struct {
	service  *service.SecretService
	repo     *repo.SecretRepository
	logger   *logrus.Entry
	interval time.Duration
}

// NewSecretRotationWorker constructs the worker.
func NewSecretRotationWorker(secretSvc *service.SecretService, secretRepo *repo.SecretRepository, interval time.Duration, logger *logrus.Entry) *SecretRotationWorker {
	if interval <= 0 {
		interval = time.Hour
	}
	if logger == nil {
		logger = logrus.WithField("component", "integration.secret_rotation_worker")
	}
	return &SecretRotationWorker{
		service:  secretSvc,
		repo:     secretRepo,
		logger:   logger,
		interval: interval,
	}
}

// Name returns job name.
func (w *SecretRotationWorker) Name() string {
	return "integration.secret_rotation"
}

// Interval returns run interval.
func (w *SecretRotationWorker) Interval() time.Duration {
	return w.interval
}

// Run fetches due secrets and schedules rotation reminders.
func (w *SecretRotationWorker) Run(ctx context.Context) error {
	if w.repo == nil || w.service == nil {
		w.logger.Warn("secret rotation worker dependencies not configured")
		return w.service.RefreshRotationMetrics(ctx)
	}

	now := time.Now().UTC()
	soon := now.Add(24 * time.Hour)

	dueSecrets, err := w.repo.ListDueForRotation(ctx, now, 0)
	if err != nil {
		return err
	}
	obs.SetSecretsDue("due_now", float64(len(dueSecrets)))

	soonSecrets, err := w.repo.ListDueForRotation(ctx, soon, 0)
	if err != nil {
		return err
	}
	obs.SetSecretsDue("due_24h", float64(len(soonSecrets)))

	for _, secret := range dueSecrets {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		_, err := w.service.RotateSecret(ctx, service.RotateSecretParams{
			TenantUuid: secret.TenantUuid,
			SecretID:   secret.ID,
			Generate:   true,
			Actor:      "system-rotation-worker",
		})
		if err != nil {
			w.logger.WithError(err).WithFields(logrus.Fields{
				"tenant_uuid": secret.TenantUuid,
				"secret_id":   secret.ID,
			}).Warn("failed to schedule automatic rotation")
			continue
		}
	}

	return nil
}
