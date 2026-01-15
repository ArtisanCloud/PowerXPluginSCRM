package integration

import (
	"strings"

	authx "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/middleware"
)

// RBACEntries 返回 Integration 管理端的权限映射。
func RBACEntries(prefix string) map[string]authx.Permission {
	base := strings.TrimRight(prefix, "/") + "/admin/integration"
	return map[string]authx.Permission{
		"GET:" + base + "/approvals":              {Resource: "integration.approvals", Action: "read"},
		"POST:" + base + "/approvals/:id/approve": {Resource: "integration.approvals", Action: "manage"},
		"POST:" + base + "/approvals/:id/reject":  {Resource: "integration.approvals", Action: "manage"},
		"GET:" + base + "/grant-matrix":           {Resource: "integration.grant_matrix", Action: "read"},
		"GET:" + base + "/webhooks":               {Resource: "integration.webhooks", Action: "read"},
		"POST:" + base + "/webhooks":              {Resource: "integration.webhooks", Action: "manage"},
		"PUT:" + base + "/webhooks/:id":           {Resource: "integration.webhooks", Action: "manage"},
		"DELETE:" + base + "/webhooks/:id":        {Resource: "integration.webhooks", Action: "manage"},
		"GET:" + base + "/webhooks/:id/attempts":  {Resource: "integration.webhooks", Action: "read"},
		"POST:" + base + "/webhooks/attempts/:attemptId/replay": {
			Resource: "integration.webhooks",
			Action:   "manage",
		},
		"GET:" + base + "/secrets":                      {Resource: "integration.secrets", Action: "read"},
		"POST:" + base + "/secrets":                     {Resource: "integration.secrets", Action: "manage"},
		"POST:" + base + "/secrets/:id/rotate":          {Resource: "integration.secrets", Action: "manage"},
		"POST:" + base + "/secrets/:id/rotate/complete": {Resource: "integration.secrets", Action: "manage"},
		"POST:" + base + "/secrets/:id/revoke":          {Resource: "integration.secrets", Action: "manage"},
		"GET:" + base + "/secrets/:id/audit":            {Resource: "integration.secrets", Action: "read"},
	}
}
