package lead_capture

import (
	"context"
	"errors"
	"strings"
	"time"

	leadmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/lead_capture"
	leadrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/lead_capture"
	leadobs "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/observability/lead_capture"
)

type WeComSyncService struct {
	taskRepo        *leadrepo.LeadSyncTaskRepository
	metrics         *leadobs.Metrics
	providerAdapter SyncTaskProviderAdapter
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
	if s.metrics != nil {
		s.metrics.RecordSyncTask(submit.Provider, submit.Status)
	}
	return created, nil
}

func (s *WeComSyncService) ListSyncTasks(ctx context.Context, tenantUUID, channelAccountUUID string, limit int) ([]*leadmodel.LeadSyncTask, error) {
	if s == nil || s.taskRepo == nil {
		return nil, errors.New("wecom sync service unavailable")
	}
	return s.taskRepo.ListByAccount(ctx, tenantUUID, channelAccountUUID, limit)
}
