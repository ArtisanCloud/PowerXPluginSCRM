package acquisition

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	plcache "github.com/ArtisanCloud/PowerLibs/v3/cache"
	openwork "github.com/ArtisanCloud/PowerWeChat/v3/src/openWork"
	openworksuit "github.com/ArtisanCloud/PowerWeChat/v3/src/openWork/suitAuth"
	work "github.com/ArtisanCloud/PowerWeChat/v3/src/work"
	pwgroupchatreq "github.com/ArtisanCloud/PowerWeChat/v3/src/work/externalContact/groupChat/request"
	acqmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/models/acquisition"
	acqrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/repository/acquisition"
	leadmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/lead_capture"
	leadrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/lead_capture"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/logger"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/wecomauth"
	"github.com/google/uuid"
	"gorm.io/datatypes"
)

var ErrGroupChatSyncServiceNotReady = errors.New("group chat sync service not ready")

var ErrUnsupportedChannelAccountType = errors.New("unsupported channel account type")

var ErrGroupChatTaskNotFound = errors.New("group chat sync task not found")

type GroupChatSyncRequest struct {
	TenantUUID         string
	ChannelAccountUUID string
	Mode               string
}

type GroupChatAccountProfile struct {
	ChannelCode string
	AppType     string
}

type GroupChatSyncResult struct {
	SyncedCount         int
	ChannelAccountUUID  string
	ResolvedChannelCode string
	ResolvedAppType     string
}

type GroupChatWebhookEvent struct {
	TenantUUID         string
	ChannelAccountUUID string
	ChatID             string
	Name               string
	OwnerUserID        string
	MemberCount        int
	SourceConfigID     string
	OccurredAt         time.Time
	Payload            map[string]any
}

type GroupChatSyncTask struct {
	TaskUUID            string `json:"task_uuid"`
	JobUUID             string `json:"job_uuid"`
	Status              string `json:"status"`
	Mode                string `json:"mode"`
	ChannelAccountUUID  string `json:"channel_account_uuid"`
	ResolvedChannelCode string `json:"resolved_channel_code,omitempty"`
	ResolvedAppType     string `json:"resolved_app_type,omitempty"`
	ProgressTotal       int    `json:"progress_total"`
	ProgressCurrent     int    `json:"progress_current"`
	ProgressPercent     int    `json:"progress_percent"`
	StatsTotal          int    `json:"stats_total"`
	StatsCreated        int    `json:"stats_created"`
	StatsUpdated        int    `json:"stats_updated"`
	ErrorMessage        string `json:"error_message,omitempty"`
	CreatedAt           string `json:"created_at,omitempty"`
	StartedAt           string `json:"started_at,omitempty"`
	FinishedAt          string `json:"finished_at,omitempty"`
}

type GroupCustomerTimelineEvent struct {
	EventType   string         `json:"event_type"`
	EventTime   string         `json:"event_time,omitempty"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Source      string         `json:"source"`
	Payload     map[string]any `json:"payload,omitempty"`
}

type GroupCustomerFollowupAggregation struct {
	Status       string                       `json:"status"`
	Reason       string                       `json:"reason,omitempty"`
	ChatID       string                       `json:"chat_id"`
	ExternalUser string                       `json:"external_userid"`
	Summary      map[string]any               `json:"summary,omitempty"`
	Items        []map[string]any             `json:"items,omitempty"`
	Meta         map[string]any               `json:"meta,omitempty"`
	TimelineHint []GroupCustomerTimelineEvent `json:"timeline_hint,omitempty"`
}

type GroupCustomerRelatedChat struct {
	ChatID         string `json:"chat_id"`
	ChatName       string `json:"chat_name,omitempty"`
	OwnerUserID    string `json:"owner_userid,omitempty"`
	SourceConfigID string `json:"source_config_id,omitempty"`
	JoinTime       string `json:"join_time,omitempty"`
	JoinScene      int    `json:"join_scene,omitempty"`
	JoinSceneText  string `json:"join_scene_text,omitempty"`
	InvitorUserID  string `json:"invitor_userid,omitempty"`
	UpdatedAt      string `json:"updated_at,omitempty"`
}

type GroupChatSyncService struct {
	repo            acqrepo.GroupChatSnapshotRepository
	taskRepo        *leadrepo.LeadSyncTaskRepository
	accountResolver GroupChatAccountResolver
	providerFactory *GroupChatProviderFactory
}

type GroupChatAccountResolver interface {
	ResolveDefaultChannelAccount(ctx context.Context, tenantUUID, channel, appType string) (string, error)
	GetChannelAccount(ctx context.Context, tenantUUID, accountUUID string) (*GroupChatAccountProfile, error)
	GetChannelAccountCredentials(ctx context.Context, tenantUUID, accountUUID string) (map[string]string, error)
}

func NewGroupChatSyncService(repo acqrepo.GroupChatSnapshotRepository, taskRepo *leadrepo.LeadSyncTaskRepository, resolver ...GroupChatAccountResolver) *GroupChatSyncService {
	factory := NewGroupChatProviderFactory()
	_ = factory.Register("wechat", "wecom", buildWeComSelfBuiltWorkApp)
	_ = factory.Register("wechat", "openwork", buildWeComOpenWorkApp)
	svc := &GroupChatSyncService{repo: repo, taskRepo: taskRepo, providerFactory: factory}
	if len(resolver) > 0 {
		svc.accountResolver = resolver[0]
	}
	return svc
}

func (s *GroupChatSyncService) TriggerSyncAsync(ctx context.Context, req GroupChatSyncRequest) (*GroupChatSyncTask, error) {
	if s == nil || s.repo == nil || s.taskRepo == nil {
		return nil, ErrGroupChatSyncServiceNotReady
	}
	mode := normalizeSyncMode(req.Mode)
	resolved, err := s.resolveRuntime(ctx, req.TenantUUID, req.ChannelAccountUUID)
	if err != nil {
		return nil, err
	}

	task := &leadmodel.LeadSyncTask{
		TenantUUID:           resolved.tenantUUID,
		Channel:              "wechat",
		AppType:              resolved.appType,
		ChannelAccountUUID:   resolved.accountUUID,
		AccountResolveSource: resolved.resolveSource,
		TaskProvider:         leadmodel.LeadSyncTaskProviderLocalFallback,
		TriggerType:          "group_chat_sync",
		Status:               "queued",
	}
	created, err := s.taskRepo.CreateTask(ctx, task)
	if err != nil {
		return nil, err
	}

	go s.executeTask(context.Background(), created.TaskUUID, resolved, mode)
	return mapTask(created, mode, resolved.channelCode, resolved.appType, ""), nil
}

func (s *GroupChatSyncService) ListTasks(ctx context.Context, tenantUUID, status string, limit int) ([]*GroupChatSyncTask, error) {
	if s == nil || s.taskRepo == nil {
		return []*GroupChatSyncTask{}, nil
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	status = strings.ToLower(strings.TrimSpace(status))
	if tenantUUID == "" {
		return []*GroupChatSyncTask{}, nil
	}
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.taskRepo.ListByFilter(ctx, tenantUUID, "", status, limit*3)
	if err != nil {
		return nil, err
	}
	items := make([]*GroupChatSyncTask, 0, minInt(limit, len(rows)))
	for _, row := range rows {
		if row == nil {
			continue
		}
		if strings.TrimSpace(row.TriggerType) != "group_chat_sync" {
			continue
		}
		mode := "incremental"
		if raw := strings.TrimSpace(row.ErrorCode); strings.HasPrefix(raw, "mode:") {
			mode = strings.TrimSpace(strings.TrimPrefix(raw, "mode:"))
		}
		items = append(items, mapTask(row, mode, strings.TrimSpace(row.Channel), strings.TrimSpace(row.AppType), ""))
		if len(items) >= limit {
			break
		}
	}
	return items, nil
}

func (s *GroupChatSyncService) ClearTasks(ctx context.Context, tenantUUID string, includeInFlight bool) (int64, error) {
	if s == nil || s.taskRepo == nil || s.taskRepo.DB == nil {
		return 0, ErrGroupChatSyncServiceNotReady
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	if tenantUUID == "" {
		return 0, errors.New("tenant_uuid is required")
	}
	query := s.taskRepo.DB.WithContext(ctx).
		Model(&leadmodel.LeadSyncTask{}).
		Where("tenant_uuid = ? AND trigger_type = ?", tenantUUID, "group_chat_sync")
	if !includeInFlight {
		query = query.Where("status IN ?", []string{"success", "failed"})
	}
	res := query.Delete(&leadmodel.LeadSyncTask{})
	if res.Error != nil {
		return 0, res.Error
	}
	return res.RowsAffected, nil
}

func (s *GroupChatSyncService) Sync(ctx context.Context, req GroupChatSyncRequest) (*GroupChatSyncResult, error) {
	// 保留同步接口签名用于兼容旧调用，内部转为异步。
	task, err := s.TriggerSyncAsync(ctx, req)
	if err != nil {
		return nil, err
	}
	return &GroupChatSyncResult{
		SyncedCount:         0,
		ChannelAccountUUID:  task.ChannelAccountUUID,
		ResolvedChannelCode: task.ResolvedChannelCode,
		ResolvedAppType:     task.ResolvedAppType,
	}, nil
}

func (s *GroupChatSyncService) List(ctx context.Context, tenantUUID string, limit int) ([]*acqmodel.GroupChatSnapshot, error) {
	if s == nil || s.repo == nil {
		return []*acqmodel.GroupChatSnapshot{}, nil
	}
	items, err := s.repo.List(ctx, tenantUUID, limit)
	if err != nil && isUndefinedRelationErr(err) {
		return []*acqmodel.GroupChatSnapshot{}, nil
	}
	return items, err
}

func (s *GroupChatSyncService) Get(ctx context.Context, tenantUUID, chatID string) (*acqmodel.GroupChatSnapshot, error) {
	if s == nil || s.repo == nil {
		return nil, ErrGroupChatSyncServiceNotReady
	}
	item, err := s.repo.GetByChatID(ctx, tenantUUID, chatID)
	if err != nil && isUndefinedRelationErr(err) {
		return nil, acqrepo.ErrRecordNotFound
	}
	return item, err
}

func (s *GroupChatSyncService) GetCustomerTimeline(ctx context.Context, tenantUUID, chatID, externalUserID string, limit int) ([]GroupCustomerTimelineEvent, error) {
	if s == nil || s.repo == nil {
		return nil, ErrGroupChatSyncServiceNotReady
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	chatID = strings.TrimSpace(chatID)
	externalUserID = strings.TrimSpace(externalUserID)
	if tenantUUID == "" || chatID == "" || externalUserID == "" {
		return []GroupCustomerTimelineEvent{}, nil
	}
	if limit <= 0 {
		limit = 20
	}
	item, err := s.repo.GetByChatID(ctx, tenantUUID, chatID)
	if err != nil {
		if isUndefinedRelationErr(err) {
			return []GroupCustomerTimelineEvent{}, nil
		}
		return nil, err
	}
	payload := map[string]any{}
	_ = json.Unmarshal(item.Payload, &payload)
	groupRaw := payload
	if nested, ok := payload["group_chat"].(map[string]any); ok && nested != nil {
		groupRaw = nested
	}
	memberList, _ := groupRaw["member_list"].([]any)
	if len(memberList) == 0 {
		return []GroupCustomerTimelineEvent{}, nil
	}
	events := make([]GroupCustomerTimelineEvent, 0, 4)
	for _, raw := range memberList {
		member, ok := raw.(map[string]any)
		if !ok || member == nil {
			continue
		}
		userID := strings.TrimSpace(fmt.Sprintf("%v", member["userid"]))
		if userID != externalUserID {
			continue
		}
		name := strings.TrimSpace(fmt.Sprintf("%v", member["name"]))
		joinScene := toInt(member["join_scene"])
		joinSceneLabel := mapJoinSceneLabel(joinScene)
		joinAt := toRFC3339FromUnix(member["join_time"])
		events = append(events, GroupCustomerTimelineEvent{
			EventType:   "group_joined",
			EventTime:   joinAt,
			Title:       "加入群聊",
			Description: fmt.Sprintf("%s 加入了 %s", firstNonEmpty(name, externalUserID), firstNonEmpty(strings.TrimSpace(item.Name), item.ChatID)),
			Source:      "group_chat_snapshot",
			Payload: map[string]any{
				"chat_id":         item.ChatID,
				"join_scene":      joinScene,
				"join_scene_text": joinSceneLabel,
			},
		})
		if invitorRaw, ok := member["invitor"].(map[string]any); ok && invitorRaw != nil {
			invitor := strings.TrimSpace(fmt.Sprintf("%v", invitorRaw["userid"]))
			if invitor != "" {
				events = append(events, GroupCustomerTimelineEvent{
					EventType:   "group_invited",
					EventTime:   joinAt,
					Title:       "被邀请入群",
					Description: fmt.Sprintf("邀请人：%s", invitor),
					Source:      "group_chat_snapshot",
					Payload: map[string]any{
						"chat_id":        item.ChatID,
						"invitor_userid": invitor,
					},
				})
			}
		}
		break
	}
	sort.SliceStable(events, func(i, j int) bool {
		return events[i].EventTime > events[j].EventTime
	})
	if len(events) > limit {
		events = events[:limit]
	}
	return events, nil
}

func (s *GroupChatSyncService) GetCustomerFollowups(ctx context.Context, tenantUUID, chatID, externalUserID string, limit int) (*GroupCustomerFollowupAggregation, error) {
	events, err := s.GetCustomerTimeline(ctx, tenantUUID, chatID, externalUserID, limit)
	if err != nil {
		return nil, err
	}
	return &GroupCustomerFollowupAggregation{
		Status:       "not_implemented",
		Reason:       "followup aggregation is pending CRM integration",
		ChatID:       strings.TrimSpace(chatID),
		ExternalUser: strings.TrimSpace(externalUserID),
		Summary: map[string]any{
			"total":       0,
			"latest_time": "",
		},
		Items: []map[string]any{},
		Meta: map[string]any{
			"source": "acquisition.group_chat_placeholder",
		},
		TimelineHint: events,
	}, nil
}

func (s *GroupChatSyncService) GetCustomerRelatedChats(ctx context.Context, tenantUUID, externalUserID string, limit int) ([]GroupCustomerRelatedChat, error) {
	if s == nil || s.repo == nil {
		return nil, ErrGroupChatSyncServiceNotReady
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	externalUserID = strings.TrimSpace(externalUserID)
	if tenantUUID == "" || externalUserID == "" {
		return []GroupCustomerRelatedChat{}, nil
	}
	if limit <= 0 {
		limit = 20
	}
	scanLimit := maxInt(limit*5, 500)
	if scanLimit > 5000 {
		scanLimit = 5000
	}
	items, err := s.repo.List(ctx, tenantUUID, scanLimit)
	if err != nil {
		if isUndefinedRelationErr(err) {
			return []GroupCustomerRelatedChat{}, nil
		}
		return nil, err
	}
	related := make([]GroupCustomerRelatedChat, 0, minInt(limit, len(items)))
	for _, item := range items {
		if item == nil {
			continue
		}
		memberList := extractMemberListFromSnapshot(item)
		if len(memberList) == 0 {
			continue
		}
		for _, raw := range memberList {
			member, ok := raw.(map[string]any)
			if !ok || member == nil {
				continue
			}
			userID := strings.TrimSpace(fmt.Sprintf("%v", member["userid"]))
			if userID != externalUserID {
				continue
			}
			invitorUserID := ""
			if invitorRaw, ok := member["invitor"].(map[string]any); ok && invitorRaw != nil {
				invitorUserID = strings.TrimSpace(fmt.Sprintf("%v", invitorRaw["userid"]))
			}
			joinScene := toInt(member["join_scene"])
			related = append(related, GroupCustomerRelatedChat{
				ChatID:         strings.TrimSpace(item.ChatID),
				ChatName:       strings.TrimSpace(item.Name),
				OwnerUserID:    strings.TrimSpace(item.OwnerUserID),
				SourceConfigID: strings.TrimSpace(item.SourceConfigID),
				JoinTime:       toRFC3339FromUnix(member["join_time"]),
				JoinScene:      joinScene,
				JoinSceneText:  mapJoinSceneLabel(joinScene),
				InvitorUserID:  invitorUserID,
				UpdatedAt:      item.UpdatedAt.UTC().Format(time.RFC3339),
			})
			break
		}
		if len(related) >= limit {
			break
		}
	}
	return related, nil
}

func extractMemberListFromSnapshot(item *acqmodel.GroupChatSnapshot) []any {
	if item == nil {
		return []any{}
	}
	payload := map[string]any{}
	_ = json.Unmarshal(item.Payload, &payload)
	groupRaw := payload
	if nested, ok := payload["group_chat"].(map[string]any); ok && nested != nil {
		groupRaw = nested
	}
	memberList, _ := groupRaw["member_list"].([]any)
	if memberList == nil {
		return []any{}
	}
	return memberList
}

func (s *GroupChatSyncService) ApplyWebhook(ctx context.Context, evt GroupChatWebhookEvent) error {
	if s == nil || s.repo == nil {
		return ErrGroupChatSyncServiceNotReady
	}
	tenantUUID := strings.ToLower(strings.TrimSpace(evt.TenantUUID))
	accountUUID := strings.ToLower(strings.TrimSpace(evt.ChannelAccountUUID))
	chatID := strings.TrimSpace(evt.ChatID)
	if tenantUUID == "" || accountUUID == "" || chatID == "" {
		return errors.New("tenant_uuid/channel_account_uuid/chat_id are required")
	}
	occurredAt := evt.OccurredAt.UTC()
	if occurredAt.IsZero() {
		occurredAt = time.Now().UTC()
	}
	payload := datatypes.JSON([]byte(`{"source":"webhook"}`))
	if evt.Payload != nil {
		if bs, err := json.Marshal(evt.Payload); err == nil {
			payload = datatypes.JSON(bs)
		}
	}
	item := &acqmodel.GroupChatSnapshot{
		SnapshotUUID:       uuid.NewString(),
		TenantUUID:         tenantUUID,
		ChannelAccountUUID: accountUUID,
		ChatID:             chatID,
		Name:               evt.Name,
		OwnerUserID:        evt.OwnerUserID,
		MemberCount:        evt.MemberCount,
		SourceConfigID:     evt.SourceConfigID,
		Payload:            payload,
		UpdatedAt:          occurredAt,
	}
	if err := s.repo.Upsert(ctx, item); err != nil {
		if isUndefinedRelationErr(err) {
			return nil
		}
		return err
	}
	return nil
}

type groupChatSyncRuntime struct {
	tenantUUID    string
	accountUUID   string
	channelCode   string
	appType       string
	resolveSource string
}

func (s *GroupChatSyncService) resolveRuntime(ctx context.Context, tenantUUID, accountUUID string) (*groupChatSyncRuntime, error) {
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	accountUUID = strings.ToLower(strings.TrimSpace(accountUUID))
	if tenantUUID == "" {
		return nil, errors.New("tenant_uuid is required")
	}
	if _, err := uuid.Parse(tenantUUID); err != nil {
		return nil, errors.New("tenant_uuid is invalid")
	}
	resolveSource := leadmodel.LeadSyncTaskResolveExplicit
	if accountUUID != "" {
		if _, parseErr := uuid.Parse(accountUUID); parseErr != nil {
			accountUUID = ""
		}
	}
	if accountUUID == "" {
		if s.accountResolver == nil {
			return nil, ErrDefaultChannelAccountNotFound
		}
		// 严格使用系统默认渠道账号，不做类型偏好推断。
		resolved, err := s.accountResolver.ResolveDefaultChannelAccount(ctx, tenantUUID, "", "")
		if err != nil || strings.TrimSpace(resolved) == "" {
			return nil, ErrDefaultChannelAccountNotFound
		}
		accountUUID = strings.ToLower(strings.TrimSpace(resolved))
		resolveSource = leadmodel.LeadSyncTaskResolveDefault
	}
	if accountUUID == "" {
		return nil, ErrDefaultChannelAccountNotFound
	}
	if _, err := uuid.Parse(accountUUID); err != nil {
		return nil, ErrDefaultChannelAccountNotFound
	}
	if s.accountResolver == nil {
		return nil, errors.New("group chat account resolver unavailable")
	}
	profile, err := s.accountResolver.GetChannelAccount(ctx, tenantUUID, accountUUID)
	if err != nil {
		return nil, err
	}
	if profile == nil {
		return nil, ErrDefaultChannelAccountNotFound
	}
	channelCode := strings.ToLower(strings.TrimSpace(profile.ChannelCode))
	appType := strings.ToLower(strings.TrimSpace(profile.AppType))
	if channelCode != "wechat" {
		return nil, fmt.Errorf("%w: channel=%s app_type=%s", ErrUnsupportedChannelAccountType, channelCode, appType)
	}
	if _, err := wecomauth.ResolveKind(channelCode, appType); err != nil {
		return nil, fmt.Errorf("%w: channel=%s app_type=%s", ErrUnsupportedChannelAccountType, channelCode, appType)
	}
	return &groupChatSyncRuntime{
		tenantUUID:    tenantUUID,
		accountUUID:   accountUUID,
		channelCode:   channelCode,
		appType:       appType,
		resolveSource: resolveSource,
	}, nil
}

func (s *GroupChatSyncService) executeTask(ctx context.Context, taskUUID string, rt *groupChatSyncRuntime, mode string) {
	taskUUID = strings.ToLower(strings.TrimSpace(taskUUID))
	if s == nil || s.taskRepo == nil || rt == nil || rt.tenantUUID == "" || taskUUID == "" {
		return
	}
	startedAt := time.Now().UTC()
	_ = s.taskRepo.UpdateStatus(ctx, rt.tenantUUID, taskUUID, "running", map[string]any{
		"started_at":       startedAt,
		"progress_percent": 5,
		"error_code":       "mode:" + mode,
		"error_message":    "",
	})

	stats, runErr := s.pullAndUpsert(ctx, rt, mode, func(current, total int) {
		percent := 10
		if total > 0 {
			percent = minInt(95, maxInt(10, int(float64(current)/float64(total)*90)))
		}
		_ = s.taskRepo.UpdateStatus(ctx, rt.tenantUUID, taskUUID, "", map[string]any{
			"progress_current": current,
			"progress_total":   total,
			"progress_percent": percent,
		})
	})
	finishedAt := time.Now().UTC()
	if runErr != nil {
		errMsg := strings.TrimSpace(runErr.Error())
		if errMsg == "" {
			errMsg = "group chat sync failed"
		}
		_ = s.taskRepo.UpdateStatus(ctx, rt.tenantUUID, taskUUID, "failed", map[string]any{
			"finished_at":      finishedAt,
			"error_message":    errMsg,
			"progress_percent": 100,
			"stats_total":      stats.Total,
			"stats_created":    stats.Synced,
			"stats_updated":    0,
			"progress_current": stats.Synced + stats.Skipped + stats.Failed,
			"progress_total":   stats.Total,
		})
		logger.WithFields(logger.Fields{
			"component":            "group_chat_sync_task",
			"tenant_uuid":          rt.tenantUUID,
			"channel_account_uuid": rt.accountUUID,
			"task_uuid":            taskUUID,
			"mode":                 mode,
			"error":                errMsg,
		}).Error("group chat sync task failed")
		return
	}
	finalStatus := "success"
	finalMessage := summarizeGroupChatPullIssues(stats)
	if stats.Synced == 0 && (stats.Failed > 0 || stats.PermissionDeny > 0) {
		finalStatus = "failed"
		if finalMessage == "" {
			finalMessage = "group chat sync failed: no group chat can be synced"
		}
	}
	_ = s.taskRepo.UpdateStatus(ctx, rt.tenantUUID, taskUUID, finalStatus, map[string]any{
		"finished_at":      finishedAt,
		"error_message":    finalMessage,
		"progress_percent": 100,
		"progress_current": stats.Synced + stats.Skipped + stats.Failed,
		"progress_total":   stats.Total,
		"stats_total":      stats.Total,
		"stats_created":    stats.Synced,
		"stats_updated":    0,
	})
	logger.WithFields(logger.Fields{
		"component":            "group_chat_sync_task",
		"tenant_uuid":          rt.tenantUUID,
		"channel_account_uuid": rt.accountUUID,
		"task_uuid":            taskUUID,
		"mode":                 mode,
		"total":                stats.Total,
		"synced":               stats.Synced,
		"skipped":              stats.Skipped,
		"failed":               stats.Failed,
		"permission_deny":      stats.PermissionDeny,
	}).Info("group chat sync task completed")
}

type groupChatPullStats struct {
	Total          int
	Synced         int
	Skipped        int
	Failed         int
	PermissionDeny int
	FailureSamples []string
}

func (s *GroupChatSyncService) pullAndUpsert(
	ctx context.Context,
	rt *groupChatSyncRuntime,
	mode string,
	onProgress func(current, total int),
) (groupChatPullStats, error) {
	if s == nil || s.repo == nil || s.accountResolver == nil || rt == nil {
		return groupChatPullStats{}, ErrGroupChatSyncServiceNotReady
	}
	credentials, err := s.accountResolver.GetChannelAccountCredentials(ctx, rt.tenantUUID, rt.accountUUID)
	if err != nil {
		return groupChatPullStats{}, err
	}
	if s.providerFactory == nil {
		return groupChatPullStats{}, ErrGroupChatSyncServiceNotReady
	}
	app, err := s.providerFactory.Build(rt.channelCode, rt.appType, credentials)
	if err != nil {
		return groupChatPullStats{}, err
	}
	if app == nil || app.ExternalContactGroupChat == nil {
		return groupChatPullStats{}, errors.New("wecom group chat client unavailable")
	}

	statusFilters := []int{0}
	if strings.EqualFold(mode, "full") {
		statusFilters = []int{0, 1, 2}
	}

	chatIDs := make([]string, 0, 256)
	seen := make(map[string]struct{}, 512)
	for _, statusFilter := range statusFilters {
		cursor := ""
		for {
			res, listErr := app.ExternalContactGroupChat.List(ctx, &pwgroupchatreq.RequestGroupChatList{
				StatusFilter: statusFilter,
				Cursor:       cursor,
				Limit:        100,
			})
			if listErr != nil {
				return groupChatPullStats{}, listErr
			}
			if res == nil {
				return groupChatPullStats{}, errors.New("wecom groupchat/list response is nil")
			}
			if res.ErrCode != 0 {
				return groupChatPullStats{}, fmt.Errorf("wecom groupchat/list failed: %d %s", res.ErrCode, strings.TrimSpace(res.ErrMsg))
			}
			for _, row := range res.GroupChatList {
				if row == nil {
					continue
				}
				chatID := strings.TrimSpace(row.ChatID)
				if chatID == "" {
					continue
				}
				if _, ok := seen[chatID]; ok {
					continue
				}
				seen[chatID] = struct{}{}
				chatIDs = append(chatIDs, chatID)
			}
			nextCursor := strings.TrimSpace(res.NextCursor)
			if nextCursor == "" || nextCursor == cursor {
				break
			}
			cursor = nextCursor
		}
	}

	stats := groupChatPullStats{Total: len(chatIDs)}
	if stats.Total == 0 {
		if onProgress != nil {
			onProgress(0, 0)
		}
		return stats, nil
	}

	now := time.Now().UTC()
	for idx, chatID := range chatIDs {
		processed := idx + 1
		res, getErr := app.ExternalContactGroupChat.Get(ctx, chatID, 1)
		if getErr != nil {
			stats.Failed++
			if len(stats.FailureSamples) < 5 {
				stats.FailureSamples = append(stats.FailureSamples, fmt.Sprintf("%s: %s", chatID, strings.TrimSpace(getErr.Error())))
			}
			if onProgress != nil {
				onProgress(processed, stats.Total)
			}
			continue
		}
		if res == nil {
			stats.Failed++
			if len(stats.FailureSamples) < 5 {
				stats.FailureSamples = append(stats.FailureSamples, fmt.Sprintf("%s: empty response", chatID))
			}
			if onProgress != nil {
				onProgress(processed, stats.Total)
			}
			continue
		}
		if res.ErrCode != 0 {
			if res.ErrCode == 60011 {
				stats.PermissionDeny++
				stats.Skipped++
			} else {
				stats.Failed++
			}
			if len(stats.FailureSamples) < 5 {
				stats.FailureSamples = append(stats.FailureSamples, fmt.Sprintf("%s: %d %s", chatID, res.ErrCode, strings.TrimSpace(res.ErrMsg)))
			}
			if onProgress != nil {
				onProgress(processed, stats.Total)
			}
			continue
		}
		if res.GroupChat == nil {
			stats.Skipped++
			if onProgress != nil {
				onProgress(processed, stats.Total)
			}
			continue
		}

		payloadBytes, _ := json.Marshal(map[string]any{
			"source":     "wecom_pull",
			"mode":       mode,
			"group_chat": res.GroupChat,
		})
		payload := datatypes.JSON([]byte(`{"source":"wecom_pull"}`))
		if len(payloadBytes) > 0 {
			payload = datatypes.JSON(payloadBytes)
		}
		item := &acqmodel.GroupChatSnapshot{
			SnapshotUUID:       uuid.NewString(),
			TenantUUID:         rt.tenantUUID,
			ChannelAccountUUID: rt.accountUUID,
			ChatID:             strings.TrimSpace(res.GroupChat.ChatID),
			Name:               strings.TrimSpace(res.GroupChat.Name),
			OwnerUserID:        strings.TrimSpace(res.GroupChat.Owner),
			MemberCount:        len(res.GroupChat.MemberList),
			CreateTime:         unixPtrToTime(res.GroupChat.CreateTime),
			Payload:            payload,
			UpdatedAt:          now,
		}
		if item.ChatID == "" {
			item.ChatID = chatID
		}
		if err := s.repo.Upsert(ctx, item); err != nil {
			if isUndefinedRelationErr(err) {
				return groupChatPullStats{}, nil
			}
			stats.Failed++
			if len(stats.FailureSamples) < 5 {
				stats.FailureSamples = append(stats.FailureSamples, fmt.Sprintf("%s: upsert failed: %s", chatID, strings.TrimSpace(err.Error())))
			}
			if onProgress != nil {
				onProgress(processed, stats.Total)
			}
			continue
		}
		stats.Synced++
		if onProgress != nil {
			onProgress(processed, stats.Total)
		}
	}
	return stats, nil
}

func normalizeSyncMode(raw string) string {
	raw = strings.ToLower(strings.TrimSpace(raw))
	if raw != "full" {
		return "incremental"
	}
	return "full"
}

func summarizeGroupChatPullIssues(stats groupChatPullStats) string {
	msgParts := make([]string, 0, 4)
	if stats.PermissionDeny > 0 {
		msgParts = append(msgParts, fmt.Sprintf("权限不足跳过 %d", stats.PermissionDeny))
	}
	if stats.Failed > 0 {
		msgParts = append(msgParts, fmt.Sprintf("失败 %d", stats.Failed))
	}
	if stats.Skipped > stats.PermissionDeny {
		msgParts = append(msgParts, fmt.Sprintf("其他跳过 %d", stats.Skipped-stats.PermissionDeny))
	}
	if len(stats.FailureSamples) > 0 {
		msgParts = append(msgParts, "示例："+strings.Join(stats.FailureSamples, " | "))
	}
	if len(msgParts) == 0 {
		return ""
	}
	return strings.Join(msgParts, "；")
}

func mapTask(task *leadmodel.LeadSyncTask, mode, channelCode, appType, _ string) *GroupChatSyncTask {
	if task == nil {
		return nil
	}
	mode = normalizeSyncMode(mode)
	return &GroupChatSyncTask{
		TaskUUID:            strings.TrimSpace(task.TaskUUID),
		JobUUID:             strings.TrimSpace(task.TaskUUID),
		Status:              strings.TrimSpace(task.Status),
		Mode:                mode,
		ChannelAccountUUID:  strings.TrimSpace(task.ChannelAccountUUID),
		ResolvedChannelCode: strings.TrimSpace(channelCode),
		ResolvedAppType:     strings.TrimSpace(appType),
		ProgressTotal:       task.ProgressTotal,
		ProgressCurrent:     task.ProgressCurrent,
		ProgressPercent:     task.ProgressPercent,
		StatsTotal:          task.StatsTotal,
		StatsCreated:        task.StatsCreated,
		StatsUpdated:        task.StatsUpdated,
		ErrorMessage:        strings.TrimSpace(task.ErrorMessage),
		CreatedAt:           formatTaskTime(task.CreatedAt),
		StartedAt:           formatTaskTimePtr(task.StartedAt),
		FinishedAt:          formatTaskTimePtr(task.FinishedAt),
	}
}

func formatTaskTimePtr(v *time.Time) string {
	if v == nil || v.IsZero() {
		return ""
	}
	return v.UTC().Format(time.RFC3339)
}

func formatTaskTime(v time.Time) string {
	if v.IsZero() {
		return ""
	}
	return v.UTC().Format(time.RFC3339)
}

func unixPtrToTime(v int) *time.Time {
	if v <= 0 {
		return nil
	}
	t := time.Unix(int64(v), 0).UTC()
	return &t
}

func buildWeComSelfBuiltWorkApp(credentials map[string]string) (*work.Work, error) {
	corpID := strings.TrimSpace(credentials["corp_id"])
	appSecret := strings.TrimSpace(credentials["app_secret"])
	agentIDRaw := strings.TrimSpace(credentials["agent_id"])
	if corpID == "" {
		corpID = strings.TrimSpace(credentials["auth_corp_id"])
	}
	if corpID == "" || appSecret == "" {
		return nil, errors.New("wecom self-built account requires corp_id and app_secret")
	}
	agentID := 0
	if agentIDRaw != "" {
		parsed, err := strconv.Atoi(agentIDRaw)
		if err != nil {
			return nil, errors.New("wecom credentials invalid agent_id")
		}
		agentID = parsed
	}
	return work.NewWork(&work.UserConfig{
		CorpID:    corpID,
		AgentID:   agentID,
		Secret:    appSecret,
		Token:     strings.TrimSpace(credentials["token"]),
		HttpDebug: parseCredentialBool(credentials["http_debug"]),
		OAuth: work.OAuth{
			Callback: strings.TrimSpace(credentials["oauth_callback"]),
			Scopes:   nil,
		},
	})
}

func buildWeComOpenWorkApp(credentials map[string]string) (*work.Work, error) {
	corpID := strings.TrimSpace(credentials["corp_id"])
	templateID := strings.TrimSpace(firstNonEmpty(credentials["template_id"], credentials["suite_id"]))
	templateSecret := strings.TrimSpace(firstNonEmpty(credentials["template_secret"], credentials["suite_secret"]))
	templateTicket := strings.TrimSpace(firstNonEmpty(credentials["template_ticket"], credentials["suite_ticket"]))
	providerCorpID := strings.TrimSpace(firstNonEmpty(credentials["provider_corpid"], credentials["provider_corp_id"]))
	providerSecret := strings.TrimSpace(credentials["provider_secret"])
	permanentCode := strings.TrimSpace(credentials["permanent_code"])
	if corpID == "" {
		corpID = strings.TrimSpace(credentials["auth_corp_id"])
	}
	delegatedReady := corpID != "" &&
		templateID != "" &&
		templateSecret != "" &&
		providerCorpID != "" &&
		providerSecret != "" &&
		permanentCode != "" &&
		templateTicket != ""
	if !delegatedReady {
		return nil, errors.New("openwork account requires delegated_template credentials")
	}
	callback := strings.TrimSpace(credentials["oauth_callback"])
	if callback == "" {
		callback = "http://localhost"
	}
	memCache := plcache.NewMemCache("scrm_group_chat_sync_openwork", 10*time.Minute, os.TempDir())
	if memCache == nil {
		return nil, errors.New("wecom delegated init cache failed")
	}
	httpDebug := parseCredentialBool(credentials["http_debug"])
	openWorkApp, err := openwork.NewOpenWork(&openwork.UserConfig{
		AppID:          templateID,
		Secret:         templateSecret,
		ProviderCorpID: providerCorpID,
		ProviderSecret: providerSecret,
		CallbackURL:    callback,
		Cache:          memCache,
		HttpDebug:      httpDebug,
		Log:            openwork.Log{Level: "debug", Stdout: httpDebug},
	})
	if err != nil {
		return nil, err
	}
	suiteTicketComponent, ok := openWorkApp.GetComponent("SuiteTicket").(*openworksuit.SuiteTicket)
	if !ok || suiteTicketComponent == nil {
		return nil, errors.New("wecom delegated SuiteTicket component unavailable")
	}
	if err := suiteTicketComponent.SetTicket(templateTicket); err != nil {
		return nil, err
	}
	return openWorkApp.ProviderClient(corpID, permanentCode, nil)
}

type GroupChatAppBuilder func(credentials map[string]string) (*work.Work, error)

type GroupChatProviderFactory struct {
	entries map[string]GroupChatAppBuilder
}

func NewGroupChatProviderFactory() *GroupChatProviderFactory {
	return &GroupChatProviderFactory{entries: map[string]GroupChatAppBuilder{}}
}

func (f *GroupChatProviderFactory) Register(channel, appType string, builder GroupChatAppBuilder) error {
	if f == nil {
		return errors.New("group chat provider factory is nil")
	}
	if builder == nil {
		return errors.New("group chat app builder is required")
	}
	key, err := normalizeGroupChatProviderKey(channel, appType)
	if err != nil {
		return err
	}
	f.entries[key] = builder
	return nil
}

func (f *GroupChatProviderFactory) Build(channel, appType string, credentials map[string]string) (*work.Work, error) {
	if f == nil {
		return nil, errors.New("group chat provider factory is nil")
	}
	key, err := normalizeGroupChatProviderKey(channel, appType)
	if err != nil {
		return nil, err
	}
	builder, ok := f.entries[key]
	if !ok || builder == nil {
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedChannelAccountType, key)
	}
	return builder(credentials)
}

func normalizeGroupChatProviderKey(channel, appType string) (string, error) {
	channel = strings.ToLower(strings.TrimSpace(channel))
	appType = strings.ToLower(strings.TrimSpace(appType))
	if channel == "" || appType == "" {
		return "", errors.New("channel and app_type are required")
	}
	return channel + ":" + appType, nil
}

func parseCredentialBool(raw string) bool {
	raw = strings.ToLower(strings.TrimSpace(raw))
	return raw == "1" || raw == "true" || raw == "yes" || raw == "on"
}

func toInt(raw any) int {
	switch v := raw.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case json.Number:
		n, _ := v.Int64()
		return int(n)
	default:
		parsed, _ := strconv.Atoi(strings.TrimSpace(fmt.Sprintf("%v", raw)))
		return parsed
	}
}

func toRFC3339FromUnix(raw any) string {
	ts := int64(toInt(raw))
	if ts <= 0 {
		return ""
	}
	return time.Unix(ts, 0).UTC().Format(time.RFC3339)
}

func mapJoinSceneLabel(scene int) string {
	switch scene {
	case 1:
		return "扫描群二维码"
	case 2:
		return "直接邀请入群"
	case 3:
		return "由群成员邀请入群"
	default:
		return "未知"
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
