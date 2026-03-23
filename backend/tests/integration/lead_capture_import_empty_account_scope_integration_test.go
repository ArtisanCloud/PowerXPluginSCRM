package integration

import (
	"context"
	"testing"

	leadrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/lead_capture"
	leadsvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/lead_capture"
	"github.com/stretchr/testify/require"
)

func TestLeadCaptureImportScopeIntegration_EmptySourceAccountScopeIsolation(t *testing.T) {
	db := openDedupMergeIntegrationDB(t, "lead_capture_import_empty_account_scope")
	tenantUUID := "00000000-0000-0000-0000-000000000001"
	repo := leadrepo.NewLeadRepository(db)
	svc := leadsvc.NewLeadService(repo)

	first, err := svc.Create(context.Background(), tenantUUID, leadsvc.LeadCreateRequest{
		DisplayName:       "Manual A",
		Phone:             "13900008888",
		SourceChannel:     "wechat",
		SourceAppType:     "wecom",
		SourceAccountUUID: "",
	})
	require.NoError(t, err)

	second, err := svc.Create(context.Background(), tenantUUID, leadsvc.LeadCreateRequest{
		DisplayName:       "Manual B",
		Phone:             "13900008888",
		SourceChannel:     "wechat",
		SourceAppType:     "wecom",
		SourceAccountUUID: "",
	})
	require.NoError(t, err)
	require.Equal(t, first.LeadUUID, second.LeadUUID)
	require.True(t, second.HasMerge)

	third, err := svc.Create(context.Background(), tenantUUID, leadsvc.LeadCreateRequest{
		DisplayName:       "Channel Sync",
		Phone:             "13900008888",
		SourceChannel:     "wechat",
		SourceAppType:     "wecom",
		SourceAccountUUID: "11111111-1111-4111-8111-111111111111",
	})
	require.NoError(t, err)
	require.NotEqual(t, first.LeadUUID, third.LeadUUID)

	leads, err := repo.List(context.Background(), tenantUUID)
	require.NoError(t, err)
	require.Len(t, leads, 2)
}
