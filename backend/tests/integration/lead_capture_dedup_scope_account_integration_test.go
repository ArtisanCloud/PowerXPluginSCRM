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

type accountScopeProviderAdapter struct{}

func (accountScopeProviderAdapter) SubmitSyncTask(_ context.Context, _ leadsvc.TriggerSyncRequest, _ string) leadsvc.SyncTaskSubmitResult {
	return leadsvc.SyncTaskSubmitResult{Provider: leadmodel.LeadSyncTaskProviderLocalFallback, Status: "queued"}
}

type accountScopeLeadAdapter struct{}

func (accountScopeLeadAdapter) FetchLeads(_ context.Context, _ leadsvc.TriggerSyncRequest, _ string) ([]leadsvc.WeComLeadRecord, error) {
	return []leadsvc.WeComLeadRecord{
		{DisplayName: "Same Phone", Phone: "13800008888"},
	}, nil
}

func TestLeadCaptureDedupScopeIntegration_DoNotMergeAcrossSourceAccountUUID(t *testing.T) {
	tenantUUID := "00000000-0000-0000-0000-000000000001"
	accountA := "11111111-1111-4111-8111-111111111111"
	accountB := "22222222-2222-4222-8222-222222222222"

	db := openWeComSyncIntegrationDB(t, "lead_capture_dedup_scope_account")
	require.NoError(t, db.Create(&socialmodel.ChannelAccount{
		AccountUUID:     accountA,
		TenantUuid:      tenantUUID,
		ChannelCode:     "wechat",
		AppType:         "wecom",
		AccountID:       "acc-a",
		DisplayName:     "A",
		Status:          socialmodel.ChannelAccountStatusConnected,
		OrgSyncDefault:  true,
		OwnerMemberUUID: "00000000-0000-0000-0000-000000000111",
	}).Error)
	require.NoError(t, db.Create(&socialmodel.ChannelAccount{
		AccountUUID:     accountB,
		TenantUuid:      tenantUUID,
		ChannelCode:     "wechat",
		AppType:         "wecom",
		AccountID:       "acc-b",
		DisplayName:     "B",
		Status:          socialmodel.ChannelAccountStatusConnected,
		OrgSyncDefault:  false,
		OwnerMemberUUID: "00000000-0000-0000-0000-000000000112",
	}).Error)

	taskRepo := leadrepo.NewLeadSyncTaskRepository(db)
	leadRepo := leadrepo.NewLeadRepository(db)
	factory := leadsvc.NewChannelSyncFactory()
	require.NoError(t, factory.Register("wechat", "wecom", accountScopeLeadAdapter{}, accountScopeProviderAdapter{}))

	svc := leadsvc.NewWeComSyncService(taskRepo, nil, nil).
		WithChannelFactory(factory).
		WithLeadIngestion(leadRepo, nil)

	_, err := svc.TriggerSync(context.Background(), leadsvc.TriggerSyncRequest{
		TenantUUID:         tenantUUID,
		Channel:            "wechat",
		AppType:            "wecom",
		ChannelAccountUUID: accountA,
	})
	require.NoError(t, err)

	_, err = svc.TriggerSync(context.Background(), leadsvc.TriggerSyncRequest{
		TenantUUID:         tenantUUID,
		Channel:            "wechat",
		AppType:            "wecom",
		ChannelAccountUUID: accountB,
	})
	require.NoError(t, err)

	leads, err := leadRepo.List(context.Background(), tenantUUID)
	require.NoError(t, err)
	require.Len(t, leads, 2)
}
