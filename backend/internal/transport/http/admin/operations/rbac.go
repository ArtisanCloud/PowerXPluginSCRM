package operations

import (
	"strings"

	authx "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/middleware"
)

// RBACEntries returns the Operations admin RBAC definitions.
func RBACEntries(prefix string) map[string]authx.Permission {
	base := strings.TrimRight(prefix, "/") + "/admin/operations"
	return map[string]authx.Permission{
		// Support Playbook
		"GET:" + base + "/support/playbook":       {Resource: "operations.support", Action: "read"},
		"PUT:" + base + "/support/playbook":       {Resource: "operations.support", Action: "manage"},
		"POST:" + base + "/support/channels/test": {Resource: "operations.support", Action: "manage"},
		"GET:" + base + "/support/metrics":        {Resource: "operations.support", Action: "read"},
		// Incident lifecycle
		"POST:" + base + "/incidents":              {Resource: "operations.incident", Action: "command"},
		"GET:" + base + "/incidents":               {Resource: "operations.incident", Action: "read"},
		"GET:" + base + "/incidents/:incidentId":   {Resource: "operations.incident", Action: "read"},
		"PATCH:" + base + "/incidents/:incidentId": {Resource: "operations.incident", Action: "command"},
		"POST:" + base + "/incidents/:incidentId/timeline": {
			Resource: "operations.incident",
			Action:   "command",
		},
		// SLA transparency
		"GET:" + base + "/sla/profiles":            {Resource: "operations.sla", Action: "read"},
		"POST:" + base + "/sla/profiles":           {Resource: "operations.sla", Action: "manage"},
		"POST:" + base + "/sla/profiles/recompute": {Resource: "operations.sla", Action: "command"},
		"PATCH:" + base + "/sla/profiles/actuals":  {Resource: "operations.sla", Action: "command"},
	}
}
