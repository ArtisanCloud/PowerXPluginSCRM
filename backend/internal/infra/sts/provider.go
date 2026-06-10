package sts

import "time"

const (
	DefaultAudience = "powerx:api"
	DefaultScope    = "access"
	MaxTTL          = 300 * time.Second
)

func NormalizeExchangeConfig(audience, scope string, ttl time.Duration) (string, string, time.Duration) {
	if audience == "" {
		audience = DefaultAudience
	}
	if scope == "" {
		scope = DefaultScope
	}
	if ttl <= 0 || ttl > MaxTTL {
		ttl = MaxTTL
	}
	return audience, scope, ttl
}
