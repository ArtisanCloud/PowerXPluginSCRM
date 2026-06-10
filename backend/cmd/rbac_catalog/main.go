package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	authx "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/middleware"
	httpregistry "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http"
	"gopkg.in/yaml.v3"
)

type permissionSpec struct {
	Resource string   `yaml:"resource"`
	Actions  []string `yaml:"actions"`
}

type routePermissionSpec struct {
	Method     string           `yaml:"method"`
	Path       string           `yaml:"path"`
	Permission permissionDetail `yaml:"permission"`
}

type permissionDetail struct {
	Resource string `yaml:"resource"`
	Action   string `yaml:"action"`
}

type routeSpec struct {
	BasePath      string                `yaml:"basePath"`
	AdminManifest string                `yaml:"adminManifest"`
	RBAC          string                `yaml:"rbac"`
	Permissions   []routePermissionSpec `yaml:"permissions,omitempty"`
}

type catalog struct {
	Permissions []permissionSpec `yaml:"permissions"`
	RBAC        struct {
		Resources []permissionSpec `yaml:"resources"`
	} `yaml:"rbac"`
	Routes routeSpec `yaml:"routes"`
}

func main() {
	prefix := flag.String("prefix", "/api/v1", "API prefix used by plugin routes")
	output := flag.String("output", "plugin.d/rbac.yaml", "output rbac catalog path")
	flag.Parse()

	doc := buildCatalog(*prefix, httpregistry.StaticRBACEntries(*prefix))
	raw, err := yaml.Marshal(doc)
	if err != nil {
		fail(err)
	}
	if err := os.WriteFile(*output, raw, 0o644); err != nil {
		fail(err)
	}
	fmt.Printf("rbac catalog written: %s\n", *output)
}

func buildCatalog(prefix string, entries map[string]authx.Permission) catalog {
	prefix = strings.TrimRight(strings.TrimSpace(prefix), "/")
	if prefix == "" {
		prefix = "/api/v1"
	}

	resourceActions := map[string]map[string]bool{}
	routeSpecs := make([]routePermissionSpec, 0, len(entries))
	for route, perm := range entries {
		method, fullPath, ok := splitRouteKey(route)
		if !ok {
			continue
		}
		method = strings.ToUpper(strings.TrimSpace(method))
		fullPath = strings.TrimSpace(fullPath)
		resource := strings.TrimSpace(perm.Resource)
		action := strings.TrimSpace(perm.Action)
		if method == "" || fullPath == "" || resource == "" || action == "" {
			continue
		}
		if _, ok := resourceActions[resource]; !ok {
			resourceActions[resource] = map[string]bool{}
		}
		resourceActions[resource][action] = true
		routeSpecs = append(routeSpecs, routePermissionSpec{
			Method: method,
			Path:   trimRoutePrefix(prefix, fullPath),
			Permission: permissionDetail{
				Resource: resource,
				Action:   action,
			},
		})
	}

	sort.Slice(routeSpecs, func(i, j int) bool {
		if routeSpecs[i].Path == routeSpecs[j].Path {
			return routeSpecs[i].Method < routeSpecs[j].Method
		}
		return routeSpecs[i].Path < routeSpecs[j].Path
	})

	resources := make([]string, 0, len(resourceActions))
	for resource := range resourceActions {
		resources = append(resources, resource)
	}
	sort.Strings(resources)

	permissionSpecs := make([]permissionSpec, 0, len(resources))
	for _, resource := range resources {
		actions := make([]string, 0, len(resourceActions[resource]))
		for action := range resourceActions[resource] {
			actions = append(actions, action)
		}
		sort.Strings(actions)
		permissionSpecs = append(permissionSpecs, permissionSpec{Resource: resource, Actions: actions})
	}

	doc := catalog{
		Permissions: permissionSpecs,
		Routes: routeSpec{
			BasePath:      prefix,
			AdminManifest: prefix + "/admin/manifest",
			RBAC:          prefix + "/admin/rbac",
			Permissions:   routeSpecs,
		},
	}
	doc.RBAC.Resources = permissionSpecs
	return doc
}

func trimRoutePrefix(prefix, path string) string {
	if prefix == "" || prefix == "/" {
		return path
	}
	if path == prefix {
		return "/"
	}
	if strings.HasPrefix(path, prefix+"/") {
		return strings.TrimPrefix(path, prefix)
	}
	return path
}

func splitRouteKey(route string) (string, string, bool) {
	route = strings.TrimSpace(route)
	if route == "" {
		return "", "", false
	}
	idx := strings.Index(route, ":/")
	if idx < 0 {
		return "", "", false
	}
	return route[:idx], route[idx+1:], true
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
