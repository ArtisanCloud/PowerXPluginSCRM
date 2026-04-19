package social_channel_governance

import (
	"strings"

	model "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/social_channel_governance"
)

// BindingPolicyService decides default binding when tenant has multiple corp bindings.
// Priority: preferred corp_id > existing is_default > latest active binding > first item.
type BindingPolicyService struct{}

func NewBindingPolicyService() *BindingPolicyService {
	return &BindingPolicyService{}
}

func (s *BindingPolicyService) ResolveDefaultBinding(bindings []*model.WeComOpenAuthBinding, preferredCorpID string) *model.WeComOpenAuthBinding {
	if len(bindings) == 0 {
		return nil
	}
	preferredCorpID = strings.TrimSpace(preferredCorpID)
	if preferredCorpID != "" {
		for _, item := range bindings {
			if item != nil && strings.TrimSpace(item.CorpID) == preferredCorpID {
				return item
			}
		}
	}
	for _, item := range bindings {
		if item != nil && item.IsDefault {
			return item
		}
	}
	for _, item := range bindings {
		if item != nil && strings.TrimSpace(item.Status) == model.WeComAuthBindingStatusActive {
			return item
		}
	}
	return bindings[0]
}
