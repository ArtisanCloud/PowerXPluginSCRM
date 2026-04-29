package social_channel_governance

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ArtisanCloud/PowerSocialite/v3/src/models"
	wecomresp "github.com/ArtisanCloud/PowerSocialite/v3/src/response/weCom"
	pwexternal "github.com/ArtisanCloud/PowerWeChat/v3/src/work/externalContact"
	pwexternalresp "github.com/ArtisanCloud/PowerWeChat/v3/src/work/externalContact/response"
	leadmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/lead_capture"
	leadrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/lead_capture"
	"gorm.io/datatypes"
)

type CustomerTagBindingTag struct {
	TagID     string `json:"tag_id"`
	TagName   string `json:"tag_name"`
	GroupName string `json:"group_name"`
}

type CustomerFollowUserBinding struct {
	UserID        string                  `json:"userid"`
	Remark        string                  `json:"remark"`
	Description   string                  `json:"description"`
	OperUserID    string                  `json:"oper_userid"`
	RemarkMobiles []string                `json:"remark_mobiles"`
	Tags          []CustomerTagBindingTag `json:"tags"`
}

type CustomerTagBindingItem struct {
	LeadUUID          string                      `json:"lead_uuid"`
	DisplayName       string                      `json:"display_name"`
	ExternalUserID    string                      `json:"external_userid"`
	OwnerUserUUID     string                      `json:"owner_user_uuid"`
	SourceAccountUUID string                      `json:"source_account_uuid"`
	ChannelSyncStatus string                      `json:"channel_sync_status"`
	FollowUsers       []CustomerFollowUserBinding `json:"follow_users"`
	RemoteError       string                      `json:"remote_error,omitempty"`
}

type weComExternalContactClient interface {
	Get(ctx context.Context, externalUserID string, cursor string) (*wecomresp.ResponseGetExternalContact, error)
	GetFollowUsers(ctx context.Context) (*pwexternalresp.ResponseGetFollowUserList, error)
	BatchGet(ctx context.Context, userID []string, cursor string, limit int) (*pwexternalresp.ResponseBatchGetByUser, error)
}

type weComExternalContactClientFactory func(appType string, credentials map[string]string) (weComExternalContactClient, error)

type powerWeComExternalContactClient struct {
	client *pwexternal.Client
}

func (c *powerWeComExternalContactClient) Get(ctx context.Context, externalUserID string, cursor string) (*wecomresp.ResponseGetExternalContact, error) {
	return c.client.Get(ctx, externalUserID, cursor)
}

func (c *powerWeComExternalContactClient) GetFollowUsers(ctx context.Context) (*pwexternalresp.ResponseGetFollowUserList, error) {
	return c.client.GetFollowUsers(ctx)
}

func (c *powerWeComExternalContactClient) BatchGet(ctx context.Context, userID []string, cursor string, limit int) (*pwexternalresp.ResponseBatchGetByUser, error) {
	return c.client.BatchGet(ctx, userID, cursor, limit)
}

var defaultWeComExternalContactClientFactory weComExternalContactClientFactory = func(appType string, credentials map[string]string) (weComExternalContactClient, error) {
	app, err := newWeComTagSyncApp("wechat", appType, credentials)
	if err != nil {
		return nil, err
	}
	if app == nil || app.ExternalContact == nil {
		return nil, errors.New("wecom external contact client unavailable")
	}
	return &powerWeComExternalContactClient{client: app.ExternalContact}, nil
}

type CustomerTagBindingService struct {
	leadRepo      *leadrepo.LeadRepository
	tagSyncSvc    *TagSyncService
	clientFactory weComExternalContactClientFactory
}

type cachedRemoteResult struct {
	followUsers []CustomerFollowUserBinding
	remoteError string
}

type cachedRemoteEntry struct {
	expiresAt time.Time
	result    cachedRemoteResult
}

var customerTagRemoteCache = struct {
	mu    sync.RWMutex
	items map[string]cachedRemoteEntry
}{
	items: map[string]cachedRemoteEntry{},
}

const customerTagRemoteCacheTTL = 5 * time.Minute

func NewCustomerTagBindingService(leadRepo *leadrepo.LeadRepository, tagSyncSvc *TagSyncService) *CustomerTagBindingService {
	return &CustomerTagBindingService{
		leadRepo:      leadRepo,
		tagSyncSvc:    tagSyncSvc,
		clientFactory: defaultWeComExternalContactClientFactory,
	}
}

func (s *CustomerTagBindingService) ListByChannel(
	ctx context.Context,
	tenantUUID, channelAccountUUID string,
	limit int,
) ([]CustomerTagBindingItem, error) {
	if s == nil || s.leadRepo == nil || s.tagSyncSvc == nil {
		return nil, errors.New("customer tag binding service unavailable")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	channelAccountUUID = strings.ToLower(strings.TrimSpace(channelAccountUUID))
	if tenantUUID == "" || channelAccountUUID == "" {
		return nil, errors.New("tenant_uuid and channel_account_uuid are required")
	}
	if limit < 0 {
		limit = 0
	}
	if limit > 1000 {
		limit = 1000
	}
	leads, err := s.leadRepo.List(ctx, tenantUUID)
	if err != nil {
		return nil, err
	}
	syncStateByLead, err := s.buildLeadSyncState(ctx, tenantUUID, leads)
	if err != nil {
		return nil, err
	}

	// 远端优先：直接按企业微信全量关系构建视图，保证与企微后台一致。
	// 若远端暂时不可用，再回退本地快照兜底。
	client, clientErr := s.resolveWeComExternalContactClient(ctx, tenantUUID, channelAccountUUID)
	if clientErr == nil && client != nil {
		items, remoteErr := s.listBindingsFromRemote(ctx, tenantUUID, channelAccountUUID, limit, leads, syncStateByLead, client)
		if remoteErr == nil {
			return items, nil
		}
	}

	outCap := limit
	if outCap <= 0 {
		outCap = 64
	}
	out := make([]CustomerTagBindingItem, 0, outCap)
	seenLead := make(map[string]struct{}, outCap)
	for _, lead := range leads {
		if lead == nil {
			continue
		}
		leadUUID := strings.TrimSpace(lead.LeadUUID)
		state := syncStateByLead[leadUUID]
		externalUserID := strings.TrimSpace(state.ExternalUserID)
		if leadUUID == "" || externalUserID == "" {
			continue
		}
		if _, ok := seenLead[leadUUID]; ok {
			continue
		}
		sourceAccountUUID := strings.TrimSpace(state.SourceAccountUUID)
		if sourceAccountUUID != "" && !strings.EqualFold(sourceAccountUUID, channelAccountUUID) {
			continue
		}

		item := CustomerTagBindingItem{
			LeadUUID:          leadUUID,
			DisplayName:       strings.TrimSpace(lead.DisplayName),
			ExternalUserID:    externalUserID,
			OwnerUserUUID:     strings.TrimSpace(lead.OwnerUserUUID),
			SourceAccountUUID: sourceAccountUUID,
			ChannelSyncStatus: strings.TrimSpace(state.ChannelSyncStatus),
			FollowUsers:       append([]CustomerFollowUserBinding{}, state.FollowUsers...),
			RemoteError:       strings.TrimSpace(state.RemoteError),
		}

		out = append(out, item)
		seenLead[leadUUID] = struct{}{}
		if limit > 0 && len(out) >= limit {
			break
		}
	}

	return out, nil
}

func (s *CustomerTagBindingService) listBindingsFromRemote(
	ctx context.Context,
	tenantUUID, channelAccountUUID string,
	limit int,
	leads []*leadmodel.Lead,
	syncStateByLead map[string]leadSyncState,
	client weComExternalContactClient,
) ([]CustomerTagBindingItem, error) {
	if client == nil {
		return nil, errors.New("wecom external contact client unavailable")
	}
	followUsersResp, err := client.GetFollowUsers(ctx)
	if err != nil {
		return nil, err
	}
	if followUsersResp == nil {
		return nil, errors.New("get_follow_user_list empty response")
	}
	if followUsersResp.ErrCode != 0 {
		return nil, fmt.Errorf("get_follow_user_list failed: %d %s", followUsersResp.ErrCode, strings.TrimSpace(followUsersResp.ErrMsg))
	}
	followUsers := compactNonEmptyStrings(followUsersResp.FollowUser)
	if len(followUsers) == 0 {
		return []CustomerTagBindingItem{}, nil
	}

	leadMetaByExternal := buildLeadMetaByExternal(tenantUUID, channelAccountUUID, leads, syncStateByLead)
	itemsByExternal := make(map[string]*CustomerTagBindingItem, 128)
	externalOrder := make([]string, 0, 128)

	cursor := ""
	const pageLimit = 100
	for {
		batchResp, batchErr := client.BatchGet(ctx, followUsers, cursor, pageLimit)
		if batchErr != nil {
			return nil, batchErr
		}
		if batchResp == nil {
			return nil, errors.New("batch_get_by_user empty response")
		}
		if batchResp.ErrCode != 0 {
			return nil, fmt.Errorf("batch_get_by_user failed: %d %s", batchResp.ErrCode, strings.TrimSpace(batchResp.ErrMsg))
		}
		for _, entry := range batchResp.ExternalContactList {
			if entry == nil || entry.ExternalContact == nil {
				continue
			}
			externalUserID := strings.TrimSpace(entry.ExternalContact.ExternalUserID)
			if externalUserID == "" {
				continue
			}
			item, ok := itemsByExternal[externalUserID]
			if !ok {
				item = &CustomerTagBindingItem{
					LeadUUID:          "",
					DisplayName:       strings.TrimSpace(entry.ExternalContact.Name),
					ExternalUserID:    externalUserID,
					OwnerUserUUID:     "",
					SourceAccountUUID: channelAccountUUID,
					ChannelSyncStatus: "synced",
					FollowUsers:       []CustomerFollowUserBinding{},
					RemoteError:       "",
				}
				if meta, exists := leadMetaByExternal[externalUserID]; exists {
					item.LeadUUID = meta.LeadUUID
					item.OwnerUserUUID = meta.OwnerUserUUID
					if strings.TrimSpace(meta.DisplayName) != "" {
						item.DisplayName = strings.TrimSpace(meta.DisplayName)
					}
					if strings.TrimSpace(meta.SourceAccountUUID) != "" {
						item.SourceAccountUUID = strings.TrimSpace(meta.SourceAccountUUID)
					}
				}
				itemsByExternal[externalUserID] = item
				externalOrder = append(externalOrder, externalUserID)
			}
			if entry.FollowInfo != nil {
				mergeFollowUserBinding(item, entry.FollowInfo)
			}
		}
		nextCursor := strings.TrimSpace(batchResp.NextCursor)
		if nextCursor == "" || nextCursor == cursor {
			break
		}
		cursor = nextCursor
	}

	out := make([]CustomerTagBindingItem, 0, len(externalOrder))
	for _, externalUserID := range externalOrder {
		item := itemsByExternal[externalUserID]
		if item == nil {
			continue
		}
		// 过滤掉没有任何跟进关系的记录，避免噪声。
		if len(item.FollowUsers) == 0 {
			continue
		}
		out = append(out, *item)
	}
	sort.SliceStable(out, func(i, j int) bool {
		ni := strings.TrimSpace(out[i].DisplayName)
		nj := strings.TrimSpace(out[j].DisplayName)
		if ni != nj {
			return ni < nj
		}
		return strings.TrimSpace(out[i].ExternalUserID) < strings.TrimSpace(out[j].ExternalUserID)
	})
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

type leadMeta struct {
	LeadUUID          string
	DisplayName       string
	OwnerUserUUID     string
	SourceAccountUUID string
}

func buildLeadMetaByExternal(
	tenantUUID, channelAccountUUID string,
	leads []*leadmodel.Lead,
	syncStateByLead map[string]leadSyncState,
) map[string]leadMeta {
	out := make(map[string]leadMeta, len(leads))
	for _, lead := range leads {
		if lead == nil {
			continue
		}
		leadUUID := strings.TrimSpace(lead.LeadUUID)
		if leadUUID == "" {
			continue
		}
		state := syncStateByLead[leadUUID]
		externalUserID := strings.TrimSpace(state.ExternalUserID)
		if externalUserID == "" {
			continue
		}
		sourceAccountUUID := strings.TrimSpace(state.SourceAccountUUID)
		if sourceAccountUUID != "" && !strings.EqualFold(sourceAccountUUID, channelAccountUUID) {
			continue
		}
		if _, exists := out[externalUserID]; exists {
			continue
		}
		out[externalUserID] = leadMeta{
			LeadUUID:          leadUUID,
			DisplayName:       strings.TrimSpace(lead.DisplayName),
			OwnerUserUUID:     strings.TrimSpace(lead.OwnerUserUUID),
			SourceAccountUUID: strings.TrimSpace(sourceAccountUUID),
		}
	}
	return out
}

func mergeFollowUserBinding(item *CustomerTagBindingItem, follow *models.FollowUser) {
	if item == nil || follow == nil {
		return
	}
	userID := strings.TrimSpace(follow.UserID)
	if userID == "" {
		return
	}
	tags := make([]CustomerTagBindingTag, 0, len(follow.Tags))
	for _, tag := range follow.Tags {
		tagID := strings.TrimSpace(tag.TagID)
		tagName := strings.TrimSpace(tag.TagName)
		groupName := strings.TrimSpace(tag.GroupName)
		if tagID == "" && tagName == "" {
			continue
		}
		tags = append(tags, CustomerTagBindingTag{
			TagID:     tagID,
			TagName:   tagName,
			GroupName: groupName,
		})
	}
	for idx := range item.FollowUsers {
		if strings.TrimSpace(item.FollowUsers[idx].UserID) != userID {
			continue
		}
		item.FollowUsers[idx].Remark = strings.TrimSpace(follow.Remark)
		item.FollowUsers[idx].Description = strings.TrimSpace(follow.Description)
		item.FollowUsers[idx].OperUserID = strings.TrimSpace(follow.OperUserID)
		item.FollowUsers[idx].RemarkMobiles = append([]string{}, follow.RemarkMobiles...)
		item.FollowUsers[idx].Tags = tags
		return
	}
	item.FollowUsers = append(item.FollowUsers, CustomerFollowUserBinding{
		UserID:        userID,
		Remark:        strings.TrimSpace(follow.Remark),
		Description:   strings.TrimSpace(follow.Description),
		OperUserID:    strings.TrimSpace(follow.OperUserID),
		RemarkMobiles: append([]string{}, follow.RemarkMobiles...),
		Tags:          tags,
	})
}

func (s *CustomerTagBindingService) RefreshSnapshotByChannel(
	ctx context.Context,
	tenantUUID, channelAccountUUID string,
) (int, error) {
	if s == nil || s.leadRepo == nil || s.tagSyncSvc == nil {
		return 0, errors.New("customer tag binding service unavailable")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	channelAccountUUID = strings.ToLower(strings.TrimSpace(channelAccountUUID))
	if tenantUUID == "" || channelAccountUUID == "" {
		return 0, errors.New("tenant_uuid and channel_account_uuid are required")
	}
	client, err := s.resolveWeComExternalContactClient(ctx, tenantUUID, channelAccountUUID)
	if err != nil {
		return 0, err
	}
	leads, err := s.leadRepo.List(ctx, tenantUUID)
	if err != nil {
		return 0, err
	}
	syncStateByLead, err := s.buildLeadSyncState(ctx, tenantUUID, leads)
	if err != nil {
		return 0, err
	}
	leadUUIDsByExternalUserID := make(map[string][]string, len(leads))
	externalUserIDs := make([]string, 0, len(leads))
	for _, lead := range leads {
		if lead == nil {
			continue
		}
		leadUUID := strings.TrimSpace(lead.LeadUUID)
		if leadUUID == "" {
			continue
		}
		state := syncStateByLead[leadUUID]
		externalUserID := strings.TrimSpace(state.ExternalUserID)
		if externalUserID == "" {
			continue
		}
		sourceAccountUUID := strings.TrimSpace(state.SourceAccountUUID)
		if sourceAccountUUID != "" && !strings.EqualFold(sourceAccountUUID, channelAccountUUID) {
			continue
		}
		if len(leadUUIDsByExternalUserID[externalUserID]) == 0 {
			externalUserIDs = append(externalUserIDs, externalUserID)
		}
		leadUUIDsByExternalUserID[externalUserID] = append(leadUUIDsByExternalUserID[externalUserID], leadUUID)
	}
	remoteByExternalUserID := fetchRemoteBindingsConcurrently(ctx, client, tenantUUID, channelAccountUUID, externalUserIDs)
	if err := s.persistRemoteSnapshot(ctx, tenantUUID, channelAccountUUID, leadUUIDsByExternalUserID, remoteByExternalUserID); err != nil {
		return 0, err
	}
	return len(externalUserIDs), nil
}

func fetchRemoteBindingsConcurrently(
	ctx context.Context,
	client weComExternalContactClient,
	tenantUUID string,
	channelAccountUUID string,
	externalUserIDs []string,
) map[string]cachedRemoteResult {
	result := make(map[string]cachedRemoteResult, len(externalUserIDs))
	if client == nil || len(externalUserIDs) == 0 {
		return result
	}

	dedup := make([]string, 0, len(externalUserIDs))
	seen := make(map[string]struct{}, len(externalUserIDs))
	for _, id := range externalUserIDs {
		externalUserID := strings.TrimSpace(id)
		if externalUserID == "" {
			continue
		}
		if _, ok := seen[externalUserID]; ok {
			continue
		}
		seen[externalUserID] = struct{}{}
		dedup = append(dedup, externalUserID)
	}
	if len(dedup) == 0 {
		return result
	}

	now := time.Now()
	toFetch := make([]string, 0, len(dedup))
	customerTagRemoteCache.mu.RLock()
	for _, externalUserID := range dedup {
		key := remoteCacheKey(tenantUUID, channelAccountUUID, externalUserID)
		entry, ok := customerTagRemoteCache.items[key]
		if ok && now.Before(entry.expiresAt) {
			result[externalUserID] = entry.result
			continue
		}
		toFetch = append(toFetch, externalUserID)
	}
	customerTagRemoteCache.mu.RUnlock()
	if len(toFetch) == 0 {
		return result
	}

	const workerCount = 6
	var (
		wg sync.WaitGroup
		mu sync.Mutex
		ch = make(chan string)
	)
	worker := func() {
		defer wg.Done()
		for externalUserID := range ch {
			remote := cachedRemoteResult{followUsers: []CustomerFollowUserBinding{}}
			reqCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
			resp, getErr := client.Get(reqCtx, externalUserID, "")
			cancel()
			if getErr != nil {
				remote.remoteError = getErr.Error()
			} else if resp == nil {
				remote.remoteError = "empty response"
			} else if resp.ResponseWeCom != nil && resp.ErrCode != 0 {
				remote.remoteError = fmt.Sprintf("%d %s", resp.ErrCode, strings.TrimSpace(resp.ErrMSG))
			} else {
				remote.followUsers = flattenFollowUsers(resp)
			}
			mu.Lock()
			result[externalUserID] = remote
			mu.Unlock()

			customerTagRemoteCache.mu.Lock()
			customerTagRemoteCache.items[remoteCacheKey(tenantUUID, channelAccountUUID, externalUserID)] = cachedRemoteEntry{
				expiresAt: time.Now().Add(customerTagRemoteCacheTTL),
				result:    remote,
			}
			customerTagRemoteCache.mu.Unlock()
		}
	}

	n := workerCount
	if len(toFetch) < n {
		n = len(toFetch)
	}
	for i := 0; i < n; i++ {
		wg.Add(1)
		go worker()
	}
	for _, externalUserID := range toFetch {
		ch <- externalUserID
	}
	close(ch)
	wg.Wait()
	return result
}

func remoteCacheKey(tenantUUID, channelAccountUUID, externalUserID string) string {
	return strings.ToLower(strings.TrimSpace(tenantUUID)) + "::" +
		strings.ToLower(strings.TrimSpace(channelAccountUUID)) + "::" +
		strings.TrimSpace(externalUserID)
}

func flattenFollowUsers(resp *wecomresp.ResponseGetExternalContact) []CustomerFollowUserBinding {
	if resp == nil {
		return []CustomerFollowUserBinding{}
	}
	out := make([]CustomerFollowUserBinding, 0, len(resp.FollowUsers))
	for _, fu := range resp.FollowUsers {
		if fu == nil {
			continue
		}
		binding := CustomerFollowUserBinding{
			UserID:        strings.TrimSpace(fu.UserID),
			Remark:        strings.TrimSpace(fu.Remark),
			Description:   strings.TrimSpace(fu.Description),
			OperUserID:    strings.TrimSpace(fu.OperUserID),
			RemarkMobiles: append([]string{}, fu.RemarkMobiles...),
			Tags:          []CustomerTagBindingTag{},
		}
		for _, tag := range fu.Tags {
			if strings.TrimSpace(tag.TagID) == "" && strings.TrimSpace(tag.TagName) == "" {
				continue
			}
			binding.Tags = append(binding.Tags, CustomerTagBindingTag{
				TagID:     strings.TrimSpace(tag.TagID),
				TagName:   strings.TrimSpace(tag.TagName),
				GroupName: strings.TrimSpace(tag.GroupName),
			})
		}
		out = append(out, binding)
	}
	return out
}

type leadSyncState struct {
	ExternalUserID    string
	SourceAccountUUID string
	ChannelSyncStatus string
	FollowUsers       []CustomerFollowUserBinding
	RemoteError       string
}

func (s *CustomerTagBindingService) buildLeadSyncState(
	ctx context.Context,
	tenantUUID string,
	leads []*leadmodel.Lead,
) (map[string]leadSyncState, error) {
	states := make(map[string]leadSyncState, len(leads))
	leadUUIDs := make([]string, 0, len(leads))
	for _, lead := range leads {
		if lead == nil {
			continue
		}
		leadUUID := strings.TrimSpace(lead.LeadUUID)
		if leadUUID == "" {
			continue
		}
		leadUUIDs = append(leadUUIDs, leadUUID)
		states[leadUUID] = leadSyncState{
			ExternalUserID:    "",
			SourceAccountUUID: sourceAccountUUIDFromLead(lead),
			ChannelSyncStatus: "unsynced",
			FollowUsers:       []CustomerFollowUserBinding{},
			RemoteError:       "",
		}
	}
	if len(leadUUIDs) == 0 {
		return states, nil
	}

	var activities []leadmodel.LeadActivity
	if err := s.leadRepo.DB.WithContext(ctx).
		Where("tenant_uuid = ? AND activity_type = ? AND lead_uuid IN ?", tenantUUID, leadmodel.LeadActivityTypeSyncTrace, leadUUIDs).
		Order("updated_at DESC").
		Find(&activities).Error; err != nil {
		return nil, err
	}
	seen := make(map[string]struct{}, len(leadUUIDs))
	for _, activity := range activities {
		leadUUID := strings.TrimSpace(activity.LeadUUID)
		if leadUUID == "" {
			continue
		}
		if _, ok := seen[leadUUID]; ok {
			continue
		}
		state := states[leadUUID]
		if externalUserID := payloadStringValue(activity.Payload, "external_lead_id"); externalUserID != "" {
			state.ExternalUserID = externalUserID
			state.ChannelSyncStatus = "synced"
		}
		if state.ExternalUserID == "" {
			if wechatID := payloadStringValue(activity.Payload, "external_wechat_id"); wechatID != "" {
				state.ExternalUserID = wechatID
				state.ChannelSyncStatus = "synced"
			}
		}
		if state.SourceAccountUUID == "" {
			state.SourceAccountUUID = payloadStringValue(activity.Payload, "source_account_uuid")
		}
		state.RemoteError = payloadStringValue(activity.Payload, "customer_follow_users_error")
		state.FollowUsers = parseFollowUsersFromPayload(activity.Payload["customer_follow_users"])
		states[leadUUID] = state
		seen[leadUUID] = struct{}{}
	}
	return states, nil
}

func (s *CustomerTagBindingService) persistRemoteSnapshot(
	ctx context.Context,
	tenantUUID string,
	channelAccountUUID string,
	leadUUIDsByExternalUserID map[string][]string,
	remoteByExternalUserID map[string]cachedRemoteResult,
) error {
	if len(leadUUIDsByExternalUserID) == 0 || len(remoteByExternalUserID) == 0 {
		return nil
	}
	leadUUIDs := make([]string, 0, len(leadUUIDsByExternalUserID))
	for _, ids := range leadUUIDsByExternalUserID {
		leadUUIDs = append(leadUUIDs, ids...)
	}
	var activities []leadmodel.LeadActivity
	if err := s.leadRepo.DB.WithContext(ctx).
		Where("tenant_uuid = ? AND activity_type = ? AND lead_uuid IN ?", tenantUUID, leadmodel.LeadActivityTypeSyncTrace, leadUUIDs).
		Order("updated_at DESC").
		Find(&activities).Error; err != nil {
		return err
	}
	latestByLead := make(map[string]leadmodel.LeadActivity, len(leadUUIDs))
	for _, activity := range activities {
		leadUUID := strings.TrimSpace(activity.LeadUUID)
		if leadUUID == "" {
			continue
		}
		if _, exists := latestByLead[leadUUID]; exists {
			continue
		}
		latestByLead[leadUUID] = activity
	}
	now := time.Now().UTC()
	tx := s.leadRepo.DB.WithContext(ctx)
	for externalUserID, remote := range remoteByExternalUserID {
		for _, leadUUID := range leadUUIDsByExternalUserID[externalUserID] {
			leadUUID = strings.TrimSpace(leadUUID)
			if leadUUID == "" {
				continue
			}
			existing, exists := latestByLead[leadUUID]
			payload := datatypes.JSONMap{}
			if exists {
				for k, v := range existing.Payload {
					payload[k] = v
				}
			}
			payload["external_lead_id"] = strings.TrimSpace(externalUserID)
			payload["source_account_uuid"] = strings.TrimSpace(channelAccountUUID)
			payload["customer_follow_users"] = toFollowUsersPayload(remote.followUsers)
			payload["customer_follow_users_error"] = strings.TrimSpace(remote.remoteError)
			payload["customer_follow_users_synced_at"] = now.Format(time.RFC3339Nano)

			if exists {
				if err := tx.Model(&leadmodel.LeadActivity{}).
					Where("activity_uuid = ?", existing.ActivityUUID).
					Updates(map[string]any{
						"payload":    payload,
						"updated_at": now,
					}).Error; err != nil {
					return err
				}
				continue
			}
			if err := tx.Create(&leadmodel.LeadActivity{
				LeadUUID:     leadUUID,
				TenantUUID:   tenantUUID,
				ActivityType: leadmodel.LeadActivityTypeSyncTrace,
				Payload:      payload,
				CreatedAt:    now,
				UpdatedAt:    now,
			}).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func toFollowUsersPayload(items []CustomerFollowUserBinding) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		tags := make([]map[string]any, 0, len(item.Tags))
		for _, tag := range item.Tags {
			tags = append(tags, map[string]any{
				"tag_id":     strings.TrimSpace(tag.TagID),
				"tag_name":   strings.TrimSpace(tag.TagName),
				"group_name": strings.TrimSpace(tag.GroupName),
			})
		}
		out = append(out, map[string]any{
			"userid":         strings.TrimSpace(item.UserID),
			"remark":         strings.TrimSpace(item.Remark),
			"description":    strings.TrimSpace(item.Description),
			"oper_userid":    strings.TrimSpace(item.OperUserID),
			"remark_mobiles": append([]string{}, item.RemarkMobiles...),
			"tags":           tags,
		})
	}
	return out
}

func parseFollowUsersFromPayload(raw any) []CustomerFollowUserBinding {
	list, ok := raw.([]any)
	if !ok || len(list) == 0 {
		return []CustomerFollowUserBinding{}
	}
	out := make([]CustomerFollowUserBinding, 0, len(list))
	for _, item := range list {
		m, ok := item.(map[string]any)
		if !ok || m == nil {
			continue
		}
		tags := parseTagsFromPayload(m["tags"])
		out = append(out, CustomerFollowUserBinding{
			UserID:        strings.TrimSpace(payloadStringValue(m, "userid")),
			Remark:        strings.TrimSpace(payloadStringValue(m, "remark")),
			Description:   strings.TrimSpace(payloadStringValue(m, "description")),
			OperUserID:    strings.TrimSpace(payloadStringValue(m, "oper_userid")),
			RemarkMobiles: parseStringSliceFromPayload(m["remark_mobiles"]),
			Tags:          tags,
		})
	}
	return out
}

func parseTagsFromPayload(raw any) []CustomerTagBindingTag {
	list, ok := raw.([]any)
	if !ok || len(list) == 0 {
		return []CustomerTagBindingTag{}
	}
	out := make([]CustomerTagBindingTag, 0, len(list))
	for _, item := range list {
		m, ok := item.(map[string]any)
		if !ok || m == nil {
			continue
		}
		tagID := strings.TrimSpace(payloadStringValue(m, "tag_id"))
		tagName := strings.TrimSpace(payloadStringValue(m, "tag_name"))
		groupName := strings.TrimSpace(payloadStringValue(m, "group_name"))
		if tagID == "" && tagName == "" {
			continue
		}
		out = append(out, CustomerTagBindingTag{
			TagID:     tagID,
			TagName:   tagName,
			GroupName: groupName,
		})
	}
	return out
}

func parseStringSliceFromPayload(raw any) []string {
	list, ok := raw.([]any)
	if !ok || len(list) == 0 {
		return []string{}
	}
	out := make([]string, 0, len(list))
	for _, item := range list {
		value := strings.TrimSpace(fmt.Sprintf("%v", item))
		if value == "" {
			continue
		}
		out = append(out, value)
	}
	return out
}

func compactNonEmptyStrings(items []string) []string {
	if len(items) == 0 {
		return []string{}
	}
	out := make([]string, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
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
	return out
}

func payloadStringValue(payload map[string]any, key string) string {
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

func sourceAccountUUIDFromLead(lead *leadmodel.Lead) string {
	if lead == nil || lead.SourceAccountUUID == nil {
		return ""
	}
	return strings.TrimSpace(*lead.SourceAccountUUID)
}

func (s *CustomerTagBindingService) resolveWeComExternalContactClient(
	ctx context.Context,
	tenantUUID, channelAccountUUID string,
) (weComExternalContactClient, error) {
	if s == nil || s.tagSyncSvc == nil || s.tagSyncSvc.accountRepo == nil {
		return nil, errors.New("tag sync account repository unavailable")
	}
	if s.clientFactory == nil {
		return nil, errors.New("wecom external contact client factory unavailable")
	}
	account, err := s.tagSyncSvc.accountRepo.GetByAccountUUID(ctx, tenantUUID, channelAccountUUID)
	if err != nil {
		return nil, err
	}
	if account == nil {
		return nil, errors.New("channel account not found")
	}
	credentials, appType, err := s.tagSyncSvc.resolveCredentialMap(ctx, tenantUUID, channelAccountUUID)
	if err != nil {
		return nil, err
	}
	return s.clientFactory(appType, credentials)
}
