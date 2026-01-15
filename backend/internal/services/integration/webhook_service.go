package integration

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/config"
	model "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/integration"
	repo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/integration"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/logger"
	obs "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/observability/integration"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/crypto"
	"github.com/sirupsen/logrus"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// WebhookService orchestrates subscription management and delivery bookkeeping.
type WebhookService struct {
	cfg           *config.Config
	subscriptions *repo.WebhookSubscriptionRepository
	attempts      *repo.DeliveryAttemptRepository
	approvalSvc   *ApprovalService
	log           *logrus.Entry
	now           func() time.Time
}

// NewWebhookService constructs a webhook service instance.
func NewWebhookService(
	cfg *config.Config,
	subRepo *repo.WebhookSubscriptionRepository,
	attemptRepo *repo.DeliveryAttemptRepository,
	approvalRepo *repo.ApprovalRepository,
) *WebhookService {
	var approvalSvc *ApprovalService
	if approvalRepo != nil {
		approvalSvc = NewApprovalService(approvalRepo, logger.WithField("component", "integration.webhook_approval"))
	}
	return &WebhookService{
		cfg:           cfg,
		subscriptions: subRepo,
		attempts:      attemptRepo,
		approvalSvc:   approvalSvc,
		log:           logger.WithField("component", "integration.webhook_service"),
		now:           time.Now,
	}
}

// CreateSubscriptionParams captures creation parameters.
type CreateSubscriptionParams struct {
	TenantUuid      string
	EventType       string
	TargetURL       string
	SecretPlaintext string
	RetryPolicy     []int
	Metadata        map[string]any
	Status          string
}

// UpdateSubscriptionParams captures updatable fields.
type UpdateSubscriptionParams struct {
	TenantUuid     string
	SubscriptionID string
	TargetURL      *string
	Status         *string
	NewSecretPlain *string
	RetryPolicy    []int
	Metadata       map[string]any
}

// DeliveryResultParams captures attempt recording inputs.
type DeliveryResultParams struct {
	SubscriptionID string
	EnvelopeID     string
	Status         string
	LastError      string
	NextRetryAt    *time.Time
	RetryCount     int
	Payload        any
	TenantUuid     string
}

// CreateSubscription registers or updates a subscription entry.
func (s *WebhookService) CreateSubscription(ctx context.Context, p CreateSubscriptionParams) (*model.WebhookSubscription, error) {
	if s.subscriptions == nil {
		return nil, errors.New("subscription repository not configured")
	}
	sub := &model.WebhookSubscription{
		TenantUuid: strings.TrimSpace(p.TenantUuid),
		EventType:  strings.TrimSpace(p.EventType),
		TargetURL:  strings.TrimSpace(p.TargetURL),
		Status:     strings.ToUpper(strings.TrimSpace(p.Status)),
		Metadata:   datatypes.JSONMap{},
	}
	if sub.Status == "" {
		sub.Status = model.WebhookStatusActive
	}
	if len(p.Metadata) > 0 {
		sub.Metadata = datatypes.JSONMap(p.Metadata)
	}
	if len(p.RetryPolicy) > 0 {
		raw, err := json.Marshal(p.RetryPolicy)
		if err != nil {
			return nil, err
		}
		sub.RetryPolicy = datatypes.JSON(raw)
	}

	if strings.TrimSpace(p.SecretPlaintext) != "" {
		secret, err := s.encryptSecret(sub.TenantUuid, sub.EventType, sub.TargetURL, p.SecretPlaintext)
		if err != nil {
			return nil, err
		}
		sub.Secret = secret
	}

	saved, err := s.subscriptions.Upsert(ctx, sub)
	if err != nil {
		return nil, err
	}
	s.submitApproval(ctx, saved.TenantUuid, saved.ID, "create", sanitizeSubscription(saved))
	return sanitizeSubscription(saved), nil
}

// UpdateSubscription modifies mutable attributes for a subscription.
func (s *WebhookService) UpdateSubscription(ctx context.Context, p UpdateSubscriptionParams) (*model.WebhookSubscription, error) {
	if s.subscriptions == nil {
		return nil, errors.New("subscription repository not configured")
	}
	sub, err := s.subscriptions.GetByID(ctx, p.TenantUuid, p.SubscriptionID)
	if err != nil {
		return nil, err
	}
	if sub == nil {
		return nil, gorm.ErrRecordNotFound
	}

	if p.TargetURL != nil {
		sub.TargetURL = strings.TrimSpace(*p.TargetURL)
	}
	if p.Status != nil {
		sub.Status = strings.ToUpper(strings.TrimSpace(*p.Status))
	}
	if p.NewSecretPlain != nil && strings.TrimSpace(*p.NewSecretPlain) != "" {
		secret, err := s.encryptSecret(sub.TenantUuid, sub.EventType, sub.TargetURL, *p.NewSecretPlain)
		if err != nil {
			return nil, err
		}
		sub.Secret = secret
	}
	if p.Metadata != nil {
		sub.Metadata = datatypes.JSONMap(p.Metadata)
	}
	if len(p.RetryPolicy) > 0 {
		raw, err := json.Marshal(p.RetryPolicy)
		if err != nil {
			return nil, err
		}
		sub.RetryPolicy = datatypes.JSON(raw)
	}

	saved, err := s.subscriptions.Upsert(ctx, sub)
	if err != nil {
		return nil, err
	}
	s.submitApproval(ctx, saved.TenantUuid, saved.ID, "update", sanitizeSubscription(saved))
	return sanitizeSubscription(saved), nil
}

// ListSubscriptions retrieves subscriptions for a tenant.
func (s *WebhookService) ListSubscriptions(ctx context.Context, tenantID string, statuses []string) ([]*model.WebhookSubscription, error) {
	subs, err := s.subscriptions.ListByTenant(ctx, tenantID, statuses)
	if err != nil {
		return nil, err
	}
	result := make([]*model.WebhookSubscription, 0, len(subs))
	for _, sub := range subs {
		result = append(result, sanitizeSubscription(sub))
	}
	return result, nil
}

// GetSubscription returns a single subscription sanitized.
func (s *WebhookService) GetSubscription(ctx context.Context, tenantID, subscriptionID string) (*model.WebhookSubscription, error) {
	sub, err := s.subscriptions.GetByID(ctx, tenantID, subscriptionID)
	if err != nil || sub == nil {
		return sub, err
	}
	return sanitizeSubscription(sub), nil
}

// DeleteSubscription removes a subscription.
func (s *WebhookService) DeleteSubscription(ctx context.Context, tenantID, subscriptionID string) error {
	var snapshot *model.WebhookSubscription
	if existing, err := s.subscriptions.GetByID(ctx, tenantID, subscriptionID); err == nil {
		snapshot = existing
	}
	if err := s.subscriptions.Delete(ctx, tenantID, subscriptionID); err != nil {
		return err
	}
	if snapshot != nil {
		s.submitApproval(ctx, tenantID, subscriptionID, "delete", sanitizeSubscription(snapshot))
	}
	return nil
}

// GetSubscriptionByID retrieves a subscription regardless of tenant scoping (internal use).
func (s *WebhookService) GetSubscriptionByID(ctx context.Context, subscriptionID string) (*model.WebhookSubscription, error) {
	if s.subscriptions == nil {
		return nil, errors.New("subscription repository not configured")
	}
	sub, err := s.subscriptions.GetBySubscriptionID(ctx, subscriptionID)
	if err != nil {
		return nil, err
	}
	if sub == nil {
		return nil, nil
	}
	return sanitizeSubscription(sub), nil
}

// RecordDeliveryAttempt persists a new attempt row.
func (s *WebhookService) RecordDeliveryAttempt(ctx context.Context, params DeliveryResultParams) (*model.DeliveryAttempt, error) {
	if s.attempts == nil {
		return nil, errors.New("attempt repository not configured")
	}
	attempt := &model.DeliveryAttempt{
		SubscriptionID: params.SubscriptionID,
		EnvelopeID:     params.EnvelopeID,
		Status:         params.Status,
		RetryCount:     params.RetryCount,
		LastError:      params.LastError,
	}
	if params.NextRetryAt != nil {
		next := params.NextRetryAt.UTC()
		attempt.NextDeliveryAt = &next
	}
	if params.Payload != nil {
		if raw, err := json.Marshal(params.Payload); err == nil {
			attempt.PayloadSnapshot = datatypes.JSON(raw)
		}
	}
	created, err := s.attempts.Create(ctx, attempt)
	if err == nil && params.TenantUuid != "" {
		obs.RecordWebhookAttempt(strings.ToLower(params.Status), params.TenantUuid)
	}
	return created, err
}

// UpdateAttemptStatus updates attempt metadata.
func (s *WebhookService) UpdateAttemptStatus(ctx context.Context, attemptID, status string, retryCount int, next *time.Time, lastError string, tenantID string) error {
	if s.attempts == nil {
		return errors.New("attempt repository not configured")
	}
	var nextTime time.Time
	if next != nil {
		nextTime = next.UTC()
	}
	if err := s.attempts.UpdateStatus(ctx, attemptID, status, nextTime, lastError, retryCount); err != nil {
		return err
	}
	if tenantID != "" {
		obs.RecordWebhookAttempt(strings.ToLower(status), tenantID)
	}
	return nil
}

// ListAttemptsBySubscription returns recent attempts.
func (s *WebhookService) ListAttemptsBySubscription(ctx context.Context, subscriptionID string, limit int) ([]*model.DeliveryAttempt, error) {
	return s.attempts.ListBySubscription(ctx, subscriptionID, limit)
}

// GetAttempt retrieves attempt details by ID.
func (s *WebhookService) GetAttempt(ctx context.Context, attemptID string) (*model.DeliveryAttempt, error) {
	if s.attempts == nil {
		return nil, errors.New("attempt repository not configured")
	}
	return s.attempts.GetByID(ctx, attemptID)
}

// SignPayload generates a HMAC-SHA256 signature with the subscription secret.
func (s *WebhookService) SignPayload(ctx context.Context, tenantID, subscriptionID string, payload []byte) (string, error) {
	sub, err := s.subscriptions.GetByID(ctx, tenantID, subscriptionID)
	if err != nil {
		return "", err
	}
	if sub == nil {
		return "", gorm.ErrRecordNotFound
	}
	secret, err := s.decryptSecret(sub.TenantUuid, sub.EventType, sub.TargetURL, sub.Secret)
	if err != nil {
		return "", err
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return base64.StdEncoding.EncodeToString(mac.Sum(nil)), nil
}

// NextRetry computes the next retry time and returns whether the attempt should enter DLQ.
func (s *WebhookService) NextRetry(sub *model.WebhookSubscription, currentRetry int) (time.Time, bool) {
	policy := extractRetryPolicy(sub)
	if currentRetry >= len(policy) {
		return time.Time{}, true
	}
	delay := policy[currentRetry]
	if delay <= 0 {
		delay = 60
	}
	return s.now().Add(time.Duration(delay) * time.Second), false
}

func (s *WebhookService) submitApproval(ctx context.Context, tenantID, targetID, action string, payload any) {
	if s.approvalSvc == nil {
		return
	}
	_, err := s.approvalSvc.SubmitChange(ctx, SubmitChangeRequest{
		TargetType:  "webhook_subscription",
		TargetID:    targetID,
		Payload:     map[string]any{"tenant_uuid": tenantID, "action": action, "data": payload},
		SubmittedBy: fmt.Sprintf("system:%s", tenantID),
	})
	if err != nil {
		s.log.WithError(err).WithField("target_id", targetID).Warn("failed to record webhook subscription approval")
	}
}

func (s *WebhookService) encryptSecret(tenantID, eventType, targetURL, plaintext string) (string, error) {
	keyMaterial := s.cfg.Server.SecretKey
	if strings.TrimSpace(keyMaterial) == "" {
		if s.cfg.IsProduction() {
			return "", errors.New("server.secret_key not configured")
		}
		logger.Warn("server.secret_key is empty; using development fallback")
		keyMaterial = "dev-only-change-me"
	}
	key := crypto.DeriveKey32(keyMaterial)
	aad := []byte(fmt.Sprintf("tenant:%s|event:%s|target:%s", tenantID, eventType, targetURL))
	ct, nonce, err := crypto.EncryptAESGCM(key, []byte(plaintext), aad)
	if err != nil {
		return "", err
	}
	payload := append(nonce, ct...)
	return base64.StdEncoding.EncodeToString(payload), nil
}

func (s *WebhookService) decryptSecret(tenantID, eventType, targetURL, encoded string) (string, error) {
	if strings.TrimSpace(encoded) == "" {
		return "", nil
	}
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}
	if len(raw) <= 12 {
		return "", errors.New("secret payload malformed")
	}
	nonce := raw[:12]
	ct := raw[12:]

	keyMaterial := s.cfg.Server.SecretKey
	if strings.TrimSpace(keyMaterial) == "" {
		if s.cfg.IsProduction() {
			return "", errors.New("server.secret_key not configured")
		}
		keyMaterial = "dev-only-change-me"
	}
	key := crypto.DeriveKey32(keyMaterial)
	aad := []byte(fmt.Sprintf("tenant:%s|event:%s|target:%s", tenantID, eventType, targetURL))
	plain, err := crypto.DecryptAESGCM(key, ct, nonce, aad)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

func extractRetryPolicy(sub *model.WebhookSubscription) []int {
	if sub == nil || len(sub.RetryPolicy) == 0 {
		return []int{60, 300, 900}
	}
	var values []int
	if err := json.Unmarshal(sub.RetryPolicy, &values); err != nil || len(values) == 0 {
		return []int{60, 300, 900}
	}
	return values
}

func sanitizeSubscription(sub *model.WebhookSubscription) *model.WebhookSubscription {
	if sub == nil {
		return nil
	}
	cloned := *sub
	cloned.Secret = ""
	return &cloned
}
