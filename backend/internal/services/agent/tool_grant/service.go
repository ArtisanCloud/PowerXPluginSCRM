package tool_grant

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/config"
	tgmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/tool_grant"
	tgrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/tool_grant"
	seclog "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/observability/security"
	"github.com/golang-jwt/jwt/v5"
	"github.com/sirupsen/logrus"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Service orchestrates ToolGrant issuance and validation.
type Service struct {
	repo       *tgrepo.Repository
	cfg        *config.Config
	logger     *logrus.Entry
	signingKey []byte
}

func NewService(db *gorm.DB, cfg *config.Config, logger *logrus.Entry, signingKey []byte) *Service {
	return &Service{
		repo:       tgrepo.NewRepository(db),
		cfg:        cfg,
		logger:     logger,
		signingKey: signingKey,
	}
}

// Claims defines ToolGrant JWT payload.
type Claims struct {
	TenantUuid   string   `json:"tenant_uuid"`
	ToolID       string   `json:"tool_id"`
	Capabilities []string `json:"capabilities"`
	AgentID      string   `json:"agent_id"`
	jwt.RegisteredClaims
}

// Issue generates a ToolGrant JWT and records issuance event.
func (s *Service) Issue(ctx context.Context, tenantID, toolID, agentID string, caps []string, initiatedBy string) (string, error) {
	if tenantID == "" || toolID == "" {
		return "", errors.New("tenant_uuid and tool_id required")
	}
	if len(s.signingKey) == 0 {
		return "", errors.New("toolgrant signing key not configured")
	}
	ttl := s.cfg.ToolGrantTTL()
	now := time.Now().UTC()
	claims := Claims{
		TenantUuid:   tenantID,
		ToolID:       toolID,
		Capabilities: caps,
		AgentID:      agentID,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			ID:        fmt.Sprintf("%s:%s:%d", tenantID, toolID, now.UnixNano()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(s.signingKey)
	if err != nil {
		seclog.RecordToolGrantEvent("issuance_failed", tenantID)
		return "", err
	}
	_, recErr := s.repo.RecordUsageEvent(ctx, &tgmodel.UsageEvent{
		TenantUuid:  tenantID,
		ToolGrantID: claims.ID,
		EventType:   "ISSUED",
		Capability:  toolID,
		AgentID:     agentID,
	}, datatypes.JSONMap{"initiated_by": initiatedBy})
	if recErr != nil && s.logger != nil {
		s.logger.WithError(recErr).Warn("failed to record issuance event")
	}
	seclog.EmitToolGrantEvent(s.logger, "toolgrant.issued", tenantID, map[string]interface{}{
		"tool_id":      toolID,
		"agent_id":     agentID,
		"toolgrant_id": claims.ID,
	})
	seclog.RecordToolGrantEvent("issued", tenantID)
	return signed, nil
}

// Revoke records revocation and emits usage event.
func (s *Service) Revoke(ctx context.Context, tenantID, toolGrantID, reason, actor string, ttlExpiry time.Time) error {
	if toolGrantID == "" {
		return errors.New("toolgrant_id required")
	}
	if ttlExpiry.IsZero() {
		ttlExpiry = time.Now().UTC()
	}
	_, err := s.repo.RecordRevocation(ctx, &tgmodel.Revocation{
		TenantUuid:  tenantID,
		ToolGrantID: toolGrantID,
		RevokedBy:   actor,
		Reason:      reason,
		TtlExpiry:   ttlExpiry,
	})
	if err != nil {
		seclog.RecordToolGrantEvent("revocation_failed", tenantID)
		return err
	}
	_, recErr := s.repo.RecordUsageEvent(ctx, &tgmodel.UsageEvent{
		TenantUuid:  tenantID,
		ToolGrantID: toolGrantID,
		EventType:   "REVOKED",
		Capability:  "",
		AgentID:     actor,
	}, datatypes.JSONMap{"reason": reason})
	if recErr != nil && s.logger != nil {
		s.logger.WithError(recErr).Warn("failed to record revocation event")
	}
	seclog.EmitToolGrantEvent(s.logger, "toolgrant.revoked", tenantID, map[string]interface{}{
		"toolgrant_id": toolGrantID,
		"reason":       reason,
		"actor":        actor,
	})
	seclog.RecordToolGrantEvent("revoked", tenantID)
	return nil
}

// Validate checks a ToolGrant JWT for signature, expiration, and revocation.
func (s *Service) Validate(ctx context.Context, tenantID string, tokenString string) (*Claims, error) {
	if tokenString == "" {
		return nil, errors.New("token missing")
	}
	if len(s.signingKey) == 0 {
		return nil, errors.New("toolgrant signing key not configured")
	}
	parsed, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, fmt.Errorf("unexpected signing method: %s", token.Method.Alg())
		}
		return s.signingKey, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := parsed.Claims.(*Claims)
	if !ok || !parsed.Valid {
		return nil, errors.New("invalid token claims")
	}
	if tenantID != "" && claims.TenantUuid != tenantID {
		return nil, errors.New("tenant mismatch")
	}
	// Revocation check
	revs, err := s.repo.ListRevocations(ctx, claims.TenantUuid, 0)
	if err != nil {
		return nil, err
	}
	for _, rev := range revs {
		if rev.ToolGrantID == claims.ID {
			return nil, errors.New("toolgrant revoked")
		}
	}
	return claims, nil
}

// RevocationHistory returns recent revocations for tenant.
func (s *Service) RevocationHistory(ctx context.Context, tenantID string, limit int) ([]*tgmodel.Revocation, error) {
	return s.repo.ListRevocations(ctx, tenantID, limit)
}

// UsageHistory returns usage events for given ToolGrant.
func (s *Service) UsageHistory(ctx context.Context, tenantID, toolGrantID string, limit int) ([]*tgmodel.UsageEvent, error) {
	return s.repo.ListUsageEvents(ctx, tenantID, toolGrantID, limit)
}
