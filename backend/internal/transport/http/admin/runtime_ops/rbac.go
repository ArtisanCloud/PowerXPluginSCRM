package runtime_ops

import (
	"strings"

	authx "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/middleware"
)

// RBACEntries exposes route-to-permission mappings for runtime ops admin APIs.
func RBACEntries(prefix string) map[string]authx.Permission {
	base := strings.TrimRight(prefix, "/") + "/admin/runtime"
	return map[string]authx.Permission{
		"POST:" + base + "/bootstrap":                    {Resource: "runtime.ops", Action: "manage"},
		"POST:" + base + "/sessions/register":            {Resource: "runtime.ops", Action: "manage"},
		"POST:" + base + "/sessions/*/ack":               {Resource: "runtime.ops", Action: "manage"},
		"POST:" + base + "/sessions/*/heartbeat":         {Resource: "runtime.ops", Action: "observe"},
		"POST:" + base + "/sessions/*/close":             {Resource: "runtime.ops", Action: "manage"},
		"POST:" + base + "/sessions/*/invoke":            {Resource: "runtime.ops", Action: "invoke"},
		"GET:" + base + "/quota/status":                  {Resource: "runtime.ops", Action: "read"},
		"POST:" + base + "/quota/overrides":              {Resource: "runtime.ops", Action: "manage"},
		"GET:" + base + "/metrics":                       {Resource: "runtime.ops", Action: "observe"},
		"GET:" + base + "/dictionaries":                  {Resource: "runtime.ops", Action: "read"},
		"POST:" + base + "/dictionaries":                 {Resource: "runtime.ops", Action: "manage"},
		"PATCH:" + base + "/dictionaries/:item_id":       {Resource: "runtime.ops", Action: "manage"},
		"DELETE:" + base + "/dictionaries/:item_id":      {Resource: "runtime.ops", Action: "manage"},
		"POST:" + base + "/event-bridge/emit":            {Resource: "runtime.ops", Action: "invoke"},
		"POST:" + base + "/internal/event-fabric/topics": {Resource: "runtime.ops", Action: "invoke"},
		"POST:" + base + "/internal/ws-bus/publish":      {Resource: "runtime.ops", Action: "invoke"},
		"POST:" + base + "/internal/ws-bus/grant":        {Resource: "runtime.ops", Action: "invoke"},
		"POST:" + base + "/internal/ws-bus/register":     {Resource: "runtime.ops", Action: "invoke"},
	}
}
