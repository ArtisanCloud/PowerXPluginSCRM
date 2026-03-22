package lead_capture

import (
	"context"
	"testing"

	leadmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/lead_capture"
	"github.com/stretchr/testify/require"
)

type factoryTestProvider struct{}

func (factoryTestProvider) SubmitSyncTask(_ context.Context, _ TriggerSyncRequest, _ string) SyncTaskSubmitResult {
	return SyncTaskSubmitResult{Provider: leadmodel.LeadSyncTaskProviderLocalFallback, Status: "queued"}
}

type factoryTestLeadAdapter struct{}

func (factoryTestLeadAdapter) FetchLeads(_ context.Context, _ TriggerSyncRequest, _ string) ([]WeComLeadRecord, error) {
	return []WeComLeadRecord{{DisplayName: "factory"}}, nil
}

func TestChannelSyncFactory_ResolveUnknownChannelAppType(t *testing.T) {
	factory := NewChannelSyncFactory()

	_, err := factory.ResolveLeadAdapter("wechat", "unknown")
	require.Error(t, err)
	require.ErrorIs(t, err, ErrLeadSyncAdapterNotRegistered)

	_, err = factory.ResolveTaskProvider("wechat", "unknown")
	require.Error(t, err)
	require.ErrorIs(t, err, ErrTaskProviderNotRegistered)
}

func TestChannelSyncFactory_ResolveRegisteredChannelAppType(t *testing.T) {
	factory := NewChannelSyncFactory()
	require.NoError(t, factory.Register("WECHAT", "WECOM", factoryTestLeadAdapter{}, factoryTestProvider{}))

	leadAdapter, err := factory.ResolveLeadAdapter("wechat", "wecom")
	require.NoError(t, err)
	require.NotNil(t, leadAdapter)

	providerAdapter, err := factory.ResolveTaskProvider("wechat", "wecom")
	require.NoError(t, err)
	require.NotNil(t, providerAdapter)
}
