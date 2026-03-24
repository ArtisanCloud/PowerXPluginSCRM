package lead_capture

import (
	"context"
	"errors"
	"strings"
	"time"

	leadmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/models/lead_capture"
	"gorm.io/gorm"
)

var (
	ErrRepositoryDBNotReady = errors.New("lead capture repository database is not initialized")
	ErrTenantUUIDRequired   = errors.New("tenant_uuid is required")
	ErrCodeUUIDRequired     = errors.New("code_uuid is required")
	ErrRecordNotFound       = errors.New("record not found")
)

type ChannelCodeRepository interface {
	Create(ctx context.Context, item *leadmodel.ChannelCode) error
	GetByCodeUUID(ctx context.Context, tenantUUID, codeUUID string) (*leadmodel.ChannelCode, error)
	GetByCodeKey(ctx context.Context, tenantUUID, channel, codeKey string) (*leadmodel.ChannelCode, error)
	List(ctx context.Context, tenantUUID string, filter ChannelCodeListFilter) ([]*leadmodel.ChannelCode, error)
	UpdateStatus(ctx context.Context, tenantUUID, codeUUID, status, updatedBy string) (*leadmodel.ChannelCode, error)
}

type CodeWelcomeConfigRepository interface {
	Save(ctx context.Context, item *leadmodel.CodeWelcomeConfig) error
	GetByCodeUUID(ctx context.Context, tenantUUID, codeUUID string) (*leadmodel.CodeWelcomeConfig, error)
}

type CodeWelcomeSyncAttemptRepository interface {
	Create(ctx context.Context, item *leadmodel.CodeWelcomeSyncAttempt) error
	ListByCodeUUID(ctx context.Context, tenantUUID, codeUUID string, limit int) ([]*leadmodel.CodeWelcomeSyncAttempt, error)
}

type ChannelCodeEventRepository interface {
	Create(ctx context.Context, item *leadmodel.ChannelCodeEvent) error
	GetByIdempotencyKey(ctx context.Context, tenantUUID, idempotencyKey string) (*leadmodel.ChannelCodeEvent, error)
	ListByCodeUUID(ctx context.Context, tenantUUID, codeUUID string, limit int) ([]*leadmodel.ChannelCodeEvent, error)
	CountByCodeUUID(ctx context.Context, tenantUUID, codeUUID string) (int64, error)
}

type LeadAttributionRepository interface {
	Create(ctx context.Context, item *leadmodel.LeadAttributionRecord) error
	ListByLeadUUID(ctx context.Context, tenantUUID, leadUUID string) ([]*leadmodel.LeadAttributionRecord, error)
	ListByEventUUID(ctx context.Context, tenantUUID, eventUUID string) ([]*leadmodel.LeadAttributionRecord, error)
	CountByCodeUUID(ctx context.Context, tenantUUID, codeUUID string) (int64, error)
}

type CodeConfigChangeLogRepository interface {
	Create(ctx context.Context, item *leadmodel.CodeConfigChangeLog) error
	ListByCodeUUID(ctx context.Context, tenantUUID, codeUUID string, limit int) ([]*leadmodel.CodeConfigChangeLog, error)
}

type ChannelCodeListFilter struct {
	Channel            string
	AppType            string
	ChannelAccountUUID string
	Status             string
	Limit              int
}

type Bundle struct {
	ChannelCodes       ChannelCodeRepository
	WelcomeConfigs     CodeWelcomeConfigRepository
	WelcomeSyncAttempt CodeWelcomeSyncAttemptRepository
	ChannelCodeEvents  ChannelCodeEventRepository
	Attributions       LeadAttributionRepository
	ConfigChangeLogs   CodeConfigChangeLogRepository
}

func NewBundle(db *gorm.DB) *Bundle {
	if db == nil {
		return &Bundle{}
	}
	return &Bundle{
		ChannelCodes:       NewChannelCodeRepository(db),
		WelcomeConfigs:     NewCodeWelcomeConfigRepository(db),
		WelcomeSyncAttempt: NewCodeWelcomeSyncAttemptRepository(db),
		ChannelCodeEvents:  NewChannelCodeEventRepository(db),
		Attributions:       NewLeadAttributionRepository(db),
		ConfigChangeLogs:   NewCodeConfigChangeLogRepository(db),
	}
}

type gormStore struct {
	db *gorm.DB
}

func normalizeTenant(tenantUUID string) (string, error) {
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	if tenantUUID == "" {
		return "", ErrTenantUUIDRequired
	}
	return tenantUUID, nil
}

func normalizeCodeUUID(codeUUID string) (string, error) {
	codeUUID = strings.ToLower(strings.TrimSpace(codeUUID))
	if codeUUID == "" {
		return "", ErrCodeUUIDRequired
	}
	return codeUUID, nil
}

func ensureLimit(limit, fallback int) int {
	if limit > 0 {
		return limit
	}
	if fallback > 0 {
		return fallback
	}
	return 20
}

func utcNow() time.Time {
	return time.Now().UTC()
}
