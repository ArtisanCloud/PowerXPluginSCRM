package customer

import "context"

// CustomerStatus defines lifecycle state for a customer identity.
type CustomerStatus string

const (
	CustomerStatusActive   CustomerStatus = "active"
	CustomerStatusDisabled CustomerStatus = "disabled"
	CustomerStatusDeleted  CustomerStatus = "deleted"
)

// CustomerAuthMode selects the authenticator implementation.
type CustomerAuthMode string

const (
	CustomerAuthModeLocal    CustomerAuthMode = "local"
	CustomerAuthModeDelegate CustomerAuthMode = "delegate"
)

// CustomerContext is the runtime identity attached to each /mini-app request.
type CustomerContext struct {
	TenantUUID    string           `json:"tenant_uuid"`
	CustomerUUID  string           `json:"customer_uuid"`
	Roles         []string         `json:"roles,omitempty"`
	SourceMode    CustomerAuthMode `json:"source_mode"`
	Attributes    map[string]any   `json:"attributes,omitempty"`
	RawClaims     map[string]any   `json:"raw_claims,omitempty"`
	Authenticated bool             `json:"authenticated"`
}

type customerContextKey struct{}

// SetContext stores CustomerContext into request context.
func SetContext(ctx context.Context, cc *CustomerContext) context.Context {
	if ctx == nil {
		return context.WithValue(context.Background(), customerContextKey{}, cc)
	}
	return context.WithValue(ctx, customerContextKey{}, cc)
}

// ContextFrom reads CustomerContext from request context.
func ContextFrom(ctx context.Context) (*CustomerContext, bool) {
	if ctx == nil {
		return nil, false
	}
	cc, ok := ctx.Value(customerContextKey{}).(*CustomerContext)
	return cc, ok && cc != nil
}

// CustomerAuthConfig is configuration for local/delegate customer auth.
type CustomerAuthConfig struct {
	Mode             CustomerAuthMode `json:"mode" yaml:"mode" mapstructure:"mode"`
	DelegateEndpoint string           `json:"delegate_endpoint" yaml:"delegate_endpoint" mapstructure:"delegate_endpoint"`
	DelegateTimeout  string           `json:"delegate_timeout" yaml:"delegate_timeout" mapstructure:"delegate_timeout"`
	JWTIssuer        string           `json:"jwt_issuer" yaml:"jwt_issuer" mapstructure:"jwt_issuer"`
	JWTAudience      string           `json:"jwt_audience" yaml:"jwt_audience" mapstructure:"jwt_audience"`
	JWTSecret        string           `json:"jwt_secret" yaml:"jwt_secret" mapstructure:"jwt_secret"`
	CacheTTLSeconds  int              `json:"cache_ttl_seconds" yaml:"cache_ttl_seconds" mapstructure:"cache_ttl_seconds"`
}
