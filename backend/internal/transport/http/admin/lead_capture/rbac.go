package lead_capture

import (
	"strings"

	authx "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/middleware"
)

// RBACEntries returns RBAC mappings for lead capture admin APIs.
func RBACEntries(prefix string) map[string]authx.Permission {
	base := strings.TrimRight(prefix, "/") + "/admin/leads"
	return map[string]authx.Permission{
		"GET:" + base:                       {Resource: "scrm.leads", Action: "read"},
		"POST:" + base:                      {Resource: "scrm.leads", Action: "write"},
		"GET:" + base + "/*":                {Resource: "scrm.leads", Action: "read"},
		"PUT:" + base + "/:lead_id":         {Resource: "scrm.leads", Action: "write"},
		"DELETE:" + base + "/:lead_id":      {Resource: "scrm.leads", Action: "write"},
		"POST:" + base + "/:lead_id/assign": {Resource: "scrm.leads", Action: "write"},
		"POST:" + base + "/:lead_id/status": {Resource: "scrm.leads", Action: "write"},
	}
}
