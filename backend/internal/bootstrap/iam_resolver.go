package bootstrap

import (
	"fmt"
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/config"
	iamservice "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/iam"
)

// ProviderResolver determines whether business providers use local services or
// delegated PowerX Core capabilities. POWERX_PROXY is intentionally unrelated.
type ProviderResolver struct {
	mode   iamservice.ProviderMode
	source string
}

func NewProviderResolver(cfg *config.Config) (*ProviderResolver, error) {
	if cfg != nil && cfg.Context != nil {
		if parsed, ok := parseProviderMode(cfg.Context.ProviderMode); ok {
			return &ProviderResolver{mode: parsed, source: "context.provider_mode"}, nil
		}
		if strings.TrimSpace(cfg.Context.ProviderMode) != "" {
			return nil, fmt.Errorf("invalid context.provider_mode %q: expected local or delegated", cfg.Context.ProviderMode)
		}
	}

	return nil, fmt.Errorf("POWERX_PROVIDER_MODE or context.provider_mode is required")
}

func (r *ProviderResolver) Mode() iamservice.ProviderMode {
	if r == nil {
		return iamservice.ProviderModeLocal
	}
	return r.mode
}

func (r *ProviderResolver) Source() string {
	if r == nil {
		return "auto"
	}
	return r.source
}

func (r *ProviderResolver) IsLocal() bool {
	return r != nil && r.mode == iamservice.ProviderModeLocal
}

func (r *ProviderResolver) IsDelegated() bool {
	return r != nil && r.mode == iamservice.ProviderModeDelegated
}

func parseProviderMode(val string) (iamservice.ProviderMode, bool) {
	v := strings.ToLower(strings.TrimSpace(val))
	switch v {
	case "delegated":
		return iamservice.ProviderModeDelegated, true
	case "local":
		return iamservice.ProviderModeLocal, true
	default:
		return iamservice.ProviderMode(""), false
	}
}
