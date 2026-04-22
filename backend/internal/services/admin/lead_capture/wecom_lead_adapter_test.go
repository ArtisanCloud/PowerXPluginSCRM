package lead_capture

import (
	"context"
	"testing"

	"github.com/ArtisanCloud/PowerSocialite/v3/src/models"
	"github.com/ArtisanCloud/PowerWeChat/v3/src/kernel/response"
	pwexternalresp "github.com/ArtisanCloud/PowerWeChat/v3/src/work/externalContact/response"
	socialmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/social_channel_governance"
	"github.com/stretchr/testify/require"
)

type stubWeComAccountLoader struct {
	account *socialmodel.ChannelAccount
	err     error
}

func (s stubWeComAccountLoader) GetByAccountUUID(_ context.Context, _ string, _ string) (*socialmodel.ChannelAccount, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.account, nil
}

type stubWeComExternalClient struct {
	followResp *pwexternalresp.ResponseGetFollowUserList
	batches    map[string]*pwexternalresp.ResponseBatchGetByUser
}

func (s *stubWeComExternalClient) GetFollowUsers(_ context.Context) (*pwexternalresp.ResponseGetFollowUserList, error) {
	return s.followResp, nil
}

func (s *stubWeComExternalClient) BatchGet(_ context.Context, _ []string, cursor string, _ int) (*pwexternalresp.ResponseBatchGetByUser, error) {
	if s.batches == nil {
		return &pwexternalresp.ResponseBatchGetByUser{ResponseWork: response.ResponseWork{}}, nil
	}
	if out, ok := s.batches[cursor]; ok {
		return out, nil
	}
	return &pwexternalresp.ResponseBatchGetByUser{ResponseWork: response.ResponseWork{}}, nil
}

func TestDefaultWeComLeadAdapter_FetchLeadsMappingAndCursor(t *testing.T) {
	adapter := &DefaultWeComLeadAdapter{
		accountLoader: stubWeComAccountLoader{
			account: &socialmodel.ChannelAccount{
				TenantUuid:  "00000000-0000-0000-0000-000000000001",
				ChannelCode: "wechat",
				AppType:     "wecom",
				Credentials: map[string]any{
					"corp_id":    "wx123",
					"app_secret": "secret",
					"agent_id":   "1000001",
				},
			},
		},
		batchLimit: 2,
		clientFactory: func(_ string, _ map[string]string) (weComExternalContactClient, error) {
			return &stubWeComExternalClient{
				followResp: &pwexternalresp.ResponseGetFollowUserList{
					ResponseWork: response.ResponseWork{},
					FollowUser:   []string{"staff_a"},
				},
				batches: map[string]*pwexternalresp.ResponseBatchGetByUser{
					"": {
						ResponseWork: response.ResponseWork{},
						NextCursor:   "cursor-1",
						ExternalContactList: []*pwexternalresp.ResponseExternalContact{
							{
								ExternalContact: &models.ExternalContact{
									ExternalUserID: "ext-001",
									Name:           "Alice",
									ExternalProfile: &models.ExternalProfile{
										ExternalAttr: []*models.ExternalAttr{
											{Name: "邮箱", Text: &models.Text{Value: "Alice@Example.com"}},
										},
									},
								},
								FollowInfo: &models.FollowUser{
									RemarkMobiles: []string{"13800000001"},
									CreateTime:    1774000000,
								},
							},
						},
					},
					"cursor-1": {
						ResponseWork: response.ResponseWork{},
						ExternalContactList: []*pwexternalresp.ResponseExternalContact{
							{
								ExternalContact: &models.ExternalContact{
									ExternalUserID: "ext-002",
									Name:           "",
									ExternalProfile: &models.ExternalProfile{
										ExternalAttr: []*models.ExternalAttr{
											{Name: "手机号", Text: &models.Text{Value: "13900000002"}},
											{Name: "微信号", Text: &models.Text{Value: "wechat_bob_02"}},
										},
									},
								},
								FollowInfo: &models.FollowUser{
									Remark: "Bob From Remark",
								},
							},
							{
								ExternalContact: &models.ExternalContact{
									ExternalUserID: "ext-001",
									Name:           "duplicate",
								},
							},
						},
					},
				},
			}, nil
		},
	}

	items, err := adapter.FetchLeads(context.Background(), TriggerSyncRequest{
		TenantUUID: "00000000-0000-0000-0000-000000000001",
		Channel:    "wechat",
		AppType:    "wecom",
	}, "11111111-1111-4111-8111-111111111111")
	require.NoError(t, err)
	require.Len(t, items, 2)
	require.Equal(t, "ext-001", items[0].ExternalLeadID)
	require.Equal(t, "Alice", items[0].DisplayName)
	require.Equal(t, "13800000001", items[0].Phone)
	require.Equal(t, "alice@example.com", items[0].Email)
	require.False(t, items[0].OccurredAt.IsZero())
	require.Equal(t, "ext-002", items[1].ExternalLeadID)
	require.Equal(t, "Bob From Remark", items[1].DisplayName)
	require.Equal(t, "13900000002", items[1].Phone)
	require.Equal(t, "wechat_bob_02", items[1].WechatID)
}
