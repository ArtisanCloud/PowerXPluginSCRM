package acquisition

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	work "github.com/ArtisanCloud/PowerWeChat/v3/src/work"
	pwgroupchatreq "github.com/ArtisanCloud/PowerWeChat/v3/src/work/externalContact/groupChat/request"
	acqmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/models/acquisition"
	acqrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/repository/acquisition"
	"github.com/google/uuid"
)

var (
	ErrGroupLiveCodeServiceNotReady = errors.New("group live code service not ready")
	ErrInvalidGroupLiveCodePayload  = errors.New("invalid group live code payload")
	ErrGroupLiveCodeNoTargetChats   = errors.New("group live code requires at least one target chat")
)

type GroupLiveCodeCreateRequest struct {
	TenantUUID         string
	Channel            string
	AppType            string
	ChannelAccountUUID string
	ActivityName       string
	JoinScene          int
	SkipVerify         bool
	AutoCreateRoom     bool
	ActorUserUUID      string
}

type GroupLiveCodeUpdateRequest struct {
	TenantUUID     string
	GroupCodeUUID  string
	ActivityName   *string
	SkipVerify     *bool
	AutoCreateRoom *bool
	Status         *string
	ActorUserUUID  string
}

type GroupLiveCodeSyncRequest struct {
	TenantUUID    string
	GroupCodeUUID string
	ActorUserUUID string
	ChatIDs       []string
}

type GroupLiveCodeService struct {
	repo            acqrepo.GroupLiveCodeRepository
	chatRepo        acqrepo.GroupChatSnapshotRepository
	accountResolver GroupChatAccountResolver
	providerFactory *GroupChatProviderFactory
}

func NewGroupLiveCodeService(repo acqrepo.GroupLiveCodeRepository, chatRepo acqrepo.GroupChatSnapshotRepository, resolver ...GroupChatAccountResolver) *GroupLiveCodeService {
	factory := NewGroupChatProviderFactory()
	_ = factory.Register("wechat", "wecom", buildWeComSelfBuiltWorkApp)
	_ = factory.Register("wechat", "openwork", buildWeComOpenWorkApp)
	svc := &GroupLiveCodeService{
		repo:            repo,
		chatRepo:        chatRepo,
		providerFactory: factory,
	}
	if len(resolver) > 0 {
		svc.accountResolver = resolver[0]
	}
	return svc
}

func (s *GroupLiveCodeService) Create(ctx context.Context, req GroupLiveCodeCreateRequest) (*acqmodel.GroupLiveCode, error) {
	if s == nil || s.repo == nil {
		return nil, ErrGroupLiveCodeServiceNotReady
	}
	req.TenantUUID = strings.ToLower(strings.TrimSpace(req.TenantUUID))
	req.Channel = strings.ToLower(strings.TrimSpace(req.Channel))
	req.AppType = strings.ToLower(strings.TrimSpace(req.AppType))
	req.ChannelAccountUUID = strings.ToLower(strings.TrimSpace(req.ChannelAccountUUID))
	req.ActivityName = strings.TrimSpace(req.ActivityName)
	req.ActorUserUUID = strings.TrimSpace(req.ActorUserUUID)
	if req.TenantUUID == "" || req.Channel == "" || req.AppType == "" || req.ChannelAccountUUID == "" || req.ActivityName == "" {
		return nil, ErrInvalidGroupLiveCodePayload
	}
	if req.JoinScene <= 0 {
		req.JoinScene = 1
	}
	if req.ActorUserUUID == "" {
		req.ActorUserUUID = "system"
	}
	now := time.Now().UTC()
	item := &acqmodel.GroupLiveCode{
		GroupCodeUUID:      uuid.NewString(),
		TenantUUID:         req.TenantUUID,
		Channel:            req.Channel,
		AppType:            req.AppType,
		ChannelAccountUUID: req.ChannelAccountUUID,
		ActivityName:       req.ActivityName,
		State:              "st-" + strings.ReplaceAll(uuid.NewString(), "-", "")[:16],
		JoinScene:          req.JoinScene,
		SkipVerify:         req.SkipVerify,
		AutoCreateRoom:     req.AutoCreateRoom,
		Status:             acqmodel.LiveCodeStatusActive,
		SyncStatus:         acqmodel.GroupSyncStatusPending,
		CapabilityStatus:   "ready",
		CreatedBy:          req.ActorUserUUID,
		UpdatedBy:          req.ActorUserUUID,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	if err := s.repo.Create(ctx, item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *GroupLiveCodeService) Get(ctx context.Context, tenantUUID, groupCodeUUID string) (*acqmodel.GroupLiveCode, error) {
	if s == nil || s.repo == nil {
		return nil, ErrGroupLiveCodeServiceNotReady
	}
	return s.repo.GetByUUID(ctx, tenantUUID, groupCodeUUID)
}

func (s *GroupLiveCodeService) Update(ctx context.Context, req GroupLiveCodeUpdateRequest) (*acqmodel.GroupLiveCode, error) {
	if s == nil || s.repo == nil {
		return nil, ErrGroupLiveCodeServiceNotReady
	}
	item, err := s.repo.GetByUUID(ctx, req.TenantUUID, req.GroupCodeUUID)
	if err != nil {
		return nil, err
	}
	if req.ActivityName != nil {
		name := strings.TrimSpace(*req.ActivityName)
		if name != "" {
			item.ActivityName = name
		}
	}
	if req.SkipVerify != nil {
		item.SkipVerify = *req.SkipVerify
	}
	if req.AutoCreateRoom != nil {
		item.AutoCreateRoom = *req.AutoCreateRoom
	}
	// 业务策略调整：状态启停不再由编辑入口驱动，统一由同步动作控制远端效果。
	if actor := strings.TrimSpace(req.ActorUserUUID); actor != "" {
		item.UpdatedBy = actor
	}
	if item.UpdatedBy == "" {
		item.UpdatedBy = "system"
	}
	if err := s.repo.Update(ctx, item); err != nil {
		return nil, err
	}
	return s.repo.GetByUUID(ctx, req.TenantUUID, req.GroupCodeUUID)
}

func (s *GroupLiveCodeService) Delete(ctx context.Context, tenantUUID, groupCodeUUID string) error {
	if s == nil || s.repo == nil {
		return ErrGroupLiveCodeServiceNotReady
	}
	return s.repo.Delete(ctx, tenantUUID, groupCodeUUID)
}

func (s *GroupLiveCodeService) Sync(ctx context.Context, req GroupLiveCodeSyncRequest) (*acqmodel.GroupLiveCode, error) {
	if s == nil || s.repo == nil {
		return nil, ErrGroupLiveCodeServiceNotReady
	}
	item, err := s.repo.GetByUUID(ctx, req.TenantUUID, req.GroupCodeUUID)
	if err != nil {
		return nil, err
	}
	chatIDs := normalizeChatIDs(req.ChatIDs)
	if len(chatIDs) == 0 && len(item.TargetChatIDs) > 0 {
		chatIDs = normalizeChatIDs(item.TargetChatIDs)
	}
	if len(chatIDs) == 0 && s.chatRepo != nil {
		chats, listErr := s.chatRepo.ListByChannelAccount(ctx, req.TenantUUID, item.ChannelAccountUUID, 5000)
		if listErr != nil {
			if !isUndefinedRelationErr(listErr) {
				return nil, listErr
			}
			// Missing snapshot table should not block publishing group live code.
			chats = nil
		}
		for _, chat := range chats {
			if chat == nil {
				continue
			}
			chatID := strings.TrimSpace(chat.ChatID)
			if chatID == "" {
				continue
			}
			chatIDs = append(chatIDs, chatID)
		}
		chatIDs = normalizeChatIDs(chatIDs)
	}
	if len(chatIDs) == 0 {
		return nil, ErrGroupLiveCodeNoTargetChats
	}
	if s.accountResolver == nil || s.providerFactory == nil {
		s.applyLocalSyncFallback(item, chatIDs)
	} else {
		if err := s.syncRemoteJoinWays(ctx, item, chatIDs); err != nil {
			s.persistSyncFailure(ctx, item, err)
			return nil, err
		}
	}
	item.Status = acqmodel.LiveCodeStatusActive
	if actor := strings.TrimSpace(req.ActorUserUUID); actor != "" {
		item.UpdatedBy = actor
	}
	if item.UpdatedBy == "" {
		item.UpdatedBy = "system"
	}
	if err := s.repo.Update(ctx, item); err != nil {
		return nil, err
	}
	return s.repo.GetByUUID(ctx, req.TenantUUID, req.GroupCodeUUID)
}

func (s *GroupLiveCodeService) applyLocalSyncFallback(item *acqmodel.GroupLiveCode, chatIDs []string) {
	now := time.Now().UTC()
	targetChatCount := len(chatIDs)
	shardCount := 1
	if targetChatCount > 0 {
		shardCount = int(math.Ceil(float64(targetChatCount) / 5.0))
	}
	shardConfigIDs := buildJoinWayShardConfigIDs(item, shardCount)
	item.SyncStatus = acqmodel.GroupSyncStatusSuccess
	item.LastSyncError = ""
	item.LastSyncedAt = &now
	item.ConfigID = shardConfigIDs[0]
	item.TargetChatCount = targetChatCount
	item.TargetChatIDs = chatIDs
	item.ShardCount = shardCount
	item.CapacityTotal = shardCount * 5
	item.CapacityUsed = targetChatCount
	item.ShardConfigIDs = shardConfigIDs
	item.QRCode = fmt.Sprintf("https://work.weixin.qq.com/qrcode/%s", item.ConfigID)
	item.CapabilityStatus = "ready"
}

func (s *GroupLiveCodeService) syncRemoteJoinWays(ctx context.Context, item *acqmodel.GroupLiveCode, chatIDs []string) error {
	if item == nil {
		return ErrInvalidGroupLiveCodePayload
	}
	rt, err := s.resolveRuntime(ctx, item)
	if err != nil {
		return err
	}
	app, err := s.buildProvider(ctx, rt)
	if err != nil {
		return err
	}
	if app == nil || app.ExternalContactGroupChat == nil {
		return errors.New("wecom group chat client unavailable")
	}
	chatIDs = normalizeChatIDs(chatIDs)
	if len(chatIDs) == 0 {
		return ErrGroupLiveCodeNoTargetChats
	}
	joinScene := item.JoinScene
	if joinScene <= 0 {
		joinScene = 1
	}
	state := strings.TrimSpace(item.State)
	if state == "" {
		state = "st-" + strings.ReplaceAll(uuid.NewString(), "-", "")[:16]
		item.State = state
	}
	remark := strings.TrimSpace(item.ActivityName)
	shards := splitStrings(chatIDs, 5)
	existingConfigs := normalizeConfigIDs(item.ShardConfigIDs)
	if len(existingConfigs) == 0 && strings.TrimSpace(item.ConfigID) != "" {
		existingConfigs = append(existingConfigs, strings.TrimSpace(item.ConfigID))
	}
	newConfigs := make([]string, 0, len(shards))
	for idx, shardChatIDs := range shards {
		configID := ""
		if idx < len(existingConfigs) {
			configID = strings.TrimSpace(existingConfigs[idx])
		}
		autoCreateRoom := 0
		if item.AutoCreateRoom {
			autoCreateRoom = 1
		}
		if configID == "" {
			addRes, addErr := app.ExternalContactGroupChat.AddJoinWay(ctx, &pwgroupchatreq.RequestAddJoinWay{
				Scene:          joinScene,
				Remark:         remark,
				AutoCreateRoom: autoCreateRoom,
				ChatIdList:     shardChatIDs,
				State:          state,
			})
			if addErr != nil {
				return fmt.Errorf("wecom add_join_way failed: %w", addErr)
			}
			if addRes == nil {
				return errors.New("wecom add_join_way failed: empty response")
			}
			if addRes.ErrCode != 0 {
				return fmt.Errorf("wecom add_join_way failed: %d %s", addRes.ErrCode, strings.TrimSpace(addRes.ErrMsg))
			}
			configID = strings.TrimSpace(addRes.ConfigId)
			if configID == "" {
				return errors.New("wecom add_join_way failed: empty config_id")
			}
		} else {
			updateRes, updateErr := app.ExternalContactGroupChat.UpdateJoinWay(ctx, &pwgroupchatreq.RequestUpdateJoinWay{
				ConfigId:       configID,
				Scene:          joinScene,
				Remark:         remark,
				AutoCreateRoom: autoCreateRoom,
				ChatIdList:     shardChatIDs,
				State:          state,
			})
			if updateErr != nil {
				return fmt.Errorf("wecom update_join_way failed: %w", updateErr)
			}
			if updateRes == nil {
				return errors.New("wecom update_join_way failed: empty response")
			}
			if updateRes.ErrCode != 0 {
				return fmt.Errorf("wecom update_join_way failed: %d %s", updateRes.ErrCode, strings.TrimSpace(updateRes.ErrMsg))
			}
		}
		newConfigs = append(newConfigs, configID)
	}
	if len(existingConfigs) > len(newConfigs) {
		for _, staleConfigID := range existingConfigs[len(newConfigs):] {
			staleConfigID = strings.TrimSpace(staleConfigID)
			if staleConfigID == "" {
				continue
			}
			delRes, delErr := app.ExternalContactGroupChat.DelJoinWay(ctx, &pwgroupchatreq.RequestDelJoinWay{ConfigId: staleConfigID})
			if delErr != nil {
				return fmt.Errorf("wecom del_join_way failed: %w", delErr)
			}
			if delRes == nil {
				return errors.New("wecom del_join_way failed: empty response")
			}
			if delRes.ErrCode != 0 {
				return fmt.Errorf("wecom del_join_way failed: %d %s", delRes.ErrCode, strings.TrimSpace(delRes.ErrMsg))
			}
		}
	}
	now := time.Now().UTC()
	item.SyncStatus = acqmodel.GroupSyncStatusSuccess
	item.LastSyncError = ""
	item.LastSyncedAt = &now
	item.ConfigID = newConfigs[0]
	item.TargetChatCount = len(chatIDs)
	item.TargetChatIDs = chatIDs
	item.ShardCount = len(newConfigs)
	item.CapacityTotal = len(newConfigs) * 5
	item.CapacityUsed = len(chatIDs)
	item.ShardConfigIDs = newConfigs
	item.CapabilityStatus = "ready"
	item.QRCode = fmt.Sprintf("https://work.weixin.qq.com/qrcode/%s", item.ConfigID)
	if firstConfig := strings.TrimSpace(item.ConfigID); firstConfig != "" {
		getRes, getErr := app.ExternalContactGroupChat.GetJoinWay(ctx, &pwgroupchatreq.RequestGetJoinWay{ConfigId: firstConfig})
		if getErr == nil && getRes != nil && getRes.ErrCode == 0 {
			if qr := strings.TrimSpace(getRes.JoinWay.QrCode); qr != "" {
				item.QRCode = qr
			}
		}
	}
	return nil
}

func (s *GroupLiveCodeService) disableRemoteJoinWays(ctx context.Context, item *acqmodel.GroupLiveCode) error {
	if item == nil {
		return ErrInvalidGroupLiveCodePayload
	}
	rt, err := s.resolveRuntime(ctx, item)
	if err != nil {
		return err
	}
	app, err := s.buildProvider(ctx, rt)
	if err != nil {
		return err
	}
	if app == nil || app.ExternalContactGroupChat == nil {
		return errors.New("wecom group chat client unavailable")
	}
	configIDs := normalizeConfigIDs(item.ShardConfigIDs)
	if len(configIDs) == 0 && strings.TrimSpace(item.ConfigID) != "" {
		configIDs = append(configIDs, strings.TrimSpace(item.ConfigID))
	}
	if len(configIDs) == 0 {
		now := time.Now().UTC()
		item.SyncStatus = acqmodel.GroupSyncStatusSuccess
		item.LastSyncError = ""
		item.LastSyncedAt = &now
		item.CapabilityStatus = "ready"
		return nil
	}
	failures := make([]string, 0, 2)
	for _, configID := range configIDs {
		res, delErr := app.ExternalContactGroupChat.DelJoinWay(ctx, &pwgroupchatreq.RequestDelJoinWay{ConfigId: configID})
		if delErr != nil {
			failures = append(failures, delErr.Error())
			continue
		}
		if res == nil {
			failures = append(failures, "empty response")
			continue
		}
		if res.ErrCode != 0 {
			failures = append(failures, fmt.Sprintf("%d %s", res.ErrCode, strings.TrimSpace(res.ErrMsg)))
		}
	}
	if len(failures) > 0 {
		return fmt.Errorf("wecom del_join_way failed: %s", strings.Join(failures, " | "))
	}
	now := time.Now().UTC()
	item.SyncStatus = acqmodel.GroupSyncStatusSuccess
	item.LastSyncError = ""
	item.LastSyncedAt = &now
	item.ConfigID = ""
	item.ShardConfigIDs = []string{}
	item.ShardCount = 0
	item.CapacityTotal = 0
	item.CapacityUsed = 0
	item.QRCode = ""
	item.CapabilityStatus = "ready"
	return nil
}

type groupLiveCodeRuntime struct {
	tenantUUID  string
	accountUUID string
	channelCode string
	appType     string
}

func (s *GroupLiveCodeService) resolveRuntime(ctx context.Context, item *acqmodel.GroupLiveCode) (*groupLiveCodeRuntime, error) {
	if s == nil || s.accountResolver == nil {
		return nil, ErrGroupLiveCodeServiceNotReady
	}
	if item == nil {
		return nil, ErrInvalidGroupLiveCodePayload
	}
	tenantUUID := strings.ToLower(strings.TrimSpace(item.TenantUUID))
	accountUUID := strings.ToLower(strings.TrimSpace(item.ChannelAccountUUID))
	if tenantUUID == "" {
		return nil, ErrInvalidGroupLiveCodePayload
	}
	if accountUUID == "" {
		resolved, err := s.accountResolver.ResolveDefaultChannelAccount(ctx, tenantUUID, strings.TrimSpace(item.Channel), strings.TrimSpace(item.AppType))
		if err != nil || strings.TrimSpace(resolved) == "" {
			return nil, ErrDefaultChannelAccountNotFound
		}
		accountUUID = strings.ToLower(strings.TrimSpace(resolved))
		item.ChannelAccountUUID = accountUUID
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
	if appType != "wecom" && appType != "openwork" {
		return nil, fmt.Errorf("%w: channel=%s app_type=%s", ErrUnsupportedChannelAccountType, channelCode, appType)
	}
	return &groupLiveCodeRuntime{
		tenantUUID:  tenantUUID,
		accountUUID: accountUUID,
		channelCode: channelCode,
		appType:     appType,
	}, nil
}

func (s *GroupLiveCodeService) buildProvider(ctx context.Context, rt *groupLiveCodeRuntime) (*work.Work, error) {
	if s == nil || s.accountResolver == nil || s.providerFactory == nil || rt == nil {
		return nil, ErrGroupLiveCodeServiceNotReady
	}
	credentials, err := s.accountResolver.GetChannelAccountCredentials(ctx, rt.tenantUUID, rt.accountUUID)
	if err != nil {
		return nil, err
	}
	workApp, err := s.providerFactory.Build(rt.channelCode, rt.appType, credentials)
	if err != nil {
		return nil, err
	}
	return workApp, nil
}

func (s *GroupLiveCodeService) persistSyncFailure(ctx context.Context, item *acqmodel.GroupLiveCode, err error) {
	if s == nil || s.repo == nil || item == nil || err == nil {
		return
	}
	now := time.Now().UTC()
	item.SyncStatus = acqmodel.GroupSyncStatusFailed
	item.LastSyncError = simplifyGroupLiveCodeError(err)
	item.LastSyncedAt = &now
	if strings.TrimSpace(item.CapabilityStatus) == "" {
		item.CapabilityStatus = "ready"
	}
	_ = s.repo.Update(ctx, item)
}

func simplifyGroupLiveCodeError(err error) string {
	if err == nil {
		return ""
	}
	msg := strings.TrimSpace(err.Error())
	lower := strings.ToLower(msg)
	switch {
	case strings.Contains(lower, "701170"):
		return "企业微信“群活码/进群方式”试用已到期（701170），请在企微后台续期或开通正式能力后再同步"
	case strings.Contains(lower, "81011"):
		return "当前账号缺少客户群活码权限（81011），请在企业微信应用权限中开通“客户群/客户联系”后重试"
	case strings.Contains(lower, "60011"):
		return "当前账号缺少客户群访问权限（60011），请检查应用可见范围与客户联系权限"
	case strings.Contains(lower, "openwork account requires delegated_template credentials"):
		return "代开发账号凭证不完整，请检查模板ID、模板Secret、Provider凭证与永久授权码"
	case strings.Contains(lower, "self-built account requires corp_id and app_secret"):
		return "自建应用凭证不完整，请检查 corp_id 与 app_secret"
	default:
		return msg
	}
}

func normalizeChatIDs(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, raw := range values {
		id := strings.TrimSpace(raw)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func normalizeConfigIDs(values []string) []string {
	return normalizeChatIDs(values)
}

func splitStrings(values []string, size int) [][]string {
	if size <= 0 {
		size = 5
	}
	values = normalizeChatIDs(values)
	if len(values) == 0 {
		return [][]string{}
	}
	chunks := make([][]string, 0, int(math.Ceil(float64(len(values))/float64(size))))
	for idx := 0; idx < len(values); idx += size {
		end := idx + size
		if end > len(values) {
			end = len(values)
		}
		part := make([]string, 0, end-idx)
		part = append(part, values[idx:end]...)
		chunks = append(chunks, part)
	}
	return chunks
}

func buildJoinWayShardConfigIDs(item *acqmodel.GroupLiveCode, shardCount int) []string {
	if shardCount <= 0 {
		shardCount = 1
	}
	base := strings.TrimSpace(item.ConfigID)
	if base == "" {
		base = "cfg-" + strings.ReplaceAll(item.GroupCodeUUID, "-", "")[:12]
	}
	configIDs := make([]string, 0, shardCount)
	for i := 1; i <= shardCount; i++ {
		if i == 1 {
			configIDs = append(configIDs, base)
			continue
		}
		configIDs = append(configIDs, fmt.Sprintf("%s-s%02d", base, i))
	}
	return configIDs
}

func (s *GroupLiveCodeService) List(ctx context.Context, tenantUUID string, limit int) ([]*acqmodel.GroupLiveCode, error) {
	if s == nil || s.repo == nil {
		return []*acqmodel.GroupLiveCode{}, nil
	}
	items, err := s.repo.List(ctx, tenantUUID, limit)
	if err != nil {
		return nil, err
	}
	return items, nil
}
