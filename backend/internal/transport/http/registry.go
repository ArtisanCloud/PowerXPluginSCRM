package http

import (
	"fmt"
	"strings"

	oprepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/operations"
	authx "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/middleware"
	opservice "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/operations"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/admin"
	admincapability "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/admin/capability"
	adminconsole "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/admin/console"
	adminiam "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/admin/iam"
	adminintegration "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/admin/integration"
	adminmarketplace "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/admin/marketplace"
	adminoperations "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/admin/operations"
	adminruntime "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/admin/runtime_ops"
	adminsecurity "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/admin/security"
	AdminSocial "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/admin/social_channel_governance"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/admin/templates"
	agentapi "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/agent"
	integrationapi "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/integration"
	publicassets "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/public/assets"
	publicmarketplace "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/public/marketplace"
	webhooksapi "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/webhooks"
	tenantmarketplace "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/tenant/marketplace"
	"github.com/gin-gonic/gin"
)

// Registry API 注册器
type Registry struct {
	engine *gin.Engine
	deps   *app.Deps
	rbac   map[string]authx.Permission
}

// NewRegistry 创建 API 注册器
func NewRegistry(engine *gin.Engine, deps *app.Deps) *Registry {
	return &Registry{
		engine: engine,
		deps:   deps,
		rbac:   map[string]authx.Permission{},
	}
}

// RegisterRoutes 注册所有路由
func (r *Registry) RegisterAPIRoutes(gApi *gin.RouterGroup) {
	admin.RegisterAPIRoutes(gApi, r.deps)
	agentapi.RegisterAPIRoutes(gApi, r.deps)
	templates.RegisterAPIRoutes(gApi, r.deps)
	integrationapi.RegisterAPIRoutes(gApi, r.deps)
	webhooksapi.RegisterRoutes(gApi, r.deps)
	r.RegisterMarketplaceRoutes(gApi)
	if isDevEnvironment(r.deps) {
		r.registerDevAssetsRoute()
	}

	r.mergeRBAC(adminruntime.RBACEntries(r.apiPrefix()))
	r.mergeRBAC(adminsecurity.RBACEntries(r.apiPrefix()))
	r.mergeRBAC(adminintegration.RBACEntries(r.apiPrefix()))
	r.mergeRBAC(adminoperations.RBACEntries(r.apiPrefix()))
	r.mergeRBAC(adminconsole.RBACEntries(r.apiPrefix()))
	r.mergeRBAC(adminmarketplace.RBACEntries(r.apiPrefix()))
	r.mergeRBAC(admincapability.RBACEntries(r.apiPrefix()))
	r.mergeRBAC(adminiam.RBACEntries(r.apiPrefix()))
	r.mergeRBAC(AdminSocial.RBACEntries(r.apiPrefix()))
	r.mergeRBAC(templates.RBACEntries(r.apiPrefix()))
	r.mergeRBAC(integrationRBACEntries(r.apiPrefix()))
	r.mergeRBAC(marketplacePublicRBACEntries(r.apiPrefix()))
}

func (r *Registry) PrintRegisteredRoutes() {
	routes := r.engine.Routes()
	fmt.Println("==== Registered Routes ====")
	for _, route := range routes {
		// 格式化输出：方法、路径、处理函数
		fmt.Printf("%-6s %-30s %s\n", route.Method, route.Path, route.Handler)
	}
	fmt.Println("===========================")
}

// RBACMap 汇总所有模块的 RBAC 声明。
func (r *Registry) RBACMap() map[string]authx.Permission {
	out := make(map[string]authx.Permission, len(r.rbac))
	for route, perm := range r.rbac {
		out[route] = perm
	}
	return out
}

func (r *Registry) mergeRBAC(entries map[string]authx.Permission) {
	if entries == nil {
		return
	}
	for route, perm := range entries {
		r.rbac[route] = perm
	}
}

// RegisterMarketplaceRoutes wires the public marketplace group so later phases can attach handlers.
func (r *Registry) RegisterMarketplaceRoutes(root *gin.RouterGroup) *gin.RouterGroup {
	if root == nil {
		return nil
	}
	group := root.Group("/marketplace")
	tenantmarketplace.RegisterRoutes(group, r.deps)
	if r.deps != nil && r.deps.DB != nil {
		slaRepo := oprepo.NewSLARepository(r.deps.DB)
		slaSvc := opservice.NewSLAService(slaRepo, r.deps.Config, r.deps.OperationsMetrics)
		publicmarketplace.Register(group, publicmarketplace.NewSLAHandler(slaRepo, slaSvc))
	}
	return group
}

func (r *Registry) apiPrefix() string {
	prefix := "/api/v1"
	if r.deps != nil && r.deps.Config != nil && r.deps.Config.Server != nil {
		if p := strings.TrimSpace(r.deps.Config.Server.APIPrefix); p != "" {
			if !strings.HasPrefix(p, "/") {
				p = "/" + p
			}
			prefix = p
		}
	}
	return prefix
}

func isDevEnvironment(deps *app.Deps) bool {
	if deps == nil || deps.Config == nil || deps.Config.Server == nil {
		return true
	}
	server := deps.Config.Server
	if server.DevMode {
		return true
	}
	mode := strings.TrimSpace(strings.ToLower(server.Mode))
	return mode == "" || mode == "debug"
}

func (r *Registry) registerDevAssetsRoute() {
	if r.engine == nil {
		return
	}
	paths := []string{"/assets/builds/meta"}
	apiPref := strings.TrimRight(r.apiPrefix(), "/")
	if apiPref != "" && apiPref != "-" {
		paths = append(paths, apiPref+"/assets/builds/meta")
	}

	for _, p := range paths {
		publicassets.RegisterBuildMetaRoutes(r.engine, p)
	}
}

func marketplacePublicRBACEntries(prefix string) map[string]authx.Permission {
	base := strings.TrimRight(prefix, "/") + "/marketplace"
	return map[string]authx.Permission{
		"GET:" + base + "/listings":                           {Resource: "marketplace.listings", Action: "read"},
		"POST:" + base + "/listings":                          {Resource: "marketplace.listings", Action: "write"},
		"GET:" + base + "/listings/*":                         {Resource: "marketplace.listings", Action: "read"},
		"PATCH:" + base + "/listings/*":                       {Resource: "marketplace.listings", Action: "write"},
		"POST:" + base + "/listings/*":                        {Resource: "marketplace.listings", Action: "review"},
		"POST:" + base + "/listings/*/status":                 {Resource: "marketplace.listings", Action: "review"},
		"POST:" + base + "/licenses":                          {Resource: "marketplace.license", Action: "purchase"},
		"GET:" + base + "/licenses/*":                         {Resource: "marketplace.license", Action: "read"},
		"POST:" + base + "/licenses/*":                        {Resource: "marketplace.license", Action: "manage"},
		"POST:" + base + "/licenses/*/offline-extend":         {Resource: "marketplace.license", Action: "manage"},
		"POST:" + base + "/usage":                             {Resource: "marketplace.usage", Action: "ingest"},
		"GET:" + base + "/usage/tenants/*/licenses/*/metrics": {Resource: "marketplace.usage", Action: "view"},
		"GET:" + base + "/revenue-share/reports":              {Resource: "marketplace.revenue", Action: "read"},
		"GET:" + base + "/sla/*":                              {Resource: "marketplace.sla", Action: "read"},
	}
}
