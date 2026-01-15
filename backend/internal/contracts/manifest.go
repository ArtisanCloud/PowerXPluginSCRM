package contracts

import "time"

// PluginManifest 插件清单契约
type PluginManifest struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Description string   `json:"description"`
	Author      string   `json:"author,omitempty"`
	Homepage    string   `json:"homepage,omitempty"`
	Repository  string   `json:"repository,omitempty"`
	License     string   `json:"license,omitempty"`
	Tags        []string `json:"tags,omitempty"`

	// 后端配置
	Backend BackendConfig `json:"backend"`

	// 前端配置
	Frontend *FrontendConfig `json:"frontend,omitempty"`

	// 菜单配置
	Menus []MenuConfig `json:"menus,omitempty"`

	// 权限配置
	Permissions []PermissionConfig `json:"permissions,omitempty"`

	// Agent 能力
	Agents []AgentConfig `json:"agents,omitempty"`

	// 工具定义
	Tools []ToolConfig `json:"tools,omitempty"`

	// 能力声明
	Capabilities *ManifestCapabilities `json:"capabilities,omitempty"`

	// 工作流定义
	Workflows []WorkflowConfig `json:"workflows,omitempty"`

	// 依赖
	Dependencies []DependencyConfig `json:"dependencies,omitempty"`

	// 配置模式
	ConfigSchema *ConfigSchema `json:"config_schema,omitempty"`

	// 生命周期
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// BackendConfig 后端配置
type BackendConfig struct {
	Entry  string `json:"entry"`  // 入口文件路径
	Port   int    `json:"port"`   // 监听端口
	Health string `json:"health"` // 健康检查端点
}

// FrontendConfig 前端配置
type FrontendConfig struct {
	Entry      string            `json:"entry"`                 // 入口文件路径
	Routes     map[string]string `json:"routes,omitempty"`      // 路由映射
	Assets     []string          `json:"assets,omitempty"`      // 静态资源
	PublicPath string            `json:"public_path,omitempty"` // 公共路径
}

// MenuConfig 菜单配置
type MenuConfig struct {
	ID       string       `json:"id"`
	Title    string       `json:"title"`
	Icon     string       `json:"icon,omitempty"`
	Path     string       `json:"path,omitempty"`
	Order    int          `json:"order,omitempty"`
	Parent   string       `json:"parent,omitempty"`
	Children []MenuConfig `json:"children,omitempty"`
	Meta     interface{}  `json:"meta,omitempty"`

	// 权限控制
	RequiredRoles       []string `json:"required_roles,omitempty"`
	RequiredPermissions []string `json:"required_permissions,omitempty"`
}

// PermissionConfig 权限配置
type PermissionConfig struct {
	Resource    string   `json:"resource"`
	Actions     []string `json:"actions"`
	Description string   `json:"description,omitempty"`
}

// AgentConfig Agent 配置
type AgentConfig struct {
	ID           string   `json:"id"`
	PluginID     string   `json:"plugin_id"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Model        string   `json:"model,omitempty"`
	Instructions string   `json:"instructions,omitempty"`
	DefaultTools []string `json:"default_tools,omitempty"`

	// 配置参数
	Config map[string]interface{} `json:"config,omitempty"`

	// 权限
	RequiredPermissions []string `json:"required_permissions,omitempty"`
}

// ToolConfig 工具配置
type ToolConfig struct {
	ID          string `json:"id"`
	PluginID    string `json:"plugin_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Transport   string `json:"transport"` // http, grpc
	Endpoint    string `json:"endpoint"`
	Method      string `json:"method,omitempty"` // HTTP 方法

	// 权限
	RBACResource string `json:"rbac_resource,omitempty"`

	// 输入输出模式
	InputSchema  *JSONSchema `json:"input_schema,omitempty"`
	OutputSchema *JSONSchema `json:"output_schema,omitempty"`

	// 配置
	Config map[string]interface{} `json:"config,omitempty"`

	// 超时设置
	Timeout int `json:"timeout,omitempty"` // 秒
}

// WorkflowConfig 工作流配置
type WorkflowConfig struct {
	ID          string `json:"id"`
	PluginID    string `json:"plugin_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Endpoint    string `json:"endpoint"`

	// 输入模式
	InputSchema *JSONSchema `json:"input_schema,omitempty"`

	// 步骤定义
	Steps []WorkflowStep `json:"steps,omitempty"`

	// 权限
	RequiredPermissions []string `json:"required_permissions,omitempty"`
}

// WorkflowStep 工作流步骤
type WorkflowStep struct {
	ID     string                 `json:"id"`
	Name   string                 `json:"name"`
	Type   string                 `json:"type"` // tool, condition, loop, etc.
	Config map[string]interface{} `json:"config,omitempty"`
	Next   []string               `json:"next,omitempty"`
}

// DependencyConfig 依赖配置
type DependencyConfig struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Type    string `json:"type"` // plugin, service, database, etc.
}

// ManifestCapabilities 提供插件能力的描述引用
type ManifestCapabilities struct {
	Provides []ManifestCapabilityRef `json:"provides,omitempty"`
	Consumes []ManifestCapabilityRef `json:"consumes,omitempty"`
}

// ManifestCapabilityRef 指向具体的能力定义文件及关联 Schema。
type ManifestCapabilityRef struct {
	ID         string              `json:"id"`
	Version    string              `json:"version,omitempty"`
	Descriptor string              `json:"descriptor,omitempty"`
	Schemas    *ManifestSchemaRefs `json:"schemas,omitempty"`
	Metadata   map[string]string   `json:"metadata,omitempty"`
}

// ManifestSchemaRefs 声明能力相关的输入输出 Schema。
type ManifestSchemaRefs struct {
	Input  []string `json:"input,omitempty"`
	Output []string `json:"output,omitempty"`
}

// ConfigSchema 配置模式定义
type ConfigSchema struct {
	Type       string                         `json:"type"`
	Properties map[string]*JSONSchemaProperty `json:"properties,omitempty"`
	Required   []string                       `json:"required,omitempty"`
}

// JSONSchema JSON Schema 定义
type JSONSchema struct {
	Type        string                         `json:"type"`
	Properties  map[string]*JSONSchemaProperty `json:"properties,omitempty"`
	Items       *JSONSchema                    `json:"items,omitempty"`
	Required    []string                       `json:"required,omitempty"`
	Enum        []interface{}                  `json:"enum,omitempty"`
	Default     interface{}                    `json:"default,omitempty"`
	Description string                         `json:"description,omitempty"`
	Format      string                         `json:"format,omitempty"`
	Minimum     *float64                       `json:"minimum,omitempty"`
	Maximum     *float64                       `json:"maximum,omitempty"`
	MinLength   *int                           `json:"minLength,omitempty"`
	MaxLength   *int                           `json:"maxLength,omitempty"`
}

// JSONSchemaProperty JSON Schema 属性
type JSONSchemaProperty struct {
	Type        string        `json:"type"`
	Description string        `json:"description,omitempty"`
	Format      string        `json:"format,omitempty"`
	Enum        []interface{} `json:"enum,omitempty"`
	Default     interface{}   `json:"default,omitempty"`
	Minimum     *float64      `json:"minimum,omitempty"`
	Maximum     *float64      `json:"maximum,omitempty"`
	MinLength   *int          `json:"minLength,omitempty"`
	MaxLength   *int          `json:"maxLength,omitempty"`
	Items       *JSONSchema   `json:"items,omitempty"`
}

// RBACInfo RBAC 信息契约
type RBACInfo struct {
	Resources   []Resource   `json:"resources"`
	Roles       []Role       `json:"roles,omitempty"`
	Permissions []Permission `json:"permissions,omitempty"`
}

// Resource 资源定义
type Resource struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Actions     []Action `json:"actions"`
}

// Action 动作定义
type Action struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// Role 角色定义
type Role struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
}

// Permission 权限定义
type Permission struct {
	Resource string `json:"resource"`
	Action   string `json:"action"`
}

// EventTopic 事件主题定义
type EventTopic struct {
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	Schema      *JSONSchema `json:"schema,omitempty"`
}

// PluginStatus 插件状态
type PluginStatus struct {
	Status     string    `json:"status"` // installed, enabled, disabled, error
	Version    string    `json:"version"`
	Health     bool      `json:"health"`
	LastCheck  time.Time `json:"last_check"`
	Message    string    `json:"message,omitempty"`
	ErrorCount int       `json:"error_count,omitempty"`
}

// PluginConfig 插件配置实例
type PluginConfig struct {
	PluginID   string                 `json:"plugin_id"`
	TenantUuid string                 `json:"tenant_uuid,omitempty"`
	Config     map[string]interface{} `json:"config"`
	Enabled    bool                   `json:"enabled"`
	UpdatedAt  time.Time              `json:"updated_at"`
}
