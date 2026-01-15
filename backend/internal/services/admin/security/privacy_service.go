package security

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/config"
	privmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/privacy"
	privrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/privacy"
	secobs "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/observability/security"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// PrivacyService encapsulates administrative operations for privacy governance.
type PrivacyService struct {
	repo   *privrepo.Repository
	cfg    *config.Config
	logger *logrus.Entry
	audit  *secobs.AuditWriter
}

// NewPrivacyService constructs the admin privacy service.
func NewPrivacyService(db *gorm.DB, cfg *config.Config, logger *logrus.Entry, audit *secobs.AuditWriter) *PrivacyService {
	return &PrivacyService{
		repo:   privrepo.NewRepository(db),
		cfg:    cfg,
		logger: logger,
		audit:  audit,
	}
}

// ListConsentTokens returns consent tokens for a tenant filtered by optional status values.
func (s *PrivacyService) ListConsentTokens(ctx context.Context, tenantID string, statuses ...string) ([]*privmodel.ConsentToken, error) {
	if tenantID == "" {
		return nil, errors.New("tenant_uuid is required")
	}
	return s.repo.ListConsentTokensByStatus(ctx, tenantID, statuses...)
}

// RevokeConsentToken marks a consent token as revoked and appends lifecycle evidence.
func (s *PrivacyService) RevokeConsentToken(ctx context.Context, tenantID, tokenID, reason, actor string) error {
	if tenantID == "" || tokenID == "" {
		return errors.New("tenant_uuid and token_id are required")
	}
	if reason == "" {
		reason = "revoked_by_admin"
	}
	if err := s.repo.RevokeConsentToken(ctx, tenantID, tokenID, reason); err != nil {
		secobs.RecordConsentEvent(privmodel.LifecycleEventConsentRevoke, privmodel.LifecycleStatusFailed)
		return err
	}
	secobs.RecordConsentEvent(privmodel.LifecycleEventConsentRevoke, privmodel.LifecycleStatusSucceeded)
	if actor == "" {
		actor = "admin"
	}
	_, err := s.repo.CreateLifecycleEvent(ctx, &privmodel.LifecycleEvent{
		TenantUuid: tenantID,
		EventType:  privmodel.LifecycleEventConsentRevoke,
		AssetKey:   "*",
		RecordedBy: actor,
		Status:     privmodel.LifecycleStatusSucceeded,
	})
	if err == nil && s.audit != nil {
		logErr := s.audit.EmitLifecycleSuccess(tenantID, privmodel.LifecycleEventConsentRevoke, actor, map[string]interface{}{
			"token_id": tokenID,
			"reason":   reason,
		})
		if logErr != nil && s.logger != nil {
			s.logger.WithError(logErr).Warn("failed to emit consent revocation audit log")
		}
	}
	return err
}

// UpsertClassification stores a data classification record for the tenant asset.
func (s *PrivacyService) UpsertClassification(ctx context.Context, record *privmodel.DataClassification) (*privmodel.DataClassification, error) {
	if record == nil {
		return nil, errors.New("record is required")
	}
	return s.repo.UpsertClassification(ctx, record)
}

// DeleteClassification removes a classification mapping.
func (s *PrivacyService) DeleteClassification(ctx context.Context, tenantID, assetKey string) error {
	if tenantID == "" || assetKey == "" {
		return errors.New("tenant_uuid and asset_key are required")
	}
	return s.repo.DeleteClassification(ctx, tenantID, assetKey)
}

// IssueConsentToken creates a new consent token and persists lifecycle evidence.
func (s *PrivacyService) IssueConsentToken(ctx context.Context, token *privmodel.ConsentToken, scope []string) (*privmodel.ConsentToken, error) {
	if token == nil {
		return nil, errors.New("token payload is required")
	}
	out, err := s.repo.IssueConsentToken(ctx, token, scope)
	if err != nil {
		secobs.RecordConsentEvent(privmodel.LifecycleEventConsentRenew, privmodel.LifecycleStatusFailed)
		return nil, err
	}
	secobs.RecordConsentEvent(privmodel.LifecycleEventConsentRenew, privmodel.LifecycleStatusSucceeded)
	_, logErr := s.repo.CreateLifecycleEvent(ctx, &privmodel.LifecycleEvent{
		TenantUuid: token.TenantUuid,
		EventType:  privmodel.LifecycleEventConsentRenew,
		AssetKey:   "*",
		RecordedBy: token.IssuedBy,
		Status:     privmodel.LifecycleStatusSucceeded,
	})
	if logErr != nil && s.logger != nil {
		s.logger.WithError(logErr).
			WithField("tenant_uuid", token.TenantUuid).
			Warn("failed to record consent issuance lifecycle event")
	}
	if s.audit != nil {
		meta := map[string]interface{}{
			"token_id":  out.ID,
			"issued_by": out.IssuedBy,
		}
		if err := s.audit.EmitLifecycleSuccess(out.TenantUuid, privmodel.LifecycleEventConsentRenew, out.IssuedBy, meta); err != nil && s.logger != nil {
			s.logger.WithError(err).
				WithField("tenant_uuid", out.TenantUuid).
				Warn("failed to emit audit log for consent issuance")
		}
	}
	return out, nil
}

// RecordLifecycleEvent appends lifecycle evidence with the supplied payload.
func (s *PrivacyService) RecordLifecycleEvent(ctx context.Context, evt *privmodel.LifecycleEvent) (*privmodel.LifecycleEvent, error) {
	if evt == nil {
		return nil, errors.New("event payload is required")
	}
	return s.repo.CreateLifecycleEvent(ctx, evt)
}

// ListLifecycleEvents returns lifecycle evidence filtered by event types and optional limit.
func (s *PrivacyService) ListLifecycleEvents(ctx context.Context, tenantID string, eventTypes []string, limit int) ([]*privmodel.LifecycleEvent, error) {
	return s.repo.ListLifecycleEvents(ctx, tenantID, eventTypes, limit)
}

// LifecycleRetentionDeadline computes the deadline for retention purge based on baseline config.
func (s *PrivacyService) LifecycleRetentionDeadline(from time.Time) time.Time {
	if from.IsZero() {
		from = time.Now().UTC()
	}
	days := s.cfg.ConsentRetentionDays()
	return from.AddDate(0, 0, days)
}

// ActiveConsentScope returns consolidated scope of all active consent tokens.
func (s *PrivacyService) ActiveConsentScope(ctx context.Context, tenantID string, now time.Time) (map[string]struct{}, error) {
	tokens, err := s.repo.ActiveConsentTokens(ctx, tenantID, now)
	if err != nil {
		return nil, err
	}
	scope := make(map[string]struct{})
	for _, token := range tokens {
		entries, e := token.ScopeValues()
		if e != nil {
			if s.logger != nil {
				s.logger.WithError(e).
					WithField("token_id", token.ID).
					Warn("failed to decode consent scope")
			}
			continue
		}
		for _, asset := range entries {
			scope[asset] = struct{}{}
		}
	}
	return scope, nil
}

// EnsureAssetsAuthorized verifies that all required assets fall within granted consent scope.
func (s *PrivacyService) EnsureAssetsAuthorized(ctx context.Context, tenantID string, assets []string) error {
	if len(assets) == 0 {
		return nil
	}
	scope, err := s.ActiveConsentScope(ctx, tenantID, time.Now().UTC())
	if err != nil {
		return err
	}
	for _, asset := range assets {
		if _, ok := scope[asset]; !ok {
			return fmt.Errorf("asset %s not authorized by consent", asset)
		}
	}
	return nil
}
