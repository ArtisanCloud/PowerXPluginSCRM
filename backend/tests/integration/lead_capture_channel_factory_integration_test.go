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

type factoryWeComLeadAdapter struct{}

func (factoryWeComLeadAdapter) FetchLeads(_ context.Context, _ leadsvc.TriggerSyncRequest, _ string) ([]leadsvc.WeComLeadRecord, error) {
	return []leadsvc.WeComLeadRecord{
		{DisplayName: "wecom-lead", Phone: "13800000011"},
	}, nil
}

type factoryMiniLeadAdapter struct{}

func (factoryMiniLeadAdapter) FetchLeads(_ context.Context, _ leadsvc.TriggerSyncRequest, _ string) ([]leadsvc.WeComLeadRecord, error) {
	return []leadsvc.WeComLeadRecord{
		{DisplayName: "mini-lead", Phone: "13800000022"},
	}, nil
}

type factoryProviderAdapter struct{}

func (factoryProviderAdapter) SubmitSyncTask(_ context.Context, _ leadsvc.TriggerSyncRequest, _ string) leadsvc.SyncTaskSubmitResult {
	return leadsvc.SyncTaskSubmitResult{Provider: leadmodel.LeadSyncTaskProviderLocalFallback, Status: "queued"}
}

func TestLeadCaptureChannelFactoryIntegration_SelectAdapterByChannelAppType(t *testing.T) {
	tenantUUID := "00000000-0000-0000-0000-000000000001"
	accountWeCom := "11111111-1111-4111-8111-111111111111"
	accountMini := "22222222-2222-4222-8222-222222222222"

	db := openWeComSyncIntegrationDB(t, "lead_capture_channel_factory_integration")
	require.NoError(t, db.Create(&socialmodel.ChannelAccount{
		AccountUUID:     accountWeCom,
		TenantUuid:      tenantUUID,
		ChannelCode:     "wechat",
		AppType:         "wecom",
		AccountID:       "wecom-main",
		DisplayName:     "企微主账号",
		Status:          socialmodel.ChannelAccountStatusConnected,
		OrgSyncDefault:  true,
		OwnerMemberUUID: "owner-001",
	}).Error)
	require.NoError(t, db.Create(&socialmodel.ChannelAccount{
		AccountUUID:     accountMini,
		TenantUuid:      tenantUUID,
		ChannelCode:     "wechat",
		AppType:         "wecom_app",
		AccountID:       "wecom-mini",
		DisplayName:     "企微应用账号",
		Status:          socialmodel.ChannelAccountStatusConnected,
		OrgSyncDefault:  true,
		OwnerMemberUUID: "owner-001",
	}).Error)

	taskRepo := leadrepo.NewLeadSyncTaskRepository(db)
	leadRepo := leadrepo.NewLeadRepository(db)

	factory := leadsvc.NewChannelSyncFactory()
	require.NoError(t, factory.Register("wechat", "wecom", factoryWeComLeadAdapter{}, factoryProviderAdapter{}))
	require.NoError(t, factory.Register("wechat", "wecom_app", factoryMiniLeadAdapter{}, factoryProviderAdapter{}))

	svc := leadsvc.NewWeComSyncService(taskRepo, nil, nil).
		WithChannelFactory(factory).
		WithLeadIngestion(leadRepo, nil)

	taskA, err := svc.TriggerSync(context.Background(), leadsvc.TriggerSyncRequest{
		TenantUUID: tenantUUID,
		Channel:    "wechat",
		AppType:    "wecom",
	})
	require.NoError(t, err)
	require.Equal(t, "success", taskA.Status)
	require.Equal(t, 1, taskA.StatsCreated)

	taskB, err := svc.TriggerSync(context.Background(), leadsvc.TriggerSyncRequest{
		TenantUUID: tenantUUID,
		Channel:    "wechat",
		AppType:    "wecom_app",
	})
	require.NoError(t, err)
	require.Equal(t, "success", taskB.Status)
	require.Equal(t, 1, taskB.StatsCreated)

	leadA, err := leadRepo.FindFirstByPhone(context.Background(), tenantUUID, "13800000011")
	require.NoError(t, err)
	require.Equal(t, "wecom", leadA.SourceAppType)

	leadB, err := leadRepo.FindFirstByPhone(context.Background(), tenantUUID, "13800000022")
	require.NoError(t, err)
	require.Equal(t, "wecom_app", leadB.SourceAppType)
}
