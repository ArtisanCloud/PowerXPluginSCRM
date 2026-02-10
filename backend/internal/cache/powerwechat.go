package cache

import (
	"strings"

	"github.com/ArtisanCloud/PowerWeChat/v3/src/kernel"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/config"
	"github.com/sirupsen/logrus"
)

// NewPowerWeChatCache builds PowerWeChat cache from unified cache config.
func NewPowerWeChatCache(cfg *config.Config, logger *logrus.Entry) kernel.CacheInterface {
	if cfg == nil || cfg.Cache == nil {
		return nil
	}
	driver := strings.ToLower(strings.TrimSpace(cfg.Cache.Driver))
	if driver == "" {
		if cfg.Server != nil && cfg.Server.DevMode {
			driver = "memory"
		} else {
			driver = "redis"
		}
	}
	if driver != "redis" {
		return nil
	}
	options, err := buildRedisOptions(cfg.Cache)
	if err != nil {
		if logger != nil {
			logger.WithError(err).Warn("wecom redis cache init failed")
		}
		return nil
	}
	cache := kernel.NewRedisClient(&kernel.UniversalOptions{
		Addrs:        []string{options.Addr},
		Username:     options.Username,
		Password:     options.Password,
		DB:           options.DB,
		DialTimeout:  options.DialTimeout,
		ReadTimeout:  options.ReadTimeout,
		WriteTimeout: options.WriteTimeout,
	})
	if cache == nil && logger != nil {
		logger.Warn("wecom redis cache unavailable")
	}
	return cache
}
