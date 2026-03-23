package cache

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/config"
	runtimeswitch "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/runtime/switches"
	"github.com/sirupsen/logrus"
)

// Store defines the unified cache interface.
type Store interface {
	Get(ctx context.Context, key string) ([]byte, bool)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	TTL(ctx context.Context, key string) (time.Duration, error)
	Incr(ctx context.Context, key string, delta int64) (int64, error)
}

// NewStore builds cache store from config.
func NewStore(cfg *config.Config, logger *logrus.Entry) Store {
	cacheCfg := resolveCacheConfig(cfg)
	driver := runtimeswitch.Resolve(cfg).Cache
	switch driver {
	case "redis":
		store, err := NewRedisStore(cacheCfg)
		if err != nil {
			if logger != nil {
				logger.WithError(err).Warn("cache redis init failed, fallback to memory")
			}
			return NewMemoryStore(cacheCfg)
		}
		return store
	case "noop":
		return NewNoopStore()
	default:
		return NewMemoryStore(cacheCfg)
	}
}

func resolveCacheConfig(cfg *config.Config) *config.CacheConfig {
	if cfg == nil || cfg.Cache == nil {
		return &config.CacheConfig{}
	}
	return cfg.Cache
}

func normalizeTTL(cacheCfg *config.CacheConfig, ttl time.Duration) time.Duration {
	if ttl > 0 {
		return ttl
	}
	if cacheCfg == nil {
		return 0
	}
	if cacheCfg.DefaultTTL > 0 {
		return cacheCfg.DefaultTTL
	}
	return 0
}

func buildKey(prefix, key string) string {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		return key
	}
	return fmt.Sprintf("%s:%s", prefix, key)
}
