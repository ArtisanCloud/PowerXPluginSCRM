package integration

import (
	"context"
	"time"

	model "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/integration"
	repo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/integration"
	obs "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/observability/integration"
	service "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/integration"
	"github.com/sirupsen/logrus"
)

// WebhookRetryWorker implements background retry logic for webhook deliveries.
type WebhookRetryWorker struct {
	service       *service.WebhookService
	subscriptions *repo.WebhookSubscriptionRepository
	attempts      *repo.DeliveryAttemptRepository
	interval      time.Duration
	logger        *logrus.Entry
}

// NewWebhookRetryWorker constructs a retry worker.
func NewWebhookRetryWorker(
	svc *service.WebhookService,
	subRepo *repo.WebhookSubscriptionRepository,
	attemptRepo *repo.DeliveryAttemptRepository,
	interval time.Duration,
	logger *logrus.Entry,
) *WebhookRetryWorker {
	if interval <= 0 {
		interval = time.Minute
	}
	if logger == nil {
		logger = logrus.WithField("component", "integration.webhook_retry_worker")
	}
	return &WebhookRetryWorker{
		service:       svc,
		subscriptions: subRepo,
		attempts:      attemptRepo,
		interval:      interval,
		logger:        logger,
	}
}

// Name returns the job name.
func (w *WebhookRetryWorker) Name() string {
	return "integration.webhook_retry"
}

// Interval returns execution frequency.
func (w *WebhookRetryWorker) Interval() time.Duration {
	return w.interval
}

// Run scans due attempts and schedules next retries or DLQ transitions.
func (w *WebhookRetryWorker) Run(ctx context.Context) error {
	if w.service == nil || w.attempts == nil || w.subscriptions == nil {
		w.logger.Warn("webhook retry worker dependencies not configured; skipping run")
		return nil
	}

	now := time.Now().UTC()
	attempts, err := w.attempts.ListDueForRetry(ctx, now, 100)
	if err != nil {
		return err
	}
	if len(attempts) == 0 {
		return nil
	}

	for _, attempt := range attempts {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		sub, err := w.subscriptions.GetBySubscriptionID(ctx, attempt.SubscriptionID)
		if err != nil {
			w.logger.WithError(err).WithField("attempt_id", attempt.ID).Warn("failed to load subscription for retry")
			continue
		}
		if sub == nil {
			_ = w.service.UpdateAttemptStatus(ctx, attempt.ID, model.AttemptStatusDLQ, attempt.RetryCount, nil, "subscription deleted", "")
			continue
		}

		next, moveToDLQ := w.service.NextRetry(sub, attempt.RetryCount)
		if moveToDLQ {
			if err := w.service.UpdateAttemptStatus(ctx, attempt.ID, model.AttemptStatusDLQ, attempt.RetryCount, nil, attempt.LastError, sub.TenantUuid); err != nil {
				w.logger.WithError(err).WithField("attempt_id", attempt.ID).Warn("failed to mark attempt as DLQ")
				continue
			}
			obs.RecordWebhookAttempt("dlq", sub.TenantUuid)
			continue
		}

		err = w.service.UpdateAttemptStatus(ctx, attempt.ID, model.AttemptStatusRetrying, attempt.RetryCount+1, &next, attempt.LastError, sub.TenantUuid)
		if err != nil {
			w.logger.WithError(err).WithField("attempt_id", attempt.ID).Warn("failed to schedule next retry")
			continue
		}
		obs.RecordWebhookAttempt("retrying", sub.TenantUuid)
	}

	return nil
}
