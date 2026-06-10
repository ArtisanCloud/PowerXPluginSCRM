package opportunity

import (
	"strings"

	authx "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/middleware"
)

func RBACEntries(prefix string) map[string]authx.Permission {
	base := strings.TrimRight(prefix, "/") + "/admin/opportunity"
	return map[string]authx.Permission{
		"GET:" + base + "/dashboard":                                                {Resource: "scrm.opportunity", Action: "read"},
		"GET:" + base + "/records":                                                  {Resource: "scrm.opportunity", Action: "read"},
		"POST:" + base + "/records":                                                 {Resource: "scrm.opportunity", Action: "write"},
		"GET:" + base + "/records/:opportunity_uuid":                                {Resource: "scrm.opportunity", Action: "read"},
		"PUT:" + base + "/records/:opportunity_uuid":                                {Resource: "scrm.opportunity", Action: "write"},
		"GET:" + base + "/records/:opportunity_uuid/line-items":                     {Resource: "scrm.opportunity", Action: "read"},
		"POST:" + base + "/records/:opportunity_uuid/line-items":                    {Resource: "scrm.opportunity", Action: "write"},
		"POST:" + base + "/records/:opportunity_uuid/quote-files":                   {Resource: "scrm.opportunity", Action: "write"},
		"GET:" + base + "/records/:opportunity_uuid/line-items/:item_uuid/download": {Resource: "scrm.opportunity", Action: "read"},
		"DELETE:" + base + "/records/:opportunity_uuid/line-items/:item_uuid":       {Resource: "scrm.opportunity", Action: "write"},
		"GET:" + base + "/records/:opportunity_uuid/tasks":                          {Resource: "scrm.opportunity", Action: "read"},
		"POST:" + base + "/records/:opportunity_uuid/tasks":                         {Resource: "scrm.opportunity", Action: "write"},
		"PATCH:" + base + "/records/:opportunity_uuid/tasks/:task_uuid/status":      {Resource: "scrm.opportunity", Action: "write"},
		"POST:" + base + "/records/:opportunity_uuid/stage":                         {Resource: "scrm.opportunity", Action: "write"},
		"POST:" + base + "/records/:opportunity_uuid/close":                         {Resource: "scrm.opportunity", Action: "write"},
		"POST:" + base + "/records/:opportunity_uuid/reopen":                        {Resource: "scrm.opportunity", Action: "write"},
		"POST:" + base + "/records/:opportunity_uuid/risk":                          {Resource: "scrm.opportunity", Action: "write"},
		"GET:" + base + "/records/:opportunity_uuid/activities":                     {Resource: "scrm.opportunity", Action: "read"},
	}
}
