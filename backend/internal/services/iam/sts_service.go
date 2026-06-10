package iam

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/config"
	authx "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/middleware"
	"github.com/golang-jwt/jwt/v5"
)

const defaultSTSTTL = 60 * time.Second

// STSService mints short-lived tokens compatible with PowerX gateway expectations.
type STSService struct {
	issuer        string
	audience      string
	secret        []byte
	ttl           time.Duration
	pluginID      string
	policyVersion string
	audit         *AuditService
}

// STSToken represents the HTTP response payload returned to callers.
type STSToken struct {
	AccessToken   string    `json:"access_token"`
	ExpiresIn     int64     `json:"expires_in"`
	ExpiresAt     time.Time `json:"expires_at"`
	IssuedAt      time.Time `json:"issued_at"`
	PluginID      string    `json:"plugin_id"`
	PolicyVersion string    `json:"policy_version"`
}

// NewSTSService builds an STS minter using the runtime config context.
func NewSTSService(cfg *config.Config, audit *AuditService, pluginID, policyVersion string) *STSService {
	ctxCfg := &config.ContextConfig{}
	if cfg != nil && cfg.Context != nil {
		ctxCfg = cfg.Context
	}
	issuer := strings.TrimSpace(ctxCfg.Issuer)
	if issuer == "" {
		issuer = "powerx-local"
	}
	audience := strings.TrimSpace(ctxCfg.Audience)
	if audience == "" {
		audience = "powerx:plugin"
	}
	secret := strings.TrimSpace(ctxCfg.HMACSecret)
	if secret == "" {
		secret = "powerx-plugin-dev"
	}
	ttl := resolveSTSTTL()
	if strings.TrimSpace(pluginID) == "" {
		pluginID = resolvePluginID()
	}
	if strings.TrimSpace(policyVersion) == "" {
		policyVersion = resolvePolicyVersionFallback()
	}
	return &STSService{
		issuer:        issuer,
		audience:      audience,
		secret:        []byte(secret),
		ttl:           ttl,
		pluginID:      pluginID,
		policyVersion: policyVersion,
		audit:         audit,
	}
}

// Mint issues a short-lived PowerX-compatible token for the provided tenant context.
func (s *STSService) Mint(ctx context.Context, tc authx.TenantContext) (*STSToken, error) {
	if s == nil {
		return nil, ErrAuthUnavailable
	}
	tenantUUID := strings.TrimSpace(tc.TenantUUID)
	if tenantUUID == "" {
		return nil, ErrInvalidArguments
	}
	userUUID := strings.TrimSpace(tc.UserUUID)
	memberUUID := strings.TrimSpace(tc.MemberUUID)
	if userUUID == "" || memberUUID == "" || tc.UserID <= 0 || tc.MemberID <= 0 {
		return nil, ErrInvalidArguments
	}
	now := time.Now()
	expires := now.Add(s.ttl)
	policyVersion := strings.TrimSpace(tc.PolicyVersion)
	if policyVersion == "" {
		policyVersion = s.policyVersion
	}
	claims := authx.PowerXClaims{
		TenantUUID:    authx.TenantClaim(tenantUUID),
		TenantID:      authx.Int64Claim(tc.TenantID),
		UserID:        authx.Int64Claim(tc.UserID),
		UserUUID:      userUUID,
		MemberID:      authx.Int64Claim(tc.MemberID),
		MemberUUID:    memberUUID,
		ActorUUID:     memberUUID,
		Roles:         tc.Roles,
		Permissions:   tc.Permissions,
		PolicyVersion: policyVersion,
		PluginID:      s.pluginID,
		Scope:         "access",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.issuer,
			Subject:   memberUUID,
			Audience:  jwt.ClaimStrings{s.audience},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expires),
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := tok.SignedString(s.secret)
	if err != nil {
		return nil, fmt.Errorf("iam: mint sts token: %w", err)
	}
	result := &STSToken{
		AccessToken:   signed,
		ExpiresIn:     int64(s.ttl.Seconds()),
		ExpiresAt:     expires,
		IssuedAt:      now,
		PluginID:      s.pluginID,
		PolicyVersion: policyVersion,
	}
	if s.audit != nil {
		var actor *uint64
		if tc.UserID > 0 {
			uid := uint64(tc.UserID)
			actor = &uid
		}
		_ = s.audit.Record(ctx, AuditEntry{
			TenantUUID:    tenantUUID,
			ActorMemberID: actor,
			Action:        "sts.mint",
			Resource:      "iam.sts",
			Diff: map[string]any{
				"audience":   s.audience,
				"expires_at": expires.UTC().Format(time.RFC3339),
			},
		})
	}
	return result, nil
}

func resolveSTSTTL() time.Duration {
	ttl := defaultSTSTTL
	if raw := strings.TrimSpace(os.Getenv("PLUGIN_IAM_STS_TTL")); raw != "" {
		if parsed, err := time.ParseDuration(raw); err == nil && parsed > 0 {
			ttl = parsed
		}
	}
	return ttl
}

func resolvePolicyVersionFallback() string {
	if v := strings.TrimSpace(os.Getenv("PLUGIN_IAM_POLICY_VERSION")); v != "" {
		return v
	}
	return defaultPolicyVersion
}
