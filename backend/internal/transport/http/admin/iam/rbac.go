package iam

import (
	"strings"

	authx "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/middleware"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
)

func RBACEntries(prefix string) map[string]authx.Permission {
	base := strings.TrimRight(prefix, "/") + "/admin/iam"
	return map[string]authx.Permission{
		"GET:" + base + "/tenants":             {Resource: app.PluginID + ":iam.tenant", Action: "read"},
		"POST:" + base + "/tenants":            {Resource: app.PluginID + ":iam.tenant", Action: "write"},
		"PATCH:" + base + "/tenants/*":         {Resource: app.PluginID + ":iam.tenant", Action: "write"},
		"GET:" + base + "/departments":         {Resource: app.PluginID + ":iam.department", Action: "read"},
		"POST:" + base + "/departments":        {Resource: app.PluginID + ":iam.department", Action: "write"},
		"PATCH:" + base + "/departments/*":     {Resource: app.PluginID + ":iam.department", Action: "write"},
		"DELETE:" + base + "/departments/*":    {Resource: app.PluginID + ":iam.department", Action: "delete"},
		"GET:" + base + "/members":             {Resource: app.PluginID + ":iam.user", Action: "read"},
		"POST:" + base + "/members":            {Resource: app.PluginID + ":iam.user", Action: "write"},
		"PATCH:" + base + "/members/*":         {Resource: app.PluginID + ":iam.user", Action: "write"},
		"POST:" + base + "/members/import":     {Resource: app.PluginID + ":iam.user", Action: "write"},
		"GET:" + base + "/user-directory":      {Resource: app.PluginID + ":iam.user", Action: "read"},
		"GET:" + base + "/users":               {Resource: app.PluginID + ":iam.user", Action: "read"},
		"POST:" + base + "/users":              {Resource: app.PluginID + ":iam.user", Action: "write"},
		"PATCH:" + base + "/users/*":           {Resource: app.PluginID + ":iam.user", Action: "write"},
		"POST:" + base + "/users/import":       {Resource: app.PluginID + ":iam.user", Action: "write"},
		"GET:" + base + "/roles":               {Resource: app.PluginID + ":iam.role", Action: "read"},
		"POST:" + base + "/roles":              {Resource: app.PluginID + ":iam.role", Action: "write"},
		"PATCH:" + base + "/roles/*":           {Resource: app.PluginID + ":iam.role", Action: "write"},
		"DELETE:" + base + "/roles/*":          {Resource: app.PluginID + ":iam.role", Action: "delete"},
		"PUT:" + base + "/roles/*/permissions": {Resource: app.PluginID + ":iam.role", Action: "write"},
		"POST:" + base + "/roles/*/members":    {Resource: app.PluginID + ":iam.role", Action: "write"},
		"DELETE:" + base + "/roles/*/members":  {Resource: app.PluginID + ":iam.role", Action: "write"},
		"GET:" + base + "/permissions":         {Resource: app.PluginID + ":iam.permission", Action: "read"},
		"GET:" + base + "/audit/logs":          {Resource: app.PluginID + ":iam.audit", Action: "read"},
		"POST:" + base + "/auth/local/sts":     {Resource: app.PluginID + ":iam.sts", Action: "mint"},
	}
}
