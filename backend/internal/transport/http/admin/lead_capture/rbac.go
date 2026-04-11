package lead_capture

import (
	"strings"

	authx "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/middleware"
)

// RBACEntries returns RBAC mappings for lead capture admin APIs.
func RBACEntries(prefix string) map[string]authx.Permission {
	base := strings.TrimRight(prefix, "/") + "/admin/leads"
	return map[string]authx.Permission{
		"GET:" + base:                                                                            {Resource: "scrm.leads", Action: "read"},
		"POST:" + base:                                                                           {Resource: "scrm.leads", Action: "write"},
		"POST:" + base + "/import":                                                               {Resource: "scrm.leads", Action: "write"},
		"POST:" + base + "/import/preview":                                                       {Resource: "scrm.leads", Action: "write"},
		"POST:" + base + "/import/confirm":                                                       {Resource: "scrm.leads", Action: "write"},
		"POST:" + base + "/assign/batch":                                                         {Resource: "scrm.leads", Action: "write"},
		"GET:" + base + "/*":                                                                     {Resource: "scrm.leads", Action: "read"},
		"PUT:" + base + "/:lead_id":                                                              {Resource: "scrm.leads", Action: "write"},
		"DELETE:" + base + "/:lead_id":                                                           {Resource: "scrm.leads", Action: "write"},
		"POST:" + base + "/:lead_id/assign":                                                      {Resource: "scrm.leads", Action: "write"},
		"POST:" + base + "/:lead_id/status":                                                      {Resource: "scrm.leads", Action: "write"},
		"GET:" + base + "/:lead_id/assignments":                                                  {Resource: "scrm.leads", Action: "read"},
		"GET:" + base + "/:lead_id/status-history":                                               {Resource: "scrm.leads", Action: "read"},
		"GET:" + base + "/:lead_id/activities":                                                   {Resource: "scrm.leads", Action: "read"},
		"GET:" + base + "/:lead_id/sources":                                                      {Resource: "scrm.leads", Action: "read"},
		"POST:" + base + "/wecom/sync":                                                           {Resource: "scrm.leads", Action: "write"},
		"GET:" + base + "/wecom/sync-tasks":                                                      {Resource: "scrm.leads", Action: "read"},
		"POST:" + base + "/wecom/sync-tasks/clear":                                               {Resource: "scrm.leads", Action: "write"},
		"GET:" + base + "/wecom/writeback-policy":                                                {Resource: "scrm.leads", Action: "read"},
		"PUT:" + base + "/wecom/writeback-policy":                                                {Resource: "scrm.leads", Action: "write"},
		"GET:" + base + "/wecom/writeback-dead-letters":                                          {Resource: "scrm.leads", Action: "read"},
		"POST:" + base + "/wecom/writeback-dead-letters/:dead_letter_uuid/replay":                {Resource: "scrm.leads", Action: "write"},
		"GET:" + base + "/channel-rules/wecom/customer-dm":                                       {Resource: "scrm.leads", Action: "read"},
		"PUT:" + base + "/channel-rules/wecom/customer-dm":                                       {Resource: "scrm.leads", Action: "write"},
		"GET:" + base + "/:lead_id/conversations":                                                {Resource: "scrm.leads", Action: "read"},
		"POST:" + base + "/:lead_id/conversations/bind":                                          {Resource: "scrm.leads", Action: "write"},
		"GET:" + base + "/source-catalogs":                                                       {Resource: "scrm.leads", Action: "read"},
		"POST:" + base + "/source-catalogs":                                                      {Resource: "scrm.leads", Action: "write"},
		"PATCH:" + base + "/source-catalogs/:catalog_id":                                         {Resource: "scrm.leads", Action: "write"},
		"DELETE:" + base + "/source-catalogs/:catalog_id":                                        {Resource: "scrm.leads", Action: "write"},
		"POST:" + base + "/channel-codes":                                                        {Resource: "channel_code", Action: "create"},
		"GET:" + base + "/channel-codes":                                                         {Resource: "channel_code", Action: "read"},
		"GET:" + base + "/channel-codes/:code_uuid/events":                                       {Resource: "channel_code", Action: "read"},
		"PATCH:" + base + "/channel-codes/:code_uuid/status":                                     {Resource: "channel_code", Action: "update"},
		"PUT:" + base + "/channel-codes/:code_uuid/welcome-config":                               {Resource: "channel_code", Action: "update"},
		"GET:" + base + "/channel-codes/:code_uuid/welcome-config/history":                       {Resource: "channel_code", Action: "read"},
		"POST:" + base + "/channel-codes/:code_uuid/welcome-config/sync":                         {Resource: "channel_welcome_sync", Action: "publish"},
		"GET:" + base + "/channel-codes/:code_uuid/welcome-config/sync-status":                   {Resource: "channel_welcome_sync", Action: "read"},
		"GET:" + strings.TrimRight(prefix, "/") + "/admin/conversations/:conversation_id/events": {Resource: "scrm.leads", Action: "read"},
	}
}
