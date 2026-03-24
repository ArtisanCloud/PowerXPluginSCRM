package lead_capture

import (
	leadrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/repository/lead_capture"
	leadobs "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/observability/lead_capture"
)

// ChannelCodeFeature bundles foundational services for channel-code acquisition flow.
type ChannelCodeFeature struct {
	ChannelCodeService      *ChannelCodeService
	ChannelCodeEventService *ChannelCodeEventService
	WelcomeSyncService      *WelcomeSyncService
}

func NewChannelCodeFeature(repos *leadrepo.Bundle, metrics *leadobs.Metrics) *ChannelCodeFeature {
	if repos == nil {
		return &ChannelCodeFeature{}
	}
	return &ChannelCodeFeature{
		ChannelCodeService:      NewChannelCodeService(repos.ChannelCodes, metrics),
		ChannelCodeEventService: NewChannelCodeEventService(repos.ChannelCodeEvents, repos.ChannelCodes, nil, nil, metrics),
		WelcomeSyncService:      NewWelcomeSyncService(repos.ChannelCodes, repos.WelcomeConfigs, repos.WelcomeSyncAttempt, NewWeComWelcomeAdapter(), metrics),
	}
}
