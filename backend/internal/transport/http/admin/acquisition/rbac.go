package acquisition

import (
	"strings"

	authx "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/middleware"
)

func RBACEntries(prefix string) map[string]authx.Permission {
	base := strings.TrimRight(prefix, "/") + "/admin/leads/acquisition"
	return map[string]authx.Permission{
		"POST:" + base + "/staff-codes":                                            {Resource: "acquisition.staff_code", Action: "create"},
		"GET:" + base + "/staff-codes":                                             {Resource: "acquisition.staff_code", Action: "read"},
		"PATCH:" + base + "/staff-codes/:staff_code_uuid/status":                   {Resource: "acquisition.staff_code", Action: "update"},
		"PUT:" + base + "/staff-codes/:staff_code_uuid/welcome-config":             {Resource: "acquisition.staff_welcome", Action: "update"},
		"POST:" + base + "/staff-codes/:staff_code_uuid/welcome-config/sync":       {Resource: "acquisition.staff_welcome", Action: "publish"},
		"GET:" + base + "/staff-codes/:staff_code_uuid/welcome-config/sync-status": {Resource: "acquisition.staff_welcome", Action: "read"},
		"GET:" + base + "/group-codes":                                             {Resource: "acquisition.group_code", Action: "read"},
	}
}
