package social_channel_governance

import (
	"strings"

	authx "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/middleware"
)

// RBACEntries returns RBAC mappings for social channel governance admin APIs.
func RBACEntries(prefix string) map[string]authx.Permission {
	base := strings.TrimRight(prefix, "/") + "/admin/social/channel-accounts"
	openworkBase := strings.TrimRight(prefix, "/") + "/admin/social/openwork/wecom"
	foundationBase := strings.TrimRight(prefix, "/") + "/admin/social/openwork/foundation"
	platformBase := strings.TrimRight(prefix, "/") + "/admin/social/channel-platform/wecom/openwork"
	return map[string]authx.Permission{
		"GET:" + base:                                                      {Resource: "scrm.social_channel_accounts", Action: "read"},
		"POST:" + base:                                                     {Resource: "scrm.social_channel_accounts", Action: "write"},
		"GET:" + base + "/deleted":                                         {Resource: "scrm.social_channel_accounts", Action: "read"},
		"GET:" + base + "/*":                                               {Resource: "scrm.social_channel_accounts", Action: "read"},
		"PUT:" + base + "/:account_uuid":                                   {Resource: "scrm.social_channel_accounts", Action: "write"},
		"POST:" + base + "/:account_uuid/test-connection":                  {Resource: "scrm.social_channel_accounts", Action: "write"},
		"POST:" + base + "/:account_uuid/test-app-secret":                  {Resource: "scrm.social_channel_accounts", Action: "write"},
		"POST:" + base + "/:account_uuid/test-contact-secret":              {Resource: "scrm.social_channel_accounts", Action: "write"},
		"DELETE:" + base + "/:account_uuid":                                {Resource: "scrm.social_channel_accounts", Action: "write"},
		"POST:" + base + "/:account_uuid/restore":                          {Resource: "scrm.social_channel_accounts", Action: "write"},
		"POST:" + base + "/:account_uuid/channel-members":                  {Resource: "scrm.social_channel_accounts", Action: "write"},
		"PATCH:" + base + "/:account_uuid/capabilities":                    {Resource: "scrm.social_channel_accounts", Action: "write"},
		"POST:" + openworkBase + "/events":                                 {Resource: "scrm.social_channel_accounts", Action: "write"},
		"POST:" + openworkBase + "/authorize/start":                        {Resource: "scrm.social_channel_accounts", Action: "write"},
		"POST:" + openworkBase + "/authorize/restart":                      {Resource: "scrm.social_channel_accounts", Action: "write"},
		"POST:" + openworkBase + "/authorize/complete":                     {Resource: "scrm.social_channel_accounts", Action: "write"},
		"GET:" + openworkBase + "/authorize/status":                        {Resource: "scrm.social_channel_accounts", Action: "read"},
		"GET:" + openworkBase + "/foundation/access/status":                {Resource: "scrm.social_channel_accounts", Action: "read"},
		"GET:" + openworkBase + "/bindings":                                {Resource: "scrm.social_channel_accounts", Action: "read"},
		"POST:" + openworkBase + "/bindings/:binding_uuid/default":         {Resource: "scrm.social_channel_accounts", Action: "write"},
		"POST:" + openworkBase + "/sync/jobs":                              {Resource: "scrm.social_channel_accounts", Action: "write"},
		"GET:" + openworkBase + "/sync/jobs":                               {Resource: "scrm.social_channel_accounts", Action: "read"},
		"GET:" + openworkBase + "/sync/conflicts":                          {Resource: "scrm.social_channel_accounts", Action: "read"},
		"POST:" + openworkBase + "/sync/conflicts/:conflict_uuid/replay":   {Resource: "scrm.social_channel_accounts", Action: "write"},
		"GET:" + openworkBase + "/sync/dashboard":                          {Resource: "scrm.social_channel_accounts", Action: "read"},
		"GET:" + openworkBase + "/go-live-gates":                           {Resource: "scrm.social_channel_accounts", Action: "read"},
		"POST:" + foundationBase + "/sync/jobs":                            {Resource: "scrm.social_channel_accounts", Action: "write"},
		"GET:" + foundationBase + "/sync/jobs":                             {Resource: "scrm.social_channel_accounts", Action: "read"},
		"GET:" + foundationBase + "/sync/overview":                         {Resource: "scrm.social_channel_accounts", Action: "read"},
		"GET:" + foundationBase + "/sync/metrics":                          {Resource: "scrm.social_channel_accounts", Action: "read"},
		"GET:" + foundationBase + "/capabilities":                          {Resource: "scrm.social_channel_accounts", Action: "read"},
		"GET:" + foundationBase + "/sync/conflicts":                        {Resource: "scrm.social_channel_accounts", Action: "read"},
		"POST:" + foundationBase + "/sync/conflicts/:conflict_uuid/replay": {Resource: "scrm.social_channel_accounts", Action: "write"},
		"GET:" + foundationBase + "/sync/dead-letters":                     {Resource: "scrm.social_channel_accounts", Action: "read"},
		"POST:" + foundationBase + "/sync/dead-letters/:dead_letter_uuid/replay": {
			Resource: "scrm.social_channel_accounts",
			Action:   "write",
		},
		"GET:" + platformBase:                                      {Resource: "scrm.social_channel_accounts", Action: "read"},
		"PUT:" + platformBase:                                      {Resource: "scrm.social_channel_accounts", Action: "write"},
		"GET:" + platformBase + "/templates":                       {Resource: "scrm.social_channel_accounts", Action: "read"},
		"POST:" + platformBase + "/templates":                      {Resource: "scrm.social_channel_accounts", Action: "write"},
		"PUT:" + platformBase + "/templates/:template_id":          {Resource: "scrm.social_channel_accounts", Action: "write"},
		"DELETE:" + platformBase + "/templates/:template_id":       {Resource: "scrm.social_channel_accounts", Action: "write"},
		"POST:" + platformBase + "/templates/:template_id/default": {Resource: "scrm.social_channel_accounts", Action: "write"},
		"POST:" + platformBase + "/suite-ticket/refresh":           {Resource: "scrm.social_channel_accounts", Action: "write"},
		"POST:" + platformBase + "/suite-ticket/verify":            {Resource: "scrm.social_channel_accounts", Action: "write"},
	}
}
