package marketplace

import (
	"strings"

	authx "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/middleware"
)

// RBACEntries declares admin marketplace route-to-permission mappings.
func RBACEntries(prefix string) map[string]authx.Permission {
	base := strings.TrimRight(prefix, "/") + "/admin/marketplace"
	return map[string]authx.Permission{
		"GET:" + base + "/listings":                    {Resource: "marketplace.listings", Action: "read"},
		"POST:" + base + "/listings":                   {Resource: "marketplace.listings", Action: "write"},
		"GET:" + base + "/listings/:id":                {Resource: "marketplace.listings", Action: "read"},
		"PATCH:" + base + "/listings/:id":              {Resource: "marketplace.listings", Action: "write"},
		"POST:" + base + "/listings/:id/review":        {Resource: "marketplace.listings", Action: "review"},
		"POST:" + base + "/listings/:id/publish":       {Resource: "marketplace.listings", Action: "review"},
		"POST:" + base + "/listings/:id/suspend":       {Resource: "marketplace.listings", Action: "review"},
		"POST:" + base + "/checklist/graphql":          {Resource: "marketplace.listings", Action: "review"},
		"GET:" + base + "/recommendation/experiments":  {Resource: "marketplace.recommendation", Action: "read"},
		"POST:" + base + "/recommendation/experiments": {Resource: "marketplace.recommendation", Action: "manage"},
		"PATCH:" + base + "/recommendation/experiments/:id": {
			Resource: "marketplace.recommendation", Action: "manage",
		},
		"POST:" + base + "/recommendation/weights/refresh": {
			Resource: "marketplace.recommendation", Action: "manage",
		},
		"POST:" + base + "/usage": {
			Resource: "marketplace.usage", Action: "ingest",
		},
		"GET:" + base + "/usage/tenants/:tenantId/licenses/:licenseId/metrics": {
			Resource: "marketplace.usage", Action: "view",
		},
		"GET:" + base + "/revenue-share/reports": {
			Resource: "marketplace.revenue", Action: "read",
		},
	}
}
