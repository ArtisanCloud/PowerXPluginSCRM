package capability

import (
	"strings"

	authx "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/middleware"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
)

// RBACEntries describes the minimal permissions required by capability routes.
func RBACEntries(prefix string) map[string]authx.Permission {
	base := strings.TrimRight(prefix, "/") + "/admin/capabilities/register"
	resource := app.PluginID + ":capability"
	listBase := strings.TrimRight(prefix, "/") + "/admin/capabilities"
	reviewBase := strings.TrimRight(prefix, "/") + "/admin/capabilities/reviews"
	reviewRes := app.PluginID + ":capability.review"
	exposureBase := strings.TrimRight(prefix, "/") + "/admin/capabilities/exposure"
	exposureRes := app.PluginID + ":capability.exposure"
	quotaBase := strings.TrimRight(prefix, "/") + "/admin/capabilities/quotas"
	quotaRes := app.PluginID + ":capability.quota"
	lifecycleBase := strings.TrimRight(prefix, "/") + "/admin/capabilities/lifecycle"
	lifecycleRes := app.PluginID + ":capability.lifecycle"
	return map[string]authx.Permission{
		"GET:" + listBase:                          {Resource: resource, Action: "read"},
		"GET:" + base + "/template":                {Resource: resource, Action: "read"},
		"POST:" + base + "/validate":               {Resource: resource, Action: "create"},
		"POST:" + base:                             {Resource: resource, Action: "create"},
		"GET:" + reviewBase + "/*":                 {Resource: reviewRes, Action: "read"},
		"POST:" + reviewBase + "/*/resubmit":       {Resource: reviewRes, Action: "update"},
		"POST:" + reviewBase + "/tasks/*/comments": {Resource: reviewRes, Action: "update"},
		"POST:" + reviewBase + "/tasks/*/decision": {Resource: reviewRes, Action: "review"},
		"GET:" + exposureBase + "/template":        {Resource: exposureRes, Action: "read"},
		"GET:" + exposureBase + "/*":               {Resource: exposureRes, Action: "read"},
		"PUT:" + exposureBase + "/*":               {Resource: exposureRes, Action: "write"},
		"GET:" + quotaBase + "/*":                  {Resource: quotaRes, Action: "read"},
		"POST:" + quotaBase + "/*":                 {Resource: quotaRes, Action: "manage"},
		"GET:" + lifecycleBase + "/template":       {Resource: lifecycleRes, Action: "read"},
		"GET:" + lifecycleBase:                     {Resource: lifecycleRes, Action: "read"},
		"POST:" + lifecycleBase:                    {Resource: lifecycleRes, Action: "write"},
		"POST:" + lifecycleBase + "/*/status":      {Resource: lifecycleRes, Action: "manage"},
	}
}
