package integration

import (
	"context"
	"encoding/json"
	"errors"

	model "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/integration"
	repository "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository"
	"github.com/sirupsen/logrus"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// ErrIdempotencyUnavailable 在未配置任何后端时返回。
var ErrIdempotencyUnavailable = errors.New("idempotency repository has no configured backend")

// IdempotencyRepository 协调主、备份幂等后端（Redis → Postgres）。
type IdempotencyRepository struct {
	*repository.BaseRepository[model.IdempotencyRecord]
	primary  IdempotencyProvider
	fallback IdempotencyProvider
	logger   *logrus.Entry
}

// NewIdempotencyRepository 构造仓储实例。
func NewIdempotencyRepository(db *gorm.DB, primary, fallback IdempotencyProvider, logger *logrus.Entry) *IdempotencyRepository {
	return &IdempotencyRepository{
		BaseRepository: repository.NewBaseRepository[model.IdempotencyRecord](db),
		primary:        primary,
		fallback:       fallback,
		logger:         logger,
	}
}

// Claim 尝试注册幂等请求，若主存储不可用则回退到备份。
func (r *IdempotencyRepository) Claim(ctx context.Context, record *model.IdempotencyRecord) (*ClaimResult, error) {
	if record == nil {
		return nil, ErrIdempotencyInvalid
	}

	if r.primary == nil {
		if r.fallback == nil {
			return nil, ErrIdempotencyUnavailable
		}
		return r.fallback.Claim(ctx, cloneModelRecord(record))
	}

	res, err := r.primary.Claim(ctx, cloneModelRecord(record))
	if err != nil {
		if r.fallback != nil {
			return r.fallback.Claim(ctx, cloneModelRecord(record))
		}
		return nil, err
	}

	if r.fallback != nil && res != nil && res.Status == ClaimStatusCreated && res.Record != nil {
		if _, ferr := r.fallback.Claim(ctx, cloneModelRecord(res.Record)); ferr != nil && r.logger != nil {
			r.logger.WithError(ferr).WithField("idempotency_key", res.Record.Key).Warn("failed to sync idempotency claim to fallback store")
		}
	}

	return res, nil
}

// SaveResponse 写入响应结果，优先主存储，失败时尝试备份。
func (r *IdempotencyRepository) SaveResponse(ctx context.Context, key string, response json.RawMessage, metadata map[string]any) (*model.IdempotencyRecord, error) {
	if r.primary == nil {
		if r.fallback == nil {
			return nil, ErrIdempotencyUnavailable
		}
		return r.fallback.SaveResponse(ctx, key, response, metadata)
	}

	record, err := r.primary.SaveResponse(ctx, key, response, metadata)
	if err != nil {
		if errors.Is(err, ErrIdempotencyNotFound) && r.fallback != nil {
			return r.fallback.SaveResponse(ctx, key, response, metadata)
		}
		if r.fallback != nil {
			if fallbackRecord, ferr := r.fallback.SaveResponse(ctx, key, response, metadata); ferr == nil {
				return fallbackRecord, nil
			}
		}
		return nil, err
	}

	if r.fallback != nil {
		if _, ferr := r.fallback.SaveResponse(ctx, key, response, metadata); ferr != nil && r.logger != nil && !errors.Is(ferr, ErrIdempotencyNotFound) {
			r.logger.WithError(ferr).WithField("idempotency_key", key).Warn("failed to propagate idempotency response to fallback store")
		}
	}
	return record, nil
}

// Delete 删除幂等记录。
func (r *IdempotencyRepository) Delete(ctx context.Context, key string) error {
	var errs []error

	if r.primary != nil {
		if err := r.primary.Delete(ctx, key); err != nil && !errors.Is(err, ErrIdempotencyNotFound) {
			errs = append(errs, err)
		}
	}

	if r.fallback != nil {
		if err := r.fallback.Delete(ctx, key); err != nil && !errors.Is(err, ErrIdempotencyNotFound) {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	if r.primary == nil && r.fallback == nil {
		return ErrIdempotencyUnavailable
	}

	return nil
}

// cloneModelRecord 复制幂等记录，避免在不同存储间共享引用。
func cloneModelRecord(src *model.IdempotencyRecord) *model.IdempotencyRecord {
	if src == nil {
		return nil
	}
	dst := *src

	if src.Metadata != nil {
		meta := make(datatypes.JSONMap, len(src.Metadata))
		for k, v := range src.Metadata {
			meta[k] = v
		}
		dst.Metadata = meta
	}

	if src.Response != nil {
		dst.Response = append(datatypes.JSON(nil), src.Response...)
	}

	if src.ExpiresAt != nil {
		ts := *src.ExpiresAt
		dst.ExpiresAt = &ts
	}

	return &dst
}

// EnsureProvider 针对缺失主存储的情况提供友好错误。
func (r *IdempotencyRepository) EnsureProvider() error {
	if r.primary == nil && r.fallback == nil {
		return ErrIdempotencyUnavailable
	}
	return nil
}
