package middleware

import (
	"net/http"
	"path"
	"strings"
)

type Permission struct{ Resource, Action string }
type RBACConfig struct {
	Enabled          bool
	DefaultDeny      bool
	SuperAdminRoles  []string
	RoutePermissions map[string]Permission // "METHOD:/api/v1/templates/*" -> {template,read}
	DelegateToPowerX bool
	PowerXIssuer     string
	PowerXAudience   string
	PluginID         string
}

func IsSuperAdmin(userRoles, superRoles []string) bool {
	if len(userRoles) == 0 || len(superRoles) == 0 {
		return false
	}
	s := map[string]struct{}{}
	for _, r := range userRoles {
		s[strings.ToLower(strings.TrimSpace(r))] = struct{}{}
	}
	for _, r := range superRoles {
		if _, ok := s[strings.ToLower(strings.TrimSpace(r))]; ok {
			return true
		}
	}
	return false
}
func HasPerm(userPerms []string, need Permission) bool {
	if len(userPerms) == 0 {
		return false
	}
	res := strings.ToLower(strings.TrimSpace(need.Resource))
	act := strings.ToLower(strings.TrimSpace(need.Action))
	want := res + ":" + act
	for _, perm := range userPerms {
		p := strings.ToLower(strings.TrimSpace(perm))
		if p == "" {
			continue
		}
		if p == "*" || p == "*:*" {
			return true
		}
		if p == want {
			return true
		}
		idx := strings.LastIndex(p, ":")
		if idx <= 0 || idx >= len(p)-1 {
			continue
		}
		pr := p[:idx]
		pa := p[idx+1:]
		if pr == "*" || pr == res {
			if pa == "*" || pa == act {
				return true
			}
		}
	}
	return false
}
func MatchRoute(method, reqPath string, table map[string]Permission) (Permission, bool) {
	if table == nil {
		return Permission{}, false
	}
	if perm, ok := table[method+":"+reqPath]; ok {
		return perm, true
	}
	for k, perm := range table {
		if i := strings.IndexByte(k, ':'); i >= 0 {
			m, pat := k[:i], k[i+1:]
			if (m == method || m == "*") && match(pat, reqPath) {
				return perm, true
			}
		}
	}
	return Permission{}, false
}
func match(pat, p string) bool { ok, _ := path.Match(pat, p); return ok }

func InferPermission(method, reqPath string) (Permission, bool) {
	action := methodToAction(method)
	if action == "" {
		return Permission{}, false
	}
	raw := reqPath
	if idx := strings.IndexByte(raw, '?'); idx >= 0 {
		raw = raw[:idx]
	}
	segments := strings.FieldsFunc(strings.Trim(raw, "/"), func(r rune) bool { return r == '/' })
	var parts []string
	for _, seg := range segments {
		seg = strings.ToLower(strings.TrimSpace(seg))
		if seg == "" || seg == "api" || seg == "v1" || seg == "v2" {
			continue
		}
		if seg == "admin" || seg == "tenant" {
			continue
		}
		if strings.HasPrefix(seg, ":") {
			continue
		}
		parts = append(parts, sanitizeSegment(seg))
		if len(parts) >= 2 {
			break
		}
	}
	if len(parts) == 0 {
		return Permission{}, false
	}
	resource := parts[0]
	if len(parts) > 1 {
		resource = resource + "." + parts[1]
	}
	return Permission{Resource: resource, Action: action}, true
}

func sanitizeSegment(seg string) string {
	seg = strings.TrimSpace(seg)
	seg = strings.ReplaceAll(seg, "-", "_")
	seg = strings.ReplaceAll(seg, ".", "_")
	return seg
}

func methodToAction(method string) string {
	switch strings.ToUpper(method) {
	case http.MethodGet, http.MethodHead:
		return "read"
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return "manage"
	default:
		return ""
	}
}

// NormalizePermission ensures permission resources follow plugin scope namespace.
func (cfg *RBACConfig) NormalizePermission(perm Permission) Permission {
	perm.Resource = strings.TrimSpace(perm.Resource)
	perm.Action = strings.TrimSpace(perm.Action)
	if cfg == nil {
		return perm
	}
	pluginID := strings.TrimSpace(cfg.PluginID)
	if pluginID != "" && perm.Resource != "" && !strings.Contains(perm.Resource, ":") {
		perm.Resource = pluginID + ":" + perm.Resource
	}
	return perm
}
