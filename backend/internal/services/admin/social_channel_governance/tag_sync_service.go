package social_channel_governance

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	plcache "github.com/ArtisanCloud/PowerLibs/v3/cache"
	"github.com/ArtisanCloud/PowerWeChat/v3/src/kernel"
	pwresp "github.com/ArtisanCloud/PowerWeChat/v3/src/kernel/response"
	openwork "github.com/ArtisanCloud/PowerWeChat/v3/src/openWork"
	openworksuit "github.com/ArtisanCloud/PowerWeChat/v3/src/openWork/suitAuth"
	"github.com/ArtisanCloud/PowerWeChat/v3/src/work"
	pwtag "github.com/ArtisanCloud/PowerWeChat/v3/src/work/externalContact/tag"
	pwtagreq "github.com/ArtisanCloud/PowerWeChat/v3/src/work/externalContact/tag/request"
	pwtagresp "github.com/ArtisanCloud/PowerWeChat/v3/src/work/externalContact/tag/response"
	socialmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/social_channel_governance"
	socialrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/social_channel_governance"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/wecomauth"
)

type TagRecord struct {
	TagID     string `json:"tag_id"`
	GroupID   string `json:"group_id,omitempty"`
	GroupName string `json:"group_name,omitempty"`
	Name      string `json:"name"`
	Version   string `json:"version"`
	Order     int    `json:"order,omitempty"`
}

type CustomerTagOperation struct {
	ExternalUserID string   `json:"external_userid"`
	UserID         string   `json:"userid"`
	AddTag         []string `json:"add_tag"`
	RemoveTag      []string `json:"remove_tag"`
}

type TagOperation struct {
	Operation string `json:"operation"`
	TagID     string `json:"tag_id"`
	GroupID   string `json:"group_id,omitempty"`
	GroupName string `json:"group_name,omitempty"`
	Name      string `json:"name,omitempty"`
}

type TagSyncResult struct {
	Pulled          int    `json:"pulled"`
	Pushed          int    `json:"pushed"`
	Conflicts       int    `json:"conflicts"`
	Created         int    `json:"created"`
	Updated         int    `json:"updated"`
	Deleted         int    `json:"deleted"`
	SnapshotVersion string `json:"snapshot_version"`
	Cursor          string `json:"cursor"`
}

func (s *TagSyncService) PushTagOperationsByChannel(
	ctx context.Context,
	tenantUUID, channelAccountUUID string,
	operations []TagOperation,
	cursor string,
) (*TagSyncResult, error) {
	if s == nil {
		return nil, errors.New("tag sync service unavailable")
	}
	client, err := s.resolveWeComTagClient(ctx, tenantUUID, channelAccountUUID)
	if err != nil {
		return nil, err
	}
	remoteTags, err := s.fetchRemoteTagsWithClient(ctx, client)
	if err != nil {
		return nil, err
	}
	remoteByID := toTagMap(remoteTags)
	remoteGroupNameByID := make(map[string]string, len(remoteTags))
	for _, item := range remoteTags {
		groupID := strings.TrimSpace(item.GroupID)
		if groupID == "" {
			continue
		}
		if _, exists := remoteGroupNameByID[groupID]; exists {
			continue
		}
		remoteGroupNameByID[groupID] = strings.TrimSpace(item.GroupName)
	}
	res := &TagSyncResult{Cursor: strings.TrimSpace(cursor)}
	if res.Cursor == "" && s.mappingRepo != nil {
		if cp, cpErr := s.mappingRepo.GetCheckpoint(ctx, tenantUUID, "push"); cpErr == nil && cp != nil {
			res.Cursor = strings.TrimSpace(cp.Cursor)
		}
	}
	for _, operation := range operations {
		op := strings.TrimSpace(strings.ToLower(operation.Operation))
		if op == "" {
			continue
		}
		switch op {
		case "create", "add":
			groupID := strings.TrimSpace(operation.GroupID)
			groupName := strings.TrimSpace(operation.GroupName)
			tagName := strings.TrimSpace(operation.Name)
			if tagName == "" {
				continue
			}
			if groupID == "" && groupName == "" {
				continue
			}
			addReq := &pwtagreq.RequestTagAddCorpTag{
				GroupName: groupName,
				Order:     0,
				Tag: []pwtagreq.RequestTagAddCorpTagFieldTag{
					{Name: tagName, Order: 0},
				},
			}
			if groupID != "" {
				addReq.GroupID = &groupID
			}
			addResp, addErr := client.AddCorpTag(ctx, addReq)
			if addErr != nil || addResp == nil || addResp.ErrCode != 0 {
				res.Conflicts++
				if s.conflictSvc != nil {
					msg := ""
					if addResp != nil {
						msg = strings.TrimSpace(addResp.ErrMsg)
					}
					if addErr != nil {
						msg = strings.TrimSpace(addErr.Error())
					}
					conflictResourceID := groupID
					if conflictResourceID == "" {
						conflictResourceID = fmt.Sprintf("%s/%s", groupName, tagName)
					}
					_ = s.conflictSvc.Enqueue(ctx, tenantUUID, "tags", "tag_group", conflictResourceID,
						map[string]any{"operation": "create", "group_id": groupID, "group_name": groupName, "name": tagName},
						map[string]any{"error": "add_failed", "message": msg},
					)
				}
				continue
			}
			res.Pushed++
			res.Created++
		case "rename_group", "update_group":
			groupID := strings.TrimSpace(operation.GroupID)
			nextGroupName := strings.TrimSpace(operation.GroupName)
			if groupID == "" {
				continue
			}
			if nextGroupName == "" {
				nextGroupName = strings.TrimSpace(operation.Name)
			}
			if nextGroupName == "" {
				continue
			}
			if currentName, exists := remoteGroupNameByID[groupID]; exists && currentName == nextGroupName {
				continue
			}
			editResp, editErr := client.EditCorpTagGroup(ctx, groupID, nextGroupName)
			if shouldFallbackToStrategyTag(editResp, editErr) {
				editResp, editErr = client.EditStrategyTag(ctx, groupID, nextGroupName)
			}
			if editErr != nil || editResp == nil || editResp.ErrCode != 0 {
				res.Conflicts++
				if s.conflictSvc != nil {
					msg := ""
					if editResp != nil {
						msg = strings.TrimSpace(editResp.ErrMsg)
					}
					if editErr != nil {
						msg = strings.TrimSpace(editErr.Error())
					}
					_ = s.conflictSvc.Enqueue(ctx, tenantUUID, "tags", "tag_group", groupID,
						map[string]any{"operation": "rename_group", "group_id": groupID, "group_name": nextGroupName},
						map[string]any{"error": "edit_group_failed", "message": msg},
					)
				}
				continue
			}
			remoteGroupNameByID[groupID] = nextGroupName
			res.Pushed++
			res.Updated++
		case "rename", "update":
			tagID := strings.TrimSpace(operation.TagID)
			if tagID == "" {
				continue
			}
			name := strings.TrimSpace(operation.Name)
			if name == "" {
				continue
			}
			remote, exists := remoteByID[tagID]
			if exists && strings.TrimSpace(remote.Name) == name {
				continue
			}
			editResp, editErr := client.EditCorpTag(ctx, &pwtagreq.RequestTagEditCorpTag{
				ID:   tagID,
				Name: name,
			})
			if shouldFallbackToStrategyTag(editResp, editErr) {
				editResp, editErr = client.EditStrategyTag(ctx, tagID, name)
			}
			if editErr != nil || editResp == nil || editResp.ErrCode != 0 {
				res.Conflicts++
				if s.conflictSvc != nil {
					msg := ""
					if editResp != nil {
						msg = strings.TrimSpace(editResp.ErrMsg)
					}
					if editErr != nil {
						msg = strings.TrimSpace(editErr.Error())
					}
					_ = s.conflictSvc.Enqueue(ctx, tenantUUID, "tags", "tag", tagID,
						map[string]any{"operation": "rename", "name": name},
						map[string]any{"error": "edit_failed", "message": msg},
					)
				}
				continue
			}
			res.Pushed++
			res.Updated++
		case "delete", "remove":
			tagID := strings.TrimSpace(operation.TagID)
			if tagID == "" {
				continue
			}
			delResp, delErr := client.DelCorpTag(ctx, &pwtagreq.RequestTagDelCorpTag{
				TagID: []string{tagID},
			})
			if delErr != nil || delResp == nil || delResp.ErrCode != 0 {
				res.Conflicts++
				if s.conflictSvc != nil {
					msg := ""
					if delResp != nil {
						msg = strings.TrimSpace(delResp.ErrMsg)
					}
					if delErr != nil {
						msg = strings.TrimSpace(delErr.Error())
					}
					_ = s.conflictSvc.Enqueue(ctx, tenantUUID, "tags", "tag", tagID,
						map[string]any{"operation": "delete"},
						map[string]any{"error": "delete_failed", "message": msg},
					)
				}
				continue
			}
			res.Pushed++
			res.Deleted++
		}
	}
	if s.mappingRepo != nil {
		version, verErr := s.mappingRepo.ResolveSnapshotVersion(ctx, tenantUUID, "push", (res.Pushed > 0 || res.Conflicts > 0))
		if verErr != nil {
			return nil, verErr
		}
		res.SnapshotVersion = version
		if saveErr := s.mappingRepo.SaveCheckpoint(ctx, tenantUUID, "push", res.Cursor, version, time.Now().UTC()); saveErr != nil {
			return nil, saveErr
		}
	}
	return res, nil
}

type TagSyncService struct {
	mappingRepo   *socialrepo.TagMappingRepository
	tagRecordRepo *socialrepo.TagRecordRepository
	conflictSvc   *ConflictResolutionService
	accountRepo   tagSyncAccountLoader
	openworkRepo  tagSyncOpenWorkLoader
	platformRepo  tagSyncPlatformLoader
	clientFactory weComTagClientFactory
}

func NewTagSyncService(mappingRepo *socialrepo.TagMappingRepository, conflictSvc *ConflictResolutionService) *TagSyncService {
	return &TagSyncService{
		mappingRepo:   mappingRepo,
		conflictSvc:   conflictSvc,
		clientFactory: defaultWeComTagClientFactory,
	}
}

type tagSyncAccountLoader interface {
	GetByAccountUUID(ctx context.Context, tenantUUID, accountUUID string) (*socialmodel.ChannelAccount, error)
}

type tagSyncOpenWorkLoader interface {
	ResolveBindingByChannelAccount(ctx context.Context, tenantUUID, channelAccountUUID string) (*socialmodel.WeComOpenAuthBinding, error)
	GetLatestSuiteTicket(ctx context.Context, tenantUUID, suiteID string) (string, error)
}

type tagSyncPlatformLoader interface {
	GetByChannelProvider(ctx context.Context, channelCode, providerCode string) (*socialmodel.ChannelPlatformSetting, error)
}

type weComTagClient interface {
	GetCorpTagList(ctx context.Context, tagID []string, groupID []string) (*pwtagresp.ResponseTagGetCorpTagList, error)
	AddCorpTag(ctx context.Context, options *pwtagreq.RequestTagAddCorpTag) (*pwtagresp.ResponseTagAddCorpTag, error)
	EditCorpTag(ctx context.Context, options *pwtagreq.RequestTagEditCorpTag) (*pwresp.ResponseWork, error)
	EditStrategyTag(ctx context.Context, id, name string) (*pwresp.ResponseWork, error)
	EditCorpTagGroup(ctx context.Context, groupID, groupName string) (*pwresp.ResponseWork, error)
	DelCorpTag(ctx context.Context, options *pwtagreq.RequestTagDelCorpTag) (*pwresp.ResponseWork, error)
	MarkTag(ctx context.Context, options *pwtagreq.RequestTagMarkTag) (*pwresp.ResponseWork, error)
}

type weComTagClientFactory func(appType string, credentials map[string]string) (weComTagClient, error)

type powerWeComTagClient struct {
	client  *pwtag.Client
	agentID *int64
}

func (c *powerWeComTagClient) GetCorpTagList(ctx context.Context, tagID []string, groupID []string) (*pwtagresp.ResponseTagGetCorpTagList, error) {
	return c.client.GetCorpTagList(ctx, tagID, groupID)
}
func (c *powerWeComTagClient) AddCorpTag(ctx context.Context, options *pwtagreq.RequestTagAddCorpTag) (*pwtagresp.ResponseTagAddCorpTag, error) {
	if c != nil && c.agentID != nil && options != nil && options.AgentID == nil {
		options.AgentID = c.agentID
	}
	return c.client.AddCorpTag(ctx, options)
}
func (c *powerWeComTagClient) EditCorpTag(ctx context.Context, options *pwtagreq.RequestTagEditCorpTag) (*pwresp.ResponseWork, error) {
	if c != nil && c.agentID != nil && options != nil && options.AgentID == nil {
		options.AgentID = c.agentID
	}
	return c.client.EditCorpTag(ctx, options)
}
func (c *powerWeComTagClient) EditStrategyTag(ctx context.Context, id, name string) (*pwresp.ResponseWork, error) {
	return c.client.EditStrategyTag(ctx, &pwtagreq.RequestTagEditStrategyTag{
		ID:    strings.TrimSpace(id),
		Name:  strings.TrimSpace(name),
		Order: 0,
	})
}
func (c *powerWeComTagClient) EditCorpTagGroup(ctx context.Context, groupID, groupName string) (*pwresp.ResponseWork, error) {
	req := &pwtagreq.RequestTagEditCorpTag{
		ID:   strings.TrimSpace(groupID),
		Name: strings.TrimSpace(groupName),
	}
	if c != nil && c.agentID != nil {
		req.AgentID = c.agentID
	}
	return c.client.EditCorpTag(ctx, req)
}
func (c *powerWeComTagClient) DelCorpTag(ctx context.Context, options *pwtagreq.RequestTagDelCorpTag) (*pwresp.ResponseWork, error) {
	if c != nil && c.agentID != nil && options != nil && options.AgentID == nil {
		options.AgentID = c.agentID
	}
	return c.client.DelCorpTag(ctx, options)
}
func (c *powerWeComTagClient) MarkTag(ctx context.Context, options *pwtagreq.RequestTagMarkTag) (*pwresp.ResponseWork, error) {
	return c.client.MarkTag(ctx, options)
}

var defaultWeComTagClientFactory weComTagClientFactory = func(appType string, credentials map[string]string) (weComTagClient, error) {
	app, err := newWeComTagSyncApp("wechat", appType, credentials)
	if err != nil {
		return nil, err
	}
	if app == nil {
		return nil, errors.New("wecom external contact tag client unavailable")
	}
	client, err := pwtag.NewClient(app)
	if err != nil {
		return nil, err
	}
	if client == nil {
		return nil, errors.New("wecom external contact tag client unavailable")
	}
	var agentID *int64
	if raw := strings.TrimSpace(credentials["agent_id"]); raw != "" {
		if parsed, convErr := strconv.ParseInt(raw, 10, 64); convErr == nil && parsed > 0 {
			agentID = &parsed
		}
	}
	return &powerWeComTagClient{client: client, agentID: agentID}, nil
}

func (s *TagSyncService) WithWeComSupport(accountRepo *socialrepo.AccountRepository, factory weComTagClientFactory) *TagSyncService {
	if s == nil {
		return s
	}
	s.accountRepo = accountRepo
	if factory != nil {
		s.clientFactory = factory
	}
	return s
}

func (s *TagSyncService) WithCredentialResolvers(
	openworkRepo *socialrepo.OpenWorkFoundationRepository,
	platformRepo *socialrepo.ChannelPlatformSettingRepository,
) *TagSyncService {
	if s == nil {
		return s
	}
	s.openworkRepo = openworkRepo
	s.platformRepo = platformRepo
	return s
}

func (s *TagSyncService) WithTagRecordRepository(repo *socialrepo.TagRecordRepository) *TagSyncService {
	if s == nil {
		return s
	}
	s.tagRecordRepo = repo
	return s
}

func (s *TagSyncService) SyncRemoteToLocalByChannel(
	ctx context.Context,
	tenantUUID, channelAccountUUID string,
	localTags []TagRecord,
	cursor string,
) (*TagSyncResult, error) {
	remoteTags, err := s.fetchRemoteTags(ctx, tenantUUID, channelAccountUUID)
	if err != nil {
		return nil, err
	}
	res, syncErr := s.SyncRemoteToLocal(ctx, tenantUUID, remoteTags, localTags, cursor)
	if syncErr != nil {
		return nil, syncErr
	}
	if s.tagRecordRepo != nil {
		resolvedAppType := "wecom"
		if s.accountRepo != nil {
			if account, accountErr := s.accountRepo.GetByAccountUUID(ctx, strings.ToLower(strings.TrimSpace(tenantUUID)), strings.ToLower(strings.TrimSpace(channelAccountUUID))); accountErr == nil && account != nil {
				if v := strings.ToLower(strings.TrimSpace(account.AppType)); v != "" {
					resolvedAppType = v
				}
			}
		}
		records := make([]socialmodel.SyncTagRecord, 0, len(remoteTags))
		for _, tag := range remoteTags {
			tag = normalizeTagRecord(tag)
			if tag.TagID == "" {
				continue
			}
			records = append(records, socialmodel.SyncTagRecord{
				ChannelAccountUUID: strings.TrimSpace(channelAccountUUID),
				ChannelCode:        "wechat",
				AppType:            resolvedAppType,
				RemoteTagID:        strings.TrimSpace(tag.TagID),
				RemoteGroupID:      strings.TrimSpace(tag.GroupID),
				RemoteGroupName:    strings.TrimSpace(tag.GroupName),
				TagName:            strings.TrimSpace(tag.Name),
				Version:            strings.TrimSpace(tag.Version),
				TagOrder:           tag.Order,
				SnapshotVersion:    strings.TrimSpace(res.SnapshotVersion),
			})
		}
		if err := s.tagRecordRepo.UpsertRemoteTags(
			ctx,
			tenantUUID,
			channelAccountUUID,
			"wechat",
			resolvedAppType,
			strings.TrimSpace(res.SnapshotVersion),
			records,
			time.Now().UTC(),
		); err != nil {
			return nil, err
		}
	}
	return res, nil
}

func (s *TagSyncService) SyncLocalToRemoteByChannel(
	ctx context.Context,
	tenantUUID, channelAccountUUID string,
	localTags []TagRecord,
	cursor string,
) (*TagSyncResult, error) {
	if s == nil {
		return nil, errors.New("tag sync service unavailable")
	}
	client, err := s.resolveWeComTagClient(ctx, tenantUUID, channelAccountUUID)
	if err != nil {
		return nil, err
	}
	remoteTags, err := s.fetchRemoteTagsWithClient(ctx, client)
	if err != nil {
		return nil, err
	}
	remoteByID := toTagMap(remoteTags)
	res := &TagSyncResult{Cursor: strings.TrimSpace(cursor)}
	if res.Cursor == "" && s.mappingRepo != nil {
		if cp, cpErr := s.mappingRepo.GetCheckpoint(ctx, tenantUUID, "push"); cpErr == nil && cp != nil {
			res.Cursor = strings.TrimSpace(cp.Cursor)
		}
	}
	for _, local := range localTags {
		local = normalizeTagRecord(local)
		lid := strings.TrimSpace(local.TagID)
		if lid == "" {
			continue
		}
		remote, exists := remoteByID[lid]
		if !exists {
			if local.GroupID == "" && local.GroupName == "" {
				res.Conflicts++
				if s.conflictSvc != nil {
					_ = s.conflictSvc.Enqueue(ctx, tenantUUID, "tags", "tag", lid,
						map[string]any{"name": local.Name, "version": local.Version, "group_id": local.GroupID, "group_name": local.GroupName},
						map[string]any{"error": "missing_group_for_create"},
					)
				}
				continue
			}
			addReq := &pwtagreq.RequestTagAddCorpTag{
				GroupName: local.GroupName,
				Order:     local.Order,
				Tag: []pwtagreq.RequestTagAddCorpTagFieldTag{
					{Name: local.Name, Order: local.Order},
				},
			}
			if local.GroupID != "" {
				groupID := local.GroupID
				addReq.GroupID = &groupID
			}
			addResp, addErr := client.AddCorpTag(ctx, addReq)
			if addErr != nil || addResp == nil || addResp.ErrCode != 0 {
				res.Conflicts++
				if s.conflictSvc != nil {
					msg := ""
					if addResp != nil {
						msg = strings.TrimSpace(addResp.ErrMsg)
					}
					if addErr != nil {
						msg = strings.TrimSpace(addErr.Error())
					}
					_ = s.conflictSvc.Enqueue(ctx, tenantUUID, "tags", "tag", lid,
						map[string]any{"name": local.Name, "version": local.Version, "group_id": local.GroupID, "group_name": local.GroupName},
						map[string]any{"error": "add_failed", "message": msg},
					)
				}
				continue
			}
			res.Pushed++
			res.Created++
			continue
		}

		if remote.Name == local.Name && (local.Version == "" || remote.Version == local.Version) {
			continue
		}
		editResp, editErr := client.EditCorpTag(ctx, &pwtagreq.RequestTagEditCorpTag{
			ID:    lid,
			Name:  local.Name,
			Order: local.Order,
		})
		if editErr != nil || editResp == nil || editResp.ErrCode != 0 {
			res.Conflicts++
			if s.conflictSvc != nil {
				msg := ""
				if editResp != nil {
					msg = strings.TrimSpace(editResp.ErrMsg)
				}
				if editErr != nil {
					msg = strings.TrimSpace(editErr.Error())
				}
				_ = s.conflictSvc.Enqueue(ctx, tenantUUID, "tags", "tag", lid,
					map[string]any{"name": local.Name, "version": local.Version},
					map[string]any{"name": remote.Name, "version": remote.Version, "error": "edit_failed", "message": msg},
				)
			}
			continue
		}
		res.Pushed++
		res.Updated++
	}
	if s.mappingRepo != nil {
		version, verErr := s.mappingRepo.ResolveSnapshotVersion(ctx, tenantUUID, "push", (res.Pushed > 0 || res.Conflicts > 0))
		if verErr != nil {
			return nil, verErr
		}
		res.SnapshotVersion = version
		if saveErr := s.mappingRepo.SaveCheckpoint(ctx, tenantUUID, "push", res.Cursor, version, time.Now().UTC()); saveErr != nil {
			return nil, saveErr
		}
	}
	return res, nil
}

func (s *TagSyncService) PushCustomerTagBindingsByChannel(
	ctx context.Context,
	tenantUUID, channelAccountUUID string,
	operations []CustomerTagOperation,
	cursor string,
) (*TagSyncResult, error) {
	if s == nil {
		return nil, errors.New("tag sync service unavailable")
	}
	client, err := s.resolveWeComTagClient(ctx, tenantUUID, channelAccountUUID)
	if err != nil {
		return nil, err
	}
	res := &TagSyncResult{Cursor: strings.TrimSpace(cursor)}
	if res.Cursor == "" && s.mappingRepo != nil {
		if cp, cpErr := s.mappingRepo.GetCheckpoint(ctx, tenantUUID, "push"); cpErr == nil && cp != nil {
			res.Cursor = strings.TrimSpace(cp.Cursor)
		}
	}
	for _, operation := range operations {
		externalUserID := strings.TrimSpace(operation.ExternalUserID)
		userID := strings.TrimSpace(operation.UserID)
		if externalUserID == "" || userID == "" {
			continue
		}
		addTags := dedupeTagIDs(operation.AddTag)
		removeTags := dedupeTagIDs(operation.RemoveTag)
		if len(addTags) == 0 && len(removeTags) == 0 {
			continue
		}
		resp, markErr := client.MarkTag(ctx, &pwtagreq.RequestTagMarkTag{
			UserID:         userID,
			ExternalUserID: externalUserID,
			AddTag:         addTags,
			RemoveTag:      removeTags,
		})
		if markErr != nil || resp == nil || resp.ErrCode != 0 {
			res.Conflicts++
			if s.conflictSvc != nil {
				msg := ""
				if resp != nil {
					msg = strings.TrimSpace(resp.ErrMsg)
				}
				if markErr != nil {
					msg = strings.TrimSpace(markErr.Error())
				}
				_ = s.conflictSvc.Enqueue(ctx, tenantUUID, "tags", "customer_tag_binding", fmt.Sprintf("%s:%s", externalUserID, userID),
					map[string]any{"add_tag": addTags, "remove_tag": removeTags},
					map[string]any{"error": "mark_tag_failed", "message": msg},
				)
			}
			continue
		}
		res.Pushed++
		res.Updated += len(addTags) + len(removeTags)
	}
	if s.mappingRepo != nil {
		version, verErr := s.mappingRepo.ResolveSnapshotVersion(ctx, tenantUUID, "push", (res.Pushed > 0 || res.Conflicts > 0))
		if verErr != nil {
			return nil, verErr
		}
		res.SnapshotVersion = version
		if saveErr := s.mappingRepo.SaveCheckpoint(ctx, tenantUUID, "push", res.Cursor, version, time.Now().UTC()); saveErr != nil {
			return nil, saveErr
		}
	}
	return res, nil
}

func (s *TagSyncService) SyncRemoteToLocal(
	ctx context.Context,
	tenantUUID string,
	remoteTags []TagRecord,
	localTags []TagRecord,
	cursor string,
) (*TagSyncResult, error) {
	res := &TagSyncResult{Cursor: strings.TrimSpace(cursor)}
	if res.Cursor == "" && s.mappingRepo != nil {
		if cp, cpErr := s.mappingRepo.GetCheckpoint(ctx, tenantUUID, "pull"); cpErr == nil && cp != nil {
			res.Cursor = strings.TrimSpace(cp.Cursor)
		}
	}
	localByID := toTagMap(localTags)
	remoteByID := toTagMap(remoteTags)
	for _, remote := range remoteTags {
		rid := strings.TrimSpace(remote.TagID)
		if rid == "" {
			continue
		}
		local, ok := localByID[rid]
		if !ok {
			res.Pulled++
			res.Created++
			continue
		}
		localName := strings.TrimSpace(local.Name)
		remoteName := strings.TrimSpace(remote.Name)
		localVer := strings.TrimSpace(local.Version)
		remoteVer := strings.TrimSpace(remote.Version)
		if localName != remoteName || (remoteVer != "" && localVer != remoteVer) {
			res.Conflicts++
			// remote_first: 记录冲突但依然按远端落地，确保最终收敛。
			res.Pulled++
			res.Updated++
			if s.conflictSvc != nil {
				_ = s.conflictSvc.Enqueue(ctx, tenantUUID, "tags", "tag", rid,
					map[string]any{"name": local.Name, "version": local.Version},
					map[string]any{"name": remote.Name, "version": remote.Version},
				)
			}
			continue
		}
	}
	for _, local := range localTags {
		lid := strings.TrimSpace(local.TagID)
		if lid == "" {
			continue
		}
		if _, ok := remoteByID[lid]; !ok {
			// 全量 pull 场景可用于判定远端已删；增量场景由上游 payload 决定是否处理。
			res.Deleted++
		}
	}
	if s.mappingRepo != nil {
		version, err := s.mappingRepo.ResolveSnapshotVersion(
			ctx,
			tenantUUID,
			"pull",
			(res.Pulled > 0 || res.Conflicts > 0 || res.Deleted > 0),
		)
		if err != nil {
			return nil, err
		}
		res.SnapshotVersion = version
		if err := s.mappingRepo.SaveCheckpoint(ctx, tenantUUID, "pull", res.Cursor, version, time.Now().UTC()); err != nil {
			return nil, err
		}
	}
	return res, nil
}

func (s *TagSyncService) SyncLocalToRemote(
	ctx context.Context,
	tenantUUID string,
	localTags []TagRecord,
	remoteTags []TagRecord,
	cursor string,
) (*TagSyncResult, error) {
	res := &TagSyncResult{Cursor: strings.TrimSpace(cursor)}
	if res.Cursor == "" && s.mappingRepo != nil {
		if cp, cpErr := s.mappingRepo.GetCheckpoint(ctx, tenantUUID, "push"); cpErr == nil && cp != nil {
			res.Cursor = strings.TrimSpace(cp.Cursor)
		}
	}
	remoteByID := toTagMap(remoteTags)
	for _, local := range localTags {
		lid := strings.TrimSpace(local.TagID)
		if lid == "" {
			continue
		}
		remote, ok := remoteByID[lid]
		if !ok {
			res.Pushed++
			res.Created++
			continue
		}
		localName := strings.TrimSpace(local.Name)
		remoteName := strings.TrimSpace(remote.Name)
		localVer := strings.TrimSpace(local.Version)
		remoteVer := strings.TrimSpace(remote.Version)
		if localName != remoteName || (localVer != "" && remoteVer != localVer) {
			res.Conflicts++
			if s.conflictSvc != nil {
				_ = s.conflictSvc.Enqueue(ctx, tenantUUID, "tags", "tag", lid,
					map[string]any{"name": local.Name, "version": local.Version},
					map[string]any{"name": remote.Name, "version": remote.Version},
				)
			}
			continue
		}
	}
	if s.mappingRepo != nil {
		version, err := s.mappingRepo.ResolveSnapshotVersion(
			ctx,
			tenantUUID,
			"push",
			(res.Pushed > 0 || res.Conflicts > 0),
		)
		if err != nil {
			return nil, err
		}
		res.SnapshotVersion = version
		if err := s.mappingRepo.SaveCheckpoint(ctx, tenantUUID, "push", res.Cursor, version, time.Now().UTC()); err != nil {
			return nil, err
		}
	}
	return res, nil
}

func toTagMap(items []TagRecord) map[string]TagRecord {
	out := make(map[string]TagRecord, len(items))
	for _, item := range items {
		item = normalizeTagRecord(item)
		key := strings.TrimSpace(item.TagID)
		if key == "" {
			continue
		}
		out[key] = item
	}
	return out
}

func normalizeTagRecord(item TagRecord) TagRecord {
	item.TagID = strings.TrimSpace(item.TagID)
	item.GroupID = strings.TrimSpace(item.GroupID)
	item.GroupName = strings.TrimSpace(item.GroupName)
	item.Name = strings.TrimSpace(item.Name)
	item.Version = strings.TrimSpace(item.Version)
	return item
}

func dedupeTagIDs(input []string) []string {
	if len(input) == 0 {
		return []string{}
	}
	out := make([]string, 0, len(input))
	seen := make(map[string]struct{}, len(input))
	for _, raw := range input {
		tagID := strings.TrimSpace(raw)
		if tagID == "" {
			continue
		}
		if _, ok := seen[tagID]; ok {
			continue
		}
		seen[tagID] = struct{}{}
		out = append(out, tagID)
	}
	return out
}

func (s *TagSyncService) fetchRemoteTags(ctx context.Context, tenantUUID, channelAccountUUID string) ([]TagRecord, error) {
	if s == nil {
		return nil, errors.New("tag sync service unavailable")
	}
	client, err := s.resolveWeComTagClient(ctx, tenantUUID, channelAccountUUID)
	if err != nil {
		return nil, err
	}
	return s.fetchRemoteTagsWithClient(ctx, client)
}

func (s *TagSyncService) fetchRemoteTagsWithClient(ctx context.Context, client weComTagClient) ([]TagRecord, error) {
	if client == nil {
		return nil, errors.New("wecom tag client unavailable")
	}
	resp, err := client.GetCorpTagList(ctx, nil, nil)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return []TagRecord{}, nil
	}
	if resp.ErrCode != 0 {
		return nil, fmt.Errorf("wecom get_corp_tag_list failed: %d %s", resp.ErrCode, strings.TrimSpace(resp.ErrMsg))
	}
	out := make([]TagRecord, 0, 64)
	for _, group := range resp.TagGroups {
		if group == nil {
			continue
		}
		groupID := strings.TrimSpace(group.GroupID)
		groupName := strings.TrimSpace(group.GroupName)
		groupVer := strconv.Itoa(group.CreateTime)
		for _, tag := range group.Tags {
			if tag == nil {
				continue
			}
			tagID := strings.TrimSpace(tag.ID)
			if tagID == "" {
				continue
			}
			version := strings.TrimSpace(groupVer)
			if tag.CreateTime > 0 {
				version = strconv.Itoa(tag.CreateTime)
			}
			out = append(out, TagRecord{
				TagID:     tagID,
				GroupID:   groupID,
				GroupName: groupName,
				Name:      strings.TrimSpace(tag.Name),
				Version:   version,
				Order:     tag.Order,
			})
		}
	}
	return out, nil
}

func (s *TagSyncService) resolveWeComTagClient(ctx context.Context, tenantUUID, channelAccountUUID string) (weComTagClient, error) {
	if s == nil {
		return nil, errors.New("tag sync service unavailable")
	}
	if s.clientFactory == nil {
		return nil, errors.New("wecom tag client factory unavailable")
	}
	credentials, appType, err := s.resolveCredentialMap(ctx, tenantUUID, channelAccountUUID)
	if err != nil {
		return nil, err
	}
	return s.clientFactory(appType, credentials)
}

func (s *TagSyncService) resolveCredentialMap(ctx context.Context, tenantUUID, channelAccountUUID string) (map[string]string, string, error) {
	if s == nil || s.accountRepo == nil {
		return nil, "", errors.New("tag sync account repository unavailable")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	channelAccountUUID = strings.ToLower(strings.TrimSpace(channelAccountUUID))
	if tenantUUID == "" || channelAccountUUID == "" {
		return nil, "", errors.New("tenant_uuid and channel_account_uuid are required")
	}
	account, err := s.accountRepo.GetByAccountUUID(ctx, tenantUUID, channelAccountUUID)
	if err != nil {
		return nil, "", err
	}
	if account == nil {
		return nil, "", socialrepo.ErrAccountNotFound
	}
	channelCode := strings.ToLower(strings.TrimSpace(account.ChannelCode))
	appType := strings.ToLower(strings.TrimSpace(account.AppType))
	if _, err := wecomauth.ResolveKind(channelCode, appType); err != nil {
		return nil, "", err
	}
	credentials := credentialsToStringMap(account.Credentials)
	credentials = s.mergeDelegatedCredentials(ctx, tenantUUID, channelAccountUUID, credentials)
	return credentials, appType, nil
}

func (s *TagSyncService) mergeDelegatedCredentials(
	ctx context.Context,
	tenantUUID, channelAccountUUID string,
	input map[string]string,
) map[string]string {
	out := make(map[string]string, len(input)+10)
	for k, v := range input {
		out[k] = strings.TrimSpace(v)
	}

	if s != nil && s.openworkRepo != nil && strings.TrimSpace(tenantUUID) != "" && strings.TrimSpace(channelAccountUUID) != "" {
		binding, err := s.openworkRepo.ResolveBindingByChannelAccount(ctx, tenantUUID, channelAccountUUID)
		if err == nil && binding != nil && strings.TrimSpace(binding.Status) == socialmodel.WeComAuthBindingStatusActive {
			if strings.TrimSpace(out["corp_id"]) == "" {
				out["corp_id"] = strings.TrimSpace(binding.CorpID)
			}
			if strings.TrimSpace(out["permanent_code"]) == "" {
				out["permanent_code"] = strings.TrimSpace(binding.PermanentCode)
			}
			if strings.TrimSpace(out["template_id"]) == "" {
				out["template_id"] = strings.TrimSpace(binding.SuiteID)
			}
			if strings.TrimSpace(out["suite_id"]) == "" {
				out["suite_id"] = strings.TrimSpace(binding.SuiteID)
			}
			if strings.TrimSpace(out["agent_id"]) == "" {
				out["agent_id"] = strings.TrimSpace(binding.AgentID)
			}
			if strings.TrimSpace(out["template_ticket"]) == "" {
				out["template_ticket"] = strings.TrimSpace(binding.SuiteTicket)
			}
			if strings.TrimSpace(out["suite_ticket"]) == "" {
				out["suite_ticket"] = strings.TrimSpace(binding.SuiteTicket)
			}
			if strings.TrimSpace(binding.SuiteID) != "" {
				if latest, latestErr := s.openworkRepo.GetLatestSuiteTicket(ctx, tenantUUID, strings.TrimSpace(binding.SuiteID)); latestErr == nil {
					latest = strings.TrimSpace(latest)
					if latest != "" {
						out["template_ticket"] = latest
						out["suite_ticket"] = latest
					}
				}
			}
		}
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

func credentialsToStringMap(input map[string]interface{}) map[string]string {
	out := make(map[string]string, len(input))
	for key, value := range input {
		if value == nil {
			continue
		}
		out[key] = strings.TrimSpace(fmt.Sprintf("%v", value))
	}
	return out
}

func newWeComTagSyncApp(channelCode, appType string, credentials map[string]string) (*work.Work, error) {
	authKind, err := wecomauth.ResolveKind(channelCode, appType)
	if err != nil {
		return nil, err
	}
	corpID := strings.TrimSpace(credentials["corp_id"])
	appSecret := strings.TrimSpace(credentials["app_secret"])
	agentIDRaw := strings.TrimSpace(credentials["agent_id"])

	templateID := strings.TrimSpace(firstNonEmpty(credentials["template_id"], credentials["suite_id"]))
	templateSecret := strings.TrimSpace(firstNonEmpty(credentials["template_secret"], credentials["suite_secret"]))
	templateTicket := strings.TrimSpace(firstNonEmpty(credentials["template_ticket"], credentials["suite_ticket"]))
	providerCorpID := strings.TrimSpace(firstNonEmpty(credentials["provider_corpid"], credentials["provider_corp_id"]))
	providerSecret := strings.TrimSpace(credentials["provider_secret"])
	permanentCode := strings.TrimSpace(credentials["permanent_code"])
	if corpID == "" {
		corpID = strings.TrimSpace(credentials["auth_corp_id"])
	}
	if wecomauth.IsDelegated(authKind) {
		if corpID == "" || templateID == "" || templateSecret == "" || providerCorpID == "" || providerSecret == "" || permanentCode == "" {
			return nil, errors.New("wecom delegated credentials missing template/provider/corp/permanent_code")
		}
		if templateTicket == "" {
			return nil, errors.New("wecom delegated credentials missing template_ticket")
		}
		callback := strings.TrimSpace(credentials["oauth_callback"])
		if callback == "" {
			callback = "http://localhost"
		}
		memCache := plcache.NewMemCache("scrm_tag_sync_openwork", 10*time.Minute, "")
		if memCache == nil {
			memCache = plcache.NewMemCache("scrm_tag_sync_openwork", 10*time.Minute, os.TempDir())
		}
		if memCache == nil {
			return nil, errors.New("wecom delegated init cache failed")
		}
		httpDebug := parseTagCredentialBool(credentials["http_debug"])
		openWorkApp, err := openwork.NewOpenWork(&openwork.UserConfig{
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
		suiteTicketComponent, ok := openWorkApp.GetComponent("SuiteTicket").(*openworksuit.SuiteTicket)
		if !ok || suiteTicketComponent == nil {
			return nil, errors.New("wecom delegated SuiteTicket component unavailable")
		}
		if err := suiteTicketComponent.SetTicket(templateTicket); err != nil {
			return nil, err
		}
		return openWorkApp.ProviderClient(corpID, permanentCode, nil)
	}

	if corpID == "" || appSecret == "" {
		return nil, errors.New("wecom self-built credentials missing corp_id/app_secret")
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
		HttpDebug: parseTagCredentialBool(credentials["http_debug"]),
	})
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if s := strings.TrimSpace(v); s != "" {
			return s
		}
	}
	return ""
}

func parseTagCredentialBool(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}

func shouldFallbackToStrategyTag(resp *pwresp.ResponseWork, err error) bool {
	if err != nil {
		return false
	}
	return resp != nil && resp.ErrCode == 81011
}
