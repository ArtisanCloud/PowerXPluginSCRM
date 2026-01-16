package social_channel_governance

import (
	"strings"

	authx "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/middleware"
)

// RBACEntries returns RBAC mappings for social channel governance admin APIs.
func RBACEntries(prefix string) map[string]authx.Permission {
	base := strings.TrimRight(prefix, "/") + "/admin/social/channel-accounts"
	return map[string]authx.Permission{
		"GET:" + base:        {Resource: "scrm.social_channel_accounts", Action: "read"},
		"POST:" + base:       {Resource: "scrm.social_channel_accounts", Action: "write"},
		"GET:" + base + "/*": {Resource: "scrm.social_channel_accounts", Action: "read"},
	}
}
