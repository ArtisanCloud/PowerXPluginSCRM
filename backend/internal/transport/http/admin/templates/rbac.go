package templates

import (
	"strings"

	authx "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/middleware"
)

// RBACEntries declares route → permission mappings for template APIs.
func RBACEntries(prefix string) map[string]authx.Permission {
	base := strings.TrimRight(prefix, "/") + "/templates"
	adminBase := strings.TrimRight(prefix, "/") + "/admin/templates"
	read := authx.Permission{Resource: "template", Action: "read"}
	create := authx.Permission{Resource: "template", Action: "create"}
	update := authx.Permission{Resource: "template", Action: "update"}
	delete := authx.Permission{Resource: "template", Action: "delete"}

	return map[string]authx.Permission{
		"GET:" + base:                        read,
		"GET:" + base + "/*":                 read,
		"POST:" + base:                       create,
		"PUT:" + base + "/*":                 update,
		"DELETE:" + base + "/*":              delete,
		"POST:" + base + "/batch-clone":      create,
		"POST:" + base + "/*/validate":       update,
		"POST:" + adminBase + "/batch-clone": create,
		"POST:" + adminBase + "/*/validate":  update,
	}
}
