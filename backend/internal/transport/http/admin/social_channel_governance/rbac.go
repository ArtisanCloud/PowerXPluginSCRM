package social_channel_governance

import (
	"strings"

	authx "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/middleware"
)

// RBACEntries returns RBAC mappings for social channel governance admin APIs.
func RBACEntries(prefix string) map[string]authx.Permission {
	base := strings.TrimRight(prefix, "/") + "/admin/social/channel-accounts"
	openworkBase := strings.TrimRight(prefix, "/") + "/admin/social/openwork/wecom"
	return map[string]authx.Permission{
		"GET:" + base:                                                    {Resource: "scrm.social_channel_accounts", Action: "read"},
		"POST:" + base:                                                   {Resource: "scrm.social_channel_accounts", Action: "write"},
		"GET:" + base + "/deleted":                                       {Resource: "scrm.social_channel_accounts", Action: "read"},
		"GET:" + base + "/*":                                             {Resource: "scrm.social_channel_accounts", Action: "read"},
		"PUT:" + base + "/:account_uuid":                                 {Resource: "scrm.social_channel_accounts", Action: "write"},
		"POST:" + base + "/:account_uuid/test-connection":                {Resource: "scrm.social_channel_accounts", Action: "write"},
		"POST:" + base + "/:account_uuid/test-app-secret":                {Resource: "scrm.social_channel_accounts", Action: "write"},
		"POST:" + base + "/:account_uuid/test-contact-secret":            {Resource: "scrm.social_channel_accounts", Action: "write"},
		"DELETE:" + base + "/:account_uuid":                              {Resource: "scrm.social_channel_accounts", Action: "write"},
		"POST:" + base + "/:account_uuid/restore":                        {Resource: "scrm.social_channel_accounts", Action: "write"},
		"POST:" + base + "/:account_uuid/channel-members":                {Resource: "scrm.social_channel_accounts", Action: "write"},
		"PATCH:" + base + "/:account_uuid/capabilities":                  {Resource: "scrm.social_channel_accounts", Action: "write"},
		"POST:" + openworkBase + "/events":                               {Resource: "scrm.social_channel_accounts", Action: "write"},
		"POST:" + openworkBase + "/authorize/start":                      {Resource: "scrm.social_channel_accounts", Action: "write"},
		"POST:" + openworkBase + "/authorize/complete":                   {Resource: "scrm.social_channel_accounts", Action: "write"},
		"GET:" + openworkBase + "/authorize/status":                      {Resource: "scrm.social_channel_accounts", Action: "read"},
		"GET:" + openworkBase + "/bindings":                              {Resource: "scrm.social_channel_accounts", Action: "read"},
		"POST:" + openworkBase + "/bindings/:binding_uuid/default":       {Resource: "scrm.social_channel_accounts", Action: "write"},
		"POST:" + openworkBase + "/sync/jobs":                            {Resource: "scrm.social_channel_accounts", Action: "write"},
		"GET:" + openworkBase + "/sync/jobs":                             {Resource: "scrm.social_channel_accounts", Action: "read"},
		"GET:" + openworkBase + "/sync/conflicts":                        {Resource: "scrm.social_channel_accounts", Action: "read"},
		"POST:" + openworkBase + "/sync/conflicts/:conflict_uuid/replay": {Resource: "scrm.social_channel_accounts", Action: "write"},
		"GET:" + openworkBase + "/sync/dashboard":                        {Resource: "scrm.social_channel_accounts", Action: "read"},
		"GET:" + openworkBase + "/go-live-gates":                         {Resource: "scrm.social_channel_accounts", Action: "read"},
	}
}
