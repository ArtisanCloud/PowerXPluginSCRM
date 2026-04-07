package social_channel_governance_test

import (
	"testing"

	model "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/social_channel_governance"
	socialsvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/social_channel_governance"
	"github.com/stretchr/testify/require"
)

func TestMultiCorpBindingPolicyPrefersConfiguredCorp(t *testing.T) {
	svc := socialsvc.NewBindingPolicyService()
	bindings := []*model.WeComOpenAuthBinding{
		{BindingUUID: "b1", CorpID: "ww-a", Status: model.WeComAuthBindingStatusActive},
		{BindingUUID: "b2", CorpID: "ww-b", Status: model.WeComAuthBindingStatusActive},
	}
	selected := svc.ResolveDefaultBinding(bindings, "ww-b")
	require.NotNil(t, selected)
	require.Equal(t, "b2", selected.BindingUUID)
}
