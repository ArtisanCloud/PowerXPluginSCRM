package customer

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/config"
	customerdomain "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/customer"
	customermodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/customer"
	customerrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/customer"
	customermetrics "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/observability/customer"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/datatypes"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrCustomerDisabled   = errors.New("customer disabled")
	ErrCustomerDeleted    = errors.New("customer deleted")
	ErrCustomerMode       = errors.New("customer auth mode not supported")
)

type TenantSelectionRequiredError struct {
	Tenants []string
}

func (e *TenantSelectionRequiredError) Error() string {
	return "tenant selection required"
}

type LocalAuthService struct {
	cfg  *config.Config
	repo *customerrepo.Repository
}

func NewLocalAuthService(cfg *config.Config, repo *customerrepo.Repository) *LocalAuthService {
	return &LocalAuthService{cfg: cfg, repo: repo}
}

type RegisterInput struct {
	TenantUUID string
	Email      string
	Phone      string
	Password   string
	Metadata   map[string]any
}

type RegisterOutput struct {
	CustomerUUID string `json:"customer_uuid"`
	TenantUUID   string `json:"tenant_uuid"`
	Status       string `json:"status"`
}

type LoginInput struct {
	TenantUUID string
	Login      string
	Password   string
}

type LoginOutput struct {
	Token        string `json:"token"`
	ExpiresIn    int64  `json:"expires_in"`
	CustomerUUID string `json:"customer_uuid"`
}

func (s *LocalAuthService) Register(ctx context.Context, in RegisterInput) (*RegisterOutput, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("service is not initialized")
	}
	if !s.isLocalMode() {
		return nil, ErrCustomerMode
	}

	tenantUUID := strings.ToLower(strings.TrimSpace(in.TenantUUID))
	email := strings.TrimSpace(in.Email)
	phone := strings.TrimSpace(in.Phone)
	password := in.Password
	if tenantUUID == "" {
		return nil, errors.New("tenant_uuid is required")
	}
	if email == "" && phone == "" {
		return nil, errors.New("either email or phone is required")
	}
	if len(password) < 8 {
		return nil, errors.New("password must be at least 8 characters")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	customerUUID := uuid.NewString()
	status := string(customerdomain.CustomerStatusActive)
	meta := datatypes.JSONMap{}
	for k, v := range in.Metadata {
		meta[k] = v
	}

	entity := &customermodel.CustomerAccount{
		CustomerUUID:  customerUUID,
		Email:         email,
		Phone:         phone,
		PasswordHash:  string(hash),
		Status:        status,
		Metadata:      meta,
		EmailVerified: false,
		PhoneVerified: false,
	}
	entity.TenantUuid = tenantUUID

	if _, err := s.repo.CreateCustomer(ctx, entity); err != nil {
		return nil, err
	}

	return &RegisterOutput{
		CustomerUUID: customerUUID,
		TenantUUID:   tenantUUID,
		Status:       status,
	}, nil
}

func (s *LocalAuthService) Login(ctx context.Context, in LoginInput) (*LoginOutput, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("service is not initialized")
	}
	if !s.isLocalMode() {
		return nil, ErrCustomerMode
	}
	start := time.Now()

	tenantUUID := strings.ToLower(strings.TrimSpace(in.TenantUUID))
	login := strings.TrimSpace(in.Login)
	if login == "" {
		customermetrics.RecordLogin("", "unauthorized", time.Since(start))
		return nil, ErrInvalidCredentials
	}
	if len(in.Password) < 8 {
		customermetrics.RecordLogin("", "unauthorized", time.Since(start))
		return nil, ErrInvalidCredentials
	}

	if tenantUUID == "" {
		tenants, err := s.repo.ListTenantUUIDsByEmailOrPhone(ctx, login)
		if err != nil {
			if errors.Is(err, customerrepo.ErrCustomerNotFound) {
				customermetrics.RecordLogin("", "unauthorized", time.Since(start))
				return nil, ErrInvalidCredentials
			}
			customermetrics.RecordLogin("", "error", time.Since(start))
			return nil, err
		}
		if len(tenants) == 1 {
			tenantUUID = strings.ToLower(strings.TrimSpace(tenants[0]))
		} else if len(tenants) > 1 {
			customermetrics.RecordLogin("", "tenant_selection_required", time.Since(start))
			return nil, &TenantSelectionRequiredError{Tenants: tenants}
		}
	}

	customer, err := s.repo.FindByEmailOrPhone(ctx, tenantUUID, login)
	if err != nil {
		if errors.Is(err, customerrepo.ErrCustomerNotFound) {
			customermetrics.RecordLogin("", "unauthorized", time.Since(start))
			return nil, ErrInvalidCredentials
		}
		customermetrics.RecordLogin("", "error", time.Since(start))
		return nil, err
	}

	switch strings.ToLower(strings.TrimSpace(customer.Status)) {
	case string(customerdomain.CustomerStatusActive):
		// ok
	case string(customerdomain.CustomerStatusDisabled):
		customermetrics.RecordLogin("", "disabled", time.Since(start))
		return nil, ErrCustomerDisabled
	case string(customerdomain.CustomerStatusDeleted):
		customermetrics.RecordLogin("", "unauthorized", time.Since(start))
		return nil, ErrCustomerDeleted
	default:
		customermetrics.RecordLogin("", "unauthorized", time.Since(start))
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(customer.PasswordHash), []byte(in.Password)); err != nil {
		customermetrics.RecordLogin(tenantUUID, "unauthorized", time.Since(start))
		return nil, ErrInvalidCredentials
	}

	token, expiresIn, err := s.issueToken(tenantUUID, customer.CustomerUUID, nil)
	if err != nil {
		customermetrics.RecordLogin(tenantUUID, "error", time.Since(start))
		return nil, err
	}

	customermetrics.RecordLogin(tenantUUID, "ok", time.Since(start))
	return &LoginOutput{
		Token:        token,
		ExpiresIn:    expiresIn,
		CustomerUUID: customer.CustomerUUID,
	}, nil
}

func (s *LocalAuthService) issueToken(tenantUUID, customerUUID string, roles []string) (string, int64, error) {
	secret := s.resolveJWTSecret()
	if secret == "" {
		return "", 0, ErrCustomerAuthNotImplemented
	}

	issuer := ""
	audience := ""
	if s.cfg != nil && s.cfg.CustomerAuth != nil {
		issuer = strings.TrimSpace(s.cfg.CustomerAuth.JWTIssuer)
		audience = strings.TrimSpace(s.cfg.CustomerAuth.JWTAudience)
	}
	if issuer == "" && s.cfg != nil && s.cfg.Context != nil {
		issuer = strings.TrimSpace(s.cfg.Context.Issuer)
	}

	ttl := 2 * time.Hour
	now := time.Now()
	exp := now.Add(ttl)

	claims := jwt.MapClaims{
		"tenant_uuid":   tenantUUID,
		"customer_uuid": customerUUID,
		"roles":         roles,
		"iat":           now.Unix(),
		"exp":           exp.Unix(),
		"sub":           customerUUID,
	}
	if issuer != "" {
		claims["iss"] = issuer
	}
	if audience != "" {
		claims["aud"] = audience
	}

	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	out, err := tok.SignedString([]byte(secret))
	if err != nil {
		return "", 0, err
	}
	return out, int64(ttl.Seconds()), nil
}

func (s *LocalAuthService) resolveJWTSecret() string {
	if s == nil || s.cfg == nil {
		return ""
	}
	if s.cfg.CustomerAuth != nil {
		if v := strings.TrimSpace(s.cfg.CustomerAuth.JWTSecret); v != "" {
			return v
		}
	}
	if s.cfg.Context != nil && !s.cfg.IsProduction() {
		return strings.TrimSpace(s.cfg.Context.HMACSecret)
	}
	return ""
}

func (s *LocalAuthService) isLocalMode() bool {
	if s == nil || s.cfg == nil || s.cfg.CustomerAuth == nil {
		return true
	}
	return strings.ToLower(strings.TrimSpace(s.cfg.CustomerAuth.Mode)) != "delegate"
}
