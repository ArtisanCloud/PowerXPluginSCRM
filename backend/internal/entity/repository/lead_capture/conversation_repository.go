package lead_capture

import (
	"context"
	"errors"
	"strings"
	"time"

	model "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/lead_capture"
	socialmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/social_channel_governance"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository"
	socialrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/social_channel_governance"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrLeadSyncTaskNotFound   = errors.New("lead sync task not found")
	ErrChannelAccountRequired = errors.New("channel account is required")
	ErrDefaultAccountNotFound = errors.New("default channel account not found")
)

type LeadSyncTaskRepository struct {
	*repository.BaseRepository[model.LeadSyncTask]
	accountRepo *socialrepo.AccountRepository
}

func NewLeadSyncTaskRepository(db *gorm.DB) *LeadSyncTaskRepository {
	return &LeadSyncTaskRepository{
		BaseRepository: repository.NewBaseRepository[model.LeadSyncTask](db),
		accountRepo:    socialrepo.NewAccountRepository(db),
	}
}

func (r *LeadSyncTaskRepository) CreateTask(ctx context.Context, task *model.LeadSyncTask) (*model.LeadSyncTask, error) {
	if task == nil {
		return nil, errors.New("task is required")
	}
	task.TenantUUID = strings.ToLower(strings.TrimSpace(task.TenantUUID))
	task.Channel = strings.ToLower(strings.TrimSpace(task.Channel))
	task.AppType = strings.ToLower(strings.TrimSpace(task.AppType))
	task.ChannelAccountUUID = strings.ToLower(strings.TrimSpace(task.ChannelAccountUUID))
	if task.TenantUUID == "" || task.ChannelAccountUUID == "" {
		return nil, ErrChannelAccountRequired
	}
	if task.Channel == "" {
		task.Channel = "wechat"
	}
	if task.AppType == "" {
		task.AppType = "wecom"
	}
	if task.AccountResolveSource == "" {
		task.AccountResolveSource = model.LeadSyncTaskResolveExplicit
	}
	if task.TaskProvider == "" {
		task.TaskProvider = model.LeadSyncTaskProviderLocalFallback
	}
	if task.Status == "" {
		task.Status = "queued"
	}
	if task.TriggerType == "" {
		task.TriggerType = "manual"
	}
	if strings.TrimSpace(task.TaskUUID) == "" {
		task.TaskUUID = uuid.NewString()
	}
	if err := r.WithTenantTx(ctx, task.TenantUUID, func(tx *gorm.DB) error {
		return tx.Create(task).Error
	}); err != nil {
		return nil, err
	}
	return task, nil
}

func (r *LeadSyncTaskRepository) UpdateStatus(ctx context.Context, tenantUUID, taskUUID, status string, updates map[string]any) error {
	if r == nil || r.DB == nil {
		return errors.New("repository database is not initialized")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	taskUUID = strings.ToLower(strings.TrimSpace(taskUUID))
	if tenantUUID == "" || taskUUID == "" {
		return ErrLeadSyncTaskNotFound
	}
	if updates == nil {
		updates = map[string]any{}
	}
	if status != "" {
		updates["status"] = strings.ToLower(strings.TrimSpace(status))
	}
	updates["updated_at"] = time.Now().UTC()
	res := r.DB.WithContext(ctx).Model(&model.LeadSyncTask{}).
		Where("tenant_uuid = ? AND task_uuid = ?", tenantUUID, taskUUID).
		Updates(updates)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrLeadSyncTaskNotFound
	}
	return nil
}

func (r *LeadSyncTaskRepository) ListByAccount(ctx context.Context, tenantUUID, channelAccountUUID string, limit int) ([]*model.LeadSyncTask, error) {
	return r.ListByFilter(ctx, tenantUUID, channelAccountUUID, "", limit)
}

func (r *LeadSyncTaskRepository) ListByFilter(ctx context.Context, tenantUUID, channelAccountUUID, status string, limit int) ([]*model.LeadSyncTask, error) {
	if r == nil || r.DB == nil {
		return nil, errors.New("repository database is not initialized")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	channelAccountUUID = strings.ToLower(strings.TrimSpace(channelAccountUUID))
	status = strings.ToLower(strings.TrimSpace(status))
	if tenantUUID == "" {
		return nil, ErrLeadSyncTaskNotFound
	}
	if limit <= 0 {
		limit = 20
	}
	query := r.DB.WithContext(ctx).Where("tenant_uuid = ?", tenantUUID)
	if channelAccountUUID != "" {
		query = query.Where("channel_account_uuid = ?", channelAccountUUID)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	var out []*model.LeadSyncTask
	if err := query.Order("created_at DESC").Limit(limit).Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

func (r *LeadSyncTaskRepository) ClearByFilter(ctx context.Context, tenantUUID, channelAccountUUID, status string) (int64, error) {
	if r == nil || r.DB == nil {
		return 0, errors.New("repository database is not initialized")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	channelAccountUUID = strings.ToLower(strings.TrimSpace(channelAccountUUID))
	status = strings.ToLower(strings.TrimSpace(status))
	if tenantUUID == "" {
		return 0, repository.ErrTenantUuidRequired
	}
	query := r.DB.WithContext(ctx).
		Where("tenant_uuid = ?", tenantUUID).
		Model(&model.LeadSyncTask{})
	if channelAccountUUID != "" {
		query = query.Where("channel_account_uuid = ?", channelAccountUUID)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	res := query.Delete(&model.LeadSyncTask{})
	if res.Error != nil {
		return 0, res.Error
	}
	return res.RowsAffected, nil
}

func (r *LeadSyncTaskRepository) ResolveChannelAccount(ctx context.Context, tenantUUID, channel, appType, explicitAccountUUID string) (string, string, error) {
	if r == nil || r.accountRepo == nil {
		return "", "", errors.New("account repository not configured")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	channel = strings.ToLower(strings.TrimSpace(channel))
	appType = strings.ToLower(strings.TrimSpace(appType))
	explicitAccountUUID = strings.ToLower(strings.TrimSpace(explicitAccountUUID))
	if tenantUUID == "" {
		return "", "", repository.ErrTenantUuidRequired
	}
	if channel == "" {
		channel = "wechat"
	}
	if appType == "" {
		appType = "wecom"
	}
	if explicitAccountUUID != "" {
		acc, err := r.accountRepo.GetByAccountUUID(ctx, tenantUUID, explicitAccountUUID)
		if err != nil {
			return "", "", err
		}
		if acc == nil {
			return "", "", ErrChannelAccountRequired
		}
		return acc.AccountUUID, model.LeadSyncTaskResolveExplicit, nil
	}

	var candidates []socialmodel.ChannelAccount
	if err := r.DB.WithContext(ctx).
		Where("tenant_uuid = ? AND channel_code = ? AND app_type = ? AND status = ?", tenantUUID, channel, appType, "connected").
		Find(&candidates).Error; err != nil {
		return "", "", err
	}
	if len(candidates) > 1 {
		return "", "", errors.New("multiple wecom accounts found, channel_account_uuid is required")
	}

	var account socialmodel.ChannelAccount
	err := r.DB.WithContext(ctx).
		Where("tenant_uuid = ? AND channel_code = ? AND app_type = ? AND org_sync_default = TRUE", tenantUUID, channel, appType).
		First(&account).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", "", ErrDefaultAccountNotFound
		}
		return "", "", err
	}
	return account.AccountUUID, model.LeadSyncTaskResolveDefault, nil
}

type ConversationEventRepository struct {
	*repository.BaseRepository[model.ConversationEvent]
}

func NewConversationEventRepository(db *gorm.DB) *ConversationEventRepository {
	return &ConversationEventRepository{BaseRepository: repository.NewBaseRepository[model.ConversationEvent](db)}
}

func (r *ConversationEventRepository) CreateIdempotent(ctx context.Context, event *model.ConversationEvent) (*model.ConversationEvent, bool, error) {
	if event == nil {
		return nil, false, errors.New("event is required")
	}
	event.TenantUUID = strings.ToLower(strings.TrimSpace(event.TenantUUID))
	event.ChannelAccountUUID = strings.ToLower(strings.TrimSpace(event.ChannelAccountUUID))
	event.IdempotencyKey = strings.TrimSpace(event.IdempotencyKey)
	if event.TenantUUID == "" || event.ChannelAccountUUID == "" || event.IdempotencyKey == "" {
		return nil, false, repository.ErrTenantUuidRequired
	}
	if strings.TrimSpace(event.EventUUID) == "" {
		event.EventUUID = uuid.NewString()
	}
	created := true
	err := r.WithTenantTx(ctx, event.TenantUUID, func(tx *gorm.DB) error {
		res := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(event)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			created = false
			var existed model.ConversationEvent
			if err := tx.Where("tenant_uuid = ? AND idempotency_key = ?", event.TenantUUID, event.IdempotencyKey).First(&existed).Error; err != nil {
				return err
			}
			*event = existed
			return nil
		}
		return nil
	})
	if err != nil {
		return nil, false, err
	}
	return event, created, nil
}

func (r *ConversationEventRepository) ListByConversation(ctx context.Context, tenantUUID, conversationID string, limit int) ([]*model.ConversationEvent, error) {
	if r == nil || r.DB == nil {
		return nil, errors.New("repository database is not initialized")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	conversationID = strings.TrimSpace(conversationID)
	if tenantUUID == "" || conversationID == "" {
		return []*model.ConversationEvent{}, nil
	}
	if limit <= 0 {
		limit = 50
	}
	var out []*model.ConversationEvent
	err := r.DB.WithContext(ctx).
		Where("tenant_uuid = ? AND conversation_id = ?", tenantUUID, conversationID).
		Order("occurred_at DESC").
		Limit(limit).
		Find(&out).Error
	if err != nil {
		return nil, err
	}
	return out, nil
}

type LeadConversationBindingRepository struct {
	*repository.BaseRepository[model.LeadConversationBinding]
}

func NewLeadConversationBindingRepository(db *gorm.DB) *LeadConversationBindingRepository {
	return &LeadConversationBindingRepository{BaseRepository: repository.NewBaseRepository[model.LeadConversationBinding](db)}
}

func (r *LeadConversationBindingRepository) UpsertActive(ctx context.Context, binding *model.LeadConversationBinding) (*model.LeadConversationBinding, error) {
	if binding == nil {
		return nil, errors.New("binding is required")
	}
	binding.TenantUUID = strings.ToLower(strings.TrimSpace(binding.TenantUUID))
	binding.ChannelAccountUUID = strings.ToLower(strings.TrimSpace(binding.ChannelAccountUUID))
	if binding.TenantUUID == "" || strings.TrimSpace(binding.ConversationID) == "" {
		return nil, repository.ErrTenantUuidRequired
	}
	if binding.Status == "" {
		binding.Status = "active"
	}
	if binding.BindSource == "" {
		binding.BindSource = "auto"
	}
	if strings.TrimSpace(binding.BindingUUID) == "" {
		binding.BindingUUID = uuid.NewString()
	}
	if err := r.WithTenantTx(ctx, binding.TenantUUID, func(tx *gorm.DB) error {
		return tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "tenant_uuid"}, {Name: "conversation_id"}, {Name: "status"}},
			DoUpdates: clause.AssignmentColumns([]string{"lead_uuid", "channel_account_uuid", "bind_source", "created_by", "updated_at"}),
		}).Create(binding).Error
	}); err != nil {
		return nil, err
	}
	return binding, nil
}

func (r *LeadConversationBindingRepository) ListByLead(ctx context.Context, tenantUUID, leadUUID string) ([]*model.LeadConversationBinding, error) {
	if r == nil || r.DB == nil {
		return nil, errors.New("repository database is not initialized")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	leadUUID = strings.ToLower(strings.TrimSpace(leadUUID))
	if tenantUUID == "" || leadUUID == "" {
		return []*model.LeadConversationBinding{}, nil
	}
	var out []*model.LeadConversationBinding
	err := r.DB.WithContext(ctx).
		Where("tenant_uuid = ? AND lead_uuid = ? AND status = ?", tenantUUID, leadUUID, "active").
		Order("updated_at DESC").
		Find(&out).Error
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (r *LeadConversationBindingRepository) FindActiveByConversation(ctx context.Context, tenantUUID, conversationID string) (*model.LeadConversationBinding, error) {
	if r == nil || r.DB == nil {
		return nil, errors.New("repository database is not initialized")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	conversationID = strings.TrimSpace(conversationID)
	if tenantUUID == "" || conversationID == "" {
		return nil, gorm.ErrRecordNotFound
	}
	var out model.LeadConversationBinding
	err := r.DB.WithContext(ctx).
		Where("tenant_uuid = ? AND conversation_id = ? AND status = ?", tenantUUID, conversationID, "active").
		First(&out).Error
	if err != nil {
		return nil, err
	}
	return &out, nil
}

type LeadConversationPendingRepository struct {
	*repository.BaseRepository[model.LeadConversationPending]
}

func NewLeadConversationPendingRepository(db *gorm.DB) *LeadConversationPendingRepository {
	return &LeadConversationPendingRepository{BaseRepository: repository.NewBaseRepository[model.LeadConversationPending](db)}
}

func (r *LeadConversationPendingRepository) Create(ctx context.Context, pending *model.LeadConversationPending) (*model.LeadConversationPending, error) {
	if pending == nil {
		return nil, errors.New("pending is required")
	}
	pending.TenantUUID = strings.ToLower(strings.TrimSpace(pending.TenantUUID))
	pending.ChannelAccountUUID = strings.ToLower(strings.TrimSpace(pending.ChannelAccountUUID))
	if pending.TenantUUID == "" || pending.EventUUID == "" {
		return nil, repository.ErrTenantUuidRequired
	}
	if pending.Status == "" {
		pending.Status = "pending"
	}
	if pending.Reason == "" {
		pending.Reason = "no_match"
	}
	if strings.TrimSpace(pending.PendingUUID) == "" {
		pending.PendingUUID = uuid.NewString()
	}
	if err := r.WithTenantTx(ctx, pending.TenantUUID, func(tx *gorm.DB) error {
		return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(pending).Error
	}); err != nil {
		return nil, err
	}
	return pending, nil
}

type LeadRealtimeProjectionRepository struct {
	*repository.BaseRepository[model.LeadRealtimeProjection]
}

func NewLeadRealtimeProjectionRepository(db *gorm.DB) *LeadRealtimeProjectionRepository {
	return &LeadRealtimeProjectionRepository{BaseRepository: repository.NewBaseRepository[model.LeadRealtimeProjection](db)}
}

func (r *LeadRealtimeProjectionRepository) Upsert(ctx context.Context, projection *model.LeadRealtimeProjection) (*model.LeadRealtimeProjection, error) {
	if projection == nil {
		return nil, errors.New("projection is required")
	}
	projection.TenantUUID = strings.ToLower(strings.TrimSpace(projection.TenantUUID))
	if projection.TenantUUID == "" || projection.LeadUUID == "" || strings.TrimSpace(projection.ConversationID) == "" {
		return nil, repository.ErrTenantUuidRequired
	}
	if strings.TrimSpace(projection.ProjectionUUID) == "" {
		projection.ProjectionUUID = uuid.NewString()
	}
	if err := r.WithTenantTx(ctx, projection.TenantUUID, func(tx *gorm.DB) error {
		return tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "tenant_uuid"}, {Name: "lead_uuid"}, {Name: "conversation_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"latest_message", "latest_actor_type", "latest_at", "unread_count", "updated_at"}),
		}).Create(projection).Error
	}); err != nil {
		return nil, err
	}
	return projection, nil
}

func (r *LeadRealtimeProjectionRepository) ListByLead(ctx context.Context, tenantUUID, leadUUID string, limit int) ([]*model.LeadRealtimeProjection, error) {
	if r == nil || r.DB == nil {
		return nil, errors.New("repository database is not initialized")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	leadUUID = strings.ToLower(strings.TrimSpace(leadUUID))
	if tenantUUID == "" || leadUUID == "" {
		return []*model.LeadRealtimeProjection{}, nil
	}
	if limit <= 0 {
		limit = 50
	}
	var out []*model.LeadRealtimeProjection
	err := r.DB.WithContext(ctx).
		Where("tenant_uuid = ? AND lead_uuid = ?", tenantUUID, leadUUID).
		Order("latest_at DESC").
		Limit(limit).
		Find(&out).Error
	if err != nil {
		return nil, err
	}
	return out, nil
}
