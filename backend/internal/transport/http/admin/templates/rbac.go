package templates

import (
	"strings"

	authx "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/middleware"
)

// RBACEntries declares route → permission mappings for template APIs.
func RBACEntries(prefix string) map[string]authx.Permission {
	base := strings.TrimRight(prefix, "/") + "/templates"
	adminBase := strings.TrimRight(prefix, "/") + "/admin/templates"
	read := authx.Permission{Resource: "base.templates", Action: "read"}
	manage := authx.Permission{Resource: "base.templates", Action: "manage"}

	return map[string]authx.Permission{
		"GET:" + base:                   read,
		"GET:" + base + "/*":            read,
		"POST:" + base:                  manage,
		"PUT:" + base + "/*":            manage,
		"DELETE:" + base + "/*":         manage,
		"POST:" + base + "/batch-clone": manage,
		"POST:" + base + "/*/validate":  manage,
		"POST:" + adminBase + "/batch-clone": manage,
		"POST:" + adminBase + "/*/validate":  manage,
	}
}
