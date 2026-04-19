package social_channel_governance

import socialsvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/social_channel_governance"

// FoundationHandler keeps the canonical file path for foundation access/reauthorize endpoints.
// It currently reuses OpenWorkFoundationHandler implementation.
type FoundationHandler struct {
	*OpenWorkFoundationHandler
}

func NewFoundationHandler(svc *socialsvc.OpenWorkFoundationService) *FoundationHandler {
	return &FoundationHandler{
		OpenWorkFoundationHandler: NewOpenWorkFoundationHandler(svc),
	}
}
