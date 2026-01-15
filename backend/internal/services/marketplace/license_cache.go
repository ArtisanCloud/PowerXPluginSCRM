package marketplace

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	dbm "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/marketplace"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"gorm.io/datatypes"
)

const defaultLicenseCachePrefix = "powerx:marketplace:licenses"

// RedisLicenseCache stores license snapshots in Redis for offline validation.
type RedisLicenseCache struct {
	client *redis.Client
	prefix string
	logger *logrus.Entry
}

// NewRedisLicenseCache connects to redisURL and returns a LicenseCache implementation.
func NewRedisLicenseCache(redisURL, keyPrefix string, logger *logrus.Entry) (*RedisLicenseCache, error) {
	redisURL = strings.TrimSpace(redisURL)
	if redisURL == "" {
		return nil, errors.New("redis url is required")
	}
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("parse redis url: %w", err)
	}
	client := redis.NewClient(opts)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("ping redis: %w", err)
	}
	if strings.TrimSpace(keyPrefix) == "" {
		keyPrefix = defaultLicenseCachePrefix
	}
	if logger == nil {
		logger = logrus.New().WithField("component", "marketplace.license_cache")
	}
	return &RedisLicenseCache{
		client: client,
		prefix: keyPrefix,
		logger: logger,
	}, nil
}

// Get returns a cached license snapshot if present.
func (c *RedisLicenseCache) Get(ctx context.Context, tenantID, listingID string) (*dbm.License, bool) {
	if c == nil || c.client == nil {
		return nil, false
	}
	tenantID = strings.TrimSpace(tenantID)
	listingID = strings.TrimSpace(listingID)
	if tenantID == "" || listingID == "" {
		return nil, false
	}
	key := c.cacheKey(tenantID, listingID)
	raw, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if !errors.Is(err, redis.Nil) && c.logger != nil {
			c.logger.WithError(err).WithField("key", key).Debug("license cache get failed")
		}
		return nil, false
	}
	var payload licenseCacheEntry
	if err := json.Unmarshal(raw, &payload); err != nil {
		if c.logger != nil {
			c.logger.WithError(err).WithField("key", key).Warn("failed to decode license cache payload, deleting")
		}
		_, _ = c.client.Del(ctx, key).Result()
		return nil, false
	}
	return payload.toModel(), true
}

// Set writes a license snapshot with TTL.
func (c *RedisLicenseCache) Set(ctx context.Context, tenantID, listingID string, license *dbm.License, ttl time.Duration) {
	if c == nil || c.client == nil || license == nil {
		return
	}
	tenantID = strings.TrimSpace(tenantID)
	listingID = strings.TrimSpace(listingID)
	if tenantID == "" || listingID == "" {
		return
	}
	if ttl <= 0 {
		ttl = time.Hour
	}
	payload := licenseCacheEntryFromModel(license)
	raw, err := json.Marshal(payload)
	if err != nil {
		if c.logger != nil {
			c.logger.WithError(err).WithField("license_id", license.ID).Warn("failed to marshal license cache payload")
		}
		return
	}
	if err := c.client.Set(ctx, c.cacheKey(tenantID, listingID), raw, ttl).Err(); err != nil && c.logger != nil {
		c.logger.WithError(err).WithField("license_id", license.ID).Warn("failed to set license cache entry")
	}
}

// Delete removes the cached license snapshot.
func (c *RedisLicenseCache) Delete(ctx context.Context, tenantID, listingID string) {
	if c == nil || c.client == nil {
		return
	}
	tenantID = strings.TrimSpace(tenantID)
	listingID = strings.TrimSpace(listingID)
	if tenantID == "" || listingID == "" {
		return
	}
	if err := c.client.Del(ctx, c.cacheKey(tenantID, listingID)).Err(); err != nil && c.logger != nil && !errors.Is(err, redis.Nil) {
		c.logger.WithError(err).WithField("tenant_uuid", tenantID).WithField("listing_id", listingID).Debug("failed to delete license cache entry")
	}
}

func (c *RedisLicenseCache) cacheKey(tenantID, listingID string) string {
	return fmt.Sprintf("%s:%s:%s", c.prefix, tenantID, listingID)
}

type licenseCacheEntry struct {
	ID              string                 `json:"id"`
	TenantUuid      string                 `json:"tenant_uuid"`
	ListingID       string                 `json:"listing_id"`
	PlanID          string                 `json:"plan_id"`
	LicenseToken    string                 `json:"license_token"`
	Status          string                 `json:"status"`
	IssuedAt        time.Time              `json:"issued_at"`
	ExpiresAt       time.Time              `json:"expires_at"`
	RenewalToken    *string                `json:"renewal_token,omitempty"`
	OfflineUntil    *time.Time             `json:"offline_until,omitempty"`
	LastValidatedAt *time.Time             `json:"last_validated_at,omitempty"`
	IssuedBy        *string                `json:"issued_by,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

func licenseCacheEntryFromModel(license *dbm.License) *licenseCacheEntry {
	if license == nil {
		return nil
	}
	meta := map[string]interface{}{}
	for k, v := range license.Metadata {
		meta[k] = v
	}
	return &licenseCacheEntry{
		ID:              license.ID,
		TenantUuid:      license.TenantUuid,
		ListingID:       license.ListingID,
		PlanID:          license.PlanID,
		LicenseToken:    license.LicenseToken,
		Status:          license.Status,
		IssuedAt:        license.IssuedAt,
		ExpiresAt:       license.ExpiresAt,
		RenewalToken:    license.RenewalToken,
		OfflineUntil:    license.OfflineUntil,
		LastValidatedAt: license.LastValidatedAt,
		IssuedBy:        license.IssuedBy,
		Metadata:        meta,
	}
}

func (e *licenseCacheEntry) toModel() *dbm.License {
	if e == nil {
		return nil
	}
	meta := datatypes.JSONMap{}
	for k, v := range e.Metadata {
		meta[k] = v
	}
	return &dbm.License{
		ID:              e.ID,
		TenantUuid:      e.TenantUuid,
		ListingID:       e.ListingID,
		PlanID:          e.PlanID,
		LicenseToken:    e.LicenseToken,
		Status:          e.Status,
		IssuedAt:        e.IssuedAt,
		ExpiresAt:       e.ExpiresAt,
		RenewalToken:    e.RenewalToken,
		OfflineUntil:    e.OfflineUntil,
		LastValidatedAt: e.LastValidatedAt,
		IssuedBy:        e.IssuedBy,
		Metadata:        meta,
	}
}
