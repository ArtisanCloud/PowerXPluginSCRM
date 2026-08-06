package integration

import (
	"context"
	"testing"

	leadmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/lead_capture"
	socialmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/social_channel_governance"
	leadrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/lead_capture"
	leadsvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/lead_capture"
	"github.com/stretchr/testify/require"
)

type channelScopeProviderAdapter struct{}

func (channelScopeProviderAdapter) SubmitSyncTask(_ context.Context, _ leadsvc.TriggerSyncRequest, _ string) leadsvc.SyncTaskSubmitResult {
	return leadsvc.SyncTaskSubmitResult{Provider: leadmodel.LeadSyncTaskProviderLocalFallback, Status: "queued"}
}

type channelScopeLeadAdapter struct{}

func (channelScopeLeadAdapter) FetchLeads(_ context.Context, req leadsvc.TriggerSyncRequest, _ string) ([]leadsvc.WeComLeadRecord, error) {
	return []leadsvc.WeComLeadRecord{
		{DisplayName: req.Channel + "-" + req.AppType, Phone: "13800009999"},
	}, nil
}

func TestLeadCaptureDedupScopeIntegration_DoNotMergeAcrossChannelOrAppType(t *testing.T) {
	tenantUUID := "00000000-0000-0000-0000-000000000001"
	wechatAccount := "11111111-1111-4111-8111-111111111111"
	douyinAccount := "33333333-3333-4333-8333-333333333333"

	db := openWeComSyncIntegrationDB(t, "lead_capture_dedup_scope_channel")
	require.NoError(t, db.Create(&socialmodel.ChannelAccount{
		AccountUUID:     wechatAccount,
		TenantUuid:      tenantUUID,
		ChannelCode:     "wechat",
		AppType:         "wecom",
		AccountID:       "wechat-main",
		DisplayName:     "WechatMain",
		Status:          socialmodel.ChannelAccountStatusConnected,
		OrgSyncDefault:  true,
		OwnerMemberUUID: "00000000-0000-0000-0000-000000000121",
	}).Error)
	require.NoError(t, db.Create(&socialmodel.ChannelAccount{
		AccountUUID:     douyinAccount,
		TenantUuid:      tenantUUID,
		ChannelCode:     "douyin",
		AppType:         "short_video",
		AccountID:       "douyin-main",
		DisplayName:     "DouyinMain",
		Status:          socialmodel.ChannelAccountStatusConnected,
		OrgSyncDefault:  true,
		OwnerMemberUUID: "00000000-0000-0000-0000-000000000122",
	}).Error)

	taskRepo := leadrepo.NewLeadSyncTaskRepository(db)
	leadRepo := leadrepo.NewLeadRepository(db)
	factory := leadsvc.NewChannelSyncFactory()
	require.NoError(t, factory.Register("wechat", "wecom", channelScopeLeadAdapter{}, channelScopeProviderAdapter{}))
	require.NoError(t, factory.Register("douyin", "short_video", channelScopeLeadAdapter{}, channelScopeProviderAdapter{}))

	svc := leadsvc.NewWeComSyncService(taskRepo, nil, nil).
		WithChannelFactory(factory).
		WithLeadIngestion(leadRepo, nil)

	_, err := svc.TriggerSync(context.Background(), leadsvc.TriggerSyncRequest{
		TenantUUID:         tenantUUID,
		Channel:            "wechat",
		AppType:            "wecom",
		ChannelAccountUUID: wechatAccount,
	})
	require.NoError(t, err)

	_, err = svc.TriggerSync(context.Background(), leadsvc.TriggerSyncRequest{
		TenantUUID:         tenantUUID,
		Channel:            "douyin",
		AppType:            "short_video",
		ChannelAccountUUID: douyinAccount,
	})
	require.NoError(t, err)

	leads, err := leadRepo.List(context.Background(), tenantUUID)
	require.NoError(t, err)
	require.Len(t, leads, 2)
}
