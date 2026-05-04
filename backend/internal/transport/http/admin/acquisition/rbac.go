package acquisition

import (
	"strings"

	authx "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/middleware"
)

func RBACEntries(prefix string) map[string]authx.Permission {
	base := strings.TrimRight(prefix, "/") + "/admin/leads/acquisition"
	return map[string]authx.Permission{
		"POST:" + base + "/staff-codes":                                              {Resource: "acquisition.staff_code", Action: "create"},
		"GET:" + base + "/staff-codes":                                               {Resource: "acquisition.staff_code", Action: "read"},
		"DELETE:" + base + "/staff-codes/:staff_code_uuid":                           {Resource: "acquisition.staff_code", Action: "delete"},
		"PATCH:" + base + "/staff-codes/:staff_code_uuid/status":                     {Resource: "acquisition.staff_code", Action: "update"},
		"PUT:" + base + "/staff-codes/:staff_code_uuid/welcome-config":               {Resource: "acquisition.staff_welcome", Action: "update"},
		"POST:" + base + "/staff-codes/:staff_code_uuid/welcome-config/sync":         {Resource: "acquisition.staff_welcome", Action: "publish"},
		"GET:" + base + "/staff-codes/:staff_code_uuid/welcome-config/sync-status":   {Resource: "acquisition.staff_welcome", Action: "read"},
		"POST:" + base + "/group-codes":                                              {Resource: "acquisition.group_code", Action: "create"},
		"GET:" + base + "/group-codes":                                               {Resource: "acquisition.group_code", Action: "read"},
		"GET:" + base + "/group-codes/:group_code_uuid":                              {Resource: "acquisition.group_code", Action: "read"},
		"PUT:" + base + "/group-codes/:group_code_uuid":                              {Resource: "acquisition.group_code", Action: "update"},
		"DELETE:" + base + "/group-codes/:group_code_uuid":                           {Resource: "acquisition.group_code", Action: "delete"},
		"POST:" + base + "/group-codes/:group_code_uuid/sync":                        {Resource: "acquisition.group_code", Action: "publish"},
		"POST:" + base + "/group-chats/sync":                                         {Resource: "acquisition.group_chat", Action: "sync"},
		"GET:" + base + "/group-chats/sync/tasks":                                    {Resource: "acquisition.group_chat", Action: "read"},
		"POST:" + base + "/group-chats/sync/tasks/clear":                             {Resource: "acquisition.group_chat", Action: "sync"},
		"GET:" + base + "/group-chats":                                               {Resource: "acquisition.group_chat", Action: "read"},
		"GET:" + base + "/group-chats/:chat_id":                                      {Resource: "acquisition.group_chat", Action: "read"},
		"GET:" + base + "/group-chats/:chat_id/customers/:external_userid/timeline":  {Resource: "acquisition.group_chat", Action: "read"},
		"GET:" + base + "/group-chats/:chat_id/customers/:external_userid/followups": {Resource: "acquisition.group_chat", Action: "read"},
		"GET:" + base + "/group-chats/customers/:external_userid/relations":          {Resource: "acquisition.group_chat", Action: "read"},
		"POST:" + base + "/group-tags":                                               {Resource: "acquisition.group_tag", Action: "create"},
		"GET:" + base + "/group-tags":                                                {Resource: "acquisition.group_tag", Action: "read"},
		"POST:" + base + "/group-tags/:group_tag_uuid/bindings":                      {Resource: "acquisition.group_tag", Action: "update"},
		"GET:" + base + "/group-tags/:group_tag_uuid/bindings":                       {Resource: "acquisition.group_tag", Action: "read"},
		"POST:" + base + "/group-tags/:group_tag_uuid/rules/replay":                  {Resource: "acquisition.group_tag", Action: "update"},
	}
}
