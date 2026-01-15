package admin

import (
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/contracts"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/logger"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"

	"github.com/gin-gonic/gin"
)

// AdminHandler 管理端处理器
type AdminHandler struct {
	deps *app.Deps
}

// NewAdminHandler 创建管理端处理器
func NewAdminHandler(deps *app.Deps) *AdminHandler {
	return &AdminHandler{
		deps: deps,
	}
}

// GetManifest 获取插件清单
func (h *AdminHandler) GetManifest(c *gin.Context) {
	log := logger.HandlerLogger("admin").WithContext(c.Request.Context())

	manifest := &contracts.PluginManifest{
		ID:          "com.powerx.plugins.base",
		Name:        "Base Template Plugin",
		Version:     "0.1.0",
		Description: "A starter plugin that showcases template management capabilities for PowerX",
		Author:      "PowerX Team",
		Homepage:    "https://powerx.dev/plugins/base",
		Repository:  "https://powerx.dev/plugins/base.git",
		License:     "MIT",
		Tags:        []string{"base", "template", "starter"},

		Backend: contracts.BackendConfig{
			Entry:  "backend/bin/plugin",
			Port:   8078,
			Health: "/healthz",
		},

		Frontend: &contracts.FrontendConfig{
			Entry: "web-admin/.output",
			Routes: map[string]string{
				"/admin/*": "index.html",
			},
			PublicPath: "/_p/com.powerx.plugins.base/admin/",
		},

		Menus: []contracts.MenuConfig{
			{
				ID:    "base",
				Title: "menu.base.template",
				Icon:  "i-heroicons-clipboard-document-check",
				Path:  "/plugins/base",
				Order: 20,
				Children: []contracts.MenuConfig{
					{
						ID:    "base.intro",
						Title: "menu.base.intro",
						Icon:  "i-heroicons-information-circle",
						Path:  "/intro",
						Order: 1,
					},
					{
						ID:    "base.templates",
						Title: "menu.base.templates.title",
						Icon:  "i-heroicons-clipboard-document-list",
						Path:  "/templates",
						Order: 2,
						Children: []contracts.MenuConfig{
							{
								ID:    "base.templates.develop",
								Title: "menu.base.templates.develop",
								Icon:  "i-heroicons-document-text",
								Path:  "/templates/develop",
								Order: 1,
							},
							{
								ID:    "base.templates.crud",
								Title: "menu.base.templates.crud",
								Icon:  "i-heroicons-wrench",
								Path:  "/templates/crud",
								Order: 2,
							},
						},
					},
				},
				RequiredPermissions: []string{"base:template:read"},
			},
		},

		Permissions: []contracts.PermissionConfig{
			{
				Resource:    "base:template",
				Actions:     []string{"read", "create", "update", "delete"},
				Description: "Template management permissions",
			},
		},

		Agents: []contracts.AgentConfig{
			{
				ID:           "base.assistant",
				PluginID:     "com.powerx.plugins.base",
				Name:         "Base 助理",
				Description:  "智能的 Base 模板助手，可以帮助创建与查询模板内容",
				Model:        "gpt-4",
				Instructions: "你是一个专业的 Base 模板助手。你可以帮助用户创建、查询和管理模板信息。请始终以友好、专业的方式回应用户的请求。",
				DefaultTools: []string{
					"base.template.create",
					"base.template.query",
				},
				RequiredPermissions: []string{"base:template:read"},
			},
		},

		Tools: []contracts.ToolConfig{
			{
				ID:           "base.template.create",
				PluginID:     "com.powerx.plugins.base",
				Name:         "创建模板",
				Description:  "创建一个新的模板记录",
				Transport:    "http",
				Endpoint:     "/api/v1/templates",
				Method:       "POST",
				RBACResource: "base:template",
				InputSchema: &contracts.JSONSchema{
					Type: "object",
					Properties: map[string]*contracts.JSONSchemaProperty{
						"name": {
							Type:        "string",
							Description: "模板名称",
						},
						"description": {
							Type:        "string",
							Description: "模板描述",
						},
						"content": {
							Type:        "string",
							Description: "模板内容",
						},
					},
					Required: []string{"name", "description", "content"},
				},
				OutputSchema: &contracts.JSONSchema{
					Type: "object",
					Properties: map[string]*contracts.JSONSchemaProperty{
						"id": {
							Type:        "integer",
							Description: "模板ID",
						},
						"name": {
							Type:        "string",
							Description: "模板名称",
						},
						"description": {
							Type:        "string",
							Description: "模板描述",
						},
						"content": {
							Type:        "string",
							Description: "模板内容",
						},
					},
				},
				Timeout: 30,
			},
			{
				ID:           "base.template.query",
				PluginID:     "com.powerx.plugins.base",
				Name:         "查询模板",
				Description:  "查询模板列表",
				Transport:    "http",
				Endpoint:     "/api/v1/templates",
				Method:       "GET",
				RBACResource: "base:template",
				InputSchema: &contracts.JSONSchema{
					Type: "object",
					Properties: map[string]*contracts.JSONSchemaProperty{
						"q": {
							Type:        "string",
							Description: "按名称或描述搜索",
						},
						"page": {
							Type:        "integer",
							Minimum:     func() *float64 { v := 1.0; return &v }(),
							Default:     1,
							Description: "页码",
						},
						"limit": {
							Type:        "integer",
							Minimum:     func() *float64 { v := 1.0; return &v }(),
							Maximum:     func() *float64 { v := 100.0; return &v }(),
							Default:     20,
							Description: "每页数量",
						},
					},
				},
				OutputSchema: &contracts.JSONSchema{
					Type: "object",
					Properties: map[string]*contracts.JSONSchemaProperty{
						"list": {
							Type: "array",
							Items: &contracts.JSONSchema{
								Type: "object",
								Properties: map[string]*contracts.JSONSchemaProperty{
									"id": {
										Type:        "integer",
										Description: "模板ID",
									},
									"name": {
										Type:        "string",
										Description: "模板名称",
									},
									"description": {
										Type:        "string",
										Description: "模板描述",
									},
									"content": {
										Type:        "string",
										Description: "模板内容",
									},
								},
							},
						},
					},
				},
			},
		},

		Dependencies: []contracts.DependencyConfig{
			{
				Name:    "postgresql",
				Version: ">=12.0",
				Type:    "database",
			},
		},

		ConfigSchema: &contracts.ConfigSchema{
			Type: "object",
			Properties: map[string]*contracts.JSONSchemaProperty{
				"template_auto_publish": {
					Type:        "boolean",
					Description: "是否自动发布新建模板",
					Default:     false,
				},
				"template_notification": {
					Type:        "boolean",
					Description: "模板变更是否通知订阅者",
					Default:     true,
				},
			},
		},

		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	log.Info("Plugin manifest requested")

	contracts.ResponseSuccess(c, manifest)
}

// GetRBACInfo 获取 RBAC 信息
func (h *AdminHandler) GetRBACInfo(c *gin.Context) {
	log := logger.HandlerLogger("admin").WithContext(c.Request.Context())

	rbacInfo := &contracts.RBACInfo{
		Resources: []contracts.Resource{
			{
				Name:        "base:template",
				Description: "Base 模板管理",
				Actions: []contracts.Action{
					{Name: "read", Description: "查看模板"},
					{Name: "create", Description: "创建模板"},
					{Name: "update", Description: "更新模板"},
					{Name: "delete", Description: "删除模板"},
				},
			},
		},
		Roles: []contracts.Role{
			{
				Name:        "base_master",
				Description: "Base Master 角色",
				Permissions: []string{
					"base:template:*",
				},
			},
			{
				Name:        "template_editor",
				Description: "模板编辑角色",
				Permissions: []string{
					"base:template:read",
					"base:template:create",
					"base:template:update",
				},
			},
			{
				Name:        "template_viewer",
				Description: "模板查看角色",
				Permissions: []string{
					"base:template:read",
				},
			},
		},
		Permissions: []contracts.Permission{
			{Resource: "base:template", Action: "read"},
			{Resource: "base:template", Action: "create"},
			{Resource: "base:template", Action: "update"},
			{Resource: "base:template", Action: "delete"},
		},
	}

	log.Info("RBAC info requested")

	contracts.ResponseSuccess(c, rbacInfo)
}
