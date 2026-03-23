package lead_capture

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	leadmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/lead_capture"
	leadrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/lead_capture"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/logger"
	leadobs "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/observability/lead_capture"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type WeComSyncService struct {
	taskRepo    *leadrepo.LeadSyncTaskRepository
	leadRepo    *leadrepo.LeadRepository
	leadService *LeadService
	metrics     *leadobs.Metrics
	syncFactory *ChannelSyncFactory
}

type TriggerSyncRequest struct {
	TenantUUID         string
	Channel            string
	AppType            string
	ChannelAccountUUID string
	TraceID            string
	TriggerType        string
	TaskProvider       string
}

func NewWeComSyncService(taskRepo *leadrepo.LeadSyncTaskRepository, metrics *leadobs.Metrics, providerAdapter SyncTaskProviderAdapter) *WeComSyncService {
	factory := NewChannelSyncFactory()
	if providerAdapter == nil {
		providerAdapter = NewDefaultSyncTaskProviderAdapter(nil, nil)
	}
	_ = factory.Register("wechat", "wecom", NewDefaultWeComLeadAdapter(), providerAdapter)
	return &WeComSyncService{taskRepo: taskRepo, metrics: metrics, syncFactory: factory}
}

func (s *WeComSyncService) WithLeadIngestion(leadRepo *leadrepo.LeadRepository, adapter WeComLeadAdapter) *WeComSyncService {
	if s == nil {
		return s
	}
	s.leadRepo = leadRepo
	if adapter != nil {
		if s.syncFactory == nil {
			s.syncFactory = NewChannelSyncFactory()
		}
		_ = s.syncFactory.RegisterLeadAdapter("wechat", "wecom", adapter)
	}
	return s
}

func (s *WeComSyncService) WithLeadService(leadService *LeadService) *WeComSyncService {
	if s == nil {
		return s
	}
	s.leadService = leadService
	return s
}

func (s *WeComSyncService) WithChannelFactory(factory *ChannelSyncFactory) *WeComSyncService {
	if s == nil || factory == nil {
		return s
	}
	s.syncFactory = factory
	return s
}

type SyncIngestStats struct {
	Total   int
	Created int
	Updated int
	Merged  int
}

func (s *WeComSyncService) TriggerSync(ctx context.Context, req TriggerSyncRequest) (*leadmodel.LeadSyncTask, error) {
	req, accountUUID, submit, created, err := s.prepareSyncTask(ctx, req)
	if err != nil {
		return nil, err
	}

	if submit.Provider == leadmodel.LeadSyncTaskProviderLocalFallback {
		if runErr := s.executeLocalSyncTask(ctx, req, accountUUID, created); runErr != nil {
			return nil, runErr
		}
	}

	if s.metrics != nil && submit.Provider != leadmodel.LeadSyncTaskProviderLocalFallback {
		s.metrics.RecordSyncTask(submit.Provider, created.Status)
	}
	return created, nil
}

func (s *WeComSyncService) TriggerSyncAsync(ctx context.Context, req TriggerSyncRequest) (*leadmodel.LeadSyncTask, error) {
	req, accountUUID, submit, created, err := s.prepareSyncTask(ctx, req)
	if err != nil {
		return nil, err
	}
	if submit.Provider == leadmodel.LeadSyncTaskProviderLocalFallback {
		go func(backgroundReq TriggerSyncRequest, resolvedAccountUUID string, queuedTask *leadmodel.LeadSyncTask) {
			bgCtx := context.Background()
			if runErr := s.executeLocalSyncTask(bgCtx, backgroundReq, resolvedAccountUUID, queuedTask); runErr != nil {
				logger.WithFields(logger.Fields{
					"component":            "lead_sync",
					"tenant_uuid":          strings.TrimSpace(backgroundReq.TenantUUID),
					"task_uuid":            strings.TrimSpace(queuedTask.TaskUUID),
					"channel_account_uuid": strings.TrimSpace(resolvedAccountUUID),
					"trace_id":             strings.TrimSpace(backgroundReq.TraceID),
				}).WithError(runErr).Error("lead sync async execution failed")
			}
		}(req, accountUUID, created)
	}
	if s.metrics != nil {
		s.metrics.RecordSyncTask(submit.Provider, created.Status)
	}
	return created, nil
}

func (s *WeComSyncService) prepareSyncTask(ctx context.Context, req TriggerSyncRequest) (TriggerSyncRequest, string, SyncTaskSubmitResult, *leadmodel.LeadSyncTask, error) {
	if s == nil || s.taskRepo == nil {
		return TriggerSyncRequest{}, "", SyncTaskSubmitResult{}, nil, errors.New("wecom sync service unavailable")
	}
	req.TenantUUID = strings.ToLower(strings.TrimSpace(req.TenantUUID))
	req.Channel = strings.ToLower(strings.TrimSpace(req.Channel))
	req.AppType = strings.ToLower(strings.TrimSpace(req.AppType))
	if req.TenantUUID == "" {
		return TriggerSyncRequest{}, "", SyncTaskSubmitResult{}, nil, errors.New("tenant_uuid is required")
	}
	if req.Channel == "" {
		req.Channel = "wechat"
	}
	if req.AppType == "" {
		req.AppType = "wecom"
	}
	if req.TriggerType == "" {
		req.TriggerType = "manual"
	}

	accountUUID, resolveSource, err := s.taskRepo.ResolveChannelAccount(ctx, req.TenantUUID, req.Channel, req.AppType, req.ChannelAccountUUID)
	if err != nil {
		if s.metrics != nil {
			s.metrics.RecordSyncTask(leadmodel.LeadSyncTaskProviderLocalFallback, "resolve_failed")
		}
		return TriggerSyncRequest{}, "", SyncTaskSubmitResult{}, nil, err
	}
	adapter, resolveErr := s.resolveTaskProvider(req.Channel, req.AppType)
	if resolveErr != nil {
		return TriggerSyncRequest{}, "", SyncTaskSubmitResult{}, nil, resolveErr
	}
	submit := adapter.SubmitSyncTask(ctx, req, accountUUID)
	if submit.Provider == "" {
		submit.Provider = leadmodel.LeadSyncTaskProviderLocalFallback
	}
	if submit.Status == "" {
		submit.Status = "queued"
	}
	task := &leadmodel.LeadSyncTask{
		ExternalTaskID:       submit.ExternalTaskID,
		TenantUUID:           req.TenantUUID,
		Channel:              req.Channel,
		AppType:              req.AppType,
		ChannelAccountUUID:   accountUUID,
		AccountResolveSource: resolveSource,
		TaskProvider:         submit.Provider,
		TriggerType:          req.TriggerType,
		Status:               submit.Status,
		ProgressTotal:        0,
		ProgressCurrent:      0,
		ProgressPercent:      0,
	}
	created, err := s.taskRepo.CreateTask(ctx, task)
	if err != nil {
		if s.metrics != nil {
			s.metrics.RecordSyncTask(submit.Provider, "create_failed")
		}
		return TriggerSyncRequest{}, "", SyncTaskSubmitResult{}, nil, err
	}
	return req, accountUUID, submit, created, nil
}

func (s *WeComSyncService) executeLocalSyncTask(ctx context.Context, req TriggerSyncRequest, accountUUID string, task *leadmodel.LeadSyncTask) error {
	if err := s.updateTaskRunning(ctx, task); err != nil {
		return err
	}
	lastProgressPercent := -1
	reportProgress := func(current, total int) {
		if s == nil || s.taskRepo == nil || task == nil {
			return
		}
		if total < 0 {
			total = 0
		}
		if current < 0 {
			current = 0
		}
		if total > 0 && current > total {
			current = total
		}
		percent := 0
		if total > 0 {
			percent = int(float64(current) / float64(total) * 100.0)
		}
		if percent < 0 {
			percent = 0
		}
		if percent > 100 {
			percent = 100
		}
		if current < total && percent >= 100 {
			percent = 99
		}
		if current != total && lastProgressPercent >= 0 && percent-lastProgressPercent < 5 {
			return
		}
		if percent == lastProgressPercent && current != total {
			return
		}
		lastProgressPercent = percent
		_ = s.taskRepo.UpdateStatus(ctx, task.TenantUUID, task.TaskUUID, "", map[string]any{
			"progress_total":   total,
			"progress_current": current,
			"progress_percent": percent,
		})
	}

	stats, runErr := s.runLocalSyncIngestion(ctx, req, accountUUID, reportProgress)
	if runErr != nil {
		failErr := s.taskRepo.UpdateStatus(ctx, task.TenantUUID, task.TaskUUID, "failed", map[string]any{
			"error_message": strings.TrimSpace(runErr.Error()),
			"finished_at":   time.Now().UTC(),
		})
		if failErr != nil {
			return failErr
		}
		task.Status = "failed"
		task.ErrorMessage = strings.TrimSpace(runErr.Error())
		if s.metrics != nil {
			s.metrics.RecordSyncTask(leadmodel.LeadSyncTaskProviderLocalFallback, "failed")
		}
		return nil
	}
	finishAt := time.Now().UTC()
	okErr := s.taskRepo.UpdateStatus(ctx, task.TenantUUID, task.TaskUUID, "success", map[string]any{
		"stats_total":      stats.Total,
		"stats_created":    stats.Created,
		"stats_updated":    stats.Updated,
		"stats_merged":     stats.Merged,
		"progress_total":   stats.Total,
		"progress_current": stats.Total,
		"progress_percent": 100,
		"finished_at":      finishAt,
	})
	if okErr != nil {
		return okErr
	}
	task.Status = "success"
	task.StatsTotal = stats.Total
	task.StatsCreated = stats.Created
	task.StatsUpdated = stats.Updated
	task.StatsMerged = stats.Merged
	task.ProgressTotal = stats.Total
	task.ProgressCurrent = stats.Total
	task.ProgressPercent = 100
	task.FinishedAt = &finishAt
	if s.metrics != nil {
		s.metrics.RecordSyncTask(leadmodel.LeadSyncTaskProviderLocalFallback, "success")
	}
	return nil
}

func (s *WeComSyncService) ListSyncTasks(ctx context.Context, tenantUUID, channelAccountUUID, status string, limit int) ([]*leadmodel.LeadSyncTask, error) {
	if s == nil || s.taskRepo == nil {
		return nil, errors.New("wecom sync service unavailable")
	}
	return s.taskRepo.ListByFilter(ctx, tenantUUID, channelAccountUUID, status, limit)
}

func (s *WeComSyncService) RetryTask(ctx context.Context, tenantUUID, taskUUID string) error {
	if s == nil || s.taskRepo == nil {
		return errors.New("wecom sync service unavailable")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	taskUUID = strings.ToLower(strings.TrimSpace(taskUUID))
	if tenantUUID == "" || taskUUID == "" {
		return errors.New("tenant_uuid and task_uuid are required")
	}
	return s.taskRepo.UpdateStatus(ctx, tenantUUID, taskUUID, "queued", map[string]any{
		"trigger_type":     "retry",
		"error_message":    "",
		"error_code":       "",
		"stats_total":      0,
		"stats_created":    0,
		"stats_updated":    0,
		"stats_merged":     0,
		"progress_total":   0,
		"progress_current": 0,
		"progress_percent": 0,
		"finished_at":      nil,
		"started_at":       nil,
		"updated_at":       time.Now().UTC(),
	})
}

func (s *WeComSyncService) updateTaskRunning(ctx context.Context, task *leadmodel.LeadSyncTask) error {
	if task == nil {
		return errors.New("task is required")
	}
	startedAt := time.Now().UTC()
	if err := s.taskRepo.UpdateStatus(ctx, task.TenantUUID, task.TaskUUID, "running", map[string]any{
		"started_at":       startedAt,
		"progress_percent": 5,
	}); err != nil {
		return err
	}
	task.Status = "running"
	task.StartedAt = &startedAt
	task.ProgressPercent = 5
	return nil
}

func (s *WeComSyncService) runLocalSyncIngestion(
	ctx context.Context,
	req TriggerSyncRequest,
	channelAccountUUID string,
	onProgress func(current, total int),
) (SyncIngestStats, error) {
	if s == nil || s.leadRepo == nil {
		return SyncIngestStats{}, nil
	}
	leadAdapter, err := s.resolveLeadAdapter(req.Channel, req.AppType)
	if err != nil {
		return SyncIngestStats{}, err
	}
	items, err := leadAdapter.FetchLeads(ctx, req, channelAccountUUID)
	if err != nil {
		return SyncIngestStats{}, err
	}
	stats := SyncIngestStats{Total: len(items)}
	if onProgress != nil {
		onProgress(0, len(items))
	}
	if s.leadService != nil {
		for idx, raw := range items {
			item := normalizeWeComLeadRecord(raw)
			if item.DisplayName == "" && item.Phone == "" && item.Email == "" {
				continue
			}
			existsBefore := false
			if strings.TrimSpace(item.Phone) != "" {
				if existing, e := s.leadRepo.FindFirstByPhone(ctx, req.TenantUUID, item.Phone); e == nil && existing != nil {
					existsBefore = true
				}
			}
			if !existsBefore && strings.TrimSpace(item.Email) != "" {
				if existing, e := s.leadRepo.FindFirstByEmail(ctx, req.TenantUUID, item.Email); e == nil && existing != nil {
					existsBefore = true
				}
			}
			created, createErr := s.leadService.Create(ctx, req.TenantUUID, LeadCreateRequest{
				DisplayName:       item.DisplayName,
				Phone:             item.Phone,
				Email:             item.Email,
				SourceChannel:     req.Channel,
				SourceAppType:     req.AppType,
				SourceAccountUUID: channelAccountUUID,
			})
			if createErr != nil {
				return SyncIngestStats{}, createErr
			}
			if created != nil {
				if err := s.appendSyncTraceActivity(ctx, req, channelAccountUUID, created.LeadUUID, item, existsBefore); err != nil {
					return SyncIngestStats{}, err
				}
			}
			if existsBefore {
				stats.Updated++
				if created != nil && created.HasMerge {
					stats.Merged++
				}
			} else {
				stats.Created++
			}
			if onProgress != nil {
				onProgress(idx+1, len(items))
			}
		}
		return stats, nil
	}

	err = s.leadRepo.WithTenantTx(ctx, req.TenantUUID, func(tx *gorm.DB) error {
		for idx, raw := range items {
			item := normalizeWeComLeadRecord(raw)
			if item.DisplayName == "" && item.Phone == "" && item.Email == "" {
				continue
			}
			existing, matchErr := s.findExistingLeadForSync(
				ctx,
				tx,
				req.TenantUUID,
				item.Phone,
				item.Email,
				req.Channel,
				req.AppType,
				channelAccountUUID,
			)
			if matchErr != nil {
				return matchErr
			}
			if existing == nil {
				lead := &leadmodel.Lead{
					LeadUUID:          uuid.NewString(),
					TenantUUID:        req.TenantUUID,
					DisplayName:       item.DisplayName,
					Phone:             item.Phone,
					Email:             item.Email,
					Status:            leadmodel.LeadStatusNew,
					SourceChannel:     req.Channel,
					SourceAppType:     req.AppType,
					SourceAccountUUID: &channelAccountUUID,
					CreatedAt:         time.Now().UTC(),
					UpdatedAt:         time.Now().UTC(),
				}
				if err := tx.Create(lead).Error; err != nil {
					return err
				}
				if err := appendSyncTraceActivityTx(ctx, tx, req, channelAccountUUID, lead.LeadUUID, item, false); err != nil {
					return err
				}
				stats.Created++
				continue
			}
			updates := map[string]any{"updated_at": time.Now().UTC()}
			changed := false
			merged := false
			if strings.TrimSpace(existing.DisplayName) == "" && item.DisplayName != "" {
				updates["display_name"] = item.DisplayName
				changed = true
				merged = true
			}
			if strings.TrimSpace(existing.Phone) == "" && item.Phone != "" {
				updates["phone"] = item.Phone
				changed = true
				merged = true
			}
			if strings.TrimSpace(existing.Email) == "" && item.Email != "" {
				updates["email"] = item.Email
				changed = true
				merged = true
			}
			if existing.SourceAccountUUID == nil {
				updates["source_account_uuid"] = channelAccountUUID
				changed = true
			}
			if changed {
				if err := tx.Model(&leadmodel.Lead{}).
					Where("tenant_uuid = ? AND lead_uuid = ?", req.TenantUUID, existing.LeadUUID).
					Updates(updates).Error; err != nil {
					return err
				}
				stats.Updated++
			}
			if merged {
				stats.Merged++
			}
			if err := appendSyncTraceActivityTx(ctx, tx, req, channelAccountUUID, existing.LeadUUID, item, true); err != nil {
				return err
			}
			if onProgress != nil {
				onProgress(idx+1, len(items))
			}
		}
		return nil
	})
	if err != nil {
		return SyncIngestStats{}, err
	}
	return stats, nil
}

func (s *WeComSyncService) resolveLeadAdapter(channel, appType string) (WeComLeadAdapter, error) {
	if s == nil || s.syncFactory == nil {
		return nil, errors.New("channel sync factory unavailable")
	}
	adapter, err := s.syncFactory.ResolveLeadAdapter(channel, appType)
	if err != nil {
		return nil, fmt.Errorf("resolve lead adapter failed: %w", err)
	}
	return adapter, nil
}

func (s *WeComSyncService) resolveTaskProvider(channel, appType string) (SyncTaskProviderAdapter, error) {
	if s == nil || s.syncFactory == nil {
		return nil, errors.New("channel sync factory unavailable")
	}
	adapter, err := s.syncFactory.ResolveTaskProvider(channel, appType)
	if err != nil {
		return nil, fmt.Errorf("resolve task provider failed: %w", err)
	}
	return adapter, nil
}

func (s *WeComSyncService) findExistingLeadForSync(
	ctx context.Context,
	tx *gorm.DB,
	tenantUUID, phone, email, sourceChannel, sourceAppType, sourceAccountUUID string,
) (*leadmodel.Lead, error) {
	phone = strings.TrimSpace(phone)
	email = strings.ToLower(strings.TrimSpace(email))
	sourceChannel = strings.ToLower(strings.TrimSpace(sourceChannel))
	sourceAppType = strings.ToLower(strings.TrimSpace(sourceAppType))
	sourceAccountUUID = strings.TrimSpace(sourceAccountUUID)
	if phone == "" && email == "" {
		return nil, nil
	}
	var out leadmodel.Lead
	query := tx.WithContext(ctx).
		Where("tenant_uuid = ? AND source_channel = ? AND source_app_type = ?", tenantUUID, sourceChannel, sourceAppType)
	if sourceAccountUUID == "" {
		query = query.Where("source_account_uuid IS NULL")
	} else {
		query = query.Where("source_account_uuid = ?", sourceAccountUUID)
	}
	if phone != "" {
		query = query.Where("phone = ?", phone)
	} else {
		query = query.Where("email = ?", email)
	}
	if err := query.Order("created_at ASC").First(&out).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("query existing lead failed: %w", err)
	}
	return &out, nil
}

func (s *WeComSyncService) appendSyncTraceActivity(
	ctx context.Context,
	req TriggerSyncRequest,
	channelAccountUUID, leadUUID string,
	item WeComLeadRecord,
	existsBefore bool,
) error {
	if s == nil || s.leadRepo == nil {
		return nil
	}
	return s.leadRepo.WithTenantTx(ctx, req.TenantUUID, func(tx *gorm.DB) error {
		return appendSyncTraceActivityTx(ctx, tx, req, channelAccountUUID, leadUUID, item, existsBefore)
	})
}

func appendSyncTraceActivityTx(
	ctx context.Context,
	tx *gorm.DB,
	req TriggerSyncRequest,
	channelAccountUUID, leadUUID string,
	item WeComLeadRecord,
	existsBefore bool,
) error {
	leadUUID = strings.TrimSpace(leadUUID)
	if tx == nil || leadUUID == "" {
		return nil
	}
	if !tx.Migrator().HasTable(&leadmodel.LeadActivity{}) {
		return fmt.Errorf("lead activity table missing: %s", leadmodel.LeadActivity{}.TableName())
	}
	payload := datatypes.JSONMap{
		"trace_id":             strings.TrimSpace(req.TraceID),
		"source_channel":       strings.ToLower(strings.TrimSpace(req.Channel)),
		"source_app_type":      strings.ToLower(strings.TrimSpace(req.AppType)),
		"source_account_uuid":  strings.TrimSpace(channelAccountUUID),
		"external_lead_id":     strings.TrimSpace(item.ExternalLeadID),
		"external_wechat_id":   strings.TrimSpace(item.WechatID),
		"display_name":         strings.TrimSpace(item.DisplayName),
		"phone":                strings.TrimSpace(item.Phone),
		"email":                strings.ToLower(strings.TrimSpace(item.Email)),
		"dedup_exists_before":  existsBefore,
		"ingestion_entrypoint": "wecom_sync",
	}
	if !item.OccurredAt.IsZero() {
		payload["occurred_at"] = item.OccurredAt.UTC().Format(time.RFC3339)
	}
	return upsertSyncTraceActivityTx(ctx, tx, req.TenantUUID, leadUUID, payload)
}

func upsertSyncTraceActivityTx(
	ctx context.Context,
	tx *gorm.DB,
	tenantUUID, leadUUID string,
	payload datatypes.JSONMap,
) error {
	if tx == nil {
		return errors.New("database transaction is nil")
	}
	tenantUUID = strings.TrimSpace(tenantUUID)
	leadUUID = strings.TrimSpace(leadUUID)
	if tenantUUID == "" || leadUUID == "" {
		return nil
	}
	extLeadID := payloadString(payload, "external_lead_id")
	wechatID := payloadString(payload, "external_wechat_id")
	accountUUID := payloadString(payload, "source_account_uuid")
	// 缺少稳定外部标识时退化为 append，避免误覆盖历史记录。
	if extLeadID == "" && wechatID == "" {
		return createLeadActivity(ctx, tx, tenantUUID, leadUUID, leadmodel.LeadActivityTypeSyncTrace, payload)
	}
	var candidates []*leadmodel.LeadActivity
	if err := tx.WithContext(ctx).
		Where("tenant_uuid = ? AND lead_uuid = ? AND activity_type = ?", tenantUUID, leadUUID, leadmodel.LeadActivityTypeSyncTrace).
		Order("updated_at DESC").
		Limit(30).
		Find(&candidates).Error; err != nil {
		return err
	}
	for _, item := range candidates {
		if item == nil {
			continue
		}
		existingPayload := datatypes.JSONMap{}
		if item.Payload != nil {
			existingPayload = item.Payload
		}
		if payloadString(existingPayload, "source_account_uuid") != accountUUID {
			continue
		}
		if extLeadID != "" && payloadString(existingPayload, "external_lead_id") != extLeadID {
			continue
		}
		if extLeadID == "" && wechatID != "" && payloadString(existingPayload, "external_wechat_id") != wechatID {
			continue
		}
		return tx.WithContext(ctx).
			Model(&leadmodel.LeadActivity{}).
			Where("tenant_uuid = ? AND activity_uuid = ?", tenantUUID, item.ActivityUUID).
			Updates(map[string]any{
				"payload":    payload,
				"updated_at": time.Now().UTC(),
			}).Error
	}
	// 某些存储实现里 JSONMap 反序列化为非预期类型，会导致精确匹配失效；
	// 若该 lead 仅存在一条 sync_trace，直接回退为更新该条，确保 sync_trace 行为为 upsert。
	if len(candidates) == 1 && candidates[0] != nil {
		return tx.WithContext(ctx).
			Model(&leadmodel.LeadActivity{}).
			Where("tenant_uuid = ? AND activity_uuid = ?", tenantUUID, candidates[0].ActivityUUID).
			Updates(map[string]any{
				"payload":    payload,
				"updated_at": time.Now().UTC(),
			}).Error
	}
	return createLeadActivity(ctx, tx, tenantUUID, leadUUID, leadmodel.LeadActivityTypeSyncTrace, payload)
}

func payloadString(payload datatypes.JSONMap, key string) string {
	if payload == nil {
		return ""
	}
	raw, ok := payload[key]
	if !ok || raw == nil {
		return ""
	}
	switch v := raw.(type) {
	case string:
		return strings.TrimSpace(v)
	case fmt.Stringer:
		return strings.TrimSpace(v.String())
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", v))
	}
}
