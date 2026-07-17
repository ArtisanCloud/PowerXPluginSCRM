package app

import (
	"context"

	fwwsbus "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/wsbus"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/capabilities"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/config"
	leadrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/repository/lead_capture"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/grpc/client"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/integrations/gateway"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/logger"
	authx "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/middleware"
	adminmetrics "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/observability/admin_console"
	capmetrics "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/observability/capability"
	leadmetrics "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/observability/lead_capture"
	opsmetrics "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/observability/operations"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/authproxy"
	iamservice "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/iam"
	marketplacesvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/marketplace"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	fweventbridge "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/eventbridge"
)

type DelegatedAuthProxy interface {
	Login(ctx context.Context, req iamservice.LoginRequest) (*iamservice.AuthTokens, error)
	Refresh(ctx context.Context, refreshToken string) (*iamservice.AuthTokens, error)
	Logout(ctx context.Context, refreshToken string) error
	MeContext(ctx context.Context, accessToken string) (*authproxy.MeContext, error)
	ProxyRequest(ctx context.Context, method, path string, payload any, out any, extraHeaders map[string]string) error
}

// Deps bundles shared infrastructure dependencies for handlers and services.
type Deps struct {
	DB                  *gorm.DB
	Ctx                 context.Context
	PowerXClient        *client.PowerXServiceClient
	CapabilityGateway   gatewayClient
	Config              *config.Config
	CapabilitiesManager capabilities.Manager
	CapabilityMetrics   *capmetrics.Metrics
	TaxProviderClient   *marketplacesvc.TaxProviderClient
	MarketplaceBilling  marketplacesvc.BillingClient
	LicenseAuthority    marketplacesvc.LicenseAuthority
	LicenseCache        marketplacesvc.LicenseCache
	OperationsMetrics   *opsmetrics.Metrics
	AdminConsoleMetrics *adminmetrics.Metrics
	LeadCaptureMetrics  *leadmetrics.Metrics
	LeadCaptureRepos    *leadrepo.Bundle
	EventEmitter        fweventbridge.Emitter
	WSBusHub            fwwsbus.LocalHub
	ProviderMode        iamservice.ProviderMode
	ProviderModeSource  string
	AuthProxy           DelegatedAuthProxy
	IAMDirectory        iamservice.IAMDirectory
}

type LeadSyncChannelIdentity struct {
	Channel string
	AppType string
}

type gatewayClient interface {
	Enabled() bool
	Invoke(ctx context.Context, params gateway.InvokeParams) (*gateway.InvokeResult, error)
	ListPlatformCapabilities(ctx context.Context, opts gateway.ListPlatformCapabilitiesOptions) ([]gateway.PlatformCapabilityRecord, error)
	Close() error
}

// RuntimeDefaults returns the configured runtime ops defaults (if any).
func (d *Deps) RuntimeDefaults() *config.RuntimeOpsDefaults {
	if d == nil || d.Config == nil {
		return nil
	}
	return d.Config.RuntimeOps
}

// DefaultLeadSyncChannelIdentity defines the baseline channel/app pair for sync factory bootstrap.
func (d *Deps) DefaultLeadSyncChannelIdentity() LeadSyncChannelIdentity {
	_ = d
	return LeadSyncChannelIdentity{
		Channel: "wechat",
		AppType: "wecom",
	}
}

// RuntimeLogger provides a structured logger enriched with runtime metadata.
func (d *Deps) RuntimeLogger(ctx context.Context, component string, extra logger.Fields) *logrus.Entry {
	if extra == nil {
		extra = logger.Fields{}
	}
	if ctx == nil && d != nil {
		ctx = d.Ctx
	}

	var tenantID string
	if tid, ok := authx.TenantUUIDFromContext(ctx); ok && tid != "" {
		tenantID = tid
	}

	traceID := ""
	if ctx != nil {
		if v := ctx.Value("request_id"); v != nil {
			if s, ok := v.(string); ok {
				traceID = s
			}
		}
	}

	return logger.WithRuntimeFields(PluginID, tenantID, traceID, component, extra)
}

func (d *Deps) LocalIAMEnabled() bool {
	return d != nil && d.ProviderMode == iamservice.ProviderModeLocal && d.IAMDirectory != nil
}

func (d *Deps) DelegatedIAMEnabled() bool {
	return d != nil && d.ProviderMode == iamservice.ProviderModeDelegated && d.AuthProxy != nil
}

func (d *Deps) LocalDirectory() iamservice.IAMDirectory {
	if d.LocalIAMEnabled() {
		return d.IAMDirectory
	}
	return nil
}

func (d *Deps) DelegatedProxy() DelegatedAuthProxy {
	if d.DelegatedIAMEnabled() {
		return d.AuthProxy
	}
	return nil
}

// EnsureLeadCaptureRepos initializes lead-capture repositories lazily.
func (d *Deps) EnsureLeadCaptureRepos() *leadrepo.Bundle {
	if d == nil || d.DB == nil {
		return nil
	}
	if d.LeadCaptureRepos == nil {
		d.LeadCaptureRepos = leadrepo.NewBundle(d.DB)
	}
	return d.LeadCaptureRepos
}
