package bootstrap

import (
	"os"
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/config"
	iamservice "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/iam"
)

// IAMResolver determines whether the plugin should rely on delegated (PowerX Core)
// or local IAM. Priority: config.context.iam_mode > POWERX_RBAC_DELEGATE > POWERX_PROXY.
type IAMResolver struct {
	mode   iamservice.IAMMode
	source string
}

func NewIAMResolver(cfg *config.Config) *IAMResolver {
	mode := iamservice.IAMModeLocal
	source := "auto"

	if cfg != nil && cfg.Context != nil {
		if parsed, ok := parseIAMMode(cfg.Context.IAMMode); ok {
			return &IAMResolver{mode: parsed, source: "config"}
		}
	}

	if truthy(os.Getenv("POWERX_RBAC_DELEGATE")) {
		return &IAMResolver{mode: iamservice.IAMModeDelegated, source: "env:POWERX_RBAC_DELEGATE"}
	}

	if os.Getenv("POWERX_PROXY") == "1" {
		mode = iamservice.IAMModeDelegated
		source = "env:POWERX_PROXY"
	}

	return &IAMResolver{mode: mode, source: source}
}

func (r *IAMResolver) Mode() iamservice.IAMMode {
	if r == nil {
		return iamservice.IAMModeLocal
	}
	return r.mode
}

func (r *IAMResolver) Source() string {
	if r == nil {
		return "auto"
	}
	return r.source
}

func (r *IAMResolver) IsLocal() bool {
	return r != nil && r.mode == iamservice.IAMModeLocal
}

func (r *IAMResolver) IsDelegated() bool {
	return r != nil && r.mode == iamservice.IAMModeDelegated
}

func parseIAMMode(val string) (iamservice.IAMMode, bool) {
	v := strings.ToLower(strings.TrimSpace(val))
	switch v {
	case "delegated":
		return iamservice.IAMModeDelegated, true
	case "local":
		return iamservice.IAMModeLocal, true
	default:
		return iamservice.IAMMode(""), false
	}
}

func truthy(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
