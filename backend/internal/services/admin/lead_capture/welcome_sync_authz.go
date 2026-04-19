package lead_capture

import (
	"errors"
	"strings"

	authx "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/middleware"
)

var ErrWelcomeSyncPublishForbidden = errors.New("welcome sync publish forbidden")

type WelcomeSyncAuthz struct{}

func NewWelcomeSyncAuthz() *WelcomeSyncAuthz {
	return &WelcomeSyncAuthz{}
}

func (a *WelcomeSyncAuthz) EnsureCanPublish(tc authx.TenantContext) error {
	if canPublishWelcomeSync(tc) {
		return nil
	}
	return ErrWelcomeSyncPublishForbidden
}

func canPublishWelcomeSync(tc authx.TenantContext) bool {
	for _, role := range tc.Roles {
		switch strings.ToLower(strings.TrimSpace(role)) {
		case "superadmin", "system.admin", "system_admin", "tenant.admin", "tenant_admin", "role_admin", "role_owner", "channel.operator", "channel_operator":
			return true
		}
	}
	for _, perm := range tc.Permissions {
		candidate := strings.ToLower(strings.TrimSpace(perm))
		switch candidate {
		case "*", "*:*", "channel_welcome_sync:publish", "channel_welcome_sync.publish", "channel_welcome_sync:*":
			return true
		}
	}
	return false
}
