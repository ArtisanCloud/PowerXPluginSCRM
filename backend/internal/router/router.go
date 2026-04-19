package router

import (
	stdhttp "net/http"
	"os"
	"strings"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/config"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/logger"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/middleware"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http"
	mcptransport "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/mcp"
	middleware2 "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/middleware"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/mini-app"
	publicauth "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/public"
	webhooksapi "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/webhooks"
	wstransport "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/websocket"

	"github.com/gin-gonic/gin"
)

// Router 路由器结构
type Router struct {
	engine *gin.Engine
	cfg    *config.Config
	deps   *app.Deps
}

// New 创建新的路由器
func NewRouter(cfg *config.Config, deps *app.Deps) *Router {
	return &Router{
		cfg:  cfg,
		deps: deps,
	}
}

// Setup 设置路由
func (r *Router) Setup() *gin.Engine {
	// 设置 Gin 模式
	if r.cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	// 创建 Gin 引擎
	r.engine = gin.New()

	// 设置全局中间件
	r.setupGlobalMiddleware()

	// 设置业务路由
	r.setupRoutes()

	logger.Info("Router setup completed")
	return r.engine
}

// setupGlobalMiddleware 设置全局中间件
func (r *Router) setupGlobalMiddleware() {

	// 健康检查（在其它中间件前放行 /healthz）
	r.engine.Use(middleware.HealthCheck("/healthz"))

	// 恢复
	r.engine.Use(middleware.Recovery())

	// 请求日志
	r.engine.Use(middleware.RequestLogger())

	// 安全头
	r.engine.Use(middleware.SecurityHeaders())

	// CORS
	r.engine.Use(middleware.CORS())

	// 请求 ID
	r.engine.Use(middleware.RequestID())

	// 超时（30 秒）
	r.engine.Use(middleware.Timeout(30 * time.Second))

	// 速率限制（每分钟最多 100 个请求）
	r.engine.Use(middleware.RateLimiter(100, time.Minute))

	// —— 仅在“不在 PowerX 宿主内”且“非生产”时，才启用 DevSwitch —— //
	// 避免 PowerX 模式被 DevSwitch 绕过鉴权。
	if !r.cfg.IsProduction() && os.Getenv("POWERX_PROXY") != "1" {
		tenantUUID := "00000000-0000-0000-0000-000000000001"
		if r.cfg.GRPCUpstream != nil && strings.TrimSpace(r.cfg.GRPCUpstream.TenantUUID) != "" {
			tenantUUID = strings.TrimSpace(r.cfg.GRPCUpstream.TenantUUID)
		}
		r.engine.Use(middleware2.DevSwitch(true, middleware.TenantContext{
			TenantUUID:  tenantUUID,
			UserID:      0,
			Roles:       []string{"superadmin"},
			Permissions: []string{"*"},
		}))
	}
}

// setupRoutes 设置路由
func (r *Router) setupRoutes() {
	// —— API 前缀：默认 /api/v1，并确保带前导斜杠 —— //
	prefix := r.cfg.Server.APIPrefix
	if strings.TrimSpace(prefix) == "" {
		prefix = "/api/v1"
	}
	if !strings.HasPrefix(prefix, "/") {
		prefix = "/" + prefix
	}

	// 鉴权/权限组件
	jwtCfg := r.buildJWT()
	rbacCfg := r.buildRBAC()

	publicauth.RegisterAuthRoutes(r.engine.Group(prefix), r.deps)

	// Mini-app routes use customer auth and should not be guarded by admin JWT/RBAC middleware.
	gMiniApp := r.engine.Group(prefix)
	gMiniApp.Use(middleware2.RequestTrace())
	miniapp.RegisterAPIRoutes(gMiniApp, r.deps)
	// Public webhooks (e.g. WeCom OpenWork callback verify/event) must be reachable without admin JWT.
	gPublicWebhook := r.engine.Group(prefix)
	gPublicWebhook.Use(middleware2.RequestTrace())
	webhooksapi.RegisterPublicRoutes(gPublicWebhook, r.deps)

	// 使用 API 注册器注册所有路由（保持你现有的注册逻辑）
	apiRegistry := http.NewRegistry(r.engine, r.deps)

	// API 分组 + 鉴权 + RBAC（Admin / Integration / Marketplace 等）
	gProtected := r.engine.Group(prefix)
	gProtected.Use(middleware2.RequestTrace())
	gProtected.Use(middleware.AdminSignatureGuard())
	gProtected.Use(middleware2.JWTAuth(jwtCfg))
	gProtected.Use(middleware2.RBAC(rbacCfg, nil, nil))
	apiRegistry.RegisterAPIRoutes(gProtected)
	r.injectRBACFromRegistry(rbacCfg, apiRegistry)
	r.inferRBACFromRoutes(rbacCfg, prefix)

	mcptransport.RegisterRoutes(r.engine, prefix)
	wstransport.RegisterWSRoutes(r.engine, middleware2.JWTAuth(jwtCfg), r.cfg)

	// 如需调试：打印已注册路由
	// apiRegistry.PrintRegisteredRoutes()
}

// GetEngine 获取 Gin 引擎
func (r *Router) GetEngine() *gin.Engine {
	return r.engine
}

// injectRBACFromRegistry 将各模块声明的 RBAC 合并到配置中。
func (r *Router) injectRBACFromRegistry(rbacCfg *middleware.RBACConfig, reg *http.Registry) {
	if rbacCfg == nil || reg == nil {
		return
	}
	if rbacCfg.DelegateToPowerX {
		return
	}
	for route, perm := range reg.RBACMap() {
		rbacCfg.RoutePermissions[route] = rbacCfg.NormalizePermission(perm)
	}
}

func (r *Router) inferRBACFromRoutes(rbacCfg *middleware.RBACConfig, prefix string) {
	if r == nil || r.engine == nil || rbacCfg == nil || rbacCfg.DelegateToPowerX {
		return
	}
	basis := strings.TrimRight(prefix, "/")
	for _, route := range r.engine.Routes() {
		if route.Method == stdhttp.MethodOptions {
			continue
		}
		if basis != "" && !strings.HasPrefix(route.Path, basis) {
			continue
		}
		key := route.Method + ":" + route.Path
		if _, exists := rbacCfg.RoutePermissions[key]; exists {
			continue
		}
		perm, ok := middleware.InferPermission(route.Method, route.Path)
		if !ok {
			continue
		}
		rbacCfg.RoutePermissions[key] = rbacCfg.NormalizePermission(perm)
	}
}

// RegisterCustomRoutes 注册自定义路由
func (r *Router) RegisterCustomRoutes(fn func(*gin.Engine)) {
	if r.engine != nil && fn != nil {
		fn(r.engine)
	}
}

// RegisterMiddleware 注册中间件
func (r *Router) RegisterMiddleware(m gin.HandlerFunc) {
	if r.engine != nil {
		r.engine.Use(m)
	}
}

// —— 从配置构造 JWT 配置（自动区分 PowerX 宿主/本地直连） —— //
func (r *Router) buildJWT() middleware.JWTAuthConfig {
	inPX := os.Getenv("POWERX_PROXY") == "1"
	if inPX {
		// PowerX 网关严格模式：使用宿主注入的安全参数
		pid := strings.TrimSpace(os.Getenv("POWERX_PLUGIN_ID"))
		aud := strings.TrimSpace(os.Getenv("POWERX_SECURITY_JWT_AUDIENCE"))
		if aud == "" && pid != "" {
			aud = "plugin:" + pid
		}
		return middleware.JWTAuthConfig{
			Issuer:             strings.TrimSpace(os.Getenv("POWERX_SECURITY_JWT_ISSUER")),
			AcceptAudiences:    []string{aud},
			HMACSecret:         strings.TrimSpace(os.Getenv("POWERX_SECURITY_JWT_SECRET")), // 可为空：只走签名上下文
			ContextHMACSecret:  strings.TrimSpace(os.Getenv("POWERX_SECURITY_CTX_HMAC_SECRET")),
			AllowSignedContext: true,  // 允许 X-PowerX-CTX / X-PowerX-CTX-SIG
			Optional:           false, // 严格：失败即 401
			ClockSkewSeconds:   60,
			MaxCtxAgeSeconds:   300,
		}
	}

	// 本地直连开发：默认也严格验证，可通过 POWERX_AUTH_OPTIONAL 手动放宽
	prod := r.cfg.IsProduction()
	optional := false
	if v := strings.TrimSpace(os.Getenv("POWERX_AUTH_OPTIONAL")); v != "" {
		optional = v == "1" || strings.EqualFold(v, "true")
	} else if !prod {
		optional = true
	}

	issuer := "powerx-local"
	audiences := []string{"powerx:plugin"}
	if ctx := r.cfg.Context; ctx != nil {
		if v := strings.TrimSpace(ctx.Issuer); v != "" {
			issuer = v
		}
		if v := strings.TrimSpace(ctx.Audience); v != "" {
			audiences = splitAudiences(v)
		}
	}

	cfg := middleware.JWTAuthConfig{
		Issuer:             issuer,
		AcceptAudiences:    audiences,
		HMACSecret:         "", // 如需本地校验 HS256，可在 config.Context.HMACSecret 配置
		ClockSkewSeconds:   60,
		Optional:           optional,
		AllowSignedContext: false, // 本地通常不走签名上下文；如要测试，置 true 并填 ContextHMACSecret
		ContextHMACSecret:  "",
		MaxCtxAgeSeconds:   300,
	}

	if ctx := r.cfg.Context; ctx != nil {
		if v := strings.TrimSpace(ctx.HMACSecret); v != "" {
			cfg.HMACSecret = v
			if cfg.ContextHMACSecret == "" {
				cfg.ContextHMACSecret = v
			}
		}
	}

	return cfg
}

func splitAudiences(raw string) []string {
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ';'
	})
	var cleaned []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			cleaned = append(cleaned, p)
		}
	}
	if len(cleaned) == 0 {
		return []string{strings.TrimSpace(raw)}
	}
	return cleaned
}

// —— 从配置构造 RBAC 配置 —— //
func (r *Router) buildRBAC() *middleware.RBACConfig {
	delegate := shouldDelegateToPowerX(r.cfg)
	issuer := strings.TrimSpace(os.Getenv("POWERX_SECURITY_JWT_ISSUER"))
	aud := strings.TrimSpace(os.Getenv("POWERX_SECURITY_JWT_AUDIENCE"))
	if aud == "" {
		if pid := strings.TrimSpace(os.Getenv("POWERX_PLUGIN_ID")); pid != "" {
			aud = "plugin:" + pid
		}
	}
	return &middleware.RBACConfig{
		Enabled:          true,
		DefaultDeny:      true,
		SuperAdminRoles:  []string{"superadmin", "admin"},
		RoutePermissions: map[string]middleware.Permission{},
		DelegateToPowerX: delegate,
		PowerXIssuer:     issuer,
		PowerXAudience:   aud,
		PluginID:         app.PluginID,
	}
}

func shouldDelegateToPowerX(cfg *config.Config) bool {
	if cfg != nil && cfg.Context != nil {
		switch strings.ToLower(strings.TrimSpace(cfg.Context.IAMMode)) {
		case "delegated":
			return true
		case "local":
			return false
		}
	}
	return os.Getenv("POWERX_PROXY") == "1"
}
