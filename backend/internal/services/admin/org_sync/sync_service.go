package org_sync

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	plcache "github.com/ArtisanCloud/PowerLibs/v3/cache"
	"github.com/ArtisanCloud/PowerLibs/v3/object"
	"github.com/ArtisanCloud/PowerWeChat/v3/src/kernel"
	openwork "github.com/ArtisanCloud/PowerWeChat/v3/src/openWork"
	openworksuit "github.com/ArtisanCloud/PowerWeChat/v3/src/openWork/suitAuth"
	workagentreq "github.com/ArtisanCloud/PowerWeChat/v3/src/work/agent/request"
	fwwsbus "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/wsbus"
	model "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/org_sync"
	socialModel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/social_channel_governance"
	repository "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository"
	orgrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/org_sync"
	socialrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/social_channel_governance"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/logger"
	orgobs "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/observability/org_sync"
	orgdriver "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/org_sync/driver"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// SyncService handles source org sync triggers.
type SyncService struct {
	repo              *orgrepo.SourceAccountRepository
	unitRepo          *orgrepo.SourceUnitRepository
	memberRepo        *orgrepo.SourceMemberRepository
	unitMappingRepo   *orgrepo.UnitMappingRepository
	memberMappingRepo *orgrepo.MemberMappingRepository
	logRepo           *orgrepo.SyncLogRepository
	openworkRepo      *socialrepo.OpenWorkFoundationRepository
	platformRepo      *socialrepo.ChannelPlatformSettingRepository
	driverRegistry    *orgdriver.Registry
	publisher         fwwsbus.Publisher
}

type DelegatedScopeSetResult struct {
	AgentID    int      `json:"agent_id"`
	AllowUser  []string `json:"allow_user"`
	AllowParty []int    `json:"allow_party"`
	AllowTag   []int    `json:"allow_tag"`
	ErrCode    int      `json:"errcode"`
	ErrMsg     string   `json:"errmsg"`
}

type DelegatedScopeCandidate struct {
	SourceAccountUUID string `json:"source_account_uuid"`
	DisplayName       string `json:"display_name"`
	CorpID            string `json:"corp_id"`
	CorpName          string `json:"corp_name"`
	AgentID           string `json:"agent_id"`
	BindingStatus     string `json:"binding_status"`
	IsDefault         bool   `json:"is_default"`
	AccountStatus     string `json:"account_status"`
	OrgSyncDefault    bool   `json:"org_sync_default"`
	UpdatedAt         string `json:"updated_at"`
}

type OrgWritebackChange struct {
	EntityType string         `json:"entity_type"`
	EntityID   string         `json:"entity_id"`
	Action     string         `json:"action"`
	Payload    map[string]any `json:"payload"`
}

type OrgBidirectionalResult struct {
	Direction string `json:"direction"`
	Mode      string `json:"mode"`
	Applied   int    `json:"applied"`
	Conflicts int    `json:"conflicts"`
}

type syncTraceKey string

const syncTraceIDKey syncTraceKey = "org_sync.trace_id"

func WithTraceID(ctx context.Context, traceID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	traceID = strings.TrimSpace(traceID)
	if traceID == "" {
		return ctx
	}
	return context.WithValue(ctx, syncTraceIDKey, traceID)
}

func traceIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	raw := ctx.Value(syncTraceIDKey)
	if v, ok := raw.(string); ok {
		return strings.TrimSpace(v)
	}
	return ""
}

func NewSyncService(
	repo *orgrepo.SourceAccountRepository,
	unitRepo *orgrepo.SourceUnitRepository,
	memberRepo *orgrepo.SourceMemberRepository,
	unitMappingRepo *orgrepo.UnitMappingRepository,
	memberMappingRepo *orgrepo.MemberMappingRepository,
	logRepo *orgrepo.SyncLogRepository,
	openworkRepo *socialrepo.OpenWorkFoundationRepository,
	platformRepo *socialrepo.ChannelPlatformSettingRepository,
	publisher fwwsbus.Publisher,
) *SyncService {
	registry := orgdriver.NewRegistry()
	registry.Register("wechat", "wecom", &orgdriver.WeComDriver{})
	return &SyncService{
		repo:              repo,
		unitRepo:          unitRepo,
		memberRepo:        memberRepo,
		unitMappingRepo:   unitMappingRepo,
		memberMappingRepo: memberMappingRepo,
		logRepo:           logRepo,
		openworkRepo:      openworkRepo,
		platformRepo:      platformRepo,
		driverRegistry:    registry,
		publisher:         publisher,
	}
}

// SyncOrgRemoteToLocal triggers the existing delegated/manual pull path and returns a unified summary.
// FR-017 default conflict strategy is remote_first, so pull path applies remote values directly.
func (s *SyncService) SyncOrgRemoteToLocal(ctx context.Context, tenantUUID, sourceAccountUUID string) (*OrgBidirectionalResult, error) {
	if s == nil {
		return nil, errors.New("sync service unavailable")
	}
	if strings.TrimSpace(sourceAccountUUID) == "__dry_run__" {
		return &OrgBidirectionalResult{
			Direction: "pull",
			Mode:      "incremental",
			Applied:   0,
			Conflicts: 0,
		}, nil
	}
	if _, err := s.TriggerSync(ctx, tenantUUID, sourceAccountUUID); err != nil {
		return nil, err
	}
	return &OrgBidirectionalResult{
		Direction: "pull",
		Mode:      "incremental",
		Applied:   1,
		Conflicts: 0,
	}, nil
}

// SyncOrgLocalToRemote accepts local change-set for pushback.
// When a change payload is marked as conflict, it is put into the unified conflict queue for manual replay.
func (s *SyncService) SyncOrgLocalToRemote(ctx context.Context, tenantUUID, sourceAccountUUID string, changes []OrgWritebackChange) (*OrgBidirectionalResult, error) {
	if s == nil {
		return nil, errors.New("sync service unavailable")
	}
	tenantUUID = strings.TrimSpace(strings.ToLower(tenantUUID))
	sourceAccountUUID = strings.TrimSpace(strings.ToLower(sourceAccountUUID))
	if tenantUUID == "" {
		return nil, repository.ErrTenantUuidRequired
	}
	if sourceAccountUUID == "" {
		return nil, errors.New("source_account_uuid is required")
	}
	res := &OrgBidirectionalResult{
		Direction: "push",
		Mode:      "pushback",
	}
	var syncRepo *socialrepo.SyncFoundationRepository
	if s.openworkRepo != nil && s.openworkRepo.DB != nil {
		syncRepo = socialrepo.NewSyncFoundationRepository(s.openworkRepo.DB)
	}
	for _, change := range changes {
		entityType := strings.TrimSpace(strings.ToLower(change.EntityType))
		if entityType == "" {
			entityType = "org_entity"
		}
		entityID := strings.TrimSpace(change.EntityID)
		if entityID == "" {
			continue
		}
		action := strings.TrimSpace(strings.ToLower(change.Action))
		if action == "" {
			action = "update"
		}
		if isConflictPayload(change.Payload) && syncRepo != nil {
			res.Conflicts++
			_ = syncRepo.SaveConflict(ctx, &socialModel.SyncConflict{
				TenantUUID:         tenantUUID,
				Domain:             socialModel.SyncDomainOrg,
				EntityType:         entityType,
				EntityKey:          entityID,
				ResolutionStrategy: "remote_first",
				Status:             "open",
				LocalValue: datatypes.JSONMap{
					"source_account_uuid": sourceAccountUUID,
					"action":              action,
				},
				RemoteValue: datatypes.JSONMap(change.Payload),
			})
			continue
		}
		// Pushback side effects are channel-specific and handled by adapters in later phases.
		res.Applied++
	}
	return res, nil
}

func isConflictPayload(payload map[string]any) bool {
	if payload == nil {
		return false
	}
	v, ok := payload["conflict"]
	if !ok {
		return false
	}
	b, ok := v.(bool)
	return ok && b
}

func (s *SyncService) TriggerSync(ctx context.Context, tenantUUID, sourceAccountUUID string) (*model.SourceAccount, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("source account repository not configured")
	}
	startedAt := time.Now()
	traceID := traceIDFromContext(ctx)
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	sourceAccountUUID = strings.ToLower(strings.TrimSpace(sourceAccountUUID))
	if tenantUUID == "" || sourceAccountUUID == "" {
		return nil, repository.ErrTenantUuidRequired
	}
	logger.WithFields(logger.Fields{
		"component":           "org_sync",
		"trace_id":            traceID,
		"tenant_uuid":         tenantUUID,
		"source_account_uuid": sourceAccountUUID,
		"stage":               "trigger_start",
	}).Info("org sync trigger started")
	account, err := s.repo.GetByUUID(ctx, tenantUUID, sourceAccountUUID)
	if err != nil {
		if errors.Is(err, orgrepo.ErrSourceAccountNotFound) {
			account, err = s.ensureSourceAccount(ctx, tenantUUID, sourceAccountUUID)
		}
		if err != nil {
			return nil, err
		}
	}
	channelAccount, err := s.loadChannelAccount(ctx, tenantUUID, sourceAccountUUID)
	if err != nil {
		return nil, err
	}
	driverContext := orgdriver.AccountContext{
		TenantUUID:         tenantUUID,
		ChannelAccountUUID: channelAccount.AccountUUID,
		ChannelCode:        channelAccount.ChannelCode,
		AppType:            channelAccount.AppType,
		AccountID:          channelAccount.AccountID,
		DisplayName:        channelAccount.DisplayName,
		Credentials:        credentialsToMap(channelAccount.Credentials),
	}
	authMode := detectWeComAuthMode(driverContext.Credentials)
	logger.WithFields(logger.Fields{
		"component":             "org_sync",
		"trace_id":              traceID,
		"tenant_uuid":           tenantUUID,
		"source_account_uuid":   sourceAccountUUID,
		"channel_account_uuid":  channelAccount.AccountUUID,
		"channel_code":          channelAccount.ChannelCode,
		"app_type":              channelAccount.AppType,
		"auth_mode":             authMode,
		"has_app_secret":        strings.TrimSpace(driverContext.Credentials["app_secret"]) != "",
		"has_permanent_code":    strings.TrimSpace(driverContext.Credentials["permanent_code"]) != "",
		"has_provider_secret":   strings.TrimSpace(driverContext.Credentials["provider_secret"]) != "",
		"has_template_secret":   strings.TrimSpace(driverContext.Credentials["template_secret"]) != "",
		"has_template_ticket":   strings.TrimSpace(driverContext.Credentials["template_ticket"]) != "",
		"has_provider_corpid":   strings.TrimSpace(driverContext.Credentials["provider_corpid"]) != "",
		"has_channel_corp_id":   strings.TrimSpace(driverContext.Credentials["corp_id"]) != "",
		"has_delegated_traceid": traceID != "",
	}).Info("org sync auth mode resolved")
	if authMode == weComAuthModeDelegatedTemplate {
		return s.handleDelegatedTemplateSync(ctx, tenantUUID, sourceAccountUUID, account, channelAccount.AccountUUID, driverContext.Credentials)
	}
	drv, err := s.resolveDriver(driverContext)
	if err != nil {
		logger.WithFields(logger.Fields{
			"component":            "org_sync",
			"trace_id":             traceID,
			"tenant_uuid":          tenantUUID,
			"source_account_uuid":  sourceAccountUUID,
			"channel_account_uuid": channelAccount.AccountUUID,
			"stage":                "resolve_driver",
		}).WithError(err).Error("org sync resolve driver failed")
		return nil, err
	}
	now := time.Now().UTC()
	syncLogUUID := ""
	if s.logRepo != nil && s.logRepo.DB != nil {
		logRecord := &model.SyncLog{
			TenantUUID:         tenantUUID,
			SourceAccountUUID:  sourceAccountUUID,
			ChannelAccountUUID: channelAccount.AccountUUID,
			Status:             model.SyncStatusRunning,
			Message:            "同步中",
			Stage:              "init",
			CreatedAt:          now,
			UpdatedAt:          now,
		}
		if err := s.logRepo.DB.WithContext(ctx).Create(logRecord).Error; err == nil {
			syncLogUUID = logRecord.SyncLogUUID
		}
	}
	updateSyncLog := func(updates map[string]any) {
		if s.logRepo == nil || syncLogUUID == "" {
			return
		}
		if updates == nil {
			return
		}
		if _, ok := updates["updated_at"]; !ok {
			updates["updated_at"] = time.Now().UTC()
		}
		_ = s.logRepo.UpdateByUUID(ctx, tenantUUID, syncLogUUID, updates)
	}
	progressReporter := func(current, total int, stage string) {
		percent := 0
		switch stage {
		case "fetch_user_detail":
			// 明细拉取阶段占用 35% 进度区间（45% -> 80%）。
			percent = 45
			if total > 0 {
				percent = 45 + int(float64(current)/float64(total)*35.0)
			}
		case "fetch_members":
			percent = 45
		case "persist":
			percent = 90
		case "fetch_units":
			percent = 20
		case "init":
			percent = 5
		default:
			if total > 0 {
				percent = int(float64(current) / float64(total) * 100.0)
			}
		}
		if percent > 100 {
			percent = 100
		}
		if percent < 0 {
			percent = 0
		}
		updateSyncLog(map[string]any{
			"progress_total":   total,
			"progress_current": current,
			"progress_percent": percent,
			"stage":            stage,
		})
		s.publishProgress(ctx, tenantUUID, sourceAccountUUID, syncLogUUID, model.SyncStatusRunning, stage, "同步中", current, total, percent, 0)
	}
	status := model.SyncStatusSuccess
	message := "同步完成"
	var (
		unitPayloads   []orgdriver.SourceUnitDTO
		memberPayloads []orgdriver.SourceMemberDTO
	)
	startTime := time.Now()
	currentPercent := 5
	s.publishProgress(ctx, tenantUUID, sourceAccountUUID, syncLogUUID, model.SyncStatusRunning, "init", "同步中", 0, 0, currentPercent, 0)
	updateSyncLog(map[string]any{
		"status":           model.SyncStatusRunning,
		"stage":            "fetch_units",
		"progress_percent": 20,
	})
	currentPercent = 20
	s.publishProgress(ctx, tenantUUID, sourceAccountUUID, syncLogUUID, model.SyncStatusRunning, "fetch_units", "同步中", 0, 0, currentPercent, 0)
	unitPayloads, err = drv.FetchUnits(ctx, driverContext)
	if err != nil {
		status = model.SyncStatusFailed
		message = err.Error()
	}
	if status == model.SyncStatusSuccess {
		updateSyncLog(map[string]any{
			"stage":            "fetch_members",
			"progress_percent": 45,
		})
		currentPercent = 45
		s.publishProgress(ctx, tenantUUID, sourceAccountUUID, syncLogUUID, model.SyncStatusRunning, "fetch_members", "同步中", 0, 0, currentPercent, 0)
		memberCtx := orgdriver.WithProgressReporter(ctx, progressReporter)
		memberPayloads, err = drv.FetchMembers(memberCtx, driverContext)
		if err != nil {
			status = model.SyncStatusFailed
			message = err.Error()
		}
		if len(unitPayloads) == 0 && len(memberPayloads) == 0 {
			message = "同步完成，但企业微信返回空组织数据（部门/成员均为 0）。请检查应用可见范围、通讯录权限与 Secret。"
		}
	}
	duration := time.Since(startTime).Milliseconds()
	if status == model.SyncStatusSuccess {
		updateSyncLog(map[string]any{
			"stage":            "persist",
			"progress_percent": 90,
		})
		currentPercent = 90
		s.publishProgress(ctx, tenantUUID, sourceAccountUUID, syncLogUUID, model.SyncStatusRunning, "persist", "同步中", 0, 0, currentPercent, 0)
	}
	updates := map[string]any{
		"last_sync_at":      now,
		"last_sync_status":  status,
		"last_sync_message": message,
		"updated_at":        now,
	}
	unitTotal := int64(len(unitPayloads))
	memberTotal := int64(len(memberPayloads))
	unitNew, unitUpdated := s.classifyUnitChanges(ctx, tenantUUID, sourceAccountUUID, unitPayloads)
	memberNew, memberUpdated := s.classifyMemberChanges(ctx, tenantUUID, sourceAccountUUID, memberPayloads)
	unitPending, unitConflict, memberPending, memberConflict := s.collectMappingStats(ctx, tenantUUID, sourceAccountUUID, channelAccount.AccountUUID)
	logUpdates := map[string]any{
		"status":           status,
		"message":          message,
		"units_total":      int(unitTotal),
		"members_total":    int(memberTotal),
		"units_new":        int(unitNew),
		"members_new":      int(memberNew),
		"units_updated":    int(unitUpdated),
		"members_updated":  int(memberUpdated),
		"units_pending":    int(unitPending),
		"members_pending":  int(memberPending),
		"units_conflict":   int(unitConflict),
		"members_conflict": int(memberConflict),
		"duration_ms":      duration,
		"updated_at":       now,
	}
	if status == model.SyncStatusSuccess {
		logUpdates["progress_total"] = int(memberTotal)
		logUpdates["progress_current"] = int(memberTotal)
		logUpdates["progress_percent"] = 100
		logUpdates["stage"] = "done"
	} else {
		logUpdates["stage"] = "failed"
	}
	if err := s.repo.WithTenantTx(ctx, tenantUUID, func(tx *gorm.DB) error {
		res := tx.Model(&model.SourceAccount{}).
			Where("tenant_uuid = ? AND source_account_uuid = ?", tenantUUID, sourceAccountUUID).
			Updates(updates)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return orgrepo.ErrSourceAccountNotFound
		}
		if status == model.SyncStatusSuccess {
			if err := upsertSourceUnits(ctx, tx, tenantUUID, sourceAccountUUID, channelAccount.AccountUUID, unitPayloads); err != nil {
				return err
			}
			if err := upsertSourceMembers(ctx, tx, tenantUUID, sourceAccountUUID, channelAccount.AccountUUID, memberPayloads); err != nil {
				return err
			}
			if err := upsertSourceMemberUnits(ctx, tx, tenantUUID, sourceAccountUUID, channelAccount.AccountUUID, memberPayloads); err != nil {
				return err
			}
		}
		if s.logRepo != nil {
			if syncLogUUID != "" {
				if err := tx.Model(&model.SyncLog{}).
					Where("tenant_uuid = ? AND sync_log_uuid = ?", tenantUUID, syncLogUUID).
					Updates(logUpdates).Error; err != nil {
					return err
				}
			} else {
				logRecord := &model.SyncLog{
					TenantUUID:         tenantUUID,
					SourceAccountUUID:  sourceAccountUUID,
					ChannelAccountUUID: channelAccount.AccountUUID,
					Status:             status,
					Message:            message,
					UnitsTotal:         int(unitTotal),
					MembersTotal:       int(memberTotal),
					UnitsNew:           int(unitNew),
					MembersNew:         int(memberNew),
					UnitsUpdated:       int(unitUpdated),
					MembersUpdated:     int(memberUpdated),
					UnitsPending:       int(unitPending),
					MembersPending:     int(memberPending),
					UnitsConflict:      int(unitConflict),
					MembersConflict:    int(memberConflict),
					DurationMs:         duration,
					Stage:              "done",
					ProgressTotal:      int(memberTotal),
					ProgressCurrent:    int(memberTotal),
					ProgressPercent:    100,
					CreatedAt:          now,
					UpdatedAt:          now,
				}
				if err := tx.Create(logRecord).Error; err != nil {
					return err
				}
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}
	if status == model.SyncStatusSuccess {
		s.publishProgress(ctx, tenantUUID, sourceAccountUUID, syncLogUUID, status, "done", message, int(memberTotal), int(memberTotal), 100, duration)
	} else {
		s.publishProgress(ctx, tenantUUID, sourceAccountUUID, syncLogUUID, status, "failed", message, 0, int(memberTotal), currentPercent, duration)
	}
	logger.WithFields(logger.Fields{
		"component":            "org_sync",
		"trace_id":             traceID,
		"tenant_uuid":          tenantUUID,
		"source_account_uuid":  sourceAccountUUID,
		"channel_account_uuid": channelAccount.AccountUUID,
		"status":               status,
		"stage":                "trigger_end",
		"duration_ms":          time.Since(startedAt).Milliseconds(),
		"members_total":        memberTotal,
		"units_total":          unitTotal,
		"sync_log_uuid":        syncLogUUID,
	}).Info("org sync trigger completed")
	account.LastSyncAt = &now
	account.LastSyncStatus = status
	account.LastSyncMessage = message
	account.UpdatedAt = now
	orgobs.EmitSourceSyncTriggered(ctx, tenantUUID, orgobs.ResolveActorUserUUID(ctx, ""), sourceAccountUUID)
	return account, nil
}

func (s *SyncService) SetDelegatedScope(ctx context.Context, tenantUUID, sourceAccountUUID string, allowUser []string, allowParty []int, allowTag []int) (*DelegatedScopeSetResult, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("source account repository not configured")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	sourceAccountUUID = strings.ToLower(strings.TrimSpace(sourceAccountUUID))
	if tenantUUID == "" || sourceAccountUUID == "" {
		return nil, repository.ErrTenantUuidRequired
	}
	channelAccount, err := s.loadChannelAccount(ctx, tenantUUID, sourceAccountUUID)
	if err != nil {
		return nil, err
	}
	credentials := s.mergeDelegatedCredentialsFromPlatform(ctx, credentialsToMap(channelAccount.Credentials))
	if detectWeComAuthMode(credentials) != weComAuthModeDelegatedTemplate {
		return nil, errors.New("当前账号不是代开发授权模式，无法设置可见范围")
	}

	templateID := strings.TrimSpace(credentials["template_id"])
	templateSecret := strings.TrimSpace(credentials["template_secret"])
	configuredTemplateTicket := strings.TrimSpace(credentials["template_ticket"])
	templateTicket := configuredTemplateTicket
	providerCorpID := strings.TrimSpace(credentials["provider_corpid"])
	providerSecret := strings.TrimSpace(credentials["provider_secret"])
	callback := strings.TrimSpace(credentials["oauth_callback"])
	corpID := strings.TrimSpace(credentials["corp_id"])
	permanentCode := strings.TrimSpace(credentials["permanent_code"])

	if s.openworkRepo == nil {
		return nil, errors.New("代开发组织同步依赖缺失：openwork repository 未配置")
	}
	binding, err := s.openworkRepo.ResolveBindingByChannelAccount(ctx, tenantUUID, channelAccount.AccountUUID)
	if err != nil {
		return nil, err
	}
	if binding == nil || strings.TrimSpace(binding.Status) != socialModel.WeComAuthBindingStatusActive {
		return nil, errors.New("代开发组织同步缺少 active 授权绑定，请先完成授权")
	}
	corpID = strings.TrimSpace(binding.CorpID)
	permanentCode = strings.TrimSpace(binding.PermanentCode)
	if templateID == "" {
		templateID = strings.TrimSpace(binding.SuiteID)
	}
	if callback == "" {
		callback = "http://localhost"
	}
	if templateID == "" || templateSecret == "" {
		return nil, errors.New("代开发组织同步缺少 template_id/template_secret")
	}
	if s != nil && s.openworkRepo != nil && strings.TrimSpace(tenantUUID) != "" {
		latest, err := s.openworkRepo.GetLatestSuiteTicket(ctx, tenantUUID, templateID)
		if err != nil {
			return nil, err
		}
		latest = strings.TrimSpace(latest)
		if latest != "" {
			templateTicket = latest
		}
	}
	if templateTicket == "" {
		return nil, errors.New("代开发组织同步缺少 template_ticket（请先确认回调入库或平台配置）")
	}
	if providerCorpID == "" || providerSecret == "" {
		return nil, errors.New("代开发组织同步缺少 provider_corpid/provider_secret")
	}
	if corpID == "" || permanentCode == "" {
		return nil, errors.New("代开发组织同步缺少 corp_id/permanent_code")
	}

	memCache := plcache.NewMemCache("scrm_org_sync_openwork", 10*time.Minute, os.TempDir())
	if memCache == nil {
		return nil, errors.New("代开发组织同步初始化缓存失败")
	}
	httpDebug := parseDelegatedBool(credentials["http_debug"])
	app, err := openwork.NewOpenWork(&openwork.UserConfig{
		AppID:          templateID,
		Secret:         templateSecret,
		ProviderCorpID: providerCorpID,
		ProviderSecret: providerSecret,
		CallbackURL:    callback,
		Cache:          kernel.CacheInterface(memCache),
		HttpDebug:      httpDebug,
		Log: openwork.Log{
			Level:  "debug",
			Stdout: httpDebug,
		},
	})
	if err != nil {
		return nil, err
	}
	suiteTicketComponent, ok := app.GetComponent("SuiteTicket").(*openworksuit.SuiteTicket)
	if !ok || suiteTicketComponent == nil {
		return nil, errors.New("PowerWechat SuiteTicket 组件未初始化")
	}
	if err := suiteTicketComponent.SetTicket(templateTicket); err != nil {
		return nil, err
	}
	workApp, err := app.ProviderClient(corpID, permanentCode, nil)
	if err != nil {
		return nil, err
	}
	agentID, convErr := strconv.Atoi(strings.TrimSpace(binding.AgentID))
	if convErr != nil || agentID <= 0 {
		return nil, errors.New("绑定缺少有效 agent_id，无法设置可见范围")
	}
	if workApp.Agent == nil {
		return nil, errors.New("代开发应用 agent 组件未初始化")
	}

	cleanAllowUser := make([]string, 0, len(allowUser))
	for _, v := range allowUser {
		clean := strings.TrimSpace(v)
		if clean == "" {
			continue
		}
		cleanAllowUser = append(cleanAllowUser, clean)
	}
	cleanAllowParty := make([]int, 0, len(allowParty))
	for _, v := range allowParty {
		if v <= 0 {
			continue
		}
		cleanAllowParty = append(cleanAllowParty, v)
	}
	cleanAllowTag := make([]int, 0, len(allowTag))
	for _, v := range allowTag {
		if v <= 0 {
			continue
		}
		cleanAllowTag = append(cleanAllowTag, v)
	}
	if len(cleanAllowUser) == 0 && len(cleanAllowParty) == 0 && len(cleanAllowTag) == 0 {
		cleanAllowParty = []int{1}
	}

	scopeResp, scopeErr := workApp.Agent.SetScope(ctx, &workagentreq.RequestAgentSetScope{
		AgentID:    agentID,
		AllowUser:  cleanAllowUser,
		AllowParty: cleanAllowParty,
		AllowTag:   cleanAllowTag,
	})
	if scopeErr != nil {
		return nil, scopeErr
	}
	if scopeResp == nil {
		return nil, errors.New("set_scope 返回为空")
	}
	return &DelegatedScopeSetResult{
		AgentID:    agentID,
		AllowUser:  cleanAllowUser,
		AllowParty: cleanAllowParty,
		AllowTag:   cleanAllowTag,
		ErrCode:    scopeResp.ErrCode,
		ErrMsg:     strings.TrimSpace(scopeResp.ErrMsg),
	}, nil
}

func (s *SyncService) ListDelegatedScopeCandidates(ctx context.Context, tenantUUID string) ([]*DelegatedScopeCandidate, error) {
	if s == nil || s.repo == nil || s.repo.DB == nil {
		return nil, errors.New("repository database is not initialized")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	if tenantUUID == "" {
		return nil, repository.ErrTenantUuidRequired
	}

	var bindings []socialModel.WeComOpenAuthBinding
	if err := s.repo.DB.WithContext(ctx).
		Where("tenant_uuid = ? AND status = ? AND channel_account_uuid IS NOT NULL AND agent_id <> ''", tenantUUID, socialModel.WeComAuthBindingStatusActive).
		Order("is_default DESC, updated_at DESC").
		Find(&bindings).Error; err != nil {
		return nil, err
	}
	if len(bindings) == 0 {
		return []*DelegatedScopeCandidate{}, nil
	}

	accountUUIDs := make([]string, 0, len(bindings))
	seen := map[string]struct{}{}
	for _, b := range bindings {
		uuid := strings.ToLower(strings.TrimSpace(b.ChannelAccountUUID))
		if uuid == "" {
			continue
		}
		if _, ok := seen[uuid]; ok {
			continue
		}
		seen[uuid] = struct{}{}
		accountUUIDs = append(accountUUIDs, uuid)
	}
	if len(accountUUIDs) == 0 {
		return []*DelegatedScopeCandidate{}, nil
	}

	var accounts []socialModel.ChannelAccount
	if err := s.repo.DB.WithContext(ctx).
		Where("tenant_uuid = ? AND account_uuid IN ? AND status IN ?", tenantUUID, accountUUIDs, []string{
			socialModel.ChannelAccountStatusConnected,
			"active",
		}).
		Find(&accounts).Error; err != nil {
		return nil, err
	}
	accountMap := make(map[string]socialModel.ChannelAccount, len(accounts))
	for _, account := range accounts {
		key := strings.ToLower(strings.TrimSpace(account.AccountUUID))
		if key == "" {
			continue
		}
		accountMap[key] = account
	}
	if len(accountMap) == 0 {
		return []*DelegatedScopeCandidate{}, nil
	}

	bestByAccount := map[string]*DelegatedScopeCandidate{}
	for _, b := range bindings {
		key := strings.ToLower(strings.TrimSpace(b.ChannelAccountUUID))
		account, ok := accountMap[key]
		if !ok {
			continue
		}
		candidate := &DelegatedScopeCandidate{
			SourceAccountUUID: key,
			DisplayName:       strings.TrimSpace(account.DisplayName),
			CorpID:            strings.TrimSpace(b.CorpID),
			CorpName:          strings.TrimSpace(b.CorpName),
			AgentID:           strings.TrimSpace(b.AgentID),
			BindingStatus:     strings.TrimSpace(b.Status),
			IsDefault:         b.IsDefault,
			AccountStatus:     strings.TrimSpace(account.Status),
			OrgSyncDefault:    account.OrgSyncDefault,
			UpdatedAt:         b.UpdatedAt.UTC().Format(time.RFC3339),
		}
		prev := bestByAccount[key]
		if prev == nil {
			bestByAccount[key] = candidate
			continue
		}
		if candidate.OrgSyncDefault && !prev.OrgSyncDefault {
			bestByAccount[key] = candidate
			continue
		}
		if candidate.IsDefault && !prev.IsDefault {
			bestByAccount[key] = candidate
			continue
		}
		if candidate.UpdatedAt > prev.UpdatedAt {
			bestByAccount[key] = candidate
		}
	}

	result := make([]*DelegatedScopeCandidate, 0, len(bestByAccount))
	for _, item := range bestByAccount {
		result = append(result, item)
	}
	sort.Slice(result, func(i, j int) bool {
		a := result[i]
		b := result[j]
		if a.OrgSyncDefault != b.OrgSyncDefault {
			return a.OrgSyncDefault
		}
		if a.IsDefault != b.IsDefault {
			return a.IsDefault
		}
		if a.UpdatedAt != b.UpdatedAt {
			return a.UpdatedAt > b.UpdatedAt
		}
		return a.SourceAccountUUID < b.SourceAccountUUID
	})
	return result, nil
}

const (
	weComAuthModeAppDetail         = "app_detail"
	weComAuthModeDelegatedTemplate = "delegated_template"
)

func detectWeComAuthMode(credentials map[string]string) string {
	if strings.TrimSpace(credentials["template_id"]) != "" ||
		strings.TrimSpace(credentials["provider_corpid"]) != "" ||
		strings.TrimSpace(credentials["provider_secret"]) != "" ||
		strings.TrimSpace(credentials["permanent_code"]) != "" {
		return weComAuthModeDelegatedTemplate
	}
	if strings.TrimSpace(credentials["app_secret"]) != "" {
		return weComAuthModeAppDetail
	}
	return weComAuthModeAppDetail
}

func (s *SyncService) handleDelegatedTemplateSync(ctx context.Context, tenantUUID, sourceAccountUUID string, account *model.SourceAccount, channelAccountUUID string, credentials map[string]string) (*model.SourceAccount, error) {
	now := time.Now().UTC()
	traceID := traceIDFromContext(ctx)
	syncLogUUID := ""
	if s.logRepo != nil && s.logRepo.DB != nil {
		logRecord := &model.SyncLog{
			TenantUUID:         tenantUUID,
			SourceAccountUUID:  sourceAccountUUID,
			ChannelAccountUUID: channelAccountUUID,
			Status:             model.SyncStatusRunning,
			Message:            "代开发组织同步中",
			Stage:              "delegated_init",
			CreatedAt:          now,
			UpdatedAt:          now,
		}
		if err := s.logRepo.DB.WithContext(ctx).Create(logRecord).Error; err == nil {
			syncLogUUID = logRecord.SyncLogUUID
		}
	}
	updateSyncLog := func(updates map[string]any) {
		if s.logRepo == nil || syncLogUUID == "" || updates == nil {
			return
		}
		if _, ok := updates["updated_at"]; !ok {
			updates["updated_at"] = time.Now().UTC()
		}
		_ = s.logRepo.UpdateByUUID(ctx, tenantUUID, syncLogUUID, updates)
	}

	s.publishProgress(ctx, tenantUUID, sourceAccountUUID, syncLogUUID, model.SyncStatusRunning, "delegated_init", "代开发组织同步中", 0, 0, 5, 0)
	logger.WithFields(logger.Fields{
		"component":            "org_sync",
		"trace_id":             traceID,
		"tenant_uuid":          tenantUUID,
		"source_account_uuid":  sourceAccountUUID,
		"channel_account_uuid": channelAccountUUID,
		"status":               model.SyncStatusRunning,
		"stage":                "delegated_init",
		"sync_log_uuid":        syncLogUUID,
	}).Info("org sync delegated mode started")
	updateSyncLog(map[string]any{
		"status":           model.SyncStatusRunning,
		"stage":            "delegated_fetch",
		"progress_percent": 20,
	})
	s.publishProgress(ctx, tenantUUID, sourceAccountUUID, syncLogUUID, model.SyncStatusRunning, "delegated_fetch", "正在拉取企业微信组织数据", 0, 0, 20, 0)

	startTime := time.Now()
	unitPayloads, memberPayloads, fetchErr := s.fetchDelegatedOrgPayloads(ctx, tenantUUID, channelAccountUUID, credentials)
	status := model.SyncStatusSuccess
	message := "同步完成"
	if fetchErr != nil {
		status = model.SyncStatusFailed
		message = fetchErr.Error()
		logger.WithFields(logger.Fields{
			"component":            "org_sync",
			"trace_id":             traceID,
			"tenant_uuid":          tenantUUID,
			"source_account_uuid":  sourceAccountUUID,
			"channel_account_uuid": channelAccountUUID,
			"status":               status,
			"stage":                "delegated_fetch",
			"sync_log_uuid":        syncLogUUID,
		}).WithError(fetchErr).Error("org sync delegated fetch failed")
	}
	duration := time.Since(startTime).Milliseconds()
	if status == model.SyncStatusSuccess && len(unitPayloads) == 0 && len(memberPayloads) == 0 {
		message = "同步完成，但代开发接口返回空组织数据（部门/成员均为 0）。请检查授权范围与通讯录可见范围。"
	}

	if status == model.SyncStatusSuccess {
		updateSyncLog(map[string]any{
			"stage":            "persist",
			"progress_percent": 90,
		})
		s.publishProgress(ctx, tenantUUID, sourceAccountUUID, syncLogUUID, model.SyncStatusRunning, "persist", "正在写入组织数据", 0, 0, 90, 0)
	}

	updates := map[string]any{
		"last_sync_at":      now,
		"last_sync_status":  status,
		"last_sync_message": message,
		"updated_at":        now,
	}
	unitTotal := int64(len(unitPayloads))
	memberTotal := int64(len(memberPayloads))
	unitNew, unitUpdated := s.classifyUnitChanges(ctx, tenantUUID, sourceAccountUUID, unitPayloads)
	memberNew, memberUpdated := s.classifyMemberChanges(ctx, tenantUUID, sourceAccountUUID, memberPayloads)
	unitPending, unitConflict, memberPending, memberConflict := s.collectMappingStats(ctx, tenantUUID, sourceAccountUUID, channelAccountUUID)
	logUpdates := map[string]any{
		"status":           status,
		"message":          message,
		"units_total":      int(unitTotal),
		"members_total":    int(memberTotal),
		"units_new":        int(unitNew),
		"members_new":      int(memberNew),
		"units_updated":    int(unitUpdated),
		"members_updated":  int(memberUpdated),
		"units_pending":    int(unitPending),
		"members_pending":  int(memberPending),
		"units_conflict":   int(unitConflict),
		"members_conflict": int(memberConflict),
		"duration_ms":      duration,
		"updated_at":       now,
	}
	if status == model.SyncStatusSuccess {
		logUpdates["progress_total"] = int(memberTotal)
		logUpdates["progress_current"] = int(memberTotal)
		logUpdates["progress_percent"] = 100
		logUpdates["stage"] = "done"
	} else {
		logUpdates["stage"] = "failed"
	}

	if err := s.repo.WithTenantTx(ctx, tenantUUID, func(tx *gorm.DB) error {
		res := tx.Model(&model.SourceAccount{}).
			Where("tenant_uuid = ? AND source_account_uuid = ?", tenantUUID, sourceAccountUUID).
			Updates(updates)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return orgrepo.ErrSourceAccountNotFound
		}
		if status == model.SyncStatusSuccess {
			if err := upsertSourceUnits(ctx, tx, tenantUUID, sourceAccountUUID, channelAccountUUID, unitPayloads); err != nil {
				return err
			}
			if err := upsertSourceMembers(ctx, tx, tenantUUID, sourceAccountUUID, channelAccountUUID, memberPayloads); err != nil {
				return err
			}
			if err := upsertSourceMemberUnits(ctx, tx, tenantUUID, sourceAccountUUID, channelAccountUUID, memberPayloads); err != nil {
				return err
			}
		}
		if s.logRepo != nil {
			if syncLogUUID != "" {
				if err := tx.Model(&model.SyncLog{}).
					Where("tenant_uuid = ? AND sync_log_uuid = ?", tenantUUID, syncLogUUID).
					Updates(logUpdates).Error; err != nil {
					return err
				}
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}

	if account == nil {
		account = &model.SourceAccount{
			SourceAccountUUID:  sourceAccountUUID,
			TenantUUID:         tenantUUID,
			ChannelAccountUUID: &channelAccountUUID,
		}
	}
	account.LastSyncAt = &now
	account.LastSyncStatus = status
	account.LastSyncMessage = message
	account.UpdatedAt = now
	if status == model.SyncStatusSuccess {
		s.publishProgress(ctx, tenantUUID, sourceAccountUUID, syncLogUUID, status, "done", message, int(memberTotal), int(memberTotal), 100, duration)
	} else {
		s.publishProgress(ctx, tenantUUID, sourceAccountUUID, syncLogUUID, status, "failed", message, 0, int(memberTotal), 20, duration)
	}
	logger.WithFields(logger.Fields{
		"component":            "org_sync",
		"trace_id":             traceID,
		"tenant_uuid":          tenantUUID,
		"source_account_uuid":  sourceAccountUUID,
		"channel_account_uuid": channelAccountUUID,
		"status":               status,
		"stage":                "delegated_end",
		"sync_log_uuid":        syncLogUUID,
		"duration_ms":          duration,
		"members_total":        memberTotal,
		"units_total":          unitTotal,
	}).Info("org sync delegated mode completed")
	orgobs.EmitSourceSyncTriggered(ctx, tenantUUID, orgobs.ResolveActorUserUUID(ctx, ""), sourceAccountUUID)
	return account, nil
}

type delegatedDepartmentListResp struct {
	ErrCode    int                       `json:"errcode"`
	ErrMsg     string                    `json:"errmsg"`
	Department []delegatedDepartmentItem `json:"department"`
}

type delegatedDepartmentItem struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	ParentID int    `json:"parentid"`
	Order    int    `json:"order"`
}

type delegatedUserListResp struct {
	ErrCode  int                `json:"errcode"`
	ErrMsg   string             `json:"errmsg"`
	UserList []delegatedUserDTO `json:"userlist"`
}

type delegatedUserDTO struct {
	UserID         string `json:"userid"`
	Name           string `json:"name"`
	Mobile         string `json:"mobile"`
	Email          string `json:"email"`
	BizMail        string `json:"biz_mail"`
	Position       string `json:"position"`
	Address        string `json:"address"`
	Avatar         string `json:"avatar"`
	Status         int    `json:"status"`
	MainDepartment int    `json:"main_department"`
	Department     []int  `json:"department"`
	Order          []int  `json:"order"`
}

type delegatedUserGetResp struct {
	ErrCode        int    `json:"errcode"`
	ErrMsg         string `json:"errmsg"`
	UserID         string `json:"userid"`
	Name           string `json:"name"`
	Mobile         string `json:"mobile"`
	Email          string `json:"email"`
	BizMail        string `json:"biz_mail"`
	Position       string `json:"position"`
	Address        string `json:"address"`
	Avatar         string `json:"avatar"`
	Status         int    `json:"status"`
	MainDepartment int    `json:"main_department"`
	Department     []int  `json:"department"`
	Order          []int  `json:"order"`
}

func (s *SyncService) fetchDelegatedOrgPayloads(ctx context.Context, tenantUUID, channelAccountUUID string, credentials map[string]string) ([]orgdriver.SourceUnitDTO, []orgdriver.SourceMemberDTO, error) {
	credentials = s.mergeDelegatedCredentialsFromPlatform(ctx, credentials)
	templateID := strings.TrimSpace(credentials["template_id"])
	templateSecret := strings.TrimSpace(credentials["template_secret"])
	configuredTemplateTicket := strings.TrimSpace(credentials["template_ticket"])
	templateTicket := configuredTemplateTicket
	providerCorpID := strings.TrimSpace(credentials["provider_corpid"])
	providerSecret := strings.TrimSpace(credentials["provider_secret"])
	corpID := strings.TrimSpace(credentials["corp_id"])
	permanentCode := strings.TrimSpace(credentials["permanent_code"])
	callback := strings.TrimSpace(credentials["oauth_callback"])
	if s == nil || s.openworkRepo == nil {
		return nil, nil, errors.New("代开发组织同步缺少 openwork 授权仓储")
	}
	if strings.TrimSpace(tenantUUID) == "" || strings.TrimSpace(channelAccountUUID) == "" {
		return nil, nil, errors.New("代开发组织同步缺少 tenant_uuid/channel_account_uuid")
	}
	binding, err := s.openworkRepo.ResolveBindingByChannelAccount(ctx, tenantUUID, channelAccountUUID)
	if err != nil {
		return nil, nil, err
	}
	if binding == nil || strings.TrimSpace(binding.Status) != socialModel.WeComAuthBindingStatusActive {
		return nil, nil, errors.New("代开发组织同步缺少 active 授权绑定，请先完成授权")
	}
	corpID = strings.TrimSpace(binding.CorpID)
	permanentCode = strings.TrimSpace(binding.PermanentCode)
	if templateID == "" {
		templateID = strings.TrimSpace(binding.SuiteID)
	}
	if callback == "" {
		callback = "http://localhost"
	}
	if templateID == "" || templateSecret == "" {
		return nil, nil, errors.New("代开发组织同步缺少 template_id/template_secret")
	}
	latestTemplateTicket := ""
	if s != nil && s.openworkRepo != nil && strings.TrimSpace(tenantUUID) != "" {
		latest, err := s.openworkRepo.GetLatestSuiteTicket(ctx, tenantUUID, templateID)
		if err != nil {
			return nil, nil, err
		}
		latest = strings.TrimSpace(latest)
		if latest != "" {
			latestTemplateTicket = latest
			templateTicket = latestTemplateTicket
		}
	}
	if templateTicket == "" {
		return nil, nil, errors.New("代开发组织同步缺少 template_ticket（请先确认回调入库或平台配置）")
	}
	if providerCorpID == "" || providerSecret == "" {
		return nil, nil, errors.New("代开发组织同步缺少 provider_corpid/provider_secret")
	}
	if corpID == "" || permanentCode == "" {
		return nil, nil, errors.New("代开发组织同步缺少 corp_id/permanent_code")
	}
	// PowerWechat v3.4.38 的 ProviderClient 会强制断言 cache，不可为 nil。
	memCache := plcache.NewMemCache("scrm_org_sync_openwork", 10*time.Minute, os.TempDir())
	if memCache == nil {
		return nil, nil, errors.New("代开发组织同步初始化缓存失败")
	}
	httpDebug := parseDelegatedBool(credentials["http_debug"])
	app, err := openwork.NewOpenWork(&openwork.UserConfig{
		AppID:          templateID,
		Secret:         templateSecret,
		ProviderCorpID: providerCorpID,
		ProviderSecret: providerSecret,
		CallbackURL:    callback,
		Cache:          kernel.CacheInterface(memCache),
		HttpDebug:      httpDebug,
		Log: openwork.Log{
			Level:  "debug",
			Stdout: httpDebug,
		},
	})
	if err != nil {
		return nil, nil, err
	}
	suiteTicketComponent, ok := app.GetComponent("SuiteTicket").(*openworksuit.SuiteTicket)
	if !ok || suiteTicketComponent == nil {
		return nil, nil, errors.New("PowerWechat SuiteTicket 组件未初始化")
	}
	if err := suiteTicketComponent.SetTicket(templateTicket); err != nil {
		return nil, nil, err
	}
	workApp, err := app.ProviderClient(corpID, permanentCode, nil)
	if err != nil {
		return nil, nil, err
	}
	if agentID, convErr := strconv.Atoi(strings.TrimSpace(binding.AgentID)); convErr == nil && agentID > 0 && workApp.Agent != nil {
		scopeResp, scopeErr := workApp.Agent.SetScope(ctx, &workagentreq.RequestAgentSetScope{
			AgentID:    agentID,
			AllowParty: []int{1},
		})
		if scopeErr != nil {
			logger.WithFields(logger.Fields{
				"component":            "org_sync",
				"tenant_uuid":          tenantUUID,
				"channel_account_uuid": channelAccountUUID,
				"agent_id":             agentID,
				"stage":                "delegated_set_scope",
			}).WithError(scopeErr).Warn("org sync delegated set_scope request failed")
		} else if scopeResp != nil && scopeResp.ErrCode != 0 {
			logger.WithFields(logger.Fields{
				"component":            "org_sync",
				"tenant_uuid":          tenantUUID,
				"channel_account_uuid": channelAccountUUID,
				"agent_id":             agentID,
				"stage":                "delegated_set_scope",
				"errcode":              scopeResp.ErrCode,
				"errmsg":               scopeResp.ErrMsg,
			}).Warn("org sync delegated set_scope rejected")
		}
	}
	corpAccessToken, err := workApp.AccessToken.GetToken(ctx, false)
	if err != nil {
		return nil, nil, err
	}
	accessTokenValue := strings.TrimSpace(corpAccessToken.AccessToken)
	if accessTokenValue == "" {
		return nil, nil, errors.New("代开发组织同步获取 corp access_token 失败")
	}
	baseClient, err := kernel.NewBaseClient(workApp, workApp.AccessToken.AccessToken)
	if err != nil {
		return nil, nil, err
	}

	deptResp := &delegatedDepartmentListResp{}
	deptQuery := object.StringMap{
		"id":           "1",
		"access_token": accessTokenValue,
	}
	if _, err := baseClient.HttpGet(ctx, "cgi-bin/department/list", &deptQuery, nil, deptResp); err != nil {
		return nil, nil, err
	}
	if deptResp.ErrCode != 0 {
		return nil, nil, fmt.Errorf("代开发 department/list failed: %d %s", deptResp.ErrCode, deptResp.ErrMsg)
	}

	units := make([]orgdriver.SourceUnitDTO, 0, len(deptResp.Department))
	deptIDs := make([]int, 0, len(deptResp.Department))
	for _, dept := range deptResp.Department {
		if dept.ID <= 0 {
			continue
		}
		deptIDs = append(deptIDs, dept.ID)
		externalID := strconv.Itoa(dept.ID)
		var parentID *string
		if dept.ParentID > 0 {
			val := strconv.Itoa(dept.ParentID)
			parentID = &val
		}
		units = append(units, orgdriver.SourceUnitDTO{
			ExternalUnitID:       externalID,
			ParentExternalUnitID: parentID,
			Name:                 strings.TrimSpace(dept.Name),
			Order:                dept.Order,
			Status:               "active",
		})
	}

	memberMap := map[string]*orgdriver.SourceMemberDTO{}
	for _, deptID := range deptIDs {
		req := object.StringMap{
			"department_id": strconv.Itoa(deptID),
			"fetch_child":   "0",
			"access_token":  accessTokenValue,
		}
		userResp := &delegatedUserListResp{}
		if _, err := baseClient.HttpGet(ctx, "cgi-bin/user/list", &req, nil, userResp); err != nil {
			return nil, nil, err
		}
		if userResp.ErrCode != 0 {
			return nil, nil, fmt.Errorf("代开发 user/list failed: %d %s", userResp.ErrCode, userResp.ErrMsg)
		}
		for _, user := range userResp.UserList {
			userID := strings.TrimSpace(user.UserID)
			if userID == "" {
				continue
			}
			dto := memberMap[userID]
			if dto == nil {
				dto = &orgdriver.SourceMemberDTO{
					ExternalMemberID: userID,
					DepartmentOrders: map[string]int{},
				}
				memberMap[userID] = dto
			}
			if name := strings.TrimSpace(user.Name); dto.Name == "" && name != "" {
				dto.Name = name
			}
			if phone := strings.TrimSpace(user.Mobile); dto.Phone == "" && phone != "" {
				dto.Phone = phone
			}
			if email := strings.TrimSpace(user.Email); dto.Email == "" && email != "" {
				dto.Email = email
			}
			if biz := strings.TrimSpace(user.BizMail); dto.BizMail == "" && biz != "" {
				dto.BizMail = biz
			}
			if pos := strings.TrimSpace(user.Position); dto.Position == "" && pos != "" {
				dto.Position = pos
			}
			if addr := strings.TrimSpace(user.Address); dto.Address == "" && addr != "" {
				dto.Address = addr
			}
			if avatar := strings.TrimSpace(user.Avatar); dto.AvatarURL == "" && avatar != "" {
				dto.AvatarURL = avatar
			}
			if user.MainDepartment > 0 {
				dto.MainDepartmentID = strconv.Itoa(user.MainDepartment)
			}
			dto.DepartmentIDs = mergeDeptIDs(dto.DepartmentIDs, user.Department, user.MainDepartment)
			for idx, depID := range user.Department {
				if depID <= 0 || idx >= len(user.Order) {
					continue
				}
				dto.DepartmentOrders[strconv.Itoa(depID)] = user.Order[idx]
			}
			dto.ProfileStatus = resolveDelegatedProfileStatus(dto.Name, dto.Phone, dto.Email)
			dto.Status = resolveDelegatedMemberStatus(user.Status)
		}
	}

	for userID, dto := range memberMap {
		if dto == nil {
			continue
		}
		if strings.TrimSpace(dto.Phone) != "" && strings.TrimSpace(dto.Email) != "" && strings.TrimSpace(dto.BizMail) != "" {
			continue
		}
		req := object.StringMap{
			"userid":       userID,
			"access_token": accessTokenValue,
		}
		detailResp := &delegatedUserGetResp{}
		if _, err := baseClient.HttpGet(ctx, "cgi-bin/user/get", &req, nil, detailResp); err != nil {
			return nil, nil, err
		}
		if detailResp.ErrCode != 0 {
			logger.WithFields(logger.Fields{
				"component":            "org_sync",
				"tenant_uuid":          tenantUUID,
				"channel_account_uuid": channelAccountUUID,
				"external_member_id":   userID,
				"stage":                "delegated_user_get",
				"errcode":              detailResp.ErrCode,
				"errmsg":               detailResp.ErrMsg,
			}).Warn("org sync delegated user/get rejected")
			continue
		}
		if name := strings.TrimSpace(detailResp.Name); name != "" {
			dto.Name = name
		}
		if phone := strings.TrimSpace(detailResp.Mobile); phone != "" {
			dto.Phone = phone
		}
		if email := strings.TrimSpace(detailResp.Email); email != "" {
			dto.Email = email
		}
		if biz := strings.TrimSpace(detailResp.BizMail); biz != "" {
			dto.BizMail = biz
		}
		if pos := strings.TrimSpace(detailResp.Position); pos != "" {
			dto.Position = pos
		}
		if addr := strings.TrimSpace(detailResp.Address); addr != "" {
			dto.Address = addr
		}
		if avatar := strings.TrimSpace(detailResp.Avatar); avatar != "" {
			dto.AvatarURL = avatar
		}
		if detailResp.MainDepartment > 0 {
			dto.MainDepartmentID = strconv.Itoa(detailResp.MainDepartment)
		}
		dto.DepartmentIDs = mergeDeptIDs(dto.DepartmentIDs, detailResp.Department, detailResp.MainDepartment)
		for idx, depID := range detailResp.Department {
			if depID <= 0 || idx >= len(detailResp.Order) {
				continue
			}
			dto.DepartmentOrders[strconv.Itoa(depID)] = detailResp.Order[idx]
		}
		dto.ProfileStatus = resolveDelegatedProfileStatus(dto.Name, dto.Phone, dto.Email)
		dto.Status = resolveDelegatedMemberStatus(detailResp.Status)
	}

	members := make([]orgdriver.SourceMemberDTO, 0, len(memberMap))
	for _, member := range memberMap {
		if member == nil || strings.TrimSpace(member.ExternalMemberID) == "" {
			continue
		}
		if member.Status == "" {
			member.Status = "active"
		}
		if member.ProfileStatus == "" {
			member.ProfileStatus = resolveDelegatedProfileStatus(member.Name, member.Phone, member.Email)
		}
		if member.DepartmentOrders == nil {
			member.DepartmentOrders = map[string]int{}
		}
		members = append(members, *member)
	}
	return units, members, nil
}

func (s *SyncService) mergeDelegatedCredentialsFromPlatform(ctx context.Context, input map[string]string) map[string]string {
	out := make(map[string]string, len(input)+8)
	for k, v := range input {
		out[k] = strings.TrimSpace(v)
	}
	if s == nil || s.platformRepo == nil {
		return out
	}
	record, err := s.platformRepo.GetByChannelProvider(ctx, "wechat", "openwork")
	if err != nil || record == nil || record.Config == nil {
		return out
	}

	cfg := record.Config
	defaultTemplateID := strings.TrimSpace(fmt.Sprintf("%v", cfg["default_template_id"]))
	if defaultTemplateID == "" {
		defaultTemplateID = strings.TrimSpace(fmt.Sprintf("%v", cfg["template_id"]))
	}

	pick := map[string]any{}
	if rows, ok := cfg["templates"].([]any); ok {
		for _, raw := range rows {
			row, ok := raw.(map[string]any)
			if !ok || row == nil {
				continue
			}
			rowTpl := strings.TrimSpace(fmt.Sprintf("%v", row["template_id"]))
			if rowTpl == "" {
				continue
			}
			if len(pick) == 0 {
				pick = row
			}
			if defaultTemplateID != "" && rowTpl == defaultTemplateID {
				pick = row
				break
			}
		}
	}

	applyIfMissing := func(key string, values ...string) {
		if strings.TrimSpace(out[key]) != "" {
			return
		}
		for _, val := range values {
			val = strings.TrimSpace(val)
			if val != "" {
				out[key] = val
				return
			}
		}
	}
	applyIfMissing("template_id",
		strings.TrimSpace(fmt.Sprintf("%v", pick["template_id"])),
		strings.TrimSpace(fmt.Sprintf("%v", cfg["template_id"])),
	)
	applyIfMissing("template_secret",
		strings.TrimSpace(fmt.Sprintf("%v", pick["template_secret"])),
		strings.TrimSpace(fmt.Sprintf("%v", cfg["template_secret"])),
	)
	applyIfMissing("template_ticket",
		strings.TrimSpace(fmt.Sprintf("%v", pick["template_ticket"])),
		strings.TrimSpace(fmt.Sprintf("%v", cfg["template_ticket"])),
	)
	applyIfMissing("provider_corpid",
		strings.TrimSpace(fmt.Sprintf("%v", pick["provider_corpid"])),
		strings.TrimSpace(fmt.Sprintf("%v", cfg["provider_corpid"])),
	)
	applyIfMissing("provider_secret",
		strings.TrimSpace(fmt.Sprintf("%v", pick["provider_secret"])),
		strings.TrimSpace(fmt.Sprintf("%v", cfg["provider_secret"])),
	)
	applyIfMissing("http_debug",
		strings.TrimSpace(fmt.Sprintf("%v", pick["http_debug"])),
		strings.TrimSpace(fmt.Sprintf("%v", cfg["http_debug"])),
	)
	return out
}

func parseDelegatedBool(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}

func mergeDeptIDs(existing []string, list []int, mainDepartment int) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(existing))
	for _, item := range existing {
		clean := strings.TrimSpace(item)
		if clean == "" {
			continue
		}
		if _, ok := seen[clean]; ok {
			continue
		}
		seen[clean] = struct{}{}
		out = append(out, clean)
	}
	for _, id := range list {
		if id <= 0 {
			continue
		}
		val := strconv.Itoa(id)
		if _, ok := seen[val]; ok {
			continue
		}
		seen[val] = struct{}{}
		out = append(out, val)
	}
	if mainDepartment > 0 {
		val := strconv.Itoa(mainDepartment)
		if _, ok := seen[val]; !ok {
			out = append(out, val)
		}
	}
	return out
}

func resolveDelegatedProfileStatus(name, phone, email string) string {
	if strings.TrimSpace(name) == "" && strings.TrimSpace(phone) == "" && strings.TrimSpace(email) == "" {
		return "limited"
	}
	return "full"
}

func resolveDelegatedMemberStatus(status int) string {
	switch status {
	case 1:
		return "active"
	case 2:
		return "disabled"
	case 4:
		return "inactive"
	case 5:
		return "quit"
	default:
		return "active"
	}
}

func (s *SyncService) collectMappingStats(ctx context.Context, tenantUUID, sourceAccountUUID, channelAccountUUID string) (int64, int64, int64, int64) {
	if s == nil {
		return 0, 0, 0, 0
	}
	var (
		unitsPending    int64
		unitsConflict   int64
		membersPending  int64
		membersConflict int64
	)
	if s.unitMappingRepo != nil {
		if channelAccountUUID != "" {
			if count, err := s.unitMappingRepo.CountByStatusForChannelAccount(ctx, tenantUUID, channelAccountUUID, model.MappingStatusPending); err == nil {
				unitsPending = count
			}
			if count, err := s.unitMappingRepo.CountByStatusForChannelAccount(ctx, tenantUUID, channelAccountUUID, model.MappingStatusConflict); err == nil {
				unitsConflict = count
			}
		} else {
			if count, err := s.unitMappingRepo.CountByStatusForAccount(ctx, tenantUUID, sourceAccountUUID, model.MappingStatusPending); err == nil {
				unitsPending = count
			}
			if count, err := s.unitMappingRepo.CountByStatusForAccount(ctx, tenantUUID, sourceAccountUUID, model.MappingStatusConflict); err == nil {
				unitsConflict = count
			}
		}
	}
	if s.memberMappingRepo != nil {
		if channelAccountUUID != "" {
			if count, err := s.memberMappingRepo.CountByStatusForChannelAccount(ctx, tenantUUID, channelAccountUUID, model.MappingStatusPending); err == nil {
				membersPending = count
			}
			if count, err := s.memberMappingRepo.CountByStatusForChannelAccount(ctx, tenantUUID, channelAccountUUID, model.MappingStatusConflict); err == nil {
				membersConflict = count
			}
		} else {
			if count, err := s.memberMappingRepo.CountByStatusForAccount(ctx, tenantUUID, sourceAccountUUID, model.MappingStatusPending); err == nil {
				membersPending = count
			}
			if count, err := s.memberMappingRepo.CountByStatusForAccount(ctx, tenantUUID, sourceAccountUUID, model.MappingStatusConflict); err == nil {
				membersConflict = count
			}
		}
	}
	return unitsPending, unitsConflict, membersPending, membersConflict
}

func (s *SyncService) resolveDriver(account orgdriver.AccountContext) (orgdriver.OrgSyncDriver, error) {
	if s == nil || s.driverRegistry == nil {
		return nil, fmt.Errorf("driver registry not configured")
	}
	return s.driverRegistry.Resolve(account.ChannelCode, account.AppType)
}

func (s *SyncService) loadChannelAccount(ctx context.Context, tenantUUID, accountUUID string) (*socialModel.ChannelAccount, error) {
	if s == nil || s.repo == nil || s.repo.DB == nil {
		return nil, errors.New("repository database is not initialized")
	}
	var channelAccount socialModel.ChannelAccount
	if err := s.repo.DB.WithContext(ctx).
		Where("tenant_uuid = ? AND account_uuid = ?", tenantUUID, accountUUID).
		First(&channelAccount).Error; err != nil {
		return nil, err
	}
	return &channelAccount, nil
}

func credentialsToMap(input map[string]interface{}) map[string]string {
	out := make(map[string]string, len(input))
	for key, value := range input {
		if value == nil {
			continue
		}
		out[key] = fmt.Sprintf("%v", value)
	}
	return out
}

func upsertSourceUnits(ctx context.Context, tx *gorm.DB, tenantUUID, sourceAccountUUID, channelAccountUUID string, units []orgdriver.SourceUnitDTO) error {
	if len(units) == 0 {
		return nil
	}
	rows := make([]*model.SourceUnit, 0, len(units))
	now := time.Now().UTC()
	for _, unit := range units {
		if strings.TrimSpace(unit.ExternalUnitID) == "" {
			continue
		}
		name := strings.TrimSpace(unit.Name)
		if name == "" {
			name = unit.ExternalUnitID
		}
		status := strings.TrimSpace(unit.Status)
		if status == "" {
			status = "active"
		}
		order := unit.Order
		rows = append(rows, &model.SourceUnit{
			TenantUUID:           tenantUUID,
			SourceAccountUUID:    sourceAccountUUID,
			ChannelAccountUUID:   channelAccountUUID,
			ExternalUnitID:       strings.TrimSpace(unit.ExternalUnitID),
			ParentExternalUnitID: unit.ParentExternalUnitID,
			Name:                 name,
			Order:                order,
			Status:               status,
			UpdatedAt:            now,
		})
	}
	if len(rows) == 0 {
		return nil
	}
	return tx.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "tenant_uuid"}, {Name: "source_account_uuid"}, {Name: "external_unit_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"channel_account_uuid", "parent_external_unit_id", "name", "order", "status", "updated_at"}),
		}).
		Create(&rows).Error
}

func upsertSourceMembers(ctx context.Context, tx *gorm.DB, tenantUUID, sourceAccountUUID, channelAccountUUID string, members []orgdriver.SourceMemberDTO) error {
	if len(members) == 0 {
		return nil
	}
	rows := make([]*model.SourceMember, 0, len(members))
	profileCandidates := make(map[string]*model.SourceMemberProfile)
	now := time.Now().UTC()
	for _, member := range members {
		if strings.TrimSpace(member.ExternalMemberID) == "" {
			continue
		}
		name := strings.TrimSpace(member.Name)
		if name == "" {
			name = member.ExternalMemberID
		}
		profileStatus := strings.TrimSpace(member.ProfileStatus)
		if profileStatus == "" {
			profileStatus = model.ProfileStatusFull
		}
		status := strings.TrimSpace(member.Status)
		if status == "" {
			status = "active"
		}
		if profileStatus == model.ProfileStatusFull {
			profileCandidates[strings.TrimSpace(member.ExternalMemberID)] = &model.SourceMemberProfile{
				TenantUUID:         tenantUUID,
				ChannelAccountUUID: channelAccountUUID,
				ExternalMemberID:   strings.TrimSpace(member.ExternalMemberID),
				Name:               name,
				Phone:              strings.TrimSpace(member.Phone),
				Email:              strings.TrimSpace(member.Email),
				BizMail:            strings.TrimSpace(member.BizMail),
				Position:           strings.TrimSpace(member.Position),
				Address:            strings.TrimSpace(member.Address),
				MainDepartmentID:   strings.TrimSpace(member.MainDepartmentID),
				AvatarURL:          strings.TrimSpace(member.AvatarURL),
				UpdatedAt:          now,
			}
		}
		rows = append(rows, &model.SourceMember{
			TenantUUID:         tenantUUID,
			SourceAccountUUID:  sourceAccountUUID,
			ChannelAccountUUID: channelAccountUUID,
			ExternalMemberID:   strings.TrimSpace(member.ExternalMemberID),
			Name:               name,
			Phone:              strings.TrimSpace(member.Phone),
			Email:              strings.TrimSpace(member.Email),
			ProfileStatus:      profileStatus,
			Status:             status,
			UpdatedAt:          now,
		})
	}
	if len(rows) == 0 {
		return nil
	}
	if err := tx.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "tenant_uuid"}, {Name: "source_account_uuid"}, {Name: "external_member_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"channel_account_uuid", "name", "phone", "email", "profile_status", "status", "updated_at"}),
		}).
		Create(&rows).Error; err != nil {
		return err
	}
	if len(profileCandidates) == 0 {
		return nil
	}
	externalIDs := make([]string, 0, len(profileCandidates))
	for externalID := range profileCandidates {
		if strings.TrimSpace(externalID) == "" {
			continue
		}
		externalIDs = append(externalIDs, externalID)
	}
	if len(externalIDs) == 0 {
		return nil
	}
	var mapping []struct {
		SourceMemberUUID string
		ExternalMemberID string
	}
	if err := tx.WithContext(ctx).
		Model(&model.SourceMember{}).
		Select("source_member_uuid, external_member_id").
		Where("tenant_uuid = ? AND source_account_uuid = ? AND external_member_id IN ?", tenantUUID, sourceAccountUUID, externalIDs).
		Find(&mapping).Error; err != nil {
		return err
	}
	profiles := make([]*model.SourceMemberProfile, 0, len(mapping))
	for _, row := range mapping {
		profile := profileCandidates[row.ExternalMemberID]
		if profile == nil || strings.TrimSpace(row.SourceMemberUUID) == "" {
			continue
		}
		profile.SourceMemberUUID = row.SourceMemberUUID
		profiles = append(profiles, profile)
	}
	if len(profiles) == 0 {
		return nil
	}
	return tx.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "tenant_uuid"}, {Name: "source_member_uuid"}},
			DoUpdates: clause.AssignmentColumns([]string{"channel_account_uuid", "external_member_id", "name", "phone", "email", "biz_mail", "position", "address", "main_department_id", "avatar_url", "updated_at"}),
		}).
		Create(&profiles).Error
}

func upsertSourceMemberUnits(ctx context.Context, tx *gorm.DB, tenantUUID, sourceAccountUUID, channelAccountUUID string, members []orgdriver.SourceMemberDTO) error {
	if len(members) == 0 {
		return nil
	}
	extMemberIDs := make([]string, 0, len(members))
	extUnitSet := map[string]struct{}{}
	memberDeptMap := map[string][]string{}
	memberDeptOrderMap := map[string]map[string]int{}
	for _, member := range members {
		memberID := strings.TrimSpace(member.ExternalMemberID)
		if memberID == "" {
			continue
		}
		if len(member.DepartmentIDs) == 0 {
			continue
		}
		deptIDs := make([]string, 0, len(member.DepartmentIDs))
		seen := map[string]struct{}{}
		for _, deptID := range member.DepartmentIDs {
			clean := strings.TrimSpace(deptID)
			if clean == "" {
				continue
			}
			if _, ok := seen[clean]; ok {
				continue
			}
			seen[clean] = struct{}{}
			deptIDs = append(deptIDs, clean)
			extUnitSet[clean] = struct{}{}
		}
		if len(deptIDs) == 0 {
			continue
		}
		memberDeptMap[memberID] = deptIDs
		if member.DepartmentOrders != nil {
			orderMap := map[string]int{}
			for deptID, order := range member.DepartmentOrders {
				clean := strings.TrimSpace(deptID)
				if clean == "" {
					continue
				}
				orderMap[clean] = order
			}
			if len(orderMap) > 0 {
				memberDeptOrderMap[memberID] = orderMap
			}
		}
		extMemberIDs = append(extMemberIDs, memberID)
	}
	if len(extMemberIDs) == 0 || len(extUnitSet) == 0 {
		return nil
	}
	extUnitIDs := make([]string, 0, len(extUnitSet))
	for id := range extUnitSet {
		extUnitIDs = append(extUnitIDs, id)
	}
	var unitRows []struct {
		SourceUnitUUID string
		ExternalUnitID string
	}
	if err := tx.WithContext(ctx).
		Model(&model.SourceUnit{}).
		Select("source_unit_uuid, external_unit_id").
		Where("tenant_uuid = ? AND source_account_uuid = ? AND external_unit_id IN ?", tenantUUID, sourceAccountUUID, extUnitIDs).
		Find(&unitRows).Error; err != nil {
		return err
	}
	unitMap := map[string]string{}
	for _, row := range unitRows {
		if strings.TrimSpace(row.SourceUnitUUID) == "" || strings.TrimSpace(row.ExternalUnitID) == "" {
			continue
		}
		unitMap[row.ExternalUnitID] = row.SourceUnitUUID
	}
	if len(unitMap) == 0 {
		return nil
	}
	var memberRows []struct {
		SourceMemberUUID string
		ExternalMemberID string
	}
	if err := tx.WithContext(ctx).
		Model(&model.SourceMember{}).
		Select("source_member_uuid, external_member_id").
		Where("tenant_uuid = ? AND source_account_uuid = ? AND external_member_id IN ?", tenantUUID, sourceAccountUUID, extMemberIDs).
		Find(&memberRows).Error; err != nil {
		return err
	}
	memberMap := map[string]string{}
	for _, row := range memberRows {
		if strings.TrimSpace(row.SourceMemberUUID) == "" || strings.TrimSpace(row.ExternalMemberID) == "" {
			continue
		}
		memberMap[row.ExternalMemberID] = row.SourceMemberUUID
	}
	if len(memberMap) == 0 {
		return nil
	}
	rows := make([]*model.SourceMemberUnit, 0, len(memberRows)*2)
	now := time.Now().UTC()
	for externalMemberID, deptIDs := range memberDeptMap {
		memberUUID := memberMap[externalMemberID]
		if memberUUID == "" {
			continue
		}
		deptOrderMap := memberDeptOrderMap[externalMemberID]
		for _, deptID := range deptIDs {
			unitUUID := unitMap[deptID]
			if unitUUID == "" {
				continue
			}
			order := 0
			if deptOrderMap != nil {
				if value, ok := deptOrderMap[deptID]; ok {
					order = value
				}
			}
			rows = append(rows, &model.SourceMemberUnit{
				TenantUUID:         tenantUUID,
				SourceAccountUUID:  sourceAccountUUID,
				ChannelAccountUUID: channelAccountUUID,
				SourceMemberUUID:   memberUUID,
				SourceUnitUUID:     unitUUID,
				ExternalMemberID:   externalMemberID,
				ExternalUnitID:     deptID,
				Order:              order,
				UpdatedAt:          now,
			})
		}
	}
	if len(rows) == 0 {
		return nil
	}
	if err := tx.WithContext(ctx).
		Where("tenant_uuid = ? AND source_account_uuid = ?", tenantUUID, sourceAccountUUID).
		Delete(&model.SourceMemberUnit{}).Error; err != nil {
		return err
	}
	return tx.WithContext(ctx).Create(&rows).Error
}

func (s *SyncService) classifyUnitChanges(ctx context.Context, tenantUUID, sourceAccountUUID string, units []orgdriver.SourceUnitDTO) (int64, int64) {
	if s == nil || s.repo == nil || s.repo.DB == nil || len(units) == 0 {
		return 0, 0
	}
	ids := make([]string, 0, len(units))
	for _, unit := range units {
		if strings.TrimSpace(unit.ExternalUnitID) != "" {
			ids = append(ids, strings.TrimSpace(unit.ExternalUnitID))
		}
	}
	if len(ids) == 0 {
		return 0, 0
	}
	var existing int64
	if err := s.repo.DB.WithContext(ctx).
		Model(&model.SourceUnit{}).
		Where("tenant_uuid = ? AND source_account_uuid = ? AND external_unit_id IN ?", tenantUUID, sourceAccountUUID, ids).
		Count(&existing).Error; err != nil {
		return 0, int64(len(ids))
	}
	newCount := int64(len(ids)) - existing
	if newCount < 0 {
		newCount = 0
	}
	return newCount, existing
}

func (s *SyncService) classifyMemberChanges(ctx context.Context, tenantUUID, sourceAccountUUID string, members []orgdriver.SourceMemberDTO) (int64, int64) {
	if s == nil || s.repo == nil || s.repo.DB == nil || len(members) == 0 {
		return 0, 0
	}
	ids := make([]string, 0, len(members))
	for _, member := range members {
		if strings.TrimSpace(member.ExternalMemberID) != "" {
			ids = append(ids, strings.TrimSpace(member.ExternalMemberID))
		}
	}
	if len(ids) == 0 {
		return 0, 0
	}
	var existing int64
	if err := s.repo.DB.WithContext(ctx).
		Model(&model.SourceMember{}).
		Where("tenant_uuid = ? AND source_account_uuid = ? AND external_member_id IN ?", tenantUUID, sourceAccountUUID, ids).
		Count(&existing).Error; err != nil {
		return 0, int64(len(ids))
	}
	newCount := int64(len(ids)) - existing
	if newCount < 0 {
		newCount = 0
	}
	return newCount, existing
}

func (s *SyncService) ensureSourceAccount(ctx context.Context, tenantUUID, channelAccountUUID string) (*model.SourceAccount, error) {
	if s == nil || s.repo == nil || s.repo.DB == nil {
		return nil, errors.New("source account repository not configured")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	channelAccountUUID = strings.ToLower(strings.TrimSpace(channelAccountUUID))
	if tenantUUID == "" || channelAccountUUID == "" {
		return nil, repository.ErrTenantUuidRequired
	}
	existing, err := s.repo.FindByChannelAccount(ctx, tenantUUID, channelAccountUUID)
	if err == nil && existing != nil {
		return existing, nil
	}
	if err != nil && !errors.Is(err, orgrepo.ErrSourceAccountNotFound) {
		return nil, err
	}
	var channelAccount socialModel.ChannelAccount
	if err := s.repo.DB.WithContext(ctx).
		Where("tenant_uuid = ? AND account_uuid = ?", tenantUUID, channelAccountUUID).
		First(&channelAccount).Error; err != nil {
		return nil, err
	}
	record := &model.SourceAccount{
		SourceAccountUUID:  channelAccount.AccountUUID,
		TenantUUID:         tenantUUID,
		Provider:           channelAccount.ChannelCode,
		AppType:            channelAccount.AppType,
		ChannelAccountUUID: &channelAccount.AccountUUID,
		DisplayName:        channelAccount.DisplayName,
		Status:             model.SourceAccountStatusActive,
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}
	if err := s.repo.WithTenantTx(ctx, tenantUUID, func(tx *gorm.DB) error {
		return tx.Create(record).Error
	}); err != nil {
		return nil, err
	}
	return record, nil
}
