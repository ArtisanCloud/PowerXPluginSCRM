package lead_capture

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	leadmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/lead_capture"
	leadrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/lead_capture"
	leadobs "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/observability/lead_capture"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type WeComSyncService struct {
	taskRepo        *leadrepo.LeadSyncTaskRepository
	leadRepo        *leadrepo.LeadRepository
	leadService     *LeadService
	metrics         *leadobs.Metrics
	providerAdapter SyncTaskProviderAdapter
	leadAdapter     WeComLeadAdapter
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
	return &WeComSyncService{taskRepo: taskRepo, metrics: metrics, providerAdapter: providerAdapter}
}

func (s *WeComSyncService) WithLeadIngestion(leadRepo *leadrepo.LeadRepository, adapter WeComLeadAdapter) *WeComSyncService {
	if s == nil {
		return s
	}
	s.leadRepo = leadRepo
	s.leadAdapter = adapter
	return s
}

func (s *WeComSyncService) WithLeadService(leadService *LeadService) *WeComSyncService {
	if s == nil {
		return s
	}
	s.leadService = leadService
	return s
}

type SyncIngestStats struct {
	Total   int
	Created int
	Updated int
	Merged  int
}

func (s *WeComSyncService) TriggerSync(ctx context.Context, req TriggerSyncRequest) (*leadmodel.LeadSyncTask, error) {
	if s == nil || s.taskRepo == nil {
		return nil, errors.New("wecom sync service unavailable")
	}
	req.TenantUUID = strings.ToLower(strings.TrimSpace(req.TenantUUID))
	req.Channel = strings.ToLower(strings.TrimSpace(req.Channel))
	req.AppType = strings.ToLower(strings.TrimSpace(req.AppType))
	if req.TenantUUID == "" {
		return nil, errors.New("tenant_uuid is required")
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
		return nil, err
	}

	adapter := s.providerAdapter
	if adapter == nil {
		adapter = NewDefaultSyncTaskProviderAdapter(nil, nil)
	}
	submit := adapter.SubmitSyncTask(ctx, req, accountUUID)
	if submit.Provider == "" {
		submit.Provider = leadmodel.LeadSyncTaskProviderLocalFallback
	}
	if submit.Status == "" {
		submit.Status = "queued"
	}

	now := time.Now().UTC()
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
		StartedAt:            &now,
	}
	created, err := s.taskRepo.CreateTask(ctx, task)
	if err != nil {
		if s.metrics != nil {
			s.metrics.RecordSyncTask(submit.Provider, "create_failed")
		}
		return nil, err
	}

	if submit.Provider == leadmodel.LeadSyncTaskProviderLocalFallback {
		if err := s.updateTaskRunning(ctx, created); err != nil {
			return nil, err
		}
		stats, runErr := s.runLocalSyncIngestion(ctx, req, accountUUID)
		if runErr != nil {
			failErr := s.taskRepo.UpdateStatus(ctx, created.TenantUUID, created.TaskUUID, "failed", map[string]any{
				"error_message": strings.TrimSpace(runErr.Error()),
				"finished_at":   time.Now().UTC(),
			})
			if failErr != nil {
				return nil, failErr
			}
			created.Status = "failed"
			created.ErrorMessage = strings.TrimSpace(runErr.Error())
		} else {
			finishAt := time.Now().UTC()
			okErr := s.taskRepo.UpdateStatus(ctx, created.TenantUUID, created.TaskUUID, "success", map[string]any{
				"stats_total":   stats.Total,
				"stats_created": stats.Created,
				"stats_updated": stats.Updated,
				"stats_merged":  stats.Merged,
				"finished_at":   finishAt,
			})
			if okErr != nil {
				return nil, okErr
			}
			created.Status = "success"
			created.StatsTotal = stats.Total
			created.StatsCreated = stats.Created
			created.StatsUpdated = stats.Updated
			created.StatsMerged = stats.Merged
			created.FinishedAt = &finishAt
		}
	}

	if s.metrics != nil {
		s.metrics.RecordSyncTask(submit.Provider, created.Status)
	}
	return created, nil
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
		"trigger_type":  "retry",
		"error_message": "",
		"error_code":    "",
		"stats_total":   0,
		"stats_created": 0,
		"stats_updated": 0,
		"stats_merged":  0,
		"finished_at":   nil,
		"started_at":    time.Now().UTC(),
		"updated_at":    time.Now().UTC(),
	})
}

func (s *WeComSyncService) updateTaskRunning(ctx context.Context, task *leadmodel.LeadSyncTask) error {
	if task == nil {
		return errors.New("task is required")
	}
	startedAt := time.Now().UTC()
	if err := s.taskRepo.UpdateStatus(ctx, task.TenantUUID, task.TaskUUID, "running", map[string]any{
		"started_at": startedAt,
	}); err != nil {
		return err
	}
	task.Status = "running"
	task.StartedAt = &startedAt
	return nil
}

func (s *WeComSyncService) runLocalSyncIngestion(ctx context.Context, req TriggerSyncRequest, channelAccountUUID string) (SyncIngestStats, error) {
	if s == nil || s.leadAdapter == nil || s.leadRepo == nil {
		return SyncIngestStats{}, nil
	}
	items, err := s.leadAdapter.FetchLeads(ctx, req, channelAccountUUID)
	if err != nil {
		return SyncIngestStats{}, err
	}
	stats := SyncIngestStats{Total: len(items)}
	if s.leadService != nil {
		for _, raw := range items {
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
			if existsBefore {
				stats.Updated++
				if created != nil && created.HasMerge {
					stats.Merged++
				}
			} else {
				stats.Created++
			}
		}
		return stats, nil
	}

	err = s.leadRepo.WithTenantTx(ctx, req.TenantUUID, func(tx *gorm.DB) error {
		for _, raw := range items {
			item := normalizeWeComLeadRecord(raw)
			if item.DisplayName == "" && item.Phone == "" && item.Email == "" {
				continue
			}
			existing, matchErr := s.findExistingLeadForSync(ctx, tx, req.TenantUUID, item.Phone, item.Email)
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
		}
		return nil
	})
	if err != nil {
		return SyncIngestStats{}, err
	}
	return stats, nil
}

func (s *WeComSyncService) findExistingLeadForSync(ctx context.Context, tx *gorm.DB, tenantUUID, phone, email string) (*leadmodel.Lead, error) {
	phone = strings.TrimSpace(phone)
	email = strings.ToLower(strings.TrimSpace(email))
	if phone == "" && email == "" {
		return nil, nil
	}
	var out leadmodel.Lead
	query := tx.WithContext(ctx).Where("tenant_uuid = ?", tenantUUID)
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
