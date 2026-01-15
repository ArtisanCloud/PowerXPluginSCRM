package customer

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/config"
	customerdomain "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/customer"
	"github.com/golang-jwt/jwt/v5"
	"github.com/sirupsen/logrus"
)

var ErrCustomerAuthNotImplemented = errors.New("customer authentication not implemented")
var ErrCustomerTokenInvalid = errors.New("customer token invalid")

// Authenticator 将 customer token 解析/校验为 CustomerContext。
// 注意：具体 local / delegate 实现会在后续任务中完善。
type Authenticator interface {
	Authenticate(ctx context.Context, tenantUUID string, token string) (*customerdomain.CustomerContext, error)
}

type Factory struct {
	cfg    *config.Config
	logger *logrus.Entry
}

func NewAuthenticatorFactory(cfg *config.Config, logger *logrus.Entry) *Factory {
	if logger == nil {
		logger = logrus.WithField("component", "customer.authenticator_factory")
	}
	return &Factory{cfg: cfg, logger: logger}
}

func (f *Factory) Build() Authenticator {
	mode := "local"
	if f != nil && f.cfg != nil && f.cfg.CustomerAuth != nil {
		if v := strings.ToLower(strings.TrimSpace(f.cfg.CustomerAuth.Mode)); v != "" {
			mode = v
		}
	}
	switch mode {
	case "delegate":
		return NewDelegateAuthenticator(f.cfg, nil, f.logger)
	default:
		return newLocalJWTAuthenticator(f.cfg, f.logger)
	}
}

type localJWTAuthenticator struct {
	secret   []byte
	issuer   string
	audience string
	logger   *logrus.Entry
}

type customerTokenClaims struct {
	TenantUUID   string   `json:"tenant_uuid,omitempty"`
	CustomerUUID string   `json:"customer_uuid,omitempty"`
	Roles        []string `json:"roles,omitempty"`
	jwt.RegisteredClaims
}

func newLocalJWTAuthenticator(cfg *config.Config, logger *logrus.Entry) Authenticator {
	secret := ""
	issuer := ""
	audience := ""
	if cfg != nil && cfg.CustomerAuth != nil {
		secret = strings.TrimSpace(cfg.CustomerAuth.JWTSecret)
		issuer = strings.TrimSpace(cfg.CustomerAuth.JWTIssuer)
		audience = strings.TrimSpace(cfg.CustomerAuth.JWTAudience)
	}
	if secret == "" && cfg != nil && cfg.Context != nil && !cfg.IsProduction() {
		secret = strings.TrimSpace(cfg.Context.HMACSecret)
	}
	return &localJWTAuthenticator{
		secret:   []byte(secret),
		issuer:   issuer,
		audience: audience,
		logger:   logger,
	}
}

func (a *localJWTAuthenticator) Authenticate(_ context.Context, requestTenantUUID string, token string) (*customerdomain.CustomerContext, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, ErrCustomerTokenInvalid
	}
	if len(a.secret) == 0 {
		return nil, ErrCustomerAuthNotImplemented
	}

	claims := &customerTokenClaims{}
	parsed, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (any, error) {
		return a.secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name}))
	if err != nil || parsed == nil || !parsed.Valid {
		return nil, ErrCustomerTokenInvalid
	}
	if a.issuer != "" && claims.Issuer != a.issuer {
		return nil, ErrCustomerTokenInvalid
	}
	if a.audience != "" && !audienceContains(claims.Audience, a.audience) {
		return nil, ErrCustomerTokenInvalid
	}

	tenantUUID := strings.ToLower(strings.TrimSpace(claims.TenantUUID))
	if tenantUUID == "" {
		tenantUUID = strings.ToLower(strings.TrimSpace(requestTenantUUID))
	}
	customerUUID := strings.ToLower(strings.TrimSpace(claims.CustomerUUID))
	if customerUUID == "" && claims.Subject != "" {
		customerUUID = strings.ToLower(strings.TrimSpace(claims.Subject))
	}

	// exp/nbf/iat: jwt/v5 会在 ParseWithClaims 中自动校验 exp/nbf（若存在）。
	// 这里补一个最小 iat 兜底，避免出现未来时间戳导致极端误判（不强制）。
	if claims.IssuedAt != nil && claims.IssuedAt.After(time.Now().Add(10*time.Minute)) {
		return nil, ErrCustomerTokenInvalid
	}

	return &customerdomain.CustomerContext{
		TenantUUID:    tenantUUID,
		CustomerUUID:  customerUUID,
		Roles:         claims.Roles,
		SourceMode:    customerdomain.CustomerAuthModeLocal,
		RawClaims:     map[string]any{"iss": claims.Issuer, "sub": claims.Subject},
		Authenticated: true,
	}, nil
}

func audienceContains(aud jwt.ClaimStrings, expected string) bool {
	expected = strings.TrimSpace(expected)
	if expected == "" {
		return true
	}
	for _, v := range aud {
		if strings.TrimSpace(v) == expected {
			return true
		}
	}
	return false
}

// delegate authenticator implementation in delegate_authenticator.go
