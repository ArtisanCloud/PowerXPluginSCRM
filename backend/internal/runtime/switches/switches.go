package switches

import (
	"os"
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/config"
)

const (
	DriverAuto   = "auto"
	DriverLocal  = "local"
	DriverHost   = "host"
	DriverRedis  = "redis"
	DriverNoop   = "noop"
	DriverMemory = "memory"
)

type Drivers struct {
	WSBus      string
	TaskBus    string
	EventTopic string
	Cache      string
}

func Resolve(cfg *config.Config) Drivers {
	return Drivers{
		WSBus:      resolveAutoDriver(runtimeDriverValue(cfg, func(r *config.RuntimeDriversConfig) string { return r.WSBus })),
		TaskBus:    resolveAutoDriver(runtimeDriverValue(cfg, func(r *config.RuntimeDriversConfig) string { return r.TaskBus })),
		EventTopic: resolveAutoDriver(runtimeDriverValue(cfg, func(r *config.RuntimeDriversConfig) string { return r.EventTopic })),
		Cache:      resolveCacheDriver(cfg, runtimeDriverValue(cfg, func(r *config.RuntimeDriversConfig) string { return r.Cache })),
	}
}

func IsHostDriver(v string) bool {
	return strings.EqualFold(strings.TrimSpace(v), DriverHost)
}

func IsLocalDriver(v string) bool {
	return strings.EqualFold(strings.TrimSpace(v), DriverLocal)
}

func runtimeDriverValue(cfg *config.Config, selector func(*config.RuntimeDriversConfig) string) string {
	if cfg == nil || cfg.Runtime == nil || cfg.Runtime.Drivers == nil || selector == nil {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(selector(cfg.Runtime.Drivers)))
}

func resolveAutoDriver(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case DriverHost:
		return DriverHost
	case DriverLocal:
		return DriverLocal
	case "", DriverAuto:
		if isProxyEnabled() {
			return DriverHost
		}
		return DriverLocal
	default:
		if isProxyEnabled() {
			return DriverHost
		}
		return DriverLocal
	}
}

func resolveCacheDriver(cfg *config.Config, runtimeOverride string) string {
	switch strings.ToLower(strings.TrimSpace(runtimeOverride)) {
	case DriverMemory, DriverRedis, DriverNoop:
		return strings.ToLower(strings.TrimSpace(runtimeOverride))
	}

	cacheDriver := ""
	if cfg != nil && cfg.Cache != nil {
		cacheDriver = strings.ToLower(strings.TrimSpace(cfg.Cache.Driver))
	}
	switch cacheDriver {
	case DriverMemory, DriverRedis, DriverNoop:
		return cacheDriver
	}

	if cfg != nil && cfg.Server != nil && cfg.Server.DevMode {
		return DriverMemory
	}
	return DriverRedis
}

func isProxyEnabled() bool {
	return strings.TrimSpace(os.Getenv("POWERX_PROXY")) == "1"
}
