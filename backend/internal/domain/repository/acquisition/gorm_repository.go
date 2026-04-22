package acquisition

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	acqmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/models/acquisition"
	entitymodels "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	orgsyncmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/org_sync"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type staffLiveCodeRepository struct{ gormStore }
type staffWelcomeConfigRepository struct{ gormStore }
type staffWelcomeSyncAttemptRepository struct{ gormStore }
type groupLiveCodeRepository struct{ gormStore }
type groupChatSnapshotRepository struct{ gormStore }
type groupTagRepository struct{ gormStore }

func NewStaffLiveCodeRepository(db *gorm.DB) StaffLiveCodeRepository {
	return &staffLiveCodeRepository{gormStore{db: db}}
}

func NewStaffWelcomeConfigRepository(db *gorm.DB) StaffWelcomeConfigRepository {
	return &staffWelcomeConfigRepository{gormStore{db: db}}
}

func NewStaffWelcomeSyncAttemptRepository(db *gorm.DB) StaffWelcomeSyncAttemptRepository {
	return &staffWelcomeSyncAttemptRepository{gormStore{db: db}}
}

func NewGroupLiveCodeRepository(db *gorm.DB) GroupLiveCodeRepository {
	return &groupLiveCodeRepository{gormStore{db: db}}
}

func NewGroupChatSnapshotRepository(db *gorm.DB) GroupChatSnapshotRepository {
	return &groupChatSnapshotRepository{gormStore{db: db}}
}

func NewGroupTagRepository(db *gorm.DB) GroupTagRepository {
	return &groupTagRepository{gormStore{db: db}}
}

func (r *staffLiveCodeRepository) Create(ctx context.Context, item *acqmodel.StaffLiveCode) error {
	if r == nil || r.db == nil {
		return ErrRepositoryDBNotReady
	}
	if item == nil {
		return errors.New("staff live code is required")
	}
	item.TenantUUID = strings.ToLower(strings.TrimSpace(item.TenantUUID))
	if _, err := normalizeTenant(item.TenantUUID); err != nil {
		return err
	}
	item.CreatedAt = utcNow()
	item.UpdatedAt = item.CreatedAt
	return r.db.WithContext(ctx).Create(item).Error
}

func (r *staffLiveCodeRepository) GetByUUID(ctx context.Context, tenantUUID, staffCodeUUID string) (*acqmodel.StaffLiveCode, error) {
	if r == nil || r.db == nil {
		return nil, ErrRepositoryDBNotReady
	}
	tenantUUID, err := normalizeTenant(tenantUUID)
	if err != nil {
		return nil, err
	}
	staffCodeUUID, err = normalizeStaffCodeUUID(staffCodeUUID)
	if err != nil {
		return nil, err
	}
	var out acqmodel.StaffLiveCode
	q := r.db.WithContext(ctx).Where("tenant_uuid = ? AND staff_code_uuid = ?", tenantUUID, staffCodeUUID).First(&out)
	if q.Error != nil {
		if errors.Is(q.Error, gorm.ErrRecordNotFound) {
			return nil, ErrRecordNotFound
		}
		return nil, q.Error
	}
	return &out, nil
}

func (r *staffLiveCodeRepository) List(ctx context.Context, tenantUUID string, filter StaffLiveCodeListFilter) ([]*acqmodel.StaffLiveCode, error) {
	if r == nil || r.db == nil {
		return nil, ErrRepositoryDBNotReady
	}
	tenantUUID, err := normalizeTenant(tenantUUID)
	if err != nil {
		return nil, err
	}
	q := r.db.WithContext(ctx).Where("tenant_uuid = ?", tenantUUID)
	if v := strings.TrimSpace(filter.ActivityName); v != "" {
		q = q.Where("activity_name ILIKE ?", "%"+v+"%")
	}
	if v := strings.TrimSpace(filter.Status); v != "" {
		q = q.Where("status = ?", strings.ToLower(v))
	}
	limit := ensureLimit(filter.Limit, 20)
	var out []*acqmodel.StaffLiveCode
	if err := q.Order("created_at desc").Limit(limit).Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

func (r *staffLiveCodeRepository) UpdateStatus(ctx context.Context, tenantUUID, staffCodeUUID, status, updatedBy string) (*acqmodel.StaffLiveCode, error) {
	if r == nil || r.db == nil {
		return nil, ErrRepositoryDBNotReady
	}
	tenantUUID, err := normalizeTenant(tenantUUID)
	if err != nil {
		return nil, err
	}
	staffCodeUUID, err = normalizeStaffCodeUUID(staffCodeUUID)
	if err != nil {
		return nil, err
	}
	status = strings.ToLower(strings.TrimSpace(status))
	if status == "" {
		return nil, errors.New("status is required")
	}
	updatedBy = strings.TrimSpace(updatedBy)
	if updatedBy == "" {
		updatedBy = "system"
	}
	now := time.Now().UTC()
	q := r.db.WithContext(ctx).Model(&acqmodel.StaffLiveCode{}).
		Where("tenant_uuid = ? AND staff_code_uuid = ?", tenantUUID, staffCodeUUID).
		Updates(map[string]any{
			"status":     status,
			"updated_by": updatedBy,
			"updated_at": now,
		})
	if q.Error != nil {
		return nil, q.Error
	}
	if q.RowsAffected == 0 {
		return nil, ErrRecordNotFound
	}
	return r.GetByUUID(ctx, tenantUUID, staffCodeUUID)
}

func (r *staffLiveCodeRepository) CountConfirmedMappings(ctx context.Context, tenantUUID string, memberUUIDs []string) (int64, error) {
	if r == nil || r.db == nil {
		return 0, ErrRepositoryDBNotReady
	}
	tenantUUID, err := normalizeTenant(tenantUUID)
	if err != nil {
		return 0, err
	}
	if len(memberUUIDs) == 0 {
		return 0, nil
	}
	cleanIDs := make([]string, 0, len(memberUUIDs))
	seen := map[string]struct{}{}
	for _, id := range memberUUIDs {
		id = strings.ToLower(strings.TrimSpace(id))
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		cleanIDs = append(cleanIDs, id)
	}
	if len(cleanIDs) == 0 {
		return 0, nil
	}
	var count int64
	err = r.db.WithContext(ctx).
		Model(&orgsyncmodel.MemberBinding{}).
		Where("tenant_uuid = ? AND main_member_id IN ?", tenantUUID, cleanIDs).
		Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (r *staffLiveCodeRepository) ExistsByCodeKey(ctx context.Context, tenantUUID, codeKey string) (bool, error) {
	if r == nil || r.db == nil {
		return false, ErrRepositoryDBNotReady
	}
	tenantUUID, err := normalizeTenant(tenantUUID)
	if err != nil {
		return false, err
	}
	codeKey = strings.ToLower(strings.TrimSpace(codeKey))
	if codeKey == "" {
		return false, errors.New("code_key is required")
	}
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&acqmodel.StaffLiveCode{}).
		Where("tenant_uuid = ? AND lower(code_key) = ?", tenantUUID, codeKey).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *staffWelcomeConfigRepository) Save(ctx context.Context, item *acqmodel.StaffWelcomeConfig) error {
	if r == nil || r.db == nil {
		return ErrRepositoryDBNotReady
	}
	if item == nil {
		return errors.New("staff welcome config is required")
	}
	item.TenantUUID = strings.ToLower(strings.TrimSpace(item.TenantUUID))
	if _, err := normalizeTenant(item.TenantUUID); err != nil {
		return err
	}
	item.StaffCodeUUID = strings.ToLower(strings.TrimSpace(item.StaffCodeUUID))
	if _, err := normalizeStaffCodeUUID(item.StaffCodeUUID); err != nil {
		return err
	}
	now := utcNow()
	if item.CreatedAt.IsZero() {
		item.CreatedAt = now
	}
	item.UpdatedAt = now
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "staff_code_uuid"}},
		DoUpdates: clause.Assignments(map[string]any{
			"welcome_mode":    item.WelcomeMode,
			"content_blocks":  item.ContentBlocks,
			"payload_preview": item.PayloadPreview,
			"sync_status":     item.SyncStatus,
			"last_sync_error": item.LastSyncError,
			"last_synced_at":  item.LastSyncedAt,
			"version":         item.Version,
			"updated_by":      item.UpdatedBy,
			"updated_at":      item.UpdatedAt,
		}),
	}).Create(item).Error
}

func (r *staffWelcomeConfigRepository) GetByStaffCodeUUID(ctx context.Context, tenantUUID, staffCodeUUID string) (*acqmodel.StaffWelcomeConfig, error) {
	if r == nil || r.db == nil {
		return nil, ErrRepositoryDBNotReady
	}
	tenantUUID, err := normalizeTenant(tenantUUID)
	if err != nil {
		return nil, err
	}
	staffCodeUUID, err = normalizeStaffCodeUUID(staffCodeUUID)
	if err != nil {
		return nil, err
	}
	var out acqmodel.StaffWelcomeConfig
	q := r.db.WithContext(ctx).Where("tenant_uuid = ? AND staff_code_uuid = ?", tenantUUID, staffCodeUUID).First(&out)
	if q.Error != nil {
		if errors.Is(q.Error, gorm.ErrRecordNotFound) {
			return nil, ErrRecordNotFound
		}
		return nil, q.Error
	}
	return &out, nil
}

func (r *staffWelcomeSyncAttemptRepository) Create(ctx context.Context, item *acqmodel.StaffWelcomeSyncAttempt) error {
	if r == nil || r.db == nil {
		return ErrRepositoryDBNotReady
	}
	if item == nil {
		return errors.New("staff welcome sync attempt is required")
	}
	item.TenantUUID = strings.ToLower(strings.TrimSpace(item.TenantUUID))
	if _, err := normalizeTenant(item.TenantUUID); err != nil {
		return err
	}
	item.StaffCodeUUID = strings.ToLower(strings.TrimSpace(item.StaffCodeUUID))
	if _, err := normalizeStaffCodeUUID(item.StaffCodeUUID); err != nil {
		return err
	}
	item.CreatedAt = utcNow()
	return r.db.WithContext(ctx).Create(item).Error
}

func (r *staffWelcomeSyncAttemptRepository) ListByStaffCodeUUID(ctx context.Context, tenantUUID, staffCodeUUID string, limit int) ([]*acqmodel.StaffWelcomeSyncAttempt, error) {
	if r == nil || r.db == nil {
		return nil, ErrRepositoryDBNotReady
	}
	tenantUUID, err := normalizeTenant(tenantUUID)
	if err != nil {
		return nil, err
	}
	staffCodeUUID, err = normalizeStaffCodeUUID(staffCodeUUID)
	if err != nil {
		return nil, err
	}
	limit = ensureLimit(limit, 20)
	var out []*acqmodel.StaffWelcomeSyncAttempt
	err = r.db.WithContext(ctx).
		Where("tenant_uuid = ? AND staff_code_uuid = ?", tenantUUID, staffCodeUUID).
		Order("created_at desc").
		Limit(limit).
		Find(&out).Error
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (r *groupLiveCodeRepository) List(ctx context.Context, tenantUUID string, limit int) ([]*acqmodel.GroupLiveCode, error) {
	if r == nil || r.db == nil {
		return nil, ErrRepositoryDBNotReady
	}
	tenantUUID, err := normalizeTenant(tenantUUID)
	if err != nil {
		return nil, err
	}
	limit = ensureLimit(limit, 20)
	var out []*acqmodel.GroupLiveCode
	err = r.db.WithContext(ctx).
		Table(entitymodels.S(entitymodels.TableAcquisitionGroupLiveCodes)).
		Where("tenant_uuid = ?", tenantUUID).
		Order("created_at desc").
		Limit(limit).
		Find(&out).Error
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (r *groupLiveCodeRepository) Create(ctx context.Context, item *acqmodel.GroupLiveCode) error {
	if r == nil || r.db == nil {
		return ErrRepositoryDBNotReady
	}
	if item == nil {
		return errors.New("group live code is required")
	}
	item.TenantUUID = strings.ToLower(strings.TrimSpace(item.TenantUUID))
	if _, err := normalizeTenant(item.TenantUUID); err != nil {
		return err
	}
	if strings.TrimSpace(item.GroupCodeUUID) == "" {
		item.GroupCodeUUID = uuid.NewString()
	}
	now := utcNow()
	item.CreatedAt = now
	item.UpdatedAt = now
	return r.db.WithContext(ctx).Create(item).Error
}

func (r *groupLiveCodeRepository) GetByUUID(ctx context.Context, tenantUUID, groupCodeUUID string) (*acqmodel.GroupLiveCode, error) {
	if r == nil || r.db == nil {
		return nil, ErrRepositoryDBNotReady
	}
	tenantUUID, err := normalizeTenant(tenantUUID)
	if err != nil {
		return nil, err
	}
	groupCodeUUID = strings.ToLower(strings.TrimSpace(groupCodeUUID))
	if groupCodeUUID == "" {
		return nil, errors.New("group_code_uuid is required")
	}
	var out acqmodel.GroupLiveCode
	q := r.db.WithContext(ctx).
		Where("tenant_uuid = ? AND group_code_uuid = ?", tenantUUID, groupCodeUUID).
		First(&out)
	if q.Error != nil {
		if errors.Is(q.Error, gorm.ErrRecordNotFound) {
			return nil, ErrRecordNotFound
		}
		return nil, q.Error
	}
	return &out, nil
}

func (r *groupLiveCodeRepository) Update(ctx context.Context, item *acqmodel.GroupLiveCode) error {
	if r == nil || r.db == nil {
		return ErrRepositoryDBNotReady
	}
	if item == nil {
		return errors.New("group live code is required")
	}
	item.TenantUUID = strings.ToLower(strings.TrimSpace(item.TenantUUID))
	if _, err := normalizeTenant(item.TenantUUID); err != nil {
		return err
	}
	item.GroupCodeUUID = strings.ToLower(strings.TrimSpace(item.GroupCodeUUID))
	if item.GroupCodeUUID == "" {
		return errors.New("group_code_uuid is required")
	}
	item.UpdatedAt = utcNow()
	var shardConfigIDs any
	if item.ShardConfigIDs == nil {
		shardConfigIDs = datatypes.JSON([]byte("[]"))
	} else {
		if buf, err := json.Marshal(item.ShardConfigIDs); err == nil {
			shardConfigIDs = datatypes.JSON(buf)
		} else {
			return err
		}
	}
	var targetChatIDs any
	if item.TargetChatIDs == nil {
		targetChatIDs = datatypes.JSON([]byte("[]"))
	} else {
		if buf, err := json.Marshal(item.TargetChatIDs); err == nil {
			targetChatIDs = datatypes.JSON(buf)
		} else {
			return err
		}
	}
	q := r.db.WithContext(ctx).
		Model(&acqmodel.GroupLiveCode{}).
		Where("tenant_uuid = ? AND group_code_uuid = ?", item.TenantUUID, item.GroupCodeUUID).
		Updates(map[string]any{
			"activity_name":        item.ActivityName,
			"state":                item.State,
			"config_id":            item.ConfigID,
			"join_scene":           item.JoinScene,
			"skip_verify":          item.SkipVerify,
			"auto_create_room":     item.AutoCreateRoom,
			"target_chat_count":    item.TargetChatCount,
			"target_chat_ids":      targetChatIDs,
			"shard_count":          item.ShardCount,
			"capacity_total":       item.CapacityTotal,
			"capacity_used":        item.CapacityUsed,
			"shard_config_ids":     shardConfigIDs,
			"qr_code":              item.QRCode,
			"status":               item.Status,
			"sync_status":          item.SyncStatus,
			"last_sync_error":      item.LastSyncError,
			"last_synced_at":       item.LastSyncedAt,
			"capability_status":    item.CapabilityStatus,
			"updated_by":           item.UpdatedBy,
			"updated_at":           item.UpdatedAt,
			"channel_account_uuid": item.ChannelAccountUUID,
		})
	if q.Error != nil {
		return q.Error
	}
	if q.RowsAffected == 0 {
		return ErrRecordNotFound
	}
	return nil
}

func (r *groupLiveCodeRepository) Delete(ctx context.Context, tenantUUID, groupCodeUUID string) error {
	if r == nil || r.db == nil {
		return ErrRepositoryDBNotReady
	}
	tenantUUID, err := normalizeTenant(tenantUUID)
	if err != nil {
		return err
	}
	groupCodeUUID = strings.ToLower(strings.TrimSpace(groupCodeUUID))
	if groupCodeUUID == "" {
		return errors.New("group_code_uuid is required")
	}
	q := r.db.WithContext(ctx).
		Where("tenant_uuid = ? AND group_code_uuid = ?", tenantUUID, groupCodeUUID).
		Delete(&acqmodel.GroupLiveCode{})
	if q.Error != nil {
		return q.Error
	}
	if q.RowsAffected == 0 {
		return ErrRecordNotFound
	}
	return nil
}

func (r *groupChatSnapshotRepository) Upsert(ctx context.Context, item *acqmodel.GroupChatSnapshot) error {
	if r == nil || r.db == nil {
		return ErrRepositoryDBNotReady
	}
	if item == nil {
		return errors.New("group chat snapshot is required")
	}
	item.TenantUUID = strings.ToLower(strings.TrimSpace(item.TenantUUID))
	if _, err := normalizeTenant(item.TenantUUID); err != nil {
		return err
	}
	item.ChatID = strings.TrimSpace(item.ChatID)
	if item.ChatID == "" {
		return errors.New("chat_id is required")
	}
	if strings.TrimSpace(item.SnapshotUUID) == "" {
		item.SnapshotUUID = uuid.NewString()
	}
	if item.UpdatedAt.IsZero() {
		item.UpdatedAt = utcNow()
	}
	if item.SourceGroupCodeUUID != nil && strings.TrimSpace(*item.SourceGroupCodeUUID) == "" {
		item.SourceGroupCodeUUID = nil
	}
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "tenant_uuid"}, {Name: "channel_account_uuid"}, {Name: "chat_id"}},
		DoUpdates: clause.Assignments(map[string]any{
			"name":                   item.Name,
			"owner_userid":           item.OwnerUserID,
			"member_count":           item.MemberCount,
			"create_time":            item.CreateTime,
			"last_activity_at":       item.LastActivityAt,
			"source_group_code_uuid": item.SourceGroupCodeUUID,
			"source_config_id":       item.SourceConfigID,
			"payload":                item.Payload,
			"updated_at":             item.UpdatedAt,
		}),
	}).Create(item).Error
}

func (r *groupChatSnapshotRepository) GetByChatID(ctx context.Context, tenantUUID, chatID string) (*acqmodel.GroupChatSnapshot, error) {
	if r == nil || r.db == nil {
		return nil, ErrRepositoryDBNotReady
	}
	tenantUUID, err := normalizeTenant(tenantUUID)
	if err != nil {
		return nil, err
	}
	chatID = strings.TrimSpace(chatID)
	if chatID == "" {
		return nil, errors.New("chat_id is required")
	}
	var out acqmodel.GroupChatSnapshot
	q := r.db.WithContext(ctx).Where("tenant_uuid = ? AND chat_id = ?", tenantUUID, chatID).First(&out)
	if q.Error != nil {
		if errors.Is(q.Error, gorm.ErrRecordNotFound) {
			return nil, ErrRecordNotFound
		}
		return nil, q.Error
	}
	return &out, nil
}

func (r *groupChatSnapshotRepository) List(ctx context.Context, tenantUUID string, limit int) ([]*acqmodel.GroupChatSnapshot, error) {
	if r == nil || r.db == nil {
		return nil, ErrRepositoryDBNotReady
	}
	tenantUUID, err := normalizeTenant(tenantUUID)
	if err != nil {
		return nil, err
	}
	limit = ensureLimit(limit, 20)
	var out []*acqmodel.GroupChatSnapshot
	err = r.db.WithContext(ctx).
		Where("tenant_uuid = ?", tenantUUID).
		Order("updated_at desc").
		Limit(limit).
		Find(&out).Error
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (r *groupChatSnapshotRepository) ListByChannelAccount(ctx context.Context, tenantUUID, channelAccountUUID string, limit int) ([]*acqmodel.GroupChatSnapshot, error) {
	if r == nil || r.db == nil {
		return nil, ErrRepositoryDBNotReady
	}
	tenantUUID, err := normalizeTenant(tenantUUID)
	if err != nil {
		return nil, err
	}
	channelAccountUUID = strings.ToLower(strings.TrimSpace(channelAccountUUID))
	if channelAccountUUID == "" {
		return nil, errors.New("channel_account_uuid is required")
	}
	limit = ensureLimit(limit, 5000)
	var out []*acqmodel.GroupChatSnapshot
	err = r.db.WithContext(ctx).
		Where("tenant_uuid = ? AND channel_account_uuid = ?", tenantUUID, channelAccountUUID).
		Order("updated_at desc").
		Limit(limit).
		Find(&out).Error
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (r *groupTagRepository) CreateDefinition(ctx context.Context, item *acqmodel.GroupTagDefinition) error {
	if r == nil || r.db == nil {
		return ErrRepositoryDBNotReady
	}
	if item == nil {
		return errors.New("group tag definition is required")
	}
	item.TenantUUID = strings.ToLower(strings.TrimSpace(item.TenantUUID))
	if _, err := normalizeTenant(item.TenantUUID); err != nil {
		return err
	}
	if strings.TrimSpace(item.GroupTagUUID) == "" {
		item.GroupTagUUID = uuid.NewString()
	}
	now := utcNow()
	item.CreatedAt = now
	item.UpdatedAt = now
	return r.db.WithContext(ctx).Create(item).Error
}

func (r *groupTagRepository) ListDefinitions(ctx context.Context, tenantUUID string, limit int) ([]*acqmodel.GroupTagDefinition, error) {
	if r == nil || r.db == nil {
		return nil, ErrRepositoryDBNotReady
	}
	tenantUUID, err := normalizeTenant(tenantUUID)
	if err != nil {
		return nil, err
	}
	limit = ensureLimit(limit, 50)
	var out []*acqmodel.GroupTagDefinition
	err = r.db.WithContext(ctx).
		Where("tenant_uuid = ?", tenantUUID).
		Order("created_at desc").
		Limit(limit).
		Find(&out).Error
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (r *groupTagRepository) GetDefinitionByUUID(ctx context.Context, tenantUUID, groupTagUUID string) (*acqmodel.GroupTagDefinition, error) {
	if r == nil || r.db == nil {
		return nil, ErrRepositoryDBNotReady
	}
	tenantUUID, err := normalizeTenant(tenantUUID)
	if err != nil {
		return nil, err
	}
	groupTagUUID = strings.ToLower(strings.TrimSpace(groupTagUUID))
	if groupTagUUID == "" {
		return nil, errors.New("group_tag_uuid is required")
	}
	var out acqmodel.GroupTagDefinition
	q := r.db.WithContext(ctx).
		Where("tenant_uuid = ? AND group_tag_uuid = ?", tenantUUID, groupTagUUID).
		First(&out)
	if q.Error != nil {
		if errors.Is(q.Error, gorm.ErrRecordNotFound) {
			return nil, ErrRecordNotFound
		}
		return nil, q.Error
	}
	return &out, nil
}

func (r *groupTagRepository) BindChats(ctx context.Context, tenantUUID, groupTagUUID string, chatIDs []string, bindSource, ruleRunUUID string) (int, error) {
	if r == nil || r.db == nil {
		return 0, ErrRepositoryDBNotReady
	}
	tenantUUID, err := normalizeTenant(tenantUUID)
	if err != nil {
		return 0, err
	}
	groupTagUUID = strings.ToLower(strings.TrimSpace(groupTagUUID))
	if groupTagUUID == "" {
		return 0, errors.New("group_tag_uuid is required")
	}
	if bindSource == "" {
		bindSource = "manual"
	}
	now := utcNow()
	bound := 0
	for _, chatID := range chatIDs {
		chatID = strings.TrimSpace(chatID)
		if chatID == "" {
			continue
		}
		item := &acqmodel.GroupTagBinding{
			BindingUUID:  uuid.NewString(),
			TenantUUID:   tenantUUID,
			GroupTagUUID: groupTagUUID,
			ChatID:       chatID,
			BindSource:   bindSource,
			RuleRunUUID:  ruleRunUUID,
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		res := r.db.WithContext(ctx).Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "tenant_uuid"}, {Name: "group_tag_uuid"}, {Name: "chat_id"}},
			DoNothing: true,
		}).Create(item)
		if res.Error != nil {
			return bound, res.Error
		}
		if res.RowsAffected > 0 {
			bound++
		}
	}
	return bound, nil
}

func (r *groupTagRepository) ListBindings(ctx context.Context, tenantUUID, groupTagUUID string, limit int) ([]*acqmodel.GroupTagBinding, error) {
	if r == nil || r.db == nil {
		return nil, ErrRepositoryDBNotReady
	}
	tenantUUID, err := normalizeTenant(tenantUUID)
	if err != nil {
		return nil, err
	}
	groupTagUUID = strings.ToLower(strings.TrimSpace(groupTagUUID))
	if groupTagUUID == "" {
		return nil, errors.New("group_tag_uuid is required")
	}
	limit = ensureLimit(limit, 200)
	var out []*acqmodel.GroupTagBinding
	err = r.db.WithContext(ctx).
		Where("tenant_uuid = ? AND group_tag_uuid = ?", tenantUUID, groupTagUUID).
		Order("created_at desc").
		Limit(limit).
		Find(&out).Error
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (r *groupTagRepository) CreateRuleRun(ctx context.Context, item *acqmodel.GroupTagRuleRun) error {
	if r == nil || r.db == nil {
		return ErrRepositoryDBNotReady
	}
	if item == nil {
		return errors.New("group tag rule run is required")
	}
	item.TenantUUID = strings.ToLower(strings.TrimSpace(item.TenantUUID))
	if _, err := normalizeTenant(item.TenantUUID); err != nil {
		return err
	}
	if strings.TrimSpace(item.RuleRunUUID) == "" {
		item.RuleRunUUID = uuid.NewString()
	}
	if item.CreatedAt.IsZero() {
		item.CreatedAt = utcNow()
	}
	return r.db.WithContext(ctx).Create(item).Error
}
