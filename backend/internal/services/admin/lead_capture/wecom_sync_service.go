package lead_capture

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	pwresponse "github.com/ArtisanCloud/PowerWeChat/v3/src/kernel/response"
	pwexternalreq "github.com/ArtisanCloud/PowerWeChat/v3/src/work/externalContact/request"
	leadmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/lead_capture"
	orgmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/org_sync"
	socialmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/social_channel_governance"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository"
	leadrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/lead_capture"
	socialrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/social_channel_governance"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/logger"
	leadobs "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/observability/lead_capture"
	socialsvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/social_channel_governance"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/wecomauth"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type WeComSyncService struct {
	taskRepo      *leadrepo.LeadSyncTaskRepository
	leadRepo      *leadrepo.LeadRepository
	leadService   *LeadService
	metrics       *leadobs.Metrics
	realtime      *LeadSyncRealtimePublisher
	syncFactory   *ChannelSyncFactory
	dedupSvc      *LeadDedupService
	syncRepo      *socialrepo.SyncFoundationRepository
	retrySvc      *socialsvc.RetryDeadletterService
	writeMu       sync.Mutex
	writeSeq      map[string]int64
	identityMu    sync.RWMutex
	identityMap   map[string]string
	remarkFactory weComRemarkClientFactory
}

type weComRemarkClient interface {
	Remark(ctx context.Context, data *pwexternalreq.RequestExternalContactRemark) (*pwresponse.ResponseWork, error)
}

type weComRemarkClientFactory func(ctx context.Context, appType string, credentials map[string]string) (weComRemarkClient, error)

type TriggerSyncRequest struct {
	TenantUUID         string
	Channel            string
	AppType            string
	ChannelAccountUUID string
	TraceID            string
	TriggerType        string
	TaskProvider       string
	Domain             string
	Direction          string
	Mode               string
	CheckpointCursor   string
	LeadWriteback      []LeadWritebackRecord
}

type LeadWritebackRecord struct {
	LeadUUID        string         `json:"lead_uuid"`
	ExternalUserID  string         `json:"external_userid"`
	CorpID          string         `json:"corp_id"`
	Phone           string         `json:"phone"`
	Fields          map[string]any `json:"fields"`
	OrderVersion    int64          `json:"order_version"`
	IdempotencyHint string         `json:"idempotency_hint"`
}

func NewWeComSyncService(taskRepo *leadrepo.LeadSyncTaskRepository, metrics *leadobs.Metrics, providerAdapter SyncTaskProviderAdapter) *WeComSyncService {
	factory := NewChannelSyncFactory()
	if providerAdapter == nil {
		providerAdapter = NewDefaultSyncTaskProviderAdapter(nil, nil)
	}
	_ = factory.Register("wechat", "wecom", NewDefaultWeComLeadAdapter(), providerAdapter)
	_ = factory.Register("wechat", "openwork", NewDefaultWeComLeadAdapter(), providerAdapter)
	return &WeComSyncService{
		taskRepo:      taskRepo,
		metrics:       metrics,
		syncFactory:   factory,
		dedupSvc:      NewLeadDedupService(),
		writeSeq:      map[string]int64{},
		identityMap:   map[string]string{},
		remarkFactory: defaultWeComRemarkClientFactory,
	}
}

func defaultWeComRemarkClientFactory(_ context.Context, appType string, credentials map[string]string) (weComRemarkClient, error) {
	app, err := newWeComLeadSyncApp("wechat", appType, credentials)
	if err != nil {
		return nil, err
	}
	if app == nil || app.ExternalContact == nil {
		return nil, errors.New("wecom external contact client unavailable")
	}
	return app.ExternalContact, nil
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
		_ = s.syncFactory.RegisterLeadAdapter("wechat", "openwork", adapter)
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

func (s *WeComSyncService) WithRealtimePublisher(realtime *LeadSyncRealtimePublisher) *WeComSyncService {
	if s == nil {
		return s
	}
	s.realtime = realtime
	return s
}

func (s *WeComSyncService) WithChannelFactory(factory *ChannelSyncFactory) *WeComSyncService {
	if s == nil || factory == nil {
		return s
	}
	s.syncFactory = factory
	return s
}

func (s *WeComSyncService) WithSyncFoundation(syncRepo *socialrepo.SyncFoundationRepository) *WeComSyncService {
	if s == nil {
		return s
	}
	s.syncRepo = syncRepo
	if syncRepo != nil {
		s.retrySvc = socialsvc.NewRetryDeadletterService(syncRepo)
	}
	return s
}

func (s *WeComSyncService) WithRemarkClientFactory(factory weComRemarkClientFactory) *WeComSyncService {
	if s == nil {
		return s
	}
	s.remarkFactory = factory
	return s
}

func (s *WeComSyncService) WithRemarkClient(client weComRemarkClient) *WeComSyncService {
	if s == nil {
		return s
	}
	if client == nil {
		s.remarkFactory = nil
		return s
	}
	s.remarkFactory = func(context.Context, string, map[string]string) (weComRemarkClient, error) {
		return client, nil
	}
	return s
}

type SyncIngestStats struct {
	Total   int
	Created int
	Updated int
	Merged  int
}

func (s *WeComSyncService) TriggerSync(ctx context.Context, req TriggerSyncRequest) (*leadmodel.LeadSyncTask, error) {
	req = normalizeTriggerSyncRequest(req)
	if req.Domain == "leads" && req.Direction == "push" {
		return s.triggerLeadWriteback(ctx, req)
	}
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
	req = normalizeTriggerSyncRequest(req)
	if req.Domain == "leads" && req.Direction == "push" {
		// writeback requires strict ordering/idempotency checks, execute synchronously.
		return s.triggerLeadWriteback(ctx, req)
	}
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

func normalizeTriggerSyncRequest(req TriggerSyncRequest) TriggerSyncRequest {
	req.Domain = strings.ToLower(strings.TrimSpace(req.Domain))
	req.Direction = strings.ToLower(strings.TrimSpace(req.Direction))
	req.Mode = strings.ToLower(strings.TrimSpace(req.Mode))
	if req.Domain == "" {
		req.Domain = "external_contacts"
	}
	if req.Direction == "" {
		req.Direction = "pull"
	}
	if req.Mode == "" {
		req.Mode = "incremental"
	}
	return req
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
	s.publishTaskProgress(ctx, created)
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
		task.ProgressTotal = total
		task.ProgressCurrent = current
		task.ProgressPercent = percent
		s.publishTaskProgress(ctx, task)
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
		s.publishTaskProgress(ctx, task)
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
	s.publishTaskProgress(ctx, task)
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

func (s *WeComSyncService) ClearSyncTasks(ctx context.Context, tenantUUID, channelAccountUUID, status string) (int64, error) {
	if s == nil || s.taskRepo == nil {
		return 0, errors.New("wecom sync service unavailable")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	channelAccountUUID = strings.ToLower(strings.TrimSpace(channelAccountUUID))
	status = strings.ToLower(strings.TrimSpace(status))
	if tenantUUID == "" {
		return 0, repository.ErrTenantUuidRequired
	}
	return s.taskRepo.ClearByFilter(ctx, tenantUUID, channelAccountUUID, status)
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
	s.publishTaskProgress(ctx, task)
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
		seenDedup := map[string]struct{}{}
		ownerBindingCache := map[string]string{}
		for idx, raw := range items {
			item := normalizeWeComLeadRecord(raw)
			if item.DisplayName == "" && item.Phone == "" && item.Email == "" {
				continue
			}
			ownerUserUUID, resolveOwnerErr := s.resolveLeadOwnerUserUUIDFromExternalMember(
				ctx,
				req.TenantUUID,
				channelAccountUUID,
				item.OwnerMemberUUID,
				ownerBindingCache,
			)
			if resolveOwnerErr != nil {
				return SyncIngestStats{}, resolveOwnerErr
			}
			if s.dedupSvc != nil {
				key := s.dedupSvc.BuildExternalContactKey(item.ExternalLeadID, item.Phone, item.CorpID, req.Channel)
				if _, ok := seenDedup[key]; ok {
					continue
				}
				seenDedup[key] = struct{}{}
			}
			identityKey := buildLeadExternalIdentityKey(req.TenantUUID, channelAccountUUID, item.ExternalLeadID, item.WechatID)
			if identityKey != "" {
				if cachedLeadUUID := s.lookupLeadUUIDByIdentity(identityKey); cachedLeadUUID != "" {
					if existing, e := s.leadRepo.GetByUUID(ctx, req.TenantUUID, cachedLeadUUID); e == nil && existing != nil {
						_ = s.appendSyncTraceActivity(ctx, req, channelAccountUUID, existing.LeadUUID, item, true)
						stats.Updated++
						if onProgress != nil {
							onProgress(idx+1, len(items))
						}
						continue
					}
				}
				if s.leadRepo != nil && s.leadRepo.DB != nil {
					if existing, e := s.findExistingLeadByExternalIdentityForSync(
						ctx,
						s.leadRepo.DB,
						req.TenantUUID,
						channelAccountUUID,
						item.ExternalLeadID,
						item.WechatID,
					); e == nil && existing != nil {
						s.rememberLeadUUIDByIdentity(identityKey, existing.LeadUUID)
						_ = s.appendSyncTraceActivity(ctx, req, channelAccountUUID, existing.LeadUUID, item, true)
						stats.Updated++
						if onProgress != nil {
							onProgress(idx+1, len(items))
						}
						continue
					}
				}
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
				OwnerUserUUID:     ownerUserUUID,
			})
			if createErr != nil {
				return SyncIngestStats{}, createErr
			}
			if created != nil {
				if err := s.appendSyncTraceActivity(ctx, req, channelAccountUUID, created.LeadUUID, item, existsBefore); err != nil {
					return SyncIngestStats{}, err
				}
				if identityKey != "" {
					s.rememberLeadUUIDByIdentity(identityKey, created.LeadUUID)
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
		s.persistExternalContactCheckpoint(ctx, req.TenantUUID, req.CheckpointCursor, stats.Total)
		return stats, nil
	}

	seenDedup := map[string]struct{}{}
	ownerBindingCache := map[string]string{}
	err = s.leadRepo.WithTenantTx(ctx, req.TenantUUID, func(tx *gorm.DB) error {
		for idx, raw := range items {
			item := normalizeWeComLeadRecord(raw)
			if item.DisplayName == "" && item.Phone == "" && item.Email == "" {
				continue
			}
			ownerUserUUID, resolveOwnerErr := s.resolveLeadOwnerUserUUIDFromExternalMember(
				ctx,
				req.TenantUUID,
				channelAccountUUID,
				item.OwnerMemberUUID,
				ownerBindingCache,
			)
			if resolveOwnerErr != nil {
				return resolveOwnerErr
			}
			if s.dedupSvc != nil {
				key := s.dedupSvc.BuildExternalContactKey(item.ExternalLeadID, item.Phone, item.CorpID, req.Channel)
				if _, ok := seenDedup[key]; ok {
					continue
				}
				seenDedup[key] = struct{}{}
			}
			identityKey := buildLeadExternalIdentityKey(req.TenantUUID, channelAccountUUID, item.ExternalLeadID, item.WechatID)
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
			if existing == nil && identityKey != "" {
				if cachedLeadUUID := s.lookupLeadUUIDByIdentity(identityKey); cachedLeadUUID != "" {
					var cached leadmodel.Lead
					if err := tx.WithContext(ctx).
						Where("tenant_uuid = ? AND lead_uuid = ?", req.TenantUUID, cachedLeadUUID).
						First(&cached).Error; err == nil {
						existing = &cached
					}
				}
			}
			if existing == nil {
				existing, matchErr = s.findExistingLeadByExternalIdentityForSync(
					ctx,
					tx,
					req.TenantUUID,
					channelAccountUUID,
					item.ExternalLeadID,
					item.WechatID,
				)
			}
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
					Status:            leadmodel.LeadStatusCaptured,
					SourceChannel:     req.Channel,
					SourceAppType:     req.AppType,
					SourceAccountUUID: &channelAccountUUID,
					OwnerUserUUID:     ownerUserUUID,
					CreatedAt:         time.Now().UTC(),
					UpdatedAt:         time.Now().UTC(),
				}
				if err := tx.Create(lead).Error; err != nil {
					return err
				}
				if err := appendSyncTraceActivityTx(ctx, tx, req, channelAccountUUID, lead.LeadUUID, item, false); err != nil {
					return err
				}
				if identityKey != "" {
					s.rememberLeadUUIDByIdentity(identityKey, lead.LeadUUID)
				}
				stats.Created++
				continue
			}
			if identityKey != "" {
				s.rememberLeadUUIDByIdentity(identityKey, existing.LeadUUID)
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
			if strings.TrimSpace(existing.OwnerUserUUID) == "" && ownerUserUUID != "" {
				updates["owner_user_uuid"] = ownerUserUUID
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
	s.persistExternalContactCheckpoint(ctx, req.TenantUUID, req.CheckpointCursor, stats.Total)
	return stats, nil
}

func (s *WeComSyncService) persistExternalContactCheckpoint(ctx context.Context, tenantUUID, cursor string, total int) {
	if s == nil || s.syncRepo == nil {
		return
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	if tenantUUID == "" {
		return
	}
	cursor = strings.TrimSpace(cursor)
	if cursor == "" {
		cursor = fmt.Sprintf("total:%d@%d", total, time.Now().UTC().Unix())
	}
	_ = s.syncRepo.UpsertCheckpoint(ctx, &socialmodel.SyncCheckpoint{
		TenantUUID:      tenantUUID,
		Domain:          "external_contacts",
		Direction:       "pull",
		Cursor:          cursor,
		SnapshotVersion: fmt.Sprintf("v%d", time.Now().UTC().Unix()),
		LastEventTime:   time.Now().UTC(),
	})
}

func (s *WeComSyncService) triggerLeadWriteback(ctx context.Context, req TriggerSyncRequest) (*leadmodel.LeadSyncTask, error) {
	if s == nil || s.taskRepo == nil {
		return nil, errors.New("wecom sync service unavailable")
	}
	req.TenantUUID = strings.ToLower(strings.TrimSpace(req.TenantUUID))
	if req.TenantUUID == "" {
		return nil, errors.New("tenant_uuid is required")
	}
	channel := strings.ToLower(strings.TrimSpace(req.Channel))
	appType := strings.ToLower(strings.TrimSpace(req.AppType))
	if channel == "" {
		channel = "wechat"
	}
	if appType == "" {
		appType = "wecom"
	}
	channelAccountUUID, resolveSource, err := s.taskRepo.ResolveChannelAccount(ctx, req.TenantUUID, channel, appType, req.ChannelAccountUUID)
	if err != nil {
		return nil, err
	}
	job := &leadmodel.LeadSyncTask{
		TenantUUID:           req.TenantUUID,
		Channel:              channel,
		AppType:              appType,
		ChannelAccountUUID:   channelAccountUUID,
		AccountResolveSource: resolveSource,
		TaskProvider:         leadmodel.LeadSyncTaskProviderLocalFallback,
		TriggerType:          "writeback",
		Status:               "running",
		StartedAt:            ptrTime(time.Now().UTC()),
	}
	created, err := s.taskRepo.CreateTask(ctx, job)
	if err != nil {
		return nil, err
	}
	s.publishTaskProgress(ctx, created)

	policy, policyErr := s.GetLeadWritebackPolicy(ctx, req.TenantUUID, channel, appType)
	if policyErr != nil || policy == nil {
		policy = &socialmodel.SyncWritebackPolicy{
			MappingRules:    datatypes.JSONMap{"whitelist": []string{"display_name", "phone", "email", "owner_user_uuid", "status"}},
			ProtectedFields: datatypes.JSONMap{"fields": []string{"lead_uuid", "tenant_uuid", "source_account_uuid", "source_channel", "source_app_type"}},
			OverwriteMode:   "safe",
			Enabled:         true,
		}
	}
	writebackPolicy := LeadWritebackPolicy{
		WhitelistedFields: stringSliceFromAny(policy.MappingRules["whitelist"]),
		ProtectedFields:   stringSliceFromAny(policy.ProtectedFields["fields"]),
	}
	credentials, err := s.resolveWeComWritebackCredentials(ctx, req.TenantUUID, channelAccountUUID)
	if err != nil {
		_ = s.taskRepo.UpdateStatus(ctx, req.TenantUUID, created.TaskUUID, "failed", map[string]any{
			"error_message": err.Error(),
			"finished_at":   time.Now().UTC(),
		})
		created.Status = "failed"
		created.ErrorMessage = err.Error()
		s.publishTaskProgress(ctx, created)
		return created, nil
	}
	remarkFactory := s.remarkFactory
	if remarkFactory == nil {
		remarkFactory = defaultWeComRemarkClientFactory
	}
	remarkClient, err := remarkFactory(ctx, req.AppType, credentials)
	if err != nil {
		_ = s.taskRepo.UpdateStatus(ctx, req.TenantUUID, created.TaskUUID, "failed", map[string]any{
			"error_message": err.Error(),
			"finished_at":   time.Now().UTC(),
		})
		created.Status = "failed"
		created.ErrorMessage = err.Error()
		s.publishTaskProgress(ctx, created)
		return created, nil
	}
	if remarkClient == nil {
		err = errors.New("wecom external contact client unavailable")
		_ = s.taskRepo.UpdateStatus(ctx, req.TenantUUID, created.TaskUUID, "failed", map[string]any{
			"error_message": err.Error(),
			"finished_at":   time.Now().UTC(),
		})
		created.Status = "failed"
		created.ErrorMessage = err.Error()
		s.publishTaskProgress(ctx, created)
		return created, nil
	}

	externalUserCache := make(map[string]string)
	operatorUserCache := make(map[string]string)
	accepted := 0
	rejected := 0
	upstreamAttempted := 0
	rejectReasons := map[string]int{
		"missing_external_userid": 0,
		"missing_operator_userid": 0,
		"out_of_order":            0,
		"payload_filtered":        0,
		"idempotent_skipped":      0,
		"upstream_call_failed":    0,
		"upstream_resp_failed":    0,
	}
	var firstErr error
	for _, item := range req.LeadWriteback {
		externalUserID := strings.TrimSpace(item.ExternalUserID)
		if externalUserID == "" {
			externalUserID = strings.TrimSpace(fieldString(item.Fields, "external_userid"))
		}
		if externalUserID == "" {
			var resolveErr error
			externalUserID, resolveErr = s.resolveLeadExternalUserID(ctx, req.TenantUUID, channelAccountUUID, item.LeadUUID, externalUserCache)
			if resolveErr != nil && firstErr == nil {
				firstErr = resolveErr
			}
		}
		if externalUserID == "" {
			rejected++
			rejectReasons["missing_external_userid"]++
			if firstErr == nil {
				firstErr = errors.New("lead writeback missing external_userid")
			}
			continue
		}

		operatorUserID := strings.TrimSpace(fieldString(item.Fields, "userid"))
		if operatorUserID == "" {
			operatorUserID = strings.TrimSpace(fieldString(item.Fields, "follow_userid"))
		}
		if operatorUserID == "" {
			var resolveErr error
			operatorUserID, resolveErr = s.resolveLeadOwnerOperatorUserID(ctx, req.TenantUUID, channelAccountUUID, item.Fields, operatorUserCache)
			if resolveErr != nil && firstErr == nil {
				firstErr = resolveErr
			}
		}
		if operatorUserID == "" {
			rejected++
			rejectReasons["missing_operator_userid"]++
			if firstErr == nil {
				firstErr = errors.New("lead writeback missing operator userid")
			}
			continue
		}

		item.ExternalUserID = externalUserID
		if !s.passWritebackOrder(req.TenantUUID, item.ExternalUserID, item.OrderVersion) {
			rejected++
			rejectReasons["out_of_order"]++
			continue
		}
		payload := BuildLeadWritebackPayload(item.Fields, writebackPolicy)
		if len(payload) == 0 {
			rejected++
			rejectReasons["payload_filtered"]++
			continue
		}
		if !s.passWritebackIdempotency(ctx, req.TenantUUID, item, payload) {
			rejected++
			rejectReasons["idempotent_skipped"]++
			continue
		}
		if failureFlag(payload) {
			firstErr = errors.New("lead writeback upstream rejected payload")
			break
		}
		remarkReq := &pwexternalreq.RequestExternalContactRemark{
			UserID:         operatorUserID,
			ExternalUserID: externalUserID,
			Remark:         fieldString(payload, "display_name"),
		}
		if phone := strings.TrimSpace(fieldString(payload, "phone")); phone != "" {
			remarkReq.RemarkMobiles = []string{phone}
		}
		email := strings.TrimSpace(fieldString(payload, "email"))
		statusText := strings.TrimSpace(fieldString(payload, "status"))
		phoneText := strings.TrimSpace(fieldString(payload, "phone"))
		descParts := make([]string, 0, 3)
		if statusText != "" {
			descParts = append(descParts, fmt.Sprintf("status=%s", statusText))
		}
		if email != "" {
			descParts = append(descParts, fmt.Sprintf("email=%s", email))
		}
		// WeCom may not persist remark_mobiles in some tenant/runtime combinations.
		// Keep mobile in description as a reliable fallback for remote visibility.
		if phoneText != "" {
			descParts = append(descParts, fmt.Sprintf("mobile=%s", phoneText))
		}
		if len(descParts) > 0 {
			remarkReq.Description = strings.Join(descParts, ",")
		}
		upstreamAttempted++
		resp, callErr := remarkClient.Remark(ctx, remarkReq)
		if callErr != nil {
			rejected++
			rejectReasons["upstream_call_failed"]++
			if firstErr == nil {
				firstErr = fmt.Errorf("lead writeback remark failed: %w", callErr)
			}
			continue
		}
		if resp != nil && resp.ErrCode == 84061 {
			// 84061: current userid is not following this external contact.
			// Retry once with owner_external_userid resolved from latest sync_trace.
			fallbackOperatorID := strings.TrimSpace(fieldString(item.Fields, "owner_external_userid"))
			if fallbackOperatorID == "" {
				var fallbackErr error
				fallbackOperatorID, fallbackErr = s.resolveLeadOwnerExternalUserIDFromSyncTrace(
					ctx,
					req.TenantUUID,
					channelAccountUUID,
					item.LeadUUID,
					externalUserID,
				)
				if fallbackErr != nil {
					logger.WithFields(logger.Fields{
						"component":            "lead_capture_sync",
						"tenant_uuid":          req.TenantUUID,
						"channel_account_uuid": channelAccountUUID,
						"lead_uuid":            item.LeadUUID,
						"external_userid":      externalUserID,
						"error":                fallbackErr.Error(),
					}).Warn("resolve owner_external_userid from sync_trace failed")
				}
			}
			if fallbackOperatorID != "" && fallbackOperatorID != operatorUserID {
				retryReq := *remarkReq
				retryReq.UserID = fallbackOperatorID
				retryResp, retryErr := remarkClient.Remark(ctx, &retryReq)
				if retryErr == nil && retryResp != nil {
					resp = retryResp
					operatorUserID = fallbackOperatorID
				}
			}
		}
		if err := validateWeComResponseCode("externalcontact.remark", *resp); err != nil {
			rejected++
			rejectReasons["upstream_resp_failed"]++
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		if traceErr := createLeadActivity(ctx, s.taskRepo.DB, req.TenantUUID, item.LeadUUID, leadmodel.LeadActivityTypeSyncTrace, datatypes.JSONMap{
			"direction":           "push",
			"source_account_uuid": channelAccountUUID,
			"external_lead_id":    externalUserID,
			"status":              "success",
			"phone":               phoneText,
		}); traceErr != nil {
			logger.WithFields(logger.Fields{
				"component":   "lead_capture_sync",
				"tenant_uuid": req.TenantUUID,
				"lead_uuid":   item.LeadUUID,
				"task_uuid":   created.TaskUUID,
				"error":       traceErr.Error(),
			}).Warn("lead writeback sync trace persist failed")
		}
		accepted++
	}

	logger.WithFields(logger.Fields{
		"component":            "lead_capture_sync",
		"tenant_uuid":          req.TenantUUID,
		"channel_account_uuid": channelAccountUUID,
		"task_uuid":            created.TaskUUID,
		"writeback_total":      len(req.LeadWriteback),
		"upstream_attempted":   upstreamAttempted,
		"accepted":             accepted,
		"rejected":             rejected,
		"reject_reasons":       rejectReasons,
	}).Info("lead writeback summary")

	idempotentSkipped := rejectReasons["idempotent_skipped"]
	if upstreamAttempted == 0 && len(req.LeadWriteback) > 0 && idempotentSkipped < len(req.LeadWriteback) {
		if firstErr == nil {
			firstErr = fmt.Errorf("no upstream write attempted (rejected=%d)", rejected)
		}
	}

	if firstErr != nil && accepted == 0 {
		_ = s.taskRepo.UpdateStatus(ctx, req.TenantUUID, created.TaskUUID, "failed", map[string]any{
			"error_message": firstErr.Error(),
			"stats_total":   len(req.LeadWriteback),
			"stats_updated": accepted,
			"stats_failed":  rejected,
			"finished_at":   time.Now().UTC(),
		})
		if s.retrySvc != nil {
			_ = s.retrySvc.HandleFailure(ctx, &socialmodel.SyncJob{
				JobUUID:      created.TaskUUID,
				TenantUUID:   req.TenantUUID,
				Domain:       "leads",
				Direction:    "push",
				MaxAttempts:  3,
				AttemptNo:    2,
				Payload:      datatypes.JSONMap{"task_uuid": created.TaskUUID, "domain": "leads"},
				ErrorCode:    "WRITEBACK_FAILED",
				ErrorMessage: firstErr.Error(),
			}, "WRITEBACK_FAILED", firstErr.Error())
		}
		created.Status = "failed"
		created.ErrorMessage = firstErr.Error()
		created.StatsTotal = len(req.LeadWriteback)
		created.StatsUpdated = accepted
		created.ProgressTotal = len(req.LeadWriteback)
		created.ProgressCurrent = len(req.LeadWriteback)
		created.ProgressPercent = 100
		s.publishTaskProgress(ctx, created)
		return created, nil
	}

	_ = s.taskRepo.UpdateStatus(ctx, req.TenantUUID, created.TaskUUID, "success", map[string]any{
		"stats_total":      len(req.LeadWriteback),
		"stats_updated":    accepted,
		"stats_failed":     rejected,
		"progress_total":   len(req.LeadWriteback),
		"progress_current": len(req.LeadWriteback),
		"progress_percent": 100,
		"finished_at":      time.Now().UTC(),
	})
	created.Status = "success"
	created.StatsTotal = len(req.LeadWriteback)
	created.StatsUpdated = accepted
	created.ProgressTotal = len(req.LeadWriteback)
	created.ProgressCurrent = len(req.LeadWriteback)
	created.ProgressPercent = 100
	s.publishTaskProgress(ctx, created)
	return created, nil
}

func (s *WeComSyncService) resolveLeadOwnerExternalUserIDFromSyncTrace(
	ctx context.Context,
	tenantUUID, channelAccountUUID, leadUUID, externalUserID string,
) (string, error) {
	if s == nil || s.taskRepo == nil || s.taskRepo.DB == nil {
		return "", errors.New("repository database is not initialized")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	channelAccountUUID = strings.ToLower(strings.TrimSpace(channelAccountUUID))
	leadUUID = strings.TrimSpace(leadUUID)
	externalUserID = strings.TrimSpace(externalUserID)
	if tenantUUID == "" || leadUUID == "" {
		return "", nil
	}

	activities := make([]leadmodel.LeadActivity, 0, 16)
	if err := s.taskRepo.DB.WithContext(ctx).
		Where(
			"tenant_uuid = ? AND lead_uuid = ? AND activity_type = ?",
			tenantUUID,
			leadUUID,
			leadmodel.LeadActivityTypeSyncTrace,
		).
		Order("updated_at DESC").
		Limit(30).
		Find(&activities).Error; err != nil {
		return "", err
	}

	for _, activity := range activities {
		payload := activity.Payload
		if payload == nil {
			continue
		}
		if channelAccountUUID != "" && payloadString(payload, "source_account_uuid") != channelAccountUUID {
			continue
		}
		if externalUserID != "" {
			payloadExternal := strings.TrimSpace(payloadString(payload, "external_lead_id"))
			if payloadExternal == "" {
				payloadExternal = strings.TrimSpace(payloadString(payload, "external_wechat_id"))
			}
			if payloadExternal != externalUserID {
				continue
			}
		}
		operatorUserID := strings.TrimSpace(payloadString(payload, "owner_external_userid"))
		if operatorUserID != "" {
			return operatorUserID, nil
		}
	}
	return "", nil
}

func (s *WeComSyncService) publishTaskProgress(ctx context.Context, task *leadmodel.LeadSyncTask) {
	if s == nil || s.realtime == nil || task == nil {
		return
	}
	tenantUUID := strings.TrimSpace(task.TenantUUID)
	if tenantUUID == "" {
		return
	}
	payload := LeadSyncProgressEvent{
		TenantUUID:         tenantUUID,
		TaskUUID:           strings.TrimSpace(task.TaskUUID),
		Channel:            strings.TrimSpace(task.Channel),
		AppType:            strings.TrimSpace(task.AppType),
		ChannelAccountUUID: strings.TrimSpace(task.ChannelAccountUUID),
		Status:             strings.TrimSpace(task.Status),
		ProgressTotal:      task.ProgressTotal,
		ProgressCurrent:    task.ProgressCurrent,
		ProgressPercent:    task.ProgressPercent,
		StatsTotal:         task.StatsTotal,
		StatsCreated:       task.StatsCreated,
		StatsUpdated:       task.StatsUpdated,
		StatsMerged:        task.StatsMerged,
		ErrorMessage:       strings.TrimSpace(task.ErrorMessage),
	}
	_ = s.realtime.PublishLeadSyncProgress(ctx, tenantUUID, payload)
}

func (s *WeComSyncService) GetLeadWritebackPolicy(ctx context.Context, tenantUUID, channel, appType string) (*socialmodel.SyncWritebackPolicy, error) {
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	channel = strings.ToLower(strings.TrimSpace(channel))
	appType = strings.ToLower(strings.TrimSpace(appType))
	if tenantUUID == "" {
		return nil, errors.New("tenant_uuid is required")
	}
	if channel == "" {
		channel = "wechat"
	}
	if appType == "" {
		appType = "wecom"
	}
	defaultPolicy := &socialmodel.SyncWritebackPolicy{
		TenantUUID:      tenantUUID,
		Domain:          "leads",
		MappingRules:    datatypes.JSONMap{"whitelist": []string{"display_name", "phone", "email", "owner_user_uuid", "status"}},
		ProtectedFields: datatypes.JSONMap{"fields": []string{"lead_uuid", "tenant_uuid", "source_account_uuid", "source_channel", "source_app_type"}},
		OverwriteMode:   "safe",
		Enabled:         true,
	}
	if _, err := wecomauth.ResolveKind(channel, appType); err != nil {
		defaultPolicy.CapabilityStatus = socialmodel.CapabilityStatusNotSupported
		return defaultPolicy, nil
	}
	defaultPolicy.CapabilityStatus = socialmodel.CapabilityStatusSupported
	if s.syncRepo == nil {
		return defaultPolicy, nil
	}
	p, err := s.syncRepo.GetWritebackPolicy(ctx, tenantUUID, "leads")
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return defaultPolicy, nil
		}
		return nil, err
	}
	if p == nil {
		return defaultPolicy, nil
	}
	return p, nil
}

func (s *WeComSyncService) UpdateLeadWritebackPolicy(
	ctx context.Context,
	tenantUUID, channel, appType string,
	mappingRules, protectedFields map[string]any,
	overwriteMode string,
	enabled bool,
) (*socialmodel.SyncWritebackPolicy, error) {
	current, err := s.GetLeadWritebackPolicy(ctx, tenantUUID, channel, appType)
	if err != nil {
		return nil, err
	}
	if current.CapabilityStatus == socialmodel.CapabilityStatusNotSupported {
		return current, nil
	}
	if s.syncRepo == nil {
		if mappingRules != nil {
			current.MappingRules = datatypes.JSONMap(mappingRules)
		}
		if protectedFields != nil {
			current.ProtectedFields = datatypes.JSONMap(protectedFields)
		}
		if strings.TrimSpace(overwriteMode) != "" {
			current.OverwriteMode = strings.TrimSpace(overwriteMode)
		}
		current.Enabled = enabled
		return current, nil
	}
	return s.syncRepo.UpsertWritebackPolicy(ctx, &socialmodel.SyncWritebackPolicy{
		TenantUUID:       strings.ToLower(strings.TrimSpace(tenantUUID)),
		Domain:           "leads",
		MappingRules:     datatypes.JSONMap(mappingRules),
		ProtectedFields:  datatypes.JSONMap(protectedFields),
		OverwriteMode:    strings.TrimSpace(overwriteMode),
		Enabled:          enabled,
		CapabilityStatus: socialmodel.CapabilityStatusSupported,
	})
}

func (s *WeComSyncService) ListLeadWritebackDeadLetters(ctx context.Context, tenantUUID string, limit int) ([]socialmodel.SyncDeadLetterItem, error) {
	if s.retrySvc == nil {
		return []socialmodel.SyncDeadLetterItem{}, nil
	}
	return s.retrySvc.List(ctx, tenantUUID, "leads", "", limit)
}

func (s *WeComSyncService) ReplayLeadWritebackDeadLetter(ctx context.Context, tenantUUID, deadLetterUUID string) (*socialmodel.SyncDeadLetterItem, error) {
	if s.retrySvc == nil {
		return nil, errors.New("retry service unavailable")
	}
	return s.retrySvc.Replay(ctx, tenantUUID, deadLetterUUID, "lead_capture_admin")
}

func (s *WeComSyncService) passWritebackOrder(tenantUUID, externalUserID string, version int64) bool {
	if strings.TrimSpace(externalUserID) == "" || version <= 0 {
		return true
	}
	key := strings.ToLower(strings.TrimSpace(tenantUUID)) + ":" + strings.ToLower(strings.TrimSpace(externalUserID))
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	last, ok := s.writeSeq[key]
	if ok && version < last {
		return false
	}
	s.writeSeq[key] = version
	return true
}

func (s *WeComSyncService) passWritebackIdempotency(ctx context.Context, tenantUUID string, item LeadWritebackRecord, payload map[string]any) bool {
	if s.dedupSvc == nil {
		return true
	}
	key := s.dedupSvc.BuildExternalContactKey(item.ExternalUserID, item.Phone, item.CorpID, "wechat")
	if strings.TrimSpace(key) == "" || s.syncRepo == nil {
		return true
	}
	cp, err := s.syncRepo.GetCheckpoint(ctx, tenantUUID, "leads", "push")
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return true
	}
	if cp != nil && strings.TrimSpace(cp.Cursor) == key {
		return false
	}
	_ = s.syncRepo.UpsertCheckpoint(ctx, &socialmodel.SyncCheckpoint{
		TenantUUID:      strings.ToLower(strings.TrimSpace(tenantUUID)),
		Domain:          "leads",
		Direction:       "push",
		Cursor:          key,
		SnapshotVersion: fmt.Sprintf("v%d", time.Now().UTC().Unix()),
		LastEventTime:   time.Now().UTC(),
	})
	_ = payload
	return true
}

func failureFlag(payload map[string]any) bool {
	if payload == nil {
		return false
	}
	v, ok := payload["force_fail"]
	if !ok {
		return false
	}
	b, ok := v.(bool)
	return ok && b
}

func ptrTime(t time.Time) *time.Time {
	return &t
}

func buildLeadExternalIdentityKey(tenantUUID, sourceAccountUUID, externalLeadID, externalWechatID string) string {
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	externalLeadID = strings.TrimSpace(externalLeadID)
	externalWechatID = strings.TrimSpace(externalWechatID)
	identity := externalLeadID
	if identity == "" {
		identity = externalWechatID
	}
	identity = strings.TrimSpace(identity)
	if tenantUUID == "" || identity == "" {
		return ""
	}
	return tenantUUID + ":" + identity
}

func (s *WeComSyncService) lookupLeadUUIDByIdentity(identityKey string) string {
	if s == nil {
		return ""
	}
	identityKey = strings.TrimSpace(identityKey)
	if identityKey == "" {
		return ""
	}
	s.identityMu.RLock()
	defer s.identityMu.RUnlock()
	return strings.TrimSpace(s.identityMap[identityKey])
}

func (s *WeComSyncService) rememberLeadUUIDByIdentity(identityKey, leadUUID string) {
	if s == nil {
		return
	}
	identityKey = strings.TrimSpace(identityKey)
	leadUUID = strings.TrimSpace(leadUUID)
	if identityKey == "" || leadUUID == "" {
		return
	}
	s.identityMu.Lock()
	defer s.identityMu.Unlock()
	if s.identityMap == nil {
		s.identityMap = map[string]string{}
	}
	s.identityMap[identityKey] = leadUUID
}

func stringSliceFromAny(v any) []string {
	switch vv := v.(type) {
	case []string:
		return vv
	case []any:
		out := make([]string, 0, len(vv))
		for _, item := range vv {
			out = append(out, strings.TrimSpace(fmt.Sprintf("%v", item)))
		}
		return out
	default:
		return []string{}
	}
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

func (s *WeComSyncService) findExistingLeadByExternalIdentityForSync(
	ctx context.Context,
	tx *gorm.DB,
	tenantUUID, sourceAccountUUID, externalLeadID, externalWechatID string,
) (*leadmodel.Lead, error) {
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	sourceAccountUUID = strings.ToLower(strings.TrimSpace(sourceAccountUUID))
	externalLeadID = strings.TrimSpace(externalLeadID)
	externalWechatID = strings.TrimSpace(externalWechatID)
	if tx == nil || tenantUUID == "" {
		return nil, nil
	}
	if externalLeadID == "" && externalWechatID == "" {
		return nil, nil
	}

	activityTable := leadmodel.LeadActivity{}.TableName()
	var leadUUID string
	dialect := strings.ToLower(strings.TrimSpace(tx.Dialector.Name()))
	switch dialect {
	case "postgres":
		query := tx.WithContext(ctx).
			Table(activityTable).
			Select("lead_uuid").
			Where("tenant_uuid = ? AND activity_type = ?", tenantUUID, leadmodel.LeadActivityTypeSyncTrace).
			Order("updated_at DESC")
		strictQuery := query.Where("payload ->> 'source_account_uuid' = ?", sourceAccountUUID)
		if externalLeadID != "" && externalWechatID != "" {
			strictQuery = strictQuery.Where("(payload ->> 'external_lead_id' = ? OR payload ->> 'external_wechat_id' = ?)", externalLeadID, externalWechatID)
			query = query.Where("(payload ->> 'external_lead_id' = ? OR payload ->> 'external_wechat_id' = ?)", externalLeadID, externalWechatID)
		} else if externalLeadID != "" {
			strictQuery = strictQuery.Where("payload ->> 'external_lead_id' = ?", externalLeadID)
			query = query.Where("payload ->> 'external_lead_id' = ?", externalLeadID)
		} else {
			strictQuery = strictQuery.Where("payload ->> 'external_wechat_id' = ?", externalWechatID)
			query = query.Where("payload ->> 'external_wechat_id' = ?", externalWechatID)
		}
		if err := strictQuery.Limit(1).Scan(&leadUUID).Error; err != nil {
			return nil, err
		}
		if strings.TrimSpace(leadUUID) == "" {
			if err := query.Limit(1).Scan(&leadUUID).Error; err != nil {
				return nil, err
			}
		}
	case "sqlite":
		query := tx.WithContext(ctx).
			Table(activityTable).
			Select("lead_uuid").
			Where("tenant_uuid = ? AND activity_type = ?", tenantUUID, leadmodel.LeadActivityTypeSyncTrace).
			Order("updated_at DESC")
		strictQuery := query.Where("json_extract(payload, '$.source_account_uuid') = ?", sourceAccountUUID)
		if externalLeadID != "" && externalWechatID != "" {
			strictQuery = strictQuery.Where("(json_extract(payload, '$.external_lead_id') = ? OR json_extract(payload, '$.external_wechat_id') = ?)", externalLeadID, externalWechatID)
			query = query.Where("(json_extract(payload, '$.external_lead_id') = ? OR json_extract(payload, '$.external_wechat_id') = ?)", externalLeadID, externalWechatID)
		} else if externalLeadID != "" {
			strictQuery = strictQuery.Where("json_extract(payload, '$.external_lead_id') = ?", externalLeadID)
			query = query.Where("json_extract(payload, '$.external_lead_id') = ?", externalLeadID)
		} else {
			strictQuery = strictQuery.Where("json_extract(payload, '$.external_wechat_id') = ?", externalWechatID)
			query = query.Where("json_extract(payload, '$.external_wechat_id') = ?", externalWechatID)
		}
		if err := strictQuery.Limit(1).Scan(&leadUUID).Error; err != nil {
			return nil, err
		}
		if strings.TrimSpace(leadUUID) == "" {
			if err := query.Limit(1).Scan(&leadUUID).Error; err != nil {
				return nil, err
			}
		}
	default:
	}

	if strings.TrimSpace(leadUUID) == "" {
		activities := make([]leadmodel.LeadActivity, 0, 200)
		if err := tx.WithContext(ctx).
			Where("tenant_uuid = ? AND activity_type = ?", tenantUUID, leadmodel.LeadActivityTypeSyncTrace).
			Order("updated_at DESC").
			Limit(200).
			Find(&activities).Error; err != nil {
			return nil, err
		}
		for _, activity := range activities {
			payload := activity.Payload
			if payload == nil {
				continue
			}
			if strings.TrimSpace(payloadString(payload, "source_account_uuid")) != sourceAccountUUID {
				continue
			}
			if externalLeadID != "" && strings.TrimSpace(payloadString(payload, "external_lead_id")) == externalLeadID {
				leadUUID = strings.TrimSpace(activity.LeadUUID)
				break
			}
			if externalWechatID != "" && strings.TrimSpace(payloadString(payload, "external_wechat_id")) == externalWechatID {
				leadUUID = strings.TrimSpace(activity.LeadUUID)
				break
			}
		}
		if strings.TrimSpace(leadUUID) == "" {
			for _, activity := range activities {
				payload := activity.Payload
				if payload == nil {
					continue
				}
				if externalLeadID != "" && strings.TrimSpace(payloadString(payload, "external_lead_id")) == externalLeadID {
					leadUUID = strings.TrimSpace(activity.LeadUUID)
					break
				}
				if externalWechatID != "" && strings.TrimSpace(payloadString(payload, "external_wechat_id")) == externalWechatID {
					leadUUID = strings.TrimSpace(activity.LeadUUID)
					break
				}
			}
		}
	}

	leadUUID = strings.TrimSpace(leadUUID)
	if leadUUID == "" {
		return nil, nil
	}
	var out leadmodel.Lead
	if err := tx.WithContext(ctx).
		Where("tenant_uuid = ? AND lead_uuid = ?", tenantUUID, leadUUID).
		First(&out).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
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
		"trace_id":               strings.TrimSpace(req.TraceID),
		"source_channel":         strings.ToLower(strings.TrimSpace(req.Channel)),
		"source_app_type":        strings.ToLower(strings.TrimSpace(req.AppType)),
		"source_account_uuid":    strings.TrimSpace(channelAccountUUID),
		"external_lead_id":       strings.TrimSpace(item.ExternalLeadID),
		"external_wechat_id":     strings.TrimSpace(item.WechatID),
		"owner_external_userid":  strings.TrimSpace(item.OwnerMemberUUID),
		"follow_external_userid": strings.TrimSpace(item.OwnerMemberUUID),
		"adder_external_userid":  strings.TrimSpace(item.AdderMemberUUID),
		"oper_userid":            strings.TrimSpace(item.AdderMemberUUID),
		"display_name":           strings.TrimSpace(item.DisplayName),
		"phone":                  strings.TrimSpace(item.Phone),
		"email":                  strings.ToLower(strings.TrimSpace(item.Email)),
		"dedup_exists_before":    existsBefore,
		"ingestion_entrypoint":   "wecom_sync",
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

func (s *WeComSyncService) resolveWeComWritebackCredentials(
	ctx context.Context,
	tenantUUID, channelAccountUUID string,
) (map[string]string, error) {
	if s == nil || s.taskRepo == nil || s.taskRepo.DB == nil {
		return nil, errors.New("account repository not configured")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	channelAccountUUID = strings.ToLower(strings.TrimSpace(channelAccountUUID))
	if tenantUUID == "" || channelAccountUUID == "" {
		return nil, errors.New("tenant_uuid and channel_account_uuid are required")
	}

	accountRepo := socialrepo.NewAccountRepository(s.taskRepo.DB)
	account, err := accountRepo.GetByAccountUUID(ctx, tenantUUID, channelAccountUUID)
	if err != nil {
		return nil, err
	}
	if account == nil {
		return nil, socialrepo.ErrAccountNotFound
	}
	if _, err := wecomauth.ResolveKind(strings.TrimSpace(account.ChannelCode), strings.TrimSpace(account.AppType)); err != nil {
		return nil, fmt.Errorf("unsupported wecom account identity: %s/%s", strings.TrimSpace(account.ChannelCode), strings.TrimSpace(account.AppType))
	}

	credentials := credentialsToStringMap(account.Credentials)
	openworkRepo := socialrepo.NewOpenWorkFoundationRepository(s.taskRepo.DB)
	platformRepo := socialrepo.NewChannelPlatformSettingRepository(s.taskRepo.DB)
	resolver := NewDefaultWeComLeadAdapterWithResolvers(accountRepo, openworkRepo, platformRepo)
	credentials = resolver.mergeDelegatedCredentials(ctx, tenantUUID, channelAccountUUID, credentials)

	return credentials, nil
}

func (s *WeComSyncService) resolveLeadExternalUserID(
	ctx context.Context,
	tenantUUID, channelAccountUUID, leadUUID string,
	cache map[string]string,
) (string, error) {
	if s == nil || s.taskRepo == nil || s.taskRepo.DB == nil {
		return "", errors.New("repository database is not initialized")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	channelAccountUUID = strings.ToLower(strings.TrimSpace(channelAccountUUID))
	leadUUID = strings.TrimSpace(leadUUID)
	if tenantUUID == "" || leadUUID == "" {
		return "", nil
	}

	cacheKey := channelAccountUUID + ":" + leadUUID
	if cache != nil {
		if hit := strings.TrimSpace(cache[cacheKey]); hit != "" {
			return hit, nil
		}
	}

	activities := make([]leadmodel.LeadActivity, 0, 16)
	if err := s.taskRepo.DB.WithContext(ctx).
		Where(
			"tenant_uuid = ? AND lead_uuid = ? AND activity_type = ?",
			tenantUUID,
			leadUUID,
			leadmodel.LeadActivityTypeSyncTrace,
		).
		Order("updated_at DESC").
		Limit(30).
		Find(&activities).Error; err != nil {
		return "", err
	}

	for _, activity := range activities {
		payload := activity.Payload
		if payload == nil {
			continue
		}
		if channelAccountUUID != "" && payloadString(payload, "source_account_uuid") != channelAccountUUID {
			continue
		}
		externalUserID := payloadString(payload, "external_lead_id")
		if externalUserID == "" {
			externalUserID = payloadString(payload, "external_wechat_id")
		}
		externalUserID = strings.TrimSpace(externalUserID)
		if externalUserID == "" {
			continue
		}
		if cache != nil {
			cache[cacheKey] = externalUserID
		}
		return externalUserID, nil
	}
	return "", nil
}

func (s *WeComSyncService) resolveLeadOwnerOperatorUserID(
	ctx context.Context,
	tenantUUID, channelAccountUUID string,
	fields map[string]any,
	cache map[string]string,
) (string, error) {
	if s == nil || s.taskRepo == nil || s.taskRepo.DB == nil {
		return "", errors.New("repository database is not initialized")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	channelAccountUUID = strings.ToLower(strings.TrimSpace(channelAccountUUID))
	if tenantUUID == "" {
		return "", nil
	}

	ownerUserUUID := strings.TrimSpace(fieldString(fields, "owner_user_uuid"))
	if ownerUserUUID == "" {
		ownerUserUUID = strings.TrimSpace(fieldString(fields, "assignee_user_uuid"))
	}
	if ownerUserUUID == "" {
		ownerUserUUID = strings.TrimSpace(fieldString(fields, "main_member_id"))
	}
	if ownerUserUUID == "" {
		return "", nil
	}

	cacheKey := channelAccountUUID + ":" + ownerUserUUID
	if cache != nil {
		if hit := strings.TrimSpace(cache[cacheKey]); hit != "" {
			return hit, nil
		}
	}

	var bound struct {
		ExternalMemberID string `gorm:"column:external_member_id"`
	}
	err := s.taskRepo.DB.WithContext(ctx).
		Table(orgmodel.MemberBinding{}.TableName()).
		Where("tenant_uuid = ? AND channel_account_uuid = ? AND main_member_id = ?", tenantUUID, channelAccountUUID, ownerUserUUID).
		Order("updated_at DESC").
		Limit(1).
		Select("external_member_id").
		Scan(&bound).Error
	if err != nil {
		return "", err
	}
	operatorUserID := strings.TrimSpace(bound.ExternalMemberID)
	if operatorUserID != "" {
		if cache != nil {
			cache[cacheKey] = operatorUserID
		}
		return operatorUserID, nil
	}
	return "", nil
}

func (s *WeComSyncService) resolveLeadOwnerUserUUIDFromExternalMember(
	ctx context.Context,
	tenantUUID, channelAccountUUID, externalMemberID string,
	cache map[string]string,
) (string, error) {
	if s == nil || s.taskRepo == nil || s.taskRepo.DB == nil {
		return "", errors.New("repository database is not initialized")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	channelAccountUUID = strings.ToLower(strings.TrimSpace(channelAccountUUID))
	externalMemberID = strings.TrimSpace(externalMemberID)
	if tenantUUID == "" || externalMemberID == "" {
		return "", nil
	}

	cacheKey := channelAccountUUID + ":" + externalMemberID
	if cache != nil {
		if hit, ok := cache[cacheKey]; ok {
			return strings.TrimSpace(hit), nil
		}
	}

	var bound struct {
		MainMemberID string `gorm:"column:main_member_id"`
	}
	err := s.taskRepo.DB.WithContext(ctx).
		Table(orgmodel.MemberBinding{}.TableName()).
		Where("tenant_uuid = ? AND channel_account_uuid = ? AND external_member_id = ?", tenantUUID, channelAccountUUID, externalMemberID).
		Order("updated_at DESC").
		Limit(1).
		Select("main_member_id").
		Scan(&bound).Error
	if err != nil {
		return "", err
	}
	ownerUserUUID := strings.TrimSpace(bound.MainMemberID)
	if cache != nil {
		cache[cacheKey] = ownerUserUUID
	}
	return ownerUserUUID, nil
}

func fieldString(input map[string]any, key string) string {
	if input == nil {
		return ""
	}
	raw, ok := input[key]
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
