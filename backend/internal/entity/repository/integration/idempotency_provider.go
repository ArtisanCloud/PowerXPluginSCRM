package integration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	model "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/integration"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ErrIdempotencyNotFound 表示幂等记录不存在。
var ErrIdempotencyNotFound = errors.New("integration idempotency record not found")

// ErrIdempotencyInvalid 在入参缺失关键字段时返回。
var ErrIdempotencyInvalid = errors.New("integration idempotency record invalid")

// ClaimStatus 表示 Claim 的结果。
type ClaimStatus int

const (
	// ClaimStatusCreated 表示成功创建了新的幂等记录。
	ClaimStatusCreated ClaimStatus = iota
	// ClaimStatusExisting 表示幂等记录已存在，返回已有数据。
	ClaimStatusExisting
)

// ClaimResult 汇总 Claim 操作的结果。
type ClaimResult struct {
	Status ClaimStatus
	Record *model.IdempotencyRecord
}

// IdempotencyProvider 定义幂等记录的核心操作。
type IdempotencyProvider interface {
	// Claim 尝试创建记录；若已存在则返回已存在的记录。
	Claim(ctx context.Context, record *model.IdempotencyRecord) (*ClaimResult, error)
	// SaveResponse 更新请求对应的响应与元数据。
	SaveResponse(ctx context.Context, key string, response json.RawMessage, metadata map[string]any) (*model.IdempotencyRecord, error)
	// Delete 删除幂等记录（通常在回滚或手动清理时使用）。
	Delete(ctx context.Context, key string) error
}

// PostgresIdempotencyProvider 使用 PostgreSQL 作为持久化后端。
type PostgresIdempotencyProvider struct {
	db  *gorm.DB
	ttl time.Duration
	now func() time.Time
}

// NewPostgresIdempotencyProvider 构造基于 PostgreSQL 的幂等存储。
func NewPostgresIdempotencyProvider(db *gorm.DB, ttl time.Duration) *PostgresIdempotencyProvider {
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}

	return &PostgresIdempotencyProvider{
		db:  db,
		ttl: ttl,
		now: time.Now,
	}
}

// WithNow allows tests to override the time source.
func (p *PostgresIdempotencyProvider) WithNow(now func() time.Time) {
	if now != nil {
		p.now = now
	}
}

// Claim 实现在 PostgreSQL 中的幂等 Claim。
func (p *PostgresIdempotencyProvider) Claim(ctx context.Context, record *model.IdempotencyRecord) (*ClaimResult, error) {
	if record == nil {
		return nil, ErrIdempotencyInvalid
	}
	key := strings.TrimSpace(record.Key)
	if key == "" {
		return nil, fmt.Errorf("%w: key must not be empty", ErrIdempotencyInvalid)
	}

	if record.Metadata == nil {
		record.Metadata = datatypes.JSONMap{}
	} else {
		record.Metadata = cloneJSONMap(record.Metadata)
	}

	if record.ExpiresAt == nil {
		expiry := p.now().UTC().Add(p.ttl)
		record.ExpiresAt = &expiry
	} else {
		expiry := record.ExpiresAt.UTC()
		record.ExpiresAt = &expiry
	}

	result := p.db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(record)
	if result.Error != nil {
		return nil, result.Error
	}

	stored, err := p.loadByKey(ctx, key)
	if err != nil {
		return nil, err
	}

	if result.RowsAffected > 0 {
		return &ClaimResult{Status: ClaimStatusCreated, Record: stored}, nil
	}

	return &ClaimResult{Status: ClaimStatusExisting, Record: stored}, nil
}

// SaveResponse 更新响应 JSON 与元数据。
func (p *PostgresIdempotencyProvider) SaveResponse(
	ctx context.Context,
	key string,
	response json.RawMessage,
	metadata map[string]any,
) (*model.IdempotencyRecord, error) {

	key = strings.TrimSpace(key)
	if key == "" {
		return nil, fmt.Errorf("%w: key must not be empty", ErrIdempotencyInvalid)
	}

	updates := map[string]any{
		"response_data": datatypes.JSON(response),
		"expires_at":    p.now().UTC().Add(p.ttl),
	}

	if metadata != nil {
		updates["metadata"] = datatypes.JSONMap(cloneGenericMap(metadata))
	}

	result := p.db.WithContext(ctx).
		Model(&model.IdempotencyRecord{}).
		Where("key = ?", key).
		Updates(updates)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, ErrIdempotencyNotFound
	}

	return p.loadByKey(ctx, key)
}

// Delete 移除幂等记录。
func (p *PostgresIdempotencyProvider) Delete(ctx context.Context, key string) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return fmt.Errorf("%w: key must not be empty", ErrIdempotencyInvalid)
	}

	result := p.db.WithContext(ctx).Delete(&model.IdempotencyRecord{}, "key = ?", key)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrIdempotencyNotFound
	}
	return nil
}

func (p *PostgresIdempotencyProvider) loadByKey(ctx context.Context, key string) (*model.IdempotencyRecord, error) {
	var record model.IdempotencyRecord
	err := p.db.WithContext(ctx).First(&record, "key = ?", key).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrIdempotencyNotFound
		}
		return nil, err
	}
	return cloneRecord(record), nil
}

// RedisClient 定义 Redis 所需的最小接口，便于单元测试替换。
type RedisClient interface {
	SetNX(ctx context.Context, key string, value string, ttl time.Duration) (bool, error)
	Get(ctx context.Context, key string) (string, error)
	TTL(ctx context.Context, key string) (time.Duration, error)
	Set(ctx context.Context, key string, value string, ttl time.Duration) error
	Del(ctx context.Context, keys ...string) (int64, error)
}

// ErrCacheMiss 表示 Redis 未命中。
var ErrCacheMiss = errors.New("idempotency cache miss")

// RedisIdempotencyProvider 使用 Redis 作为优先幂等存储。
type RedisIdempotencyProvider struct {
	client RedisClient
	ttl    time.Duration
	prefix string
	now    func() time.Time
}

// NewRedisIdempotencyProvider 构造 Redis 幂等存储实现。
func NewRedisIdempotencyProvider(client RedisClient, ttl time.Duration, prefix string) *RedisIdempotencyProvider {
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	if strings.TrimSpace(prefix) == "" {
		prefix = "integration:idempotency:"
	}
	return &RedisIdempotencyProvider{
		client: client,
		ttl:    ttl,
		prefix: prefix,
		now:    time.Now,
	}
}

// WithNow allows overriding the time source (primarily for tests).
func (p *RedisIdempotencyProvider) WithNow(now func() time.Time) {
	if now != nil {
		p.now = now
	}
}

// Claim 将请求注册到 Redis；若已存在则返回缓存内容。
func (p *RedisIdempotencyProvider) Claim(ctx context.Context, record *model.IdempotencyRecord) (*ClaimResult, error) {
	if record == nil {
		return nil, ErrIdempotencyInvalid
	}
	key := strings.TrimSpace(record.Key)
	if key == "" {
		return nil, fmt.Errorf("%w: key must not be empty", ErrIdempotencyInvalid)
	}

	payload, err := json.Marshal(toRedisRecord(record, p.now().UTC(), p.ttl))
	if err != nil {
		return nil, err
	}

	namespaced := p.namespacedKey(key)
	created, err := p.client.SetNX(ctx, namespaced, string(payload), p.ttl)
	if err != nil {
		return nil, err
	}
	if created {
		stored, err := decodeRedisRecord(string(payload))
		if err != nil {
			return nil, err
		}
		return &ClaimResult{Status: ClaimStatusCreated, Record: stored.toModel()}, nil
	}

	stored, err := p.get(ctx, key)
	if err != nil {
		return nil, err
	}
	return &ClaimResult{Status: ClaimStatusExisting, Record: stored.toModel()}, nil
}

// SaveResponse 更新 Redis 中的响应信息。
func (p *RedisIdempotencyProvider) SaveResponse(
	ctx context.Context,
	key string,
	response json.RawMessage,
	metadata map[string]any,
) (*model.IdempotencyRecord, error) {
	stored, err := p.get(ctx, key)
	if err != nil {
		return nil, err
	}

	if response != nil {
		stored.Response = append(json.RawMessage(nil), response...)
	}
	if metadata != nil {
		stored.Metadata = cloneGenericMap(metadata)
	}
	stored.ExpiresAt = p.now().UTC().Add(p.ttl)

	payload, err := json.Marshal(stored)
	if err != nil {
		return nil, err
	}

	ttl, err := p.client.TTL(ctx, p.namespacedKey(key))
	if err != nil || ttl <= 0 {
		ttl = p.ttl
	}

	if err := p.client.Set(ctx, p.namespacedKey(key), string(payload), ttl); err != nil {
		return nil, err
	}
	return stored.toModel(), nil
}

// Delete 从 Redis 移除幂等记录。
func (p *RedisIdempotencyProvider) Delete(ctx context.Context, key string) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return fmt.Errorf("%w: key must not be empty", ErrIdempotencyInvalid)
	}
	affected, err := p.client.Del(ctx, p.namespacedKey(key))
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrIdempotencyNotFound
	}
	return nil
}

func (p *RedisIdempotencyProvider) get(ctx context.Context, key string) (*redisRecord, error) {
	value, err := p.client.Get(ctx, p.namespacedKey(key))
	if err != nil {
		if errors.Is(err, ErrCacheMiss) {
			return nil, ErrIdempotencyNotFound
		}
		return nil, err
	}
	stored, err := decodeRedisRecord(value)
	if err != nil {
		return nil, err
	}
	return stored, nil
}

func (p *RedisIdempotencyProvider) namespacedKey(key string) string {
	return p.prefix + key
}

// redisRecord 是 Redis 中序列化的结构。
type redisRecord struct {
	Key         string          `json:"key"`
	TenantUuid  string          `json:"tenant_uuid"`
	Scope       string          `json:"scope"`
	Operation   string          `json:"operation"`
	PayloadHash string          `json:"payload_hash"`
	Response    json.RawMessage `json:"response"`
	Metadata    map[string]any  `json:"metadata"`
	ExpiresAt   time.Time       `json:"expires_at"`
	CreatedAt   time.Time       `json:"created_at"`
}

func toRedisRecord(record *model.IdempotencyRecord, now time.Time, ttl time.Duration) *redisRecord {
	if record == nil {
		return nil
	}

	var expires time.Time
	if record.ExpiresAt != nil && !record.ExpiresAt.IsZero() {
		expires = record.ExpiresAt.UTC()
	} else {
		expires = now.Add(ttl)
	}

	created := record.CreatedAt
	if created.IsZero() {
		created = now
	}

	return &redisRecord{
		Key:         record.Key,
		TenantUuid:  record.TenantUuid,
		Scope:       record.Scope,
		Operation:   record.Operation,
		PayloadHash: record.PayloadHash,
		Response:    cloneBytes(record.Response),
		Metadata:    cloneJSONMapToGeneric(record.Metadata),
		ExpiresAt:   expires,
		CreatedAt:   created.UTC(),
	}
}

func decodeRedisRecord(raw string) (*redisRecord, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, fmt.Errorf("%w: empty redis payload", ErrIdempotencyInvalid)
	}
	var rec redisRecord
	if err := json.Unmarshal([]byte(raw), &rec); err != nil {
		return nil, err
	}
	if rec.Metadata == nil {
		rec.Metadata = map[string]any{}
	}
	return &rec, nil
}

func (r *redisRecord) toModel() *model.IdempotencyRecord {
	if r == nil {
		return nil
	}
	expiry := r.ExpiresAt.UTC()
	return &model.IdempotencyRecord{
		Key:         r.Key,
		TenantUuid:  r.TenantUuid,
		Scope:       r.Scope,
		Operation:   r.Operation,
		PayloadHash: r.PayloadHash,
		Response:    datatypes.JSON(cloneBytes(r.Response)),
		Metadata:    datatypes.JSONMap(cloneGenericMap(r.Metadata)),
		ExpiresAt:   &expiry,
		CreatedAt:   r.CreatedAt.UTC(),
	}
}

func cloneRecord(src model.IdempotencyRecord) *model.IdempotencyRecord {
	dst := src
	if src.ExpiresAt != nil {
		expiry := src.ExpiresAt.UTC()
		dst.ExpiresAt = &expiry
	}
	dst.Response = datatypes.JSON(cloneBytes(src.Response))
	if src.Metadata != nil {
		dst.Metadata = cloneJSONMap(src.Metadata)
	}
	return &dst
}

func cloneJSONMap(src datatypes.JSONMap) datatypes.JSONMap {
	if src == nil {
		return datatypes.JSONMap{}
	}
	dst := make(datatypes.JSONMap, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func cloneJSONMapToGeneric(src datatypes.JSONMap) map[string]any {
	if src == nil {
		return map[string]any{}
	}
	dst := make(map[string]any, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func cloneGenericMap(src map[string]any) map[string]any {
	if src == nil {
		return map[string]any{}
	}
	dst := make(map[string]any, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func cloneBytes(src []byte) []byte {
	if src == nil {
		return nil
	}
	return append([]byte(nil), src...)
}
