package social_channel_governance

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	model "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/social_channel_governance"
	socialrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/social_channel_governance"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/logger"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/websocket/bus"
	"gorm.io/datatypes"
)

type SyncJobService struct {
	repo              *socialrepo.SyncFoundationRepository
	idempotency       *IdempotencyService
	scheduler         *SyncScheduler
	capabilityService *CapabilityService
	tagSyncService    *TagSyncService
	customerTagSvc    *CustomerTagBindingService
}

type TagSyncProgressEvent struct {
	JobUUID            string         `json:"job_uuid"`
	Domain             string         `json:"domain"`
	Direction          string         `json:"direction"`
	Mode               string         `json:"mode"`
	Status             string         `json:"status"`
	AttemptNo          int            `json:"attempt_no"`
	ErrorCode          string         `json:"error_code,omitempty"`
	ErrorMessage       string         `json:"error_message,omitempty"`
	ChannelAccountUUID string         `json:"channel_account_uuid,omitempty"`
	CapabilityStatus   string         `json:"capability_status,omitempty"`
	ResultSummary      map[string]any `json:"result_summary,omitempty"`
	UpdatedAt          string         `json:"updated_at"`
}

type CreateSyncJobInput struct {
	TenantUUID string
	Channel    string
	AppType    string
	Domain     string
	Direction  string
	Mode       string
	Payload    map[string]any
}

func NewSyncJobService(
	repo *socialrepo.SyncFoundationRepository,
	idempotency *IdempotencyService,
	scheduler *SyncScheduler,
	capabilityService *CapabilityService,
	tagSyncService ...*TagSyncService,
) *SyncJobService {
	var tagSvc *TagSyncService
	if len(tagSyncService) > 0 {
		tagSvc = tagSyncService[0]
	}
	return &SyncJobService{
		repo:              repo,
		idempotency:       idempotency,
		scheduler:         scheduler,
		capabilityService: capabilityService,
		tagSyncService:    tagSvc,
	}
}

func (s *SyncJobService) WithCustomerTagBindingService(customerTagSvc *CustomerTagBindingService) *SyncJobService {
	if s == nil {
		return s
	}
	s.customerTagSvc = customerTagSvc
	return s
}

func (s *SyncJobService) Create(ctx context.Context, in CreateSyncJobInput) (*model.SyncJob, bool, error) {
	if s == nil || s.repo == nil {
		return nil, false, nil
	}
	in.TenantUUID = strings.TrimSpace(strings.ToLower(in.TenantUUID))
	in.Domain = strings.TrimSpace(in.Domain)
	in.Direction = strings.TrimSpace(in.Direction)
	in.Mode = strings.TrimSpace(in.Mode)
	if in.Direction == "" {
		in.Direction = "pull"
	}
	if in.Mode == "" {
		in.Mode = "incremental"
	}
	payload := datatypes.JSONMap(in.Payload)
	if payload == nil {
		payload = datatypes.JSONMap{}
	}
	matrix := map[string]string{}
	if s.capabilityService != nil {
		matrix = s.capabilityService.Matrix(ctx, in.TenantUUID, in.Channel, in.AppType)
	}
	capabilityStatus := "supported"
	if v := strings.TrimSpace(matrix[in.Domain]); v != "" {
		capabilityStatus = v
	}
	idempotencyKey := ""
	if s.idempotency != nil {
		idempotencyKey = s.idempotency.BuildKey(in.TenantUUID, in.Domain, in.Direction, in.Mode, in.Channel, in.AppType)
	}
	job := &model.SyncJob{
		TenantUUID:      in.TenantUUID,
		Domain:          in.Domain,
		Direction:       in.Direction,
		Mode:            in.Mode,
		Status:          "pending",
		IdempotencyKey:  idempotencyKey,
		MaxAttempts:     3,
		Payload:         payload,
		CapabilityState: capabilityStatus,
	}
	createdJob, created, err := s.repo.CreateJob(ctx, job)
	if err != nil {
		return nil, false, err
	}
	s.publishTagSyncProgress(in.TenantUUID, createdJob, "pending", "", "", nil)
	// Manual retrigger should create a new job after the previous one reaches terminal status.
	// Keep idempotency for in-flight jobs, but allow replay on failed/success/dead_letter.
	if !created && createdJob != nil {
		status := strings.TrimSpace(strings.ToLower(createdJob.Status))
		if status == model.SyncJobStatusFailed || status == model.SyncJobStatusSuccess || status == model.SyncJobStatusDeadLetter {
			retryJob := *job
			retryJob.JobUUID = ""
			retryJob.Status = "pending"
			retryJob.AttemptNo = 0
			retryJob.StartedAt = nil
			retryJob.FinishedAt = nil
			retryJob.CreatedAt = time.Time{}
			retryJob.UpdatedAt = time.Time{}
			retryJob.ErrorCode = ""
			retryJob.ErrorMessage = ""
			if s.idempotency != nil {
				retryJob.IdempotencyKey = s.idempotency.BuildKey(job.IdempotencyKey, time.Now().UTC().Format(time.RFC3339Nano))
			} else {
				retryJob.IdempotencyKey = fmt.Sprintf("%s_%d", strings.TrimSpace(job.IdempotencyKey), time.Now().UTC().UnixNano())
			}
			recreatedJob, recreated, recreateErr := s.repo.CreateJob(ctx, &retryJob)
			if recreateErr == nil {
				s.publishTagSyncProgress(in.TenantUUID, recreatedJob, "pending", "", "", nil)
			}
			return recreatedJob, recreated, recreateErr
		}
	}
	return createdJob, created, nil
}

func (s *SyncJobService) List(ctx context.Context, tenantUUID, domain, status string, limit int) ([]model.SyncJob, error) {
	if s == nil || s.repo == nil {
		return []model.SyncJob{}, nil
	}
	items, err := s.repo.ListJobs(ctx, tenantUUID, domain, status, limit)
	if err != nil {
		return nil, err
	}
	for i := range items {
		payload := items[i].Payload
		if payload == nil {
			continue
		}
		raw, ok := payload["result_summary"]
		if !ok || raw == nil {
			continue
		}
		if m, ok := raw.(map[string]any); ok {
			items[i].ResultSummary = datatypes.JSONMap(m)
		}
	}
	return items, nil
}

func (s *SyncJobService) ClearTerminalJobs(ctx context.Context, tenantUUID, domain string) (int64, error) {
	return s.ClearJobs(ctx, tenantUUID, domain, false)
}

func (s *SyncJobService) ClearJobs(ctx context.Context, tenantUUID, domain string, includeInFlight bool) (int64, error) {
	if s == nil || s.repo == nil {
		return 0, errors.New("sync job service unavailable")
	}
	tenantUUID = strings.TrimSpace(strings.ToLower(tenantUUID))
	domain = strings.TrimSpace(strings.ToLower(domain))
	if tenantUUID == "" {
		return 0, errors.New("tenant_uuid is required")
	}
	if domain == "" {
		return 0, errors.New("domain is required")
	}
	return s.repo.DeleteJobsByDomain(ctx, tenantUUID, domain, includeInFlight)
}

func (s *SyncJobService) Execute(ctx context.Context, job *model.SyncJob) error {
	if s == nil || s.repo == nil {
		return errors.New("sync job service unavailable")
	}
	if job == nil {
		return errors.New("job is required")
	}
	tenantUUID := strings.TrimSpace(strings.ToLower(job.TenantUUID))
	if tenantUUID == "" {
		return errors.New("tenant_uuid is required")
	}
	domain := strings.TrimSpace(strings.ToLower(job.Domain))
	direction := strings.TrimSpace(strings.ToLower(job.Direction))
	if direction == "" {
		direction = "pull"
	}

	if err := s.repo.UpdateJobStatus(ctx, tenantUUID, job.JobUUID, model.SyncJobStatusRunning, "", "", job.AttemptNo); err != nil {
		return err
	}
	s.publishTagSyncProgress(tenantUUID, job, model.SyncJobStatusRunning, "", "", nil)

	capabilityStatus := strings.TrimSpace(strings.ToLower(job.CapabilityState))
	if capabilityStatus == model.CapabilityStatusNotSupported {
		// Tags has real execution path; do not hard-fail by capability precheck.
		if domain != model.SyncDomainTags {
			errMsg := "capability not supported for this channel/app"
			_ = s.repo.UpdateJobStatus(ctx, tenantUUID, job.JobUUID, model.SyncJobStatusFailed, "CAPABILITY_NOT_SUPPORTED", errMsg, job.AttemptNo)
			s.publishTagSyncProgress(tenantUUID, job, model.SyncJobStatusFailed, "CAPABILITY_NOT_SUPPORTED", errMsg, nil)
			return nil
		}
	}

	var (
		execErr error
		summary *TagSyncResult
	)
	switch domain {
	case model.SyncDomainTags:
		summary, execErr = s.executeTagJob(ctx, tenantUUID, direction, job.Payload)
	default:
		execErr = fmt.Errorf("domain %s execution not implemented", domain)
	}
	if execErr != nil {
		errMsg := strings.TrimSpace(execErr.Error())
		_ = s.repo.UpdateJobStatus(ctx, tenantUUID, job.JobUUID, model.SyncJobStatusFailed, "EXEC_FAILED", errMsg, job.AttemptNo)
		s.publishTagSyncProgress(tenantUUID, job, model.SyncJobStatusFailed, "EXEC_FAILED", errMsg, summary)
		logger.WithFields(logger.Fields{
			"component":            "tag_sync_job",
			"tenant_uuid":          tenantUUID,
			"job_uuid":             strings.TrimSpace(job.JobUUID),
			"domain":               strings.TrimSpace(strings.ToLower(job.Domain)),
			"direction":            strings.TrimSpace(strings.ToLower(job.Direction)),
			"mode":                 strings.TrimSpace(strings.ToLower(job.Mode)),
			"status":               model.SyncJobStatusFailed,
			"error_code":           "EXEC_FAILED",
			"error_message":        errMsg,
			"channel_account_uuid": strings.TrimSpace(fieldString(job.Payload, "channel_account_uuid")),
		}).Warn("tag sync job execution failed")
		return nil
	}
	if summary != nil {
		_ = s.repo.UpdateJobResultSummary(ctx, tenantUUID, job.JobUUID, map[string]any{
			"pulled":           summary.Pulled,
			"pushed":           summary.Pushed,
			"created":          summary.Created,
			"updated":          summary.Updated,
			"deleted":          summary.Deleted,
			"conflicts":        summary.Conflicts,
			"snapshot_version": strings.TrimSpace(summary.SnapshotVersion),
			"cursor":           strings.TrimSpace(summary.Cursor),
		})
	}
	if err := s.repo.UpdateJobStatus(ctx, tenantUUID, job.JobUUID, model.SyncJobStatusSuccess, "", "", job.AttemptNo); err != nil {
		return err
	}
	s.publishTagSyncProgress(tenantUUID, job, model.SyncJobStatusSuccess, "", "", summary)
	logger.WithFields(logger.Fields{
		"component":            "tag_sync_job",
		"tenant_uuid":          tenantUUID,
		"job_uuid":             strings.TrimSpace(job.JobUUID),
		"domain":               strings.TrimSpace(strings.ToLower(job.Domain)),
		"direction":            strings.TrimSpace(strings.ToLower(job.Direction)),
		"mode":                 strings.TrimSpace(strings.ToLower(job.Mode)),
		"status":               model.SyncJobStatusSuccess,
		"channel_account_uuid": strings.TrimSpace(fieldString(job.Payload, "channel_account_uuid")),
	}).Info("tag sync job execution succeeded")
	return nil
}

func (s *SyncJobService) publishTagSyncProgress(tenantUUID string, job *model.SyncJob, status, errorCode, errorMessage string, summary *TagSyncResult) {
	if s == nil || job == nil {
		return
	}
	if strings.TrimSpace(strings.ToLower(job.Domain)) != model.SyncDomainTags {
		return
	}
	tenantUUID = strings.TrimSpace(strings.ToLower(tenantUUID))
	if tenantUUID == "" {
		return
	}
	status = strings.TrimSpace(strings.ToLower(status))
	if status == "" {
		status = strings.TrimSpace(strings.ToLower(job.Status))
	}
	channelAccountUUID := strings.TrimSpace(fieldString(job.Payload, "channel_account_uuid"))
	payload := TagSyncProgressEvent{
		JobUUID:            strings.TrimSpace(job.JobUUID),
		Domain:             strings.TrimSpace(strings.ToLower(job.Domain)),
		Direction:          strings.TrimSpace(strings.ToLower(job.Direction)),
		Mode:               strings.TrimSpace(strings.ToLower(job.Mode)),
		Status:             status,
		AttemptNo:          job.AttemptNo,
		ErrorCode:          strings.TrimSpace(errorCode),
		ErrorMessage:       strings.TrimSpace(errorMessage),
		ChannelAccountUUID: channelAccountUUID,
		CapabilityStatus:   strings.TrimSpace(strings.ToLower(job.CapabilityState)),
		UpdatedAt:          time.Now().UTC().Format(time.RFC3339Nano),
	}
	if summary != nil {
		payload.ResultSummary = map[string]any{
			"pulled":           summary.Pulled,
			"pushed":           summary.Pushed,
			"created":          summary.Created,
			"updated":          summary.Updated,
			"deleted":          summary.Deleted,
			"conflicts":        summary.Conflicts,
			"snapshot_version": strings.TrimSpace(summary.SnapshotVersion),
			"cursor":           strings.TrimSpace(summary.Cursor),
		}
	}
	bus.DefaultHub.Publish(tenantUUID, bus.TopicTagSyncProgress, payload, "")
	bus.DefaultHub.Publish(tenantUUID, bus.TopicTagSyncProgressV1, payload, "")
}

func (s *SyncJobService) executeTagJob(ctx context.Context, tenantUUID, direction string, payload datatypes.JSONMap) (*TagSyncResult, error) {
	if s == nil || s.tagSyncService == nil {
		return nil, errors.New("tag sync service unavailable")
	}
	channelAccountUUID := strings.TrimSpace(fieldString(payload, "channel_account_uuid"))
	if channelAccountUUID == "" {
		return nil, errors.New("channel_account_uuid is required for tag sync job")
	}
	localTags := parseTagRecords(payload["local_tags"])
	customerTagOperations := parseCustomerTagOperations(payload["customer_tag_operations"])
	tagOperations := parseTagOperations(payload["tag_operations"])
	cursor := strings.TrimSpace(fieldString(payload, "cursor"))
	if direction == "push" {
		res := &TagSyncResult{Cursor: cursor}
		if len(customerTagOperations) > 0 {
			bindingRes, err := s.tagSyncService.PushCustomerTagBindingsByChannel(ctx, tenantUUID, channelAccountUUID, customerTagOperations, cursor)
			if err != nil {
				return nil, err
			}
			res = mergeTagSyncResult(res, bindingRes)
		}
		if len(tagOperations) > 0 {
			tagRes, err := s.tagSyncService.PushTagOperationsByChannel(ctx, tenantUUID, channelAccountUUID, tagOperations, res.Cursor)
			if err != nil {
				return nil, err
			}
			res = mergeTagSyncResult(res, tagRes)
		}
		if len(customerTagOperations) > 0 || len(tagOperations) > 0 {
			// pushback 完成后立即回拉标签清单并落库，保证前端标签列表可见最新远端状态。
			pullRes, pullErr := s.tagSyncService.SyncRemoteToLocalByChannel(ctx, tenantUUID, channelAccountUUID, nil, res.Cursor)
			if pullErr != nil {
				return nil, pullErr
			}
			res = mergeTagSyncResult(res, pullRes)
			// pushback 后刷新一次本地客户标签快照，避免前端仍展示旧关系。
			if s.customerTagSvc != nil {
				if _, refreshErr := s.customerTagSvc.RefreshSnapshotByChannel(ctx, tenantUUID, channelAccountUUID); refreshErr != nil {
					return nil, refreshErr
				}
			}
			return res, nil
		}
		return s.tagSyncService.SyncLocalToRemoteByChannel(ctx, tenantUUID, channelAccountUUID, localTags, cursor)
	}
	res, err := s.tagSyncService.SyncRemoteToLocalByChannel(ctx, tenantUUID, channelAccountUUID, localTags, cursor)
	if err != nil {
		return nil, err
	}
	if s.customerTagSvc != nil {
		if _, refreshErr := s.customerTagSvc.RefreshSnapshotByChannel(ctx, tenantUUID, channelAccountUUID); refreshErr != nil {
			return nil, refreshErr
		}
	}
	return res, nil
}

func mergeTagSyncResult(base *TagSyncResult, current *TagSyncResult) *TagSyncResult {
	if base == nil && current == nil {
		return &TagSyncResult{}
	}
	if base == nil {
		return current
	}
	if current == nil {
		return base
	}
	base.Pulled += current.Pulled
	base.Pushed += current.Pushed
	base.Conflicts += current.Conflicts
	base.Created += current.Created
	base.Updated += current.Updated
	base.Deleted += current.Deleted
	if strings.TrimSpace(current.SnapshotVersion) != "" {
		base.SnapshotVersion = strings.TrimSpace(current.SnapshotVersion)
	}
	if strings.TrimSpace(current.Cursor) != "" {
		base.Cursor = strings.TrimSpace(current.Cursor)
	}
	return base
}

func parseTagRecords(raw any) []TagRecord {
	list, ok := raw.([]any)
	if !ok || len(list) == 0 {
		return []TagRecord{}
	}
	out := make([]TagRecord, 0, len(list))
	for _, item := range list {
		m, ok := item.(map[string]any)
		if !ok || m == nil {
			continue
		}
		out = append(out, TagRecord{
			TagID:     strings.TrimSpace(fieldString(m, "tag_id")),
			GroupID:   strings.TrimSpace(fieldString(m, "group_id")),
			GroupName: strings.TrimSpace(fieldString(m, "group_name")),
			Name:      strings.TrimSpace(fieldString(m, "name")),
			Version:   strings.TrimSpace(fieldString(m, "version")),
		})
	}
	return out
}

func parseCustomerTagOperations(raw any) []CustomerTagOperation {
	list, ok := raw.([]any)
	if !ok || len(list) == 0 {
		return []CustomerTagOperation{}
	}
	out := make([]CustomerTagOperation, 0, len(list))
	for _, item := range list {
		m, ok := item.(map[string]any)
		if !ok || m == nil {
			continue
		}
		addTags := parseStringList(m["add_tag"])
		removeTags := parseStringList(m["remove_tag"])
		out = append(out, CustomerTagOperation{
			ExternalUserID: strings.TrimSpace(fieldString(m, "external_userid")),
			UserID:         strings.TrimSpace(fieldString(m, "userid")),
			AddTag:         addTags,
			RemoveTag:      removeTags,
		})
	}
	return out
}

func parseTagOperations(raw any) []TagOperation {
	list, ok := raw.([]any)
	if !ok || len(list) == 0 {
		return []TagOperation{}
	}
	out := make([]TagOperation, 0, len(list))
	for _, item := range list {
		m, ok := item.(map[string]any)
		if !ok || m == nil {
			continue
		}
		operation := strings.TrimSpace(fieldString(m, "operation"))
		if operation == "" {
			operation = strings.TrimSpace(fieldString(m, "op"))
		}
		if operation == "" {
			operation = strings.TrimSpace(fieldString(m, "action"))
		}
		out = append(out, TagOperation{
			Operation: strings.ToLower(operation),
			TagID:     strings.TrimSpace(fieldString(m, "tag_id")),
			GroupID:   strings.TrimSpace(fieldString(m, "group_id")),
			GroupName: strings.TrimSpace(fieldString(m, "group_name")),
			Name:      strings.TrimSpace(fieldString(m, "name")),
		})
	}
	return out
}

func parseStringList(raw any) []string {
	if raw == nil {
		return []string{}
	}
	list, ok := raw.([]any)
	if !ok || len(list) == 0 {
		return []string{}
	}
	out := make([]string, 0, len(list))
	for _, item := range list {
		if item == nil {
			continue
		}
		out = append(out, strings.TrimSpace(fmt.Sprintf("%v", item)))
	}
	return out
}

func fieldString(input map[string]any, key string) string {
	if input == nil {
		return ""
	}
	raw, ok := input[key]
	if !ok || raw == nil {
		return ""
	}
	if s, ok := raw.(string); ok {
		return strings.TrimSpace(s)
	}
	return strings.TrimSpace(fmt.Sprintf("%v", raw))
}
