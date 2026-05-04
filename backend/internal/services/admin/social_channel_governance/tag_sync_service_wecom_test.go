package social_channel_governance

import (
	"context"
	"testing"

	pwresp "github.com/ArtisanCloud/PowerWeChat/v3/src/kernel/response"
	pwtagreq "github.com/ArtisanCloud/PowerWeChat/v3/src/work/externalContact/tag/request"
	pwtagresp "github.com/ArtisanCloud/PowerWeChat/v3/src/work/externalContact/tag/response"
	socialmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/social_channel_governance"
	socialrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/social_channel_governance"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
)

type mockTagSyncAccountRepo struct {
	account *socialmodel.ChannelAccount
	err     error
}

func (m mockTagSyncAccountRepo) GetByAccountUUID(_ context.Context, _, _ string) (*socialmodel.ChannelAccount, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.account, nil
}

type mockWeComTagClient struct {
	remoteGroups []*pwtagresp.CorpTagGroup
	addCalls     int
	editCalls    int
	lastAddReq   *pwtagreq.RequestTagAddCorpTag
}

func (m *mockWeComTagClient) GetCorpTagList(_ context.Context, _ []string, _ []string) (*pwtagresp.ResponseTagGetCorpTagList, error) {
	return &pwtagresp.ResponseTagGetCorpTagList{
		ResponseWork: pwresp.ResponseWork{ErrCode: 0, ErrMsg: "ok"},
		TagGroups:    m.remoteGroups,
	}, nil
}

func (m *mockWeComTagClient) AddCorpTag(_ context.Context, req *pwtagreq.RequestTagAddCorpTag) (*pwtagresp.ResponseTagAddCorpTag, error) {
	m.addCalls++
	m.lastAddReq = req
	return &pwtagresp.ResponseTagAddCorpTag{ResponseWork: pwresp.ResponseWork{ErrCode: 0, ErrMsg: "ok"}}, nil
}

func (m *mockWeComTagClient) EditCorpTag(_ context.Context, _ *pwtagreq.RequestTagEditCorpTag) (*pwresp.ResponseWork, error) {
	m.editCalls++
	return &pwresp.ResponseWork{ErrCode: 0, ErrMsg: "ok"}, nil
}

func (m *mockWeComTagClient) EditStrategyTag(_ context.Context, _, _ string) (*pwresp.ResponseWork, error) {
	m.editCalls++
	return &pwresp.ResponseWork{ErrCode: 0, ErrMsg: "ok"}, nil
}

func (m *mockWeComTagClient) EditCorpTagGroup(_ context.Context, _, _ string) (*pwresp.ResponseWork, error) {
	m.editCalls++
	return &pwresp.ResponseWork{ErrCode: 0, ErrMsg: "ok"}, nil
}

func (m *mockWeComTagClient) DelCorpTag(_ context.Context, _ *pwtagreq.RequestTagDelCorpTag) (*pwresp.ResponseWork, error) {
	return &pwresp.ResponseWork{ErrCode: 0, ErrMsg: "ok"}, nil
}

func (m *mockWeComTagClient) MarkTag(_ context.Context, _ *pwtagreq.RequestTagMarkTag) (*pwresp.ResponseWork, error) {
	return &pwresp.ResponseWork{ErrCode: 0, ErrMsg: "ok"}, nil
}

func TestTagSyncService_SyncRemoteToLocalByChannel(t *testing.T) {
	tenantUUID := "00000000-0000-0000-0000-000000000001"
	accountUUID := "11111111-1111-4111-8111-111111111111"
	client := &mockWeComTagClient{
		remoteGroups: []*pwtagresp.CorpTagGroup{
			{
				GroupID:   "g-1",
				GroupName: "客户等级",
				Tags: []*pwtagresp.Tag{
					{ID: "t-1", Name: "重要"},
					{ID: "t-2", Name: "普通"},
				},
			},
		},
	}
	svc := NewTagSyncService(nil, nil).WithWeComSupport(
		&socialrepo.AccountRepository{}, // placeholder, overridden below
		func(appType string, credentials map[string]string) (weComTagClient, error) {
			require.Equal(t, "wecom", appType)
			require.Equal(t, "ww-demo", credentials["corp_id"])
			require.Equal(t, "demo-secret", credentials["app_secret"])
			return client, nil
		},
	)
	svc.accountRepo = mockTagSyncAccountRepo{
		account: &socialmodel.ChannelAccount{
			AccountUUID: accountUUID,
			TenantUuid:  tenantUUID,
			ChannelCode: "wechat",
			AppType:     "wecom",
			Credentials: datatypes.JSONMap{
				"corp_id":    "ww-demo",
				"app_secret": "demo-secret",
			},
		},
	}

	res, err := svc.SyncRemoteToLocalByChannel(context.Background(), tenantUUID, accountUUID,
		[]TagRecord{{TagID: "t-1", Name: "普通"}},
		"cursor-r-1",
	)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, 2, res.Pulled)
	require.Equal(t, 1, res.Conflicts)
	require.Equal(t, 1, res.Created)
	require.Equal(t, 1, res.Updated)
	require.Equal(t, "cursor-r-1", res.Cursor)
}

func TestTagSyncService_SyncLocalToRemoteByChannel(t *testing.T) {
	tenantUUID := "00000000-0000-0000-0000-000000000001"
	accountUUID := "11111111-1111-4111-8111-111111111111"
	client := &mockWeComTagClient{
		remoteGroups: []*pwtagresp.CorpTagGroup{
			{
				GroupID:   "g-1",
				GroupName: "客户等级",
				Tags: []*pwtagresp.Tag{
					{ID: "t-1", Name: "普通"},
				},
			},
		},
	}
	svc := NewTagSyncService(nil, nil).WithWeComSupport(
		&socialrepo.AccountRepository{}, // placeholder, overridden below
		func(_ string, _ map[string]string) (weComTagClient, error) {
			return client, nil
		},
	)
	svc.accountRepo = mockTagSyncAccountRepo{
		account: &socialmodel.ChannelAccount{
			AccountUUID: accountUUID,
			TenantUuid:  tenantUUID,
			ChannelCode: "wechat",
			AppType:     "wecom",
			Credentials: datatypes.JSONMap{
				"corp_id":    "ww-demo",
				"app_secret": "demo-secret",
			},
		},
	}

	res, err := svc.SyncLocalToRemoteByChannel(context.Background(), tenantUUID, accountUUID, []TagRecord{
		{TagID: "t-1", Name: "重要", GroupID: "g-1", GroupName: "客户等级"},
		{TagID: "t-2", Name: "普通", GroupID: "g-1", GroupName: "客户等级"},
	}, "cursor-p-1")
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, 2, res.Pushed)
	require.Equal(t, 0, res.Conflicts)
	require.Equal(t, 1, res.Created)
	require.Equal(t, 1, res.Updated)
	require.Equal(t, 1, client.addCalls)
	require.Equal(t, 1, client.editCalls)
	require.Equal(t, "cursor-p-1", res.Cursor)
}

func TestTagSyncService_PushTagOperationsByChannel_CreateGroupTag(t *testing.T) {
	tenantUUID := "00000000-0000-0000-0000-000000000001"
	accountUUID := "11111111-1111-4111-8111-111111111111"
	client := &mockWeComTagClient{
		remoteGroups: []*pwtagresp.CorpTagGroup{},
	}
	svc := NewTagSyncService(nil, nil).WithWeComSupport(
		&socialrepo.AccountRepository{},
		func(_ string, _ map[string]string) (weComTagClient, error) {
			return client, nil
		},
	)
	svc.accountRepo = mockTagSyncAccountRepo{
		account: &socialmodel.ChannelAccount{
			AccountUUID: accountUUID,
			TenantUuid:  tenantUUID,
			ChannelCode: "wechat",
			AppType:     "wecom",
			Credentials: datatypes.JSONMap{
				"corp_id":    "ww-demo",
				"app_secret": "demo-secret",
			},
		},
	}

	res, err := svc.PushTagOperationsByChannel(context.Background(), tenantUUID, accountUUID, []TagOperation{
		{
			Operation: "create",
			GroupName: "意向客户",
			Name:      "首标签",
		},
	}, "cursor-create-1")
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, 1, client.addCalls)
	require.NotNil(t, client.lastAddReq)
	require.Equal(t, "意向客户", client.lastAddReq.GroupName)
	require.Len(t, client.lastAddReq.Tag, 1)
	require.Equal(t, "首标签", client.lastAddReq.Tag[0].Name)
	require.Equal(t, 1, res.Pushed)
	require.Equal(t, 1, res.Created)
	require.Equal(t, 0, res.Conflicts)
	require.Equal(t, "cursor-create-1", res.Cursor)
}
