package social_channel_governance

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	pwresp "github.com/ArtisanCloud/PowerWeChat/v3/src/kernel/response"
	workuserresp "github.com/ArtisanCloud/PowerWeChat/v3/src/work/user/response"
	workusertag "github.com/ArtisanCloud/PowerWeChat/v3/src/work/user/tag"
)

type StaffTagRecord struct {
	TagID          int64  `json:"tag_id"`
	TagName        string `json:"tag_name"`
	Writable       bool   `json:"writable"`
	WritableReason string `json:"writable_reason,omitempty"`
}

type StaffTagDetail struct {
	TagID      int64    `json:"tag_id"`
	TagName    string   `json:"tag_name"`
	UserIDs    []string `json:"user_ids"`
	PartyIDs   []int    `json:"party_ids"`
	UserCount  int      `json:"user_count"`
	PartyCount int      `json:"party_count"`
}

type StaffTagMemberPatchResult struct {
	TagID            int64    `json:"tag_id"`
	AddedUsers       int      `json:"added_users"`
	RemovedUsers     int      `json:"removed_users"`
	InvalidAddUsers  []string `json:"invalid_add_users"`
	InvalidDropUsers []string `json:"invalid_remove_users"`
}

type weComStaffTagClient interface {
	List(ctx context.Context) (*workuserresp.ResponseTagList, error)
	Get(ctx context.Context, tagID int64) (*workuserresp.ResponseTagDetail, error)
	Create(ctx context.Context, tagName string, tagID int64) (*workuserresp.ResponseTagCreate, error)
	Update(ctx context.Context, tagName string, tagID int64) (*pwresp.ResponseWork, error)
	Delete(ctx context.Context, tagID int64) (*pwresp.ResponseWork, error)
	TagUsers(ctx context.Context, tagID int64, userList []string) (*workuserresp.ResponseTagCreateUser, error)
	TagDelUsers(ctx context.Context, tagID int64, userList []string, partyList []string) (*workuserresp.ResponseTagDeleteUser, error)
}

type weComStaffTagClientFactory func(appType string, credentials map[string]string) (weComStaffTagClient, error)

type powerWeComStaffTagClient struct {
	client *workusertag.Client
}

func (c *powerWeComStaffTagClient) List(ctx context.Context) (*workuserresp.ResponseTagList, error) {
	return c.client.List(ctx)
}

func (c *powerWeComStaffTagClient) Get(ctx context.Context, tagID int64) (*workuserresp.ResponseTagDetail, error) {
	return c.client.Get(ctx, tagID)
}

func (c *powerWeComStaffTagClient) Create(ctx context.Context, tagName string, tagID int64) (*workuserresp.ResponseTagCreate, error) {
	return c.client.Create(ctx, strings.TrimSpace(tagName), tagID)
}

func (c *powerWeComStaffTagClient) Update(ctx context.Context, tagName string, tagID int64) (*pwresp.ResponseWork, error) {
	return c.client.Update(ctx, strings.TrimSpace(tagName), tagID)
}

func (c *powerWeComStaffTagClient) Delete(ctx context.Context, tagID int64) (*pwresp.ResponseWork, error) {
	return c.client.Delete(ctx, tagID)
}

func (c *powerWeComStaffTagClient) TagUsers(ctx context.Context, tagID int64, userList []string) (*workuserresp.ResponseTagCreateUser, error) {
	return c.client.TagUsers(ctx, tagID, userList)
}

func (c *powerWeComStaffTagClient) TagDelUsers(ctx context.Context, tagID int64, userList []string, partyList []string) (*workuserresp.ResponseTagDeleteUser, error) {
	return c.client.TagDelUsers(ctx, tagID, userList, partyList)
}

var defaultWeComStaffTagClientFactory weComStaffTagClientFactory = func(appType string, credentials map[string]string) (weComStaffTagClient, error) {
	app, err := newWeComTagSyncApp("wechat", appType, credentials)
	if err != nil {
		return nil, err
	}
	if app == nil {
		return nil, errors.New("wecom staff tag client unavailable")
	}
	client, err := workusertag.NewClient(app)
	if err != nil {
		return nil, err
	}
	if client == nil {
		return nil, errors.New("wecom staff tag client unavailable")
	}
	return &powerWeComStaffTagClient{client: client}, nil
}

type StaffTagService struct {
	credentialResolver *TagSyncService
	clientFactory      weComStaffTagClientFactory
}

func NewStaffTagService(credentialResolver *TagSyncService, factory weComStaffTagClientFactory) *StaffTagService {
	if factory == nil {
		factory = defaultWeComStaffTagClientFactory
	}
	return &StaffTagService{credentialResolver: credentialResolver, clientFactory: factory}
}

func (s *StaffTagService) ListByChannel(ctx context.Context, tenantUUID, channelAccountUUID string, includeWritable bool) ([]StaffTagRecord, error) {
	client, err := s.resolveClient(ctx, tenantUUID, channelAccountUUID)
	if err != nil {
		return nil, err
	}
	resp, err := client.List(ctx)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return []StaffTagRecord{}, nil
	}
	if resp.ErrCode != 0 {
		return nil, fmt.Errorf("wecom tag list failed: %d %s", resp.ErrCode, strings.TrimSpace(resp.ErrMsg))
	}
	items := make([]StaffTagRecord, 0, len(resp.TagList))
	for _, item := range resp.TagList {
		if item == nil {
			continue
		}
		tagName := strings.TrimSpace(item.TagName)
		if tagName == "" {
			tagName = strconv.Itoa(item.TagID)
		}
		record := StaffTagRecord{TagID: int64(item.TagID), TagName: tagName, Writable: true}
		if includeWritable {
			writable, reason := s.probeTagWritable(ctx, client, int64(item.TagID), tagName)
			record.Writable = writable
			record.WritableReason = reason
		}
		items = append(items, record)
	}
	return items, nil
}

func (s *StaffTagService) GetDetailByChannel(ctx context.Context, tenantUUID, channelAccountUUID string, tagID int64) (*StaffTagDetail, error) {
	client, err := s.resolveClient(ctx, tenantUUID, channelAccountUUID)
	if err != nil {
		return nil, err
	}
	resp, err := client.Get(ctx, tagID)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, nil
	}
	if resp.ErrCode != 0 {
		return nil, fmt.Errorf("wecom tag get failed: %d %s", resp.ErrCode, strings.TrimSpace(resp.ErrMsg))
	}
	userIDs := make([]string, 0, len(resp.UserList))
	for _, user := range resp.UserList {
		if user == nil {
			continue
		}
		uid := strings.TrimSpace(user.UserID)
		if uid == "" {
			continue
		}
		userIDs = append(userIDs, uid)
	}
	return &StaffTagDetail{
		TagID:      tagID,
		TagName:    strings.TrimSpace(resp.TagName),
		UserIDs:    userIDs,
		PartyIDs:   resp.PartyList,
		UserCount:  len(userIDs),
		PartyCount: len(resp.PartyList),
	}, nil
}

func (s *StaffTagService) CreateByChannel(ctx context.Context, tenantUUID, channelAccountUUID, tagName string) (*StaffTagRecord, error) {
	client, err := s.resolveClient(ctx, tenantUUID, channelAccountUUID)
	if err != nil {
		return nil, err
	}
	tagName = strings.TrimSpace(tagName)
	if tagName == "" {
		return nil, errors.New("tag_name is required")
	}
	resp, err := client.Create(ctx, tagName, 0)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, errors.New("wecom create tag response is nil")
	}
	if resp.ErrCode != 0 {
		return nil, fmt.Errorf("wecom tag create failed: %d %s", resp.ErrCode, strings.TrimSpace(resp.ErrMsg))
	}
	return &StaffTagRecord{TagID: resp.TagID, TagName: tagName}, nil
}

func (s *StaffTagService) UpdateByChannel(ctx context.Context, tenantUUID, channelAccountUUID string, tagID int64, tagName string) error {
	client, err := s.resolveClient(ctx, tenantUUID, channelAccountUUID)
	if err != nil {
		return err
	}
	tagName = strings.TrimSpace(tagName)
	if tagName == "" {
		return errors.New("tag_name is required")
	}
	resp, err := client.Update(ctx, tagName, tagID)
	if err != nil {
		return err
	}
	if resp == nil {
		return errors.New("wecom update tag response is nil")
	}
	if resp.ErrCode != 0 {
		return fmt.Errorf("wecom tag update failed: %d %s", resp.ErrCode, strings.TrimSpace(resp.ErrMsg))
	}
	return nil
}

func (s *StaffTagService) DeleteByChannel(ctx context.Context, tenantUUID, channelAccountUUID string, tagID int64) error {
	client, err := s.resolveClient(ctx, tenantUUID, channelAccountUUID)
	if err != nil {
		return err
	}
	resp, err := client.Delete(ctx, tagID)
	if err != nil {
		return err
	}
	if resp == nil {
		return errors.New("wecom delete tag response is nil")
	}
	if resp.ErrCode != 0 {
		return fmt.Errorf("wecom tag delete failed: %d %s", resp.ErrCode, strings.TrimSpace(resp.ErrMsg))
	}
	return nil
}

func (s *StaffTagService) PatchTagUsersByChannel(
	ctx context.Context,
	tenantUUID, channelAccountUUID string,
	tagID int64,
	addUserIDs []string,
	removeUserIDs []string,
) (*StaffTagMemberPatchResult, error) {
	client, err := s.resolveClient(ctx, tenantUUID, channelAccountUUID)
	if err != nil {
		return nil, err
	}
	addUserIDs = normalizeStringList(addUserIDs)
	removeUserIDs = normalizeStringList(removeUserIDs)
	result := &StaffTagMemberPatchResult{TagID: tagID}
	if len(addUserIDs) > 0 {
		resp, addErr := client.TagUsers(ctx, tagID, addUserIDs)
		if addErr != nil {
			return nil, addErr
		}
		if resp == nil {
			return nil, errors.New("wecom add tag users response is nil")
		}
		if resp.ErrCode != 0 {
			return nil, fmt.Errorf("wecom add tag users failed: %d %s", resp.ErrCode, strings.TrimSpace(resp.ErrMsg))
		}
		result.AddedUsers = len(addUserIDs)
		result.InvalidAddUsers = splitPipeList(resp.InvalidUser)
	}
	if len(removeUserIDs) > 0 {
		resp, removeErr := client.TagDelUsers(ctx, tagID, removeUserIDs, []string{})
		if removeErr != nil {
			return nil, removeErr
		}
		if resp == nil {
			return nil, errors.New("wecom remove tag users response is nil")
		}
		if resp.ErrCode != 0 {
			return nil, fmt.Errorf("wecom remove tag users failed: %d %s", resp.ErrCode, strings.TrimSpace(resp.ErrMsg))
		}
		result.RemovedUsers = len(removeUserIDs)
		result.InvalidDropUsers = splitPipeList(resp.InvalidUser)
	}
	return result, nil
}

func (s *StaffTagService) resolveClient(ctx context.Context, tenantUUID, channelAccountUUID string) (weComStaffTagClient, error) {
	if s == nil || s.credentialResolver == nil {
		return nil, errors.New("staff tag service unavailable")
	}
	if s.clientFactory == nil {
		return nil, errors.New("staff tag client factory unavailable")
	}
	credentials, appType, err := s.credentialResolver.resolveCredentialMap(ctx, tenantUUID, channelAccountUUID)
	if err != nil {
		return nil, err
	}
	return s.clientFactory(appType, credentials)
}

func (s *StaffTagService) probeTagWritable(ctx context.Context, client weComStaffTagClient, tagID int64, tagName string) (bool, string) {
	if client == nil {
		return false, "client_unavailable"
	}
	resp, err := client.Update(ctx, strings.TrimSpace(tagName), tagID)
	if err != nil {
		errMsg := strings.ToLower(strings.TrimSpace(err.Error()))
		if strings.Contains(errMsg, "81011") || strings.Contains(errMsg, "no priviledge") || strings.Contains(errMsg, "no privilege") {
			return false, "permission_denied"
		}
		return false, "probe_failed"
	}
	if resp == nil {
		return false, "probe_failed"
	}
	if resp.ErrCode == 0 {
		return true, ""
	}
	if resp.ErrCode == 81011 {
		return false, "permission_denied"
	}
	return false, "probe_failed"
}

func normalizeStringList(items []string) []string {
	if len(items) == 0 {
		return []string{}
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(items))
	for _, raw := range items {
		clean := strings.TrimSpace(raw)
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

func splitPipeList(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return []string{}
	}
	parts := strings.Split(value, "|")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		clean := strings.TrimSpace(part)
		if clean == "" {
			continue
		}
		out = append(out, clean)
	}
	return out
}
