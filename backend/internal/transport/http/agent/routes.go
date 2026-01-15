package agent

import (
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
	agentsecurity "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/agent/security"
	"github.com/gin-gonic/gin"
)

// RegisterAPIRoutes 注册 Agent 相关路由
// 路由风格参考 admin/templates：以模块为前缀分组
// 最终路径示例：/api/v1/agent/tenants/:tenantId/credentials
func RegisterAPIRoutes(rg *gin.RouterGroup, deps *app.Deps) {
	h := &CredentialHandler{deps: deps}

	g := rg.Group("/agent")
	{
		g.POST("/tenants/:tenantId/credentials", h.Upsert)
		// STS 调试端点：主动触发 Exchange
		RegisterSTSRoutes(g, deps)
		agentsecurity.RegisterRoutes(g, deps)
	}
}
