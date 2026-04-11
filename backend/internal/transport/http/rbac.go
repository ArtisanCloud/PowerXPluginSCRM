package http

import (
	"strings"

	authx "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/middleware"
)

// integrationRBACEntries 返回 Integration 运行时 API 权限映射。
func integrationRBACEntries(prefix string) map[string]authx.Permission {
	base := strings.TrimRight(prefix, "/") + "/integration"
	return map[string]authx.Permission{
		"POST:" + base + "/dispatch":               {Resource: "integration.dispatch", Action: "invoke"},
		"POST:" + base + "/capabilities/invoke":    {Resource: "integration", Action: "create"},
		"GET:" + base + "/grant-matrix":            {Resource: "integration.grant_matrix", Action: "read"},
		"POST:" + base + "/grant-matrix":           {Resource: "integration.grant_matrix", Action: "manage"},
		"POST:" + base + "/webhooks/subscriptions": {Resource: "integration.webhooks", Action: "manage"},
		"GET:" + base + "/webhooks/subscriptions":  {Resource: "integration.webhooks", Action: "read"},
		"POST:" + base + "/webhooks/dlq/:attemptId/replay": {
			Resource: "integration.webhooks",
			Action:   "manage",
		},
		"POST:" + base + "/secrets":                  {Resource: "integration.secrets", Action: "manage"},
		"POST:" + base + "/secrets/:secretId/rotate": {Resource: "integration.secrets", Action: "manage"},
	}
}
