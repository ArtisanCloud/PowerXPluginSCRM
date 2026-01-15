package integration

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/config"
	model "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/integration"
	repo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/integration"
	obs "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/observability/integration"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
	"github.com/sirupsen/logrus"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// SecretService orchestrates external credential lifecycle operations.
type SecretService struct {
	cfg       *config.Config
	repo      *repo.SecretRepository
	provider  SecretProvider
	approvals *ApprovalService
	logger    *logrus.Entry
	now       func() time.Time
}

// NewSecretService constructs a new SecretService.
func NewSecretService(
	cfg *config.Config,
	repository *repo.SecretRepository,
	provider SecretProvider,
	approvalRepo *repo.ApprovalRepository,
) *SecretService {
	log := logrus.WithField("component", "integration.secret_service")
	var approvalSvc *ApprovalService
	if approvalRepo != nil {
		approvalSvc = NewApprovalService(approvalRepo, log.WithField("subcomponent", "approval"))
	}
	if provider == nil {
		provider = NewRandomSecretProvider(log.WithField("subcomponent", "provider"))
	}
	return &SecretService{
		cfg:       cfg,
		repo:      repository,
		provider:  provider,
		approvals: approvalSvc,
		logger:    log,
		now:       time.Now,
	}
}

// CreateSecretParams holds create options.
type CreateSecretParams struct {
	TenantUuid           string
	IntegrationType      string
	RotationIntervalDays int
	Metadata             map[string]any
	Generate             bool
	ExistingSecretRef    string
	Actor                string
}

// CreateSecretResult includes the persisted secret and optional plaintext.
type CreateSecretResult struct {
	Secret         *model.SecretCredential `json:"secret"`
	GeneratedPlain string                  `json:"generated_secret,omitempty"`
}

// RotateSecretParams captures rotation options.
type RotateSecretParams struct {
	TenantUuid string
	SecretID   string
	Generate   bool
	Actor      string
}

// RotateSecretResult returns pending material if generated.
type RotateSecretResult struct {
	Secret             *model.SecretCredential `json:"secret"`
	PendingSecretRef   string                  `json:"pending_secret_ref,omitempty"`
	GeneratedPlaintext string                  `json:"generated_secret,omitempty"`
}

// CreateSecret registers a new credential entry.
func (s *SecretService) CreateSecret(ctx context.Context, params CreateSecretParams) (*CreateSecretResult, error) {
	if s.repo == nil {
		return nil, errors.New("secret repository not configured")
	}
	params.TenantUuid = strings.TrimSpace(params.TenantUuid)
	params.IntegrationType = strings.TrimSpace(params.IntegrationType)
	if params.TenantUuid == "" {
		return nil, errors.New("tenant_uuid is required")
	}
	if params.IntegrationType == "" {
		return nil, errors.New("integration_type is required")
	}
	if params.RotationIntervalDays <= 0 {
		params.RotationIntervalDays = 30
	}

	secret := &model.SecretCredential{
		TenantUuid:       params.TenantUuid,
		IntegrationType:  params.IntegrationType,
		CurrentSecretRef: strings.TrimSpace(params.ExistingSecretRef),
		RotationInterval: params.RotationIntervalDays,
		Status:           model.SecretStatusActive,
		Metadata:         datatypes.JSONMap{},
	}
	if params.Metadata != nil {
		secret.Metadata = datatypes.JSONMap(params.Metadata)
	}

	var generatedPlain string
	if params.Generate {
		mat, err := s.provider.Issue(ctx, params.TenantUuid, params.IntegrationType)
		if err != nil {
			return nil, err
		}
		secret.CurrentSecretRef = mat.Reference
		generatedPlain = mat.Secret
	}

	now := s.now().UTC()
	nextDue := now.Add(time.Duration(secret.RotationInterval) * 24 * time.Hour)
	secret.NextRotationDueAt = &nextDue

	if err := s.appendAudit(secret, params.Actor, "create", map[string]any{
		"rotation_interval_days": secret.RotationInterval,
	}); err != nil {
		return nil, err
	}

	created, err := s.repo.Create(ctx, secret)
	if err != nil {
		return nil, err
	}
	s.submitApproval(ctx, params.TenantUuid, created.ID, "create", created, params.Actor)
	return &CreateSecretResult{Secret: sanitizeSecret(created), GeneratedPlain: generatedPlain}, nil
}

// RotateSecret schedules a rotation and optionally generates a pending secret.
func (s *SecretService) RotateSecret(ctx context.Context, params RotateSecretParams) (*RotateSecretResult, error) {
	secret, err := s.repo.GetByID(ctx, params.TenantUuid, params.SecretID)
	if err != nil {
		return nil, err
	}
	if secret == nil {
		return nil, gorm.ErrRecordNotFound
	}

	var generated string
	if params.Generate {
		mat, err := s.provider.Issue(ctx, params.TenantUuid, secret.IntegrationType)
		if err != nil {
			return nil, err
		}
		secret.PendingSecretRef = mat.Reference
		generated = mat.Secret
	}
	if secret.PendingSecretRef == "" {
		return nil, errors.New("no pending secret reference provided or generated")
	}
	secret.Status = model.SecretStatusRotating
	secret.UpdatedAt = s.now().UTC()
	if err := s.appendAudit(secret, params.Actor, "schedule_rotation", map[string]any{
		"pending_secret_ref": secret.PendingSecretRef,
	}); err != nil {
		return nil, err
	}
	if err := s.repo.Update(ctx, secret); err != nil {
		return nil, err
	}
	s.submitApproval(ctx, secret.TenantUuid, secret.ID, "schedule_rotation", sanitizeSecret(secret), params.Actor)
	return &RotateSecretResult{
		Secret:             sanitizeSecret(secret),
		PendingSecretRef:   secret.PendingSecretRef,
		GeneratedPlaintext: generated,
	}, nil
}

// CompleteRotation promotes pending secret to current.
func (s *SecretService) CompleteRotation(ctx context.Context, tenantID, secretID, actor string) (*model.SecretCredential, error) {
	secret, err := s.repo.GetByID(ctx, tenantID, secretID)
	if err != nil {
		return nil, err
	}
	if secret == nil {
		return nil, gorm.ErrRecordNotFound
	}
	if strings.TrimSpace(secret.PendingSecretRef) == "" {
		return nil, errors.New("no pending secret to promote")
	}

	now := s.now().UTC()
	if err := s.provider.Revoke(ctx, tenantID, secret.CurrentSecretRef); err != nil {
		s.logger.WithError(err).WithField("secret_id", secret.ID).Warn("failed to revoke previous secret_ref")
	}

	secret.CurrentSecretRef = secret.PendingSecretRef
	secret.PendingSecretRef = ""
	secret.Status = model.SecretStatusActive
	secret.LastRotatedAt = &now
	next := now.Add(time.Duration(secret.RotationInterval) * 24 * time.Hour)
	secret.NextRotationDueAt = &next
	secret.UpdatedAt = now
	if err := s.appendAudit(secret, actor, "complete_rotation", map[string]any{
		"next_due_at": next.Format(time.RFC3339),
	}); err != nil {
		return nil, err
	}
	if err := s.repo.Update(ctx, secret); err != nil {
		return nil, err
	}
	s.submitApproval(ctx, tenantID, secret.ID, "complete_rotation", sanitizeSecret(secret), actor)
	return sanitizeSecret(secret), nil
}

// RevokeSecret marks secret as revoked and clears references.
func (s *SecretService) RevokeSecret(ctx context.Context, tenantID, secretID, actor string) (*model.SecretCredential, error) {
	secret, err := s.repo.GetByID(ctx, tenantID, secretID)
	if err != nil {
		return nil, err
	}
	if secret == nil {
		return nil, gorm.ErrRecordNotFound
	}

	if secret.CurrentSecretRef != "" {
		if err := s.provider.Revoke(ctx, tenantID, secret.CurrentSecretRef); err != nil {
			s.logger.WithError(err).WithField("secret_id", secret.ID).Warn("failed to revoke current secret")
		}
	}
	if secret.PendingSecretRef != "" {
		if err := s.provider.Revoke(ctx, tenantID, secret.PendingSecretRef); err != nil {
			s.logger.WithError(err).WithField("secret_id", secret.ID).Warn("failed to revoke pending secret")
		}
	}

	secret.CurrentSecretRef = ""
	secret.PendingSecretRef = ""
	secret.Status = model.SecretStatusRevoked
	secret.NextRotationDueAt = nil
	now := s.now().UTC()
	secret.LastRotatedAt = &now
	secret.UpdatedAt = now
	if err := s.appendAudit(secret, actor, "revoke", nil); err != nil {
		return nil, err
	}
	if err := s.repo.Update(ctx, secret); err != nil {
		return nil, err
	}
	s.submitApproval(ctx, tenantID, secret.ID, "revoke", sanitizeSecret(secret), actor)
	return sanitizeSecret(secret), nil
}

// ListSecrets returns tenant secrets sanitized.
func (s *SecretService) ListSecrets(ctx context.Context, tenantID string) ([]*model.SecretCredential, error) {
	secrets, err := s.repo.ListByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	result := make([]*model.SecretCredential, 0, len(secrets))
	for _, secret := range secrets {
		result = append(result, sanitizeSecret(secret))
	}
	return result, nil
}

// GetSecret returns a tenant secret sanitized.
func (s *SecretService) GetSecret(ctx context.Context, tenantID, secretID string) (*model.SecretCredential, error) {
	secret, err := s.repo.GetByID(ctx, tenantID, secretID)
	if err != nil || secret == nil {
		return secret, err
	}
	return sanitizeSecret(secret), nil
}

// GetAuditLog returns audit entries for a secret.
func (s *SecretService) GetAuditLog(ctx context.Context, tenantID, secretID string) ([]AuditEntry, error) {
	secret, err := s.repo.GetByID(ctx, tenantID, secretID)
	if err != nil || secret == nil {
		return nil, err
	}
	return parseAudit(secret.AuditLog)
}

// RefreshRotationMetrics updates observability gauges for secrets nearing rotation.
func (s *SecretService) RefreshRotationMetrics(ctx context.Context) error {
	if s.repo == nil {
		return errors.New("secret repository not configured")
	}

	now := s.now().UTC()
	due, err := s.repo.ListDueForRotation(ctx, now, 0)
	if err != nil {
		return err
	}
	obs.SetSecretsDue("due_now", float64(len(due)))

	soonCutoff := now.Add(48 * time.Hour)
	soon, err := s.repo.ListDueForRotation(ctx, soonCutoff, 0)
	if err != nil {
		return err
	}
	obs.SetSecretsDue("due_48h", float64(len(soon)))
	return nil
}

func (s *SecretService) appendAudit(secret *model.SecretCredential, actor, action string, details map[string]any) error {
	var entries []AuditEntry
	if len(secret.AuditLog) > 0 {
		if err := json.Unmarshal(secret.AuditLog, &entries); err != nil {
			return err
		}
	}
	entry := AuditEntry{
		Action:    action,
		Actor:     actor,
		Timestamp: s.now().UTC(),
		Details:   details,
	}
	entries = append(entries, entry)
	raw, err := json.Marshal(entries)
	if err != nil {
		return err
	}
	secret.AuditLog = datatypes.JSON(raw)
	return nil
}

func (s *SecretService) submitApproval(ctx context.Context, tenantID, targetID, action string, payload any, actor string) {
	if s.approvals == nil {
		return
	}
	req := SubmitChangeRequest{
		TargetType:  "secret_credential",
		TargetID:    targetID,
		Payload:     map[string]any{"tenant_uuid": tenantID, "action": action, "snapshot": payload},
		SubmittedBy: defaultActor(actor),
	}
	if _, err := s.approvals.SubmitChange(ctx, req); err != nil {
		s.logger.WithError(err).WithField("secret_id", targetID).Warn("failed to submit approval entry")
	}
}

func sanitizeSecret(secret *model.SecretCredential) *model.SecretCredential {
	if secret == nil {
		return nil
	}
	cloned := *secret
	cloned.AuditLog = nil
	return &cloned
}

func defaultActor(actor string) string {
	actor = strings.TrimSpace(actor)
	if actor != "" {
		return actor
	}
	return "system"
}

// AuditEntry represents a secret lifecycle audit event.
type AuditEntry struct {
	Action    string         `json:"action"`
	Actor     string         `json:"actor"`
	Timestamp time.Time      `json:"timestamp"`
	Details   map[string]any `json:"details,omitempty"`
}

func parseAudit(raw datatypes.JSON) ([]AuditEntry, error) {
	if len(raw) == 0 {
		return []AuditEntry{}, nil
	}
	var entries []AuditEntry
	if err := json.Unmarshal(raw, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

// BuildSecretService assembles the secret service from shared dependencies.
func BuildSecretService(deps *app.Deps, provider SecretProvider) *SecretService {
	if deps == nil || deps.DB == nil {
		return nil
	}
	secretRepo := repo.NewSecretRepository(deps.DB)
	approvalRepo := repo.NewApprovalRepository(deps.DB)
	return NewSecretService(deps.Config, secretRepo, provider, approvalRepo)
}
