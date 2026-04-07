package social_channel_governance

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	model "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/social_channel_governance"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrBindingNotFound      = errors.New("wecom auth binding not found")
	ErrSyncJobNotFound      = errors.New("sync baseline job not found")
	ErrSyncConflictNotFound = errors.New("sync conflict record not found")
	ErrCallbackTaskNotFound = errors.New("wecom callback task not found")
)

type OpenWorkFoundationRepository struct {
	*repository.BaseRepository[model.WeComOpenAuthBinding]
}

func NewOpenWorkFoundationRepository(db *gorm.DB) *OpenWorkFoundationRepository {
	return &OpenWorkFoundationRepository{BaseRepository: repository.NewBaseRepository[model.WeComOpenAuthBinding](db)}
}

func (r *OpenWorkFoundationRepository) SaveAuthEvent(ctx context.Context, event *model.WeComOpenAuthEvent) (*model.WeComOpenAuthEvent, bool, error) {
	if r == nil || r.DB == nil {
		return nil, false, errors.New("repository database is not initialized")
	}
	if event == nil {
		return nil, false, errors.New("event is required")
	}
	event.TenantUUID = strings.ToLower(strings.TrimSpace(event.TenantUUID))
	event.SuiteID = strings.TrimSpace(event.SuiteID)
	event.EventType = strings.ToLower(strings.TrimSpace(event.EventType))
	event.CorpID = strings.TrimSpace(event.CorpID)
	event.AgentID = strings.TrimSpace(event.AgentID)
	event.EventKey = strings.TrimSpace(event.EventKey)
	if event.TenantUUID == "" {
		return nil, false, repository.ErrTenantUuidRequired
	}
	if event.EventKey == "" {
		return nil, false, errors.New("event_key is required")
	}
	if event.Payload == nil {
		event.Payload = datatypes.JSONMap{}
	}
	created := true
	err := r.WithTenantTx(ctx, event.TenantUUID, func(tx *gorm.DB) error {
		res := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(event)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected > 0 {
			return nil
		}
		created = false
		return tx.Where("tenant_uuid = ? AND event_key = ?", event.TenantUUID, event.EventKey).First(event).Error
	})
	if err != nil {
		return nil, false, err
	}
	return event, created, nil
}

func (r *OpenWorkFoundationRepository) EnqueueCallbackTask(ctx context.Context, task *model.WeComOpenCallbackTask) (*model.WeComOpenCallbackTask, bool, error) {
	if r == nil || r.DB == nil {
		return nil, false, errors.New("repository database is not initialized")
	}
	if task == nil {
		return nil, false, errors.New("task is required")
	}
	task.TenantUUID = strings.ToLower(strings.TrimSpace(task.TenantUUID))
	task.SuiteID = strings.TrimSpace(task.SuiteID)
	task.EventType = strings.ToLower(strings.TrimSpace(task.EventType))
	task.CallbackKey = strings.TrimSpace(task.CallbackKey)
	task.EventKey = strings.TrimSpace(task.EventKey)
	task.AuthCode = strings.TrimSpace(task.AuthCode)
	task.CorpID = strings.TrimSpace(task.CorpID)
	task.AgentID = strings.TrimSpace(task.AgentID)
	task.SuiteTicket = strings.TrimSpace(task.SuiteTicket)
	task.State = strings.TrimSpace(task.State)
	task.MsgSignature = strings.TrimSpace(task.MsgSignature)
	task.Nonce = strings.TrimSpace(task.Nonce)
	if strings.TrimSpace(task.TaskUUID) == "" {
		task.TaskUUID = uuid.NewString()
	}
	if task.TenantUUID == "" {
		return nil, false, repository.ErrTenantUuidRequired
	}
	if task.SuiteID == "" || task.EventType == "" {
		return nil, false, errors.New("suite_id and event_type are required")
	}
	if task.CallbackKey == "" {
		return nil, false, errors.New("callback_key is required")
	}
	if task.Status == "" {
		task.Status = model.OpenWorkCallbackTaskReceived
	}
	if task.MaxAttempts <= 0 {
		task.MaxAttempts = 3
	}
	if task.Payload == nil {
		task.Payload = datatypes.JSONMap{}
	}
	created := true
	err := r.WithTenantTx(ctx, task.TenantUUID, func(tx *gorm.DB) error {
		res := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(task)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected > 0 {
			return nil
		}
		created = false
		var existing model.WeComOpenCallbackTask
		if err := tx.Where("tenant_uuid = ? AND suite_id = ? AND callback_key = ?", task.TenantUUID, task.SuiteID, task.CallbackKey).First(&existing).Error; err != nil {
			return err
		}
		*task = existing
		return nil
	})
	if err != nil {
		return nil, false, err
	}
	return task, created, nil
}

func (r *OpenWorkFoundationRepository) ClaimNextCallbackTask(ctx context.Context) (*model.WeComOpenCallbackTask, bool, error) {
	if r == nil || r.DB == nil {
		return nil, false, errors.New("repository database is not initialized")
	}
	now := time.Now().UTC()
	var claimed *model.WeComOpenCallbackTask
	err := r.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var candidate model.WeComOpenCallbackTask
		findErr := tx.
			Where("(status = ?) OR (status = ? AND attempt_count < max_attempts AND (next_retry_at IS NULL OR next_retry_at <= ?))",
				model.OpenWorkCallbackTaskReceived,
				model.OpenWorkCallbackTaskFailed,
				now,
			).
			Order("created_at ASC").
			First(&candidate).Error
		if errors.Is(findErr, gorm.ErrRecordNotFound) {
			return nil
		}
		if findErr != nil {
			return findErr
		}
		updates := map[string]any{
			"status":                model.OpenWorkCallbackTaskProcessing,
			"attempt_count":         candidate.AttemptCount + 1,
			"processing_started_at": now,
			"updated_at":            now,
		}
		res := tx.Model(&model.WeComOpenCallbackTask{}).
			Where("task_uuid = ? AND status = ? AND attempt_count = ?", candidate.TaskUUID, candidate.Status, candidate.AttemptCount).
			Updates(updates)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return nil
		}
		candidate.Status = model.OpenWorkCallbackTaskProcessing
		candidate.AttemptCount++
		candidate.ProcessingStartedAt = &now
		claimed = &candidate
		return nil
	})
	if err != nil {
		return nil, false, err
	}
	if claimed == nil {
		return nil, false, nil
	}
	return claimed, true, nil
}

func (r *OpenWorkFoundationRepository) MarkCallbackTaskSucceeded(ctx context.Context, taskUUID, corpID, agentID string, idempotentHit bool) error {
	if r == nil || r.DB == nil {
		return errors.New("repository database is not initialized")
	}
	taskUUID = strings.ToLower(strings.TrimSpace(taskUUID))
	if taskUUID == "" {
		return errors.New("task_uuid is required")
	}
	now := time.Now().UTC()
	updates := map[string]any{
		"status":         model.OpenWorkCallbackTaskSucceeded,
		"idempotent_hit": idempotentHit,
		"corp_id":        strings.TrimSpace(corpID),
		"agent_id":       strings.TrimSpace(agentID),
		"last_error":     "",
		"next_retry_at":  nil,
		"finished_at":    now,
		"updated_at":     now,
	}
	res := r.DB.WithContext(ctx).
		Model(&model.WeComOpenCallbackTask{}).
		Where("task_uuid = ?", taskUUID).
		Updates(updates)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrCallbackTaskNotFound
	}
	return nil
}

func (r *OpenWorkFoundationRepository) MarkCallbackTaskFailed(ctx context.Context, taskUUID, lastError string, nextRetryAt *time.Time, forceReauthorize bool) error {
	if r == nil || r.DB == nil {
		return errors.New("repository database is not initialized")
	}
	taskUUID = strings.ToLower(strings.TrimSpace(taskUUID))
	if taskUUID == "" {
		return errors.New("task_uuid is required")
	}
	now := time.Now().UTC()
	status := model.OpenWorkCallbackTaskFailed
	if forceReauthorize {
		status = model.OpenWorkCallbackTaskReauth
		nextRetryAt = nil
	}
	updates := map[string]any{
		"status":        status,
		"last_error":    strings.TrimSpace(lastError),
		"next_retry_at": nextRetryAt,
		"finished_at":   now,
		"updated_at":    now,
	}
	res := r.DB.WithContext(ctx).
		Model(&model.WeComOpenCallbackTask{}).
		Where("task_uuid = ?", taskUUID).
		Updates(updates)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrCallbackTaskNotFound
	}
	return nil
}

func (r *OpenWorkFoundationRepository) UpsertBinding(ctx context.Context, binding *model.WeComOpenAuthBinding, setDefault bool) (*model.WeComOpenAuthBinding, error) {
	if r == nil || r.DB == nil {
		return nil, errors.New("repository database is not initialized")
	}
	if binding == nil {
		return nil, errors.New("binding is required")
	}
	binding.TenantUUID = strings.ToLower(strings.TrimSpace(binding.TenantUUID))
	binding.ChannelCode = strings.ToLower(strings.TrimSpace(binding.ChannelCode))
	binding.AppType = strings.ToLower(strings.TrimSpace(binding.AppType))
	binding.SuiteID = strings.TrimSpace(binding.SuiteID)
	binding.CorpID = strings.TrimSpace(binding.CorpID)
	binding.AgentID = strings.TrimSpace(binding.AgentID)
	binding.Status = strings.ToLower(strings.TrimSpace(binding.Status))
	if binding.TenantUUID == "" {
		return nil, repository.ErrTenantUuidRequired
	}
	if binding.ChannelCode == "" {
		binding.ChannelCode = "wechat"
	}
	if binding.AppType == "" {
		binding.AppType = "wecom"
	}
	if binding.Status == "" {
		binding.Status = model.WeComAuthBindingStatusPending
	}
	if binding.AuthScope == nil {
		binding.AuthScope = datatypes.JSONMap{}
	}
	if binding.Metadata == nil {
		binding.Metadata = datatypes.JSONMap{}
	}
	now := time.Now().UTC()
	var out model.WeComOpenAuthBinding
	err := r.WithTenantTx(ctx, binding.TenantUUID, func(tx *gorm.DB) error {
		if strings.TrimSpace(binding.ChannelAccountUUID) == "" && binding.CorpID != "" {
			var existing model.WeComOpenAuthBinding
			findErr := tx.Where("tenant_uuid = ? AND corp_id = ? AND agent_id = ?", binding.TenantUUID, binding.CorpID, binding.AgentID).First(&existing).Error
			if findErr == nil {
				binding.ChannelAccountUUID = strings.TrimSpace(existing.ChannelAccountUUID)
			} else if !errors.Is(findErr, gorm.ErrRecordNotFound) {
				return findErr
			}
		}
		var channelAccountUUID any
		if strings.TrimSpace(binding.ChannelAccountUUID) != "" {
			channelAccountUUID = binding.ChannelAccountUUID
		}
		if setDefault {
			if err := tx.Model(&model.WeComOpenAuthBinding{}).
				Where("tenant_uuid = ? AND channel_code = ? AND app_type = ? AND is_default = TRUE", binding.TenantUUID, binding.ChannelCode, binding.AppType).
				Updates(map[string]any{"is_default": false, "updated_at": now}).Error; err != nil {
				return err
			}
			binding.IsDefault = true
			binding.DefaultSwitchedAt = &now
		}

		if binding.CorpID != "" {
			upsert := tx.Model(&model.WeComOpenAuthBinding{}).Clauses(clause.OnConflict{
				Columns: []clause.Column{{Name: "tenant_uuid"}, {Name: "corp_id"}, {Name: "agent_id"}},
				DoUpdates: clause.Assignments(map[string]any{
					"channel_account_uuid": channelAccountUUID,
					"suite_id":             binding.SuiteID,
					"corp_name":            binding.CorpName,
					"permanent_code":       binding.PermanentCode,
					"suite_access_token":   binding.SuiteAccessToken,
					"suite_ticket":         binding.SuiteTicket,
					"status":               binding.Status,
					"is_default":           binding.IsDefault,
					"default_switched_at":  binding.DefaultSwitchedAt,
					"last_event_type":      binding.LastEventType,
					"last_event_at":        binding.LastEventAt,
					"auth_scope":           binding.AuthScope,
					"metadata":             binding.Metadata,
					"updated_at":           now,
				}),
			}).Create(map[string]any{
				"tenant_uuid":          binding.TenantUUID,
				"channel_account_uuid": channelAccountUUID,
				"channel_code":         binding.ChannelCode,
				"app_type":             binding.AppType,
				"suite_id":             binding.SuiteID,
				"corp_id":              binding.CorpID,
				"agent_id":             binding.AgentID,
				"corp_name":            binding.CorpName,
				"permanent_code":       binding.PermanentCode,
				"suite_access_token":   binding.SuiteAccessToken,
				"suite_ticket":         binding.SuiteTicket,
				"status":               binding.Status,
				"is_default":           binding.IsDefault,
				"default_switched_at":  binding.DefaultSwitchedAt,
				"last_event_type":      binding.LastEventType,
				"last_event_at":        binding.LastEventAt,
				"auth_scope":           binding.AuthScope,
				"metadata":             binding.Metadata,
				"created_at":           now,
				"updated_at":           now,
			})
			if upsert.Error != nil {
				return upsert.Error
			}
			if err := tx.Where("tenant_uuid = ? AND corp_id = ? AND agent_id = ?", binding.TenantUUID, binding.CorpID, binding.AgentID).First(&out).Error; err != nil {
				return err
			}
			return nil
		}

		if binding.BindingUUID == "" {
			return errors.New("binding_uuid or corp_id is required")
		}
		res := tx.Model(&model.WeComOpenAuthBinding{}).
			Where("tenant_uuid = ? AND binding_uuid = ?", binding.TenantUUID, binding.BindingUUID).
			Updates(map[string]any{
				"channel_account_uuid": binding.ChannelAccountUUID,
				"suite_id":             binding.SuiteID,
				"suite_ticket":         binding.SuiteTicket,
				"status":               binding.Status,
				"last_event_type":      binding.LastEventType,
				"last_event_at":        binding.LastEventAt,
				"metadata":             binding.Metadata,
				"updated_at":           now,
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrBindingNotFound
		}
		return tx.Where("tenant_uuid = ? AND binding_uuid = ?", binding.TenantUUID, binding.BindingUUID).First(&out).Error
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (r *OpenWorkFoundationRepository) ListBindings(ctx context.Context, tenantUUID string) ([]*model.WeComOpenAuthBinding, error) {
	if r == nil || r.DB == nil {
		return nil, errors.New("repository database is not initialized")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	if tenantUUID == "" {
		return nil, repository.ErrTenantUuidRequired
	}
	out := make([]*model.WeComOpenAuthBinding, 0)
	err := r.DB.WithContext(ctx).
		Where("tenant_uuid = ?", tenantUUID).
		Order("is_default DESC, updated_at DESC").
		Find(&out).Error
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (r *OpenWorkFoundationRepository) ListBindingsBySuite(ctx context.Context, tenantUUID, suiteID string, limit int) ([]*model.WeComOpenAuthBinding, error) {
	if r == nil || r.DB == nil {
		return nil, errors.New("repository database is not initialized")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	suiteID = strings.TrimSpace(suiteID)
	if tenantUUID == "" {
		return nil, repository.ErrTenantUuidRequired
	}
	if suiteID == "" {
		return nil, errors.New("suite_id is required")
	}
	if limit <= 0 || limit > 50 {
		limit = 10
	}
	out := make([]*model.WeComOpenAuthBinding, 0)
	err := r.DB.WithContext(ctx).
		Where("tenant_uuid = ? AND suite_id = ?", tenantUUID, suiteID).
		Order("updated_at DESC").
		Limit(limit).
		Find(&out).Error
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (r *OpenWorkFoundationRepository) GetActiveBindingBySuiteAndCorp(ctx context.Context, tenantUUID, suiteID, corpID string) (*model.WeComOpenAuthBinding, error) {
	if r == nil || r.DB == nil {
		return nil, errors.New("repository database is not initialized")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	suiteID = strings.TrimSpace(suiteID)
	corpID = strings.TrimSpace(corpID)
	if tenantUUID == "" {
		return nil, repository.ErrTenantUuidRequired
	}
	if suiteID == "" || corpID == "" {
		return nil, errors.New("suite_id and corp_id are required")
	}
	var out model.WeComOpenAuthBinding
	err := r.DB.WithContext(ctx).
		Where("tenant_uuid = ? AND suite_id = ? AND corp_id = ? AND status = ?", tenantUUID, suiteID, corpID, model.WeComAuthBindingStatusActive).
		Order("updated_at DESC").
		First(&out).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (r *OpenWorkFoundationRepository) GetLatestAuthEventBySuite(ctx context.Context, tenantUUID, suiteID string) (*model.WeComOpenAuthEvent, error) {
	if r == nil || r.DB == nil {
		return nil, errors.New("repository database is not initialized")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	suiteID = strings.TrimSpace(suiteID)
	if tenantUUID == "" {
		return nil, repository.ErrTenantUuidRequired
	}
	if suiteID == "" {
		return nil, errors.New("suite_id is required")
	}
	var out model.WeComOpenAuthEvent
	err := r.DB.WithContext(ctx).
		Where("tenant_uuid = ? AND suite_id = ?", tenantUUID, suiteID).
		Order("COALESCE(event_time, created_at) DESC, created_at DESC").
		First(&out).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &out, nil
}

func (r *OpenWorkFoundationRepository) SetDefaultBinding(ctx context.Context, tenantUUID, bindingUUID string) (*model.WeComOpenAuthBinding, error) {
	if r == nil || r.DB == nil {
		return nil, errors.New("repository database is not initialized")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	bindingUUID = strings.ToLower(strings.TrimSpace(bindingUUID))
	if tenantUUID == "" {
		return nil, repository.ErrTenantUuidRequired
	}
	if bindingUUID == "" {
		return nil, ErrBindingNotFound
	}
	now := time.Now().UTC()
	var out model.WeComOpenAuthBinding
	err := r.WithTenantTx(ctx, tenantUUID, func(tx *gorm.DB) error {
		if err := tx.Where("tenant_uuid = ? AND binding_uuid = ?", tenantUUID, bindingUUID).First(&out).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrBindingNotFound
			}
			return err
		}
		if err := tx.Model(&model.WeComOpenAuthBinding{}).
			Where("tenant_uuid = ? AND channel_code = ? AND app_type = ? AND is_default = TRUE", tenantUUID, out.ChannelCode, out.AppType).
			Updates(map[string]any{"is_default": false, "updated_at": now}).Error; err != nil {
			return err
		}
		if err := tx.Model(&model.WeComOpenAuthBinding{}).
			Where("tenant_uuid = ? AND binding_uuid = ?", tenantUUID, bindingUUID).
			Updates(map[string]any{"is_default": true, "default_switched_at": now, "updated_at": now}).Error; err != nil {
			return err
		}
		return tx.Where("tenant_uuid = ? AND binding_uuid = ?", tenantUUID, bindingUUID).First(&out).Error
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (r *OpenWorkFoundationRepository) ResolveBinding(ctx context.Context, tenantUUID, bindingUUID string) (*model.WeComOpenAuthBinding, string, error) {
	if r == nil || r.DB == nil {
		return nil, "", errors.New("repository database is not initialized")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	bindingUUID = strings.ToLower(strings.TrimSpace(bindingUUID))
	if tenantUUID == "" {
		return nil, "", repository.ErrTenantUuidRequired
	}
	if bindingUUID != "" {
		var out model.WeComOpenAuthBinding
		err := r.DB.WithContext(ctx).Where("tenant_uuid = ? AND binding_uuid = ?", tenantUUID, bindingUUID).First(&out).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, "", ErrBindingNotFound
			}
			return nil, "", err
		}
		return &out, "explicit", nil
	}
	var out model.WeComOpenAuthBinding
	err := r.DB.WithContext(ctx).
		Where("tenant_uuid = ? AND is_default = TRUE", tenantUUID).
		Order("default_switched_at DESC NULLS LAST, updated_at DESC").
		First(&out).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, "", ErrBindingNotFound
		}
		return nil, "", err
	}
	return &out, "default", nil
}

func (r *OpenWorkFoundationRepository) ResolveBindingByChannelAccount(ctx context.Context, tenantUUID, channelAccountUUID string) (*model.WeComOpenAuthBinding, error) {
	if r == nil || r.DB == nil {
		return nil, errors.New("repository database is not initialized")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	channelAccountUUID = strings.ToLower(strings.TrimSpace(channelAccountUUID))
	if tenantUUID == "" {
		return nil, repository.ErrTenantUuidRequired
	}
	if channelAccountUUID == "" {
		return nil, ErrBindingNotFound
	}
	var out model.WeComOpenAuthBinding
	err := r.DB.WithContext(ctx).
		Where("tenant_uuid = ? AND channel_account_uuid = ?", tenantUUID, channelAccountUUID).
		Order("is_default DESC, default_switched_at DESC NULLS LAST, updated_at DESC").
		First(&out).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrBindingNotFound
		}
		return nil, err
	}
	return &out, nil
}

func (r *OpenWorkFoundationRepository) DisableBindingsByChannelAccount(ctx context.Context, tenantUUID, channelAccountUUID, eventType string) error {
	if r == nil || r.DB == nil {
		return errors.New("repository database is not initialized")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	channelAccountUUID = strings.ToLower(strings.TrimSpace(channelAccountUUID))
	eventType = strings.TrimSpace(eventType)
	if tenantUUID == "" {
		return repository.ErrTenantUuidRequired
	}
	if channelAccountUUID == "" {
		return ErrBindingNotFound
	}
	if eventType == "" {
		eventType = "channel_account_disabled"
	}
	now := time.Now().UTC()
	res := r.DB.WithContext(ctx).
		Model(&model.WeComOpenAuthBinding{}).
		Where("tenant_uuid = ? AND channel_account_uuid = ?", tenantUUID, channelAccountUUID).
		Updates(map[string]any{
			"status":              model.WeComAuthBindingStatusDisabled,
			"is_default":          false,
			"default_switched_at": now,
			"last_event_type":     eventType,
			"last_event_at":       now,
			"updated_at":          now,
		})
	if res.Error != nil {
		return res.Error
	}
	return nil
}

func (r *OpenWorkFoundationRepository) GetLatestSuiteTicket(ctx context.Context, tenantUUID, suiteID string) (string, error) {
	if r == nil || r.DB == nil {
		return "", errors.New("repository database is not initialized")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	suiteID = strings.TrimSpace(suiteID)
	if tenantUUID == "" {
		return "", repository.ErrTenantUuidRequired
	}
	if suiteID == "" {
		return "", errors.New("suite_id is required")
	}
	var task model.WeComOpenCallbackTask
	err := r.DB.WithContext(ctx).
		Where("tenant_uuid = ? AND suite_id = ? AND event_type = ? AND suite_ticket <> ''", tenantUUID, suiteID, "suite_ticket").
		Order("event_time DESC NULLS LAST, created_at DESC").
		First(&task).Error
	if err == nil {
		return strings.TrimSpace(task.SuiteTicket), nil
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return "", err
	}

	// 兼容历史数据：若回调任务表暂无记录，则退回绑定表最近票据。
	var rec model.WeComOpenAuthBinding
	err = r.DB.WithContext(ctx).
		Where("tenant_uuid = ? AND suite_id = ? AND suite_ticket <> ''", tenantUUID, suiteID).
		Order("updated_at DESC").
		First(&rec).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil
		}
		return "", err
	}
	return strings.TrimSpace(rec.SuiteTicket), nil
}

func (r *OpenWorkFoundationRepository) CreateSyncJob(ctx context.Context, job *model.SyncBaselineJob) (*model.SyncBaselineJob, error) {
	if r == nil || r.DB == nil {
		return nil, errors.New("repository database is not initialized")
	}
	if job == nil {
		return nil, errors.New("job is required")
	}
	job.TenantUUID = strings.ToLower(strings.TrimSpace(job.TenantUUID))
	job.BindingUUID = strings.ToLower(strings.TrimSpace(job.BindingUUID))
	job.Domain = strings.ToLower(strings.TrimSpace(job.Domain))
	job.Mode = strings.ToLower(strings.TrimSpace(job.Mode))
	job.Status = strings.ToLower(strings.TrimSpace(job.Status))
	if job.TenantUUID == "" {
		return nil, repository.ErrTenantUuidRequired
	}
	if job.BindingUUID == "" || job.Domain == "" || job.Mode == "" {
		return nil, errors.New("binding_uuid, domain and mode are required")
	}
	if job.Status == "" {
		job.Status = model.SyncJobStatusQueued
	}
	if strings.TrimSpace(job.JobUUID) == "" {
		job.JobUUID = uuid.NewString()
	}
	if job.MaxRetries <= 0 {
		job.MaxRetries = 3
	}
	if job.Context == nil {
		job.Context = datatypes.JSONMap{}
	}
	if job.WriteBackFields == nil {
		job.WriteBackFields = datatypes.JSONMap{}
	}
	err := r.WithTenantTx(ctx, job.TenantUUID, func(tx *gorm.DB) error {
		return tx.Create(job).Error
	})
	if err != nil {
		return nil, err
	}
	return job, nil
}

func (r *OpenWorkFoundationRepository) UpdateSyncJob(ctx context.Context, tenantUUID, jobUUID string, updates map[string]any) (*model.SyncBaselineJob, error) {
	if r == nil || r.DB == nil {
		return nil, errors.New("repository database is not initialized")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	jobUUID = strings.ToLower(strings.TrimSpace(jobUUID))
	if tenantUUID == "" {
		return nil, repository.ErrTenantUuidRequired
	}
	if jobUUID == "" {
		return nil, ErrSyncJobNotFound
	}
	if updates == nil {
		updates = map[string]any{}
	}
	updates["updated_at"] = time.Now().UTC()
	var out model.SyncBaselineJob
	err := r.WithTenantTx(ctx, tenantUUID, func(tx *gorm.DB) error {
		res := tx.Model(&model.SyncBaselineJob{}).
			Where("tenant_uuid = ? AND job_uuid = ?", tenantUUID, jobUUID).
			Updates(updates)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrSyncJobNotFound
		}
		return tx.Where("tenant_uuid = ? AND job_uuid = ?", tenantUUID, jobUUID).First(&out).Error
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (r *OpenWorkFoundationRepository) ListSyncJobs(ctx context.Context, tenantUUID, status string, limit int) ([]*model.SyncBaselineJob, error) {
	if r == nil || r.DB == nil {
		return nil, errors.New("repository database is not initialized")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	status = strings.ToLower(strings.TrimSpace(status))
	if tenantUUID == "" {
		return nil, repository.ErrTenantUuidRequired
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	query := r.DB.WithContext(ctx).Where("tenant_uuid = ?", tenantUUID)
	if status != "" {
		query = query.Where("status = ?", status)
	}
	out := make([]*model.SyncBaselineJob, 0)
	if err := query.Order("created_at DESC").Limit(limit).Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

func (r *OpenWorkFoundationRepository) CreateSyncConflict(ctx context.Context, rec *model.SyncConflictRecord) (*model.SyncConflictRecord, error) {
	if r == nil || r.DB == nil {
		return nil, errors.New("repository database is not initialized")
	}
	if rec == nil {
		return nil, errors.New("conflict record is required")
	}
	rec.TenantUUID = strings.ToLower(strings.TrimSpace(rec.TenantUUID))
	rec.JobUUID = strings.ToLower(strings.TrimSpace(rec.JobUUID))
	rec.BindingUUID = strings.ToLower(strings.TrimSpace(rec.BindingUUID))
	rec.Domain = strings.ToLower(strings.TrimSpace(rec.Domain))
	rec.Status = strings.ToLower(strings.TrimSpace(rec.Status))
	rec.ConflictKey = strings.TrimSpace(rec.ConflictKey)
	if rec.TenantUUID == "" {
		return nil, repository.ErrTenantUuidRequired
	}
	if rec.JobUUID == "" || rec.BindingUUID == "" || rec.Domain == "" || rec.ConflictKey == "" {
		return nil, errors.New("job_uuid, binding_uuid, domain and conflict_key are required")
	}
	if rec.Status == "" {
		rec.Status = model.SyncConflictStatusOpen
	}
	if strings.TrimSpace(rec.ConflictUUID) == "" {
		rec.ConflictUUID = uuid.NewString()
	}
	if rec.Payload == nil {
		rec.Payload = datatypes.JSONMap{}
	}
	if err := r.WithTenantTx(ctx, rec.TenantUUID, func(tx *gorm.DB) error { return tx.Create(rec).Error }); err != nil {
		return nil, err
	}
	return rec, nil
}

func (r *OpenWorkFoundationRepository) ListSyncConflicts(ctx context.Context, tenantUUID, status string, limit int) ([]*model.SyncConflictRecord, error) {
	if r == nil || r.DB == nil {
		return nil, errors.New("repository database is not initialized")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	status = strings.ToLower(strings.TrimSpace(status))
	if tenantUUID == "" {
		return nil, repository.ErrTenantUuidRequired
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	query := r.DB.WithContext(ctx).Where("tenant_uuid = ?", tenantUUID)
	if status != "" {
		query = query.Where("status = ?", status)
	}
	out := make([]*model.SyncConflictRecord, 0)
	if err := query.Order("created_at DESC").Limit(limit).Find(&out).Error; err != nil {
		return nil, err
	}
	return out, nil
}

func (r *OpenWorkFoundationRepository) ReplayConflict(ctx context.Context, tenantUUID, conflictUUID, note string) (*model.SyncConflictRecord, error) {
	if r == nil || r.DB == nil {
		return nil, errors.New("repository database is not initialized")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	conflictUUID = strings.ToLower(strings.TrimSpace(conflictUUID))
	note = strings.TrimSpace(note)
	if tenantUUID == "" {
		return nil, repository.ErrTenantUuidRequired
	}
	if conflictUUID == "" {
		return nil, ErrSyncConflictNotFound
	}
	now := time.Now().UTC()
	var out model.SyncConflictRecord
	err := r.WithTenantTx(ctx, tenantUUID, func(tx *gorm.DB) error {
		res := tx.Model(&model.SyncConflictRecord{}).
			Where("tenant_uuid = ? AND conflict_uuid = ?", tenantUUID, conflictUUID).
			Updates(map[string]any{
				"status":           model.SyncConflictStatusReplayed,
				"resolution_note":  note,
				"replay_count":     gorm.Expr("replay_count + 1"),
				"last_replayed_at": now,
				"updated_at":       now,
			})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrSyncConflictNotFound
		}
		return tx.Where("tenant_uuid = ? AND conflict_uuid = ?", tenantUUID, conflictUUID).First(&out).Error
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (r *OpenWorkFoundationRepository) SyncDashboard(ctx context.Context, tenantUUID string) (map[string]any, error) {
	if r == nil || r.DB == nil {
		return nil, errors.New("repository database is not initialized")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	if tenantUUID == "" {
		return nil, repository.ErrTenantUuidRequired
	}
	statuses := []string{
		model.SyncJobStatusQueued,
		model.SyncJobStatusRunning,
		model.SyncJobStatusSuccess,
		model.SyncJobStatusFailed,
		model.SyncJobStatusDeadLetter,
	}
	statusCount := map[string]int64{}
	for _, status := range statuses {
		var c int64
		if err := r.DB.WithContext(ctx).Model(&model.SyncBaselineJob{}).Where("tenant_uuid = ? AND status = ?", tenantUUID, status).Count(&c).Error; err != nil {
			return nil, err
		}
		statusCount[status] = c
	}
	var openConflicts int64
	if err := r.DB.WithContext(ctx).Model(&model.SyncConflictRecord{}).Where("tenant_uuid = ? AND status = ?", tenantUUID, model.SyncConflictStatusOpen).Count(&openConflicts).Error; err != nil {
		return nil, err
	}
	var maxLagMins float64
	query := fmt.Sprintf(
		`SELECT COALESCE(MAX(EXTRACT(EPOCH FROM (now() - created_at)) / 60), 0) FROM %s WHERE tenant_uuid = ? AND status IN (?, ?)`,
		model.SyncBaselineJob{}.TableName(),
	)
	if r.DB.Dialector != nil && strings.EqualFold(r.DB.Dialector.Name(), "sqlite") {
		query = fmt.Sprintf(
			`SELECT COALESCE(MAX((julianday('now') - julianday(created_at)) * 24 * 60), 0) FROM %s WHERE tenant_uuid = ? AND status IN (?, ?)`,
			model.SyncBaselineJob{}.TableName(),
		)
	}
	err := r.DB.WithContext(ctx).Raw(query, tenantUUID, model.SyncJobStatusQueued, model.SyncJobStatusRunning).Scan(&maxLagMins).Error
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"jobs": map[string]any{
			"queued":      statusCount[model.SyncJobStatusQueued],
			"running":     statusCount[model.SyncJobStatusRunning],
			"success":     statusCount[model.SyncJobStatusSuccess],
			"failed":      statusCount[model.SyncJobStatusFailed],
			"dead_letter": statusCount[model.SyncJobStatusDeadLetter],
		},
		"open_conflicts":  openConflicts,
		"max_lag_minutes": int64(maxLagMins),
	}, nil
}
