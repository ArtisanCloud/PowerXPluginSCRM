package cache

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/config"
	"github.com/redis/go-redis/v9"
)

type RedisStore struct {
	client *redis.Client
	cfg    *config.CacheConfig
}

func NewRedisStore(cfg *config.CacheConfig) (*RedisStore, error) {
	options, err := buildRedisOptions(cfg)
	if err != nil {
		return nil, err
	}
	client := redis.NewClient(options)
	return &RedisStore{client: client, cfg: cfg}, nil
}

func (r *RedisStore) Get(ctx context.Context, key string) ([]byte, bool) {
	val, err := r.client.Get(ctx, buildKey(r.prefix(), key)).Bytes()
	if err != nil {
		return nil, false
	}
	return val, true
}

func (r *RedisStore) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	ttl = normalizeTTL(r.cfg, ttl)
	return r.client.Set(ctx, buildKey(r.prefix(), key), value, ttl).Err()
}

func (r *RedisStore) Delete(ctx context.Context, key string) error {
	return r.client.Del(ctx, buildKey(r.prefix(), key)).Err()
}

func (r *RedisStore) TTL(ctx context.Context, key string) (time.Duration, error) {
	return r.client.TTL(ctx, buildKey(r.prefix(), key)).Result()
}

func (r *RedisStore) Incr(ctx context.Context, key string, delta int64) (int64, error) {
	return r.client.IncrBy(ctx, buildKey(r.prefix(), key), delta).Result()
}

func (r *RedisStore) prefix() string {
	if r.cfg == nil {
		return ""
	}
	return r.cfg.Prefix
}

func buildRedisOptions(cfg *config.CacheConfig) (*redis.Options, error) {
	if cfg == nil {
		return nil, fmt.Errorf("cache config missing")
	}
	if urlStr := strings.TrimSpace(cfg.RedisURL); urlStr != "" {
		options, err := redis.ParseURL(urlStr)
		if err != nil {
			return nil, err
		}
		applyRedisTimeouts(cfg, options)
		return options, nil
	}
	host := strings.TrimSpace(cfg.Host)
	if host == "" {
		return nil, fmt.Errorf("cache host missing")
	}
	port := cfg.Port
	if port == 0 {
		port = 6379
	}
	options := &redis.Options{
		Addr:     fmt.Sprintf("%s:%d", host, port),
		Password: cfg.Password,
		DB:       cfg.DB,
	}
	applyRedisTimeouts(cfg, options)
	return options, nil
}

func applyRedisTimeouts(cfg *config.CacheConfig, options *redis.Options) {
	if cfg == nil || options == nil {
		return
	}
	if cfg.DialTimeout > 0 {
		options.DialTimeout = cfg.DialTimeout
	}
	if cfg.ReadTimeout > 0 {
		options.ReadTimeout = cfg.ReadTimeout
	}
	if cfg.WriteTimeout > 0 {
		options.WriteTimeout = cfg.WriteTimeout
	}
}
