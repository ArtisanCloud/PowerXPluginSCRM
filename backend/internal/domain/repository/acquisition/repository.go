package acquisition

import (
	"context"
	"errors"
	"strings"
	"time"

	acqmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/models/acquisition"
	"gorm.io/gorm"
)

var (
	ErrRepositoryDBNotReady = errors.New("acquisition repository database is not initialized")
	ErrTenantUUIDRequired   = errors.New("tenant_uuid is required")
	ErrStaffCodeUUIDRequire = errors.New("staff_code_uuid is required")
	ErrRecordNotFound       = errors.New("record not found")
)

type StaffLiveCodeListFilter struct {
	ActivityName string
	Status       string
	Limit        int
}

type StaffLiveCodeRepository interface {
	Create(ctx context.Context, item *acqmodel.StaffLiveCode) error
	GetByUUID(ctx context.Context, tenantUUID, staffCodeUUID string) (*acqmodel.StaffLiveCode, error)
	List(ctx context.Context, tenantUUID string, filter StaffLiveCodeListFilter) ([]*acqmodel.StaffLiveCode, error)
	UpdateStatus(ctx context.Context, tenantUUID, staffCodeUUID, status, updatedBy string) (*acqmodel.StaffLiveCode, error)
	CountConfirmedMappings(ctx context.Context, tenantUUID string, memberUUIDs []string) (int64, error)
	ExistsByCodeKey(ctx context.Context, tenantUUID, codeKey string) (bool, error)
}

type StaffWelcomeConfigRepository interface {
	Save(ctx context.Context, item *acqmodel.StaffWelcomeConfig) error
	GetByStaffCodeUUID(ctx context.Context, tenantUUID, staffCodeUUID string) (*acqmodel.StaffWelcomeConfig, error)
}

type StaffWelcomeSyncAttemptRepository interface {
	Create(ctx context.Context, item *acqmodel.StaffWelcomeSyncAttempt) error
	ListByStaffCodeUUID(ctx context.Context, tenantUUID, staffCodeUUID string, limit int) ([]*acqmodel.StaffWelcomeSyncAttempt, error)
}

type GroupLiveCodeRepository interface {
	Create(ctx context.Context, item *acqmodel.GroupLiveCode) error
	GetByUUID(ctx context.Context, tenantUUID, groupCodeUUID string) (*acqmodel.GroupLiveCode, error)
	Update(ctx context.Context, item *acqmodel.GroupLiveCode) error
	Delete(ctx context.Context, tenantUUID, groupCodeUUID string) error
	List(ctx context.Context, tenantUUID string, limit int) ([]*acqmodel.GroupLiveCode, error)
}

type GroupChatSnapshotRepository interface {
	Upsert(ctx context.Context, item *acqmodel.GroupChatSnapshot) error
	GetByChatID(ctx context.Context, tenantUUID, chatID string) (*acqmodel.GroupChatSnapshot, error)
	List(ctx context.Context, tenantUUID string, limit int) ([]*acqmodel.GroupChatSnapshot, error)
}

type GroupTagRepository interface {
	CreateDefinition(ctx context.Context, item *acqmodel.GroupTagDefinition) error
	ListDefinitions(ctx context.Context, tenantUUID string, limit int) ([]*acqmodel.GroupTagDefinition, error)
	GetDefinitionByUUID(ctx context.Context, tenantUUID, groupTagUUID string) (*acqmodel.GroupTagDefinition, error)
	BindChats(ctx context.Context, tenantUUID, groupTagUUID string, chatIDs []string, bindSource, ruleRunUUID string) (int, error)
	ListBindings(ctx context.Context, tenantUUID, groupTagUUID string, limit int) ([]*acqmodel.GroupTagBinding, error)
	CreateRuleRun(ctx context.Context, item *acqmodel.GroupTagRuleRun) error
}

type Bundle struct {
	StaffLiveCodes      StaffLiveCodeRepository
	StaffWelcomeConfigs StaffWelcomeConfigRepository
	StaffWelcomeAttempt StaffWelcomeSyncAttemptRepository
	GroupLiveCodes      GroupLiveCodeRepository
	GroupChatSnapshots  GroupChatSnapshotRepository
	GroupTags           GroupTagRepository
}

func NewBundle(db *gorm.DB) *Bundle {
	if db == nil {
		return &Bundle{}
	}
	return &Bundle{
		StaffLiveCodes:      NewStaffLiveCodeRepository(db),
		StaffWelcomeConfigs: NewStaffWelcomeConfigRepository(db),
		StaffWelcomeAttempt: NewStaffWelcomeSyncAttemptRepository(db),
		GroupLiveCodes:      NewGroupLiveCodeRepository(db),
		GroupChatSnapshots:  NewGroupChatSnapshotRepository(db),
		GroupTags:           NewGroupTagRepository(db),
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

func normalizeStaffCodeUUID(staffCodeUUID string) (string, error) {
	staffCodeUUID = strings.ToLower(strings.TrimSpace(staffCodeUUID))
	if staffCodeUUID == "" {
		return "", ErrStaffCodeUUIDRequire
	}
	return staffCodeUUID, nil
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
