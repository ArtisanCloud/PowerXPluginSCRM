package agent

import (
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/contracts"
	repo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/plugin"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/logger"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/agent"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// 凭证投递请求
type UpsertCredentialsRequest struct {
	PluginID     string `json:"plugin_id" binding:"required"`
	ClientID     string `json:"client_id" binding:"required"`
	ClientSecret string `json:"client_secret" binding:"required"`
	// 可选：rotate/version/issued_at 等字段，后续扩展
}

type UpsertCredentialsResponse struct {
	PluginID string `json:"plugin_id"`
}

type CredentialHandler struct{ deps *app.Deps }

// Upsert 接收宿主发送的明文凭证，落库（加密）
func (h *CredentialHandler) Upsert(c *gin.Context) {
	var req UpsertCredentialsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid body: "+err.Error())
		return
	}
	tenantUUID := strings.TrimSpace(c.Param("tenantId"))
	if tenantUUID == "" {
		contracts.ResponseBadRequest(c, "invalid tenant_uuid")
		return
	}
	if _, err := uuid.Parse(tenantUUID); err != nil {
		contracts.ResponseBadRequest(c, "invalid tenant_uuid")
		return
	}

	// TODO: 鉴权（生产环境需要 JWT/HMAC/mTLS）；开发模式可放宽，由上层中间件控制

	svc := agent.NewCredentialService(h.deps.Config, repo.NewCredentialsRepository(h.deps.DB))
	if err := svc.SavePlainCredentials(c.Request.Context(), tenantUUID, req.PluginID, req.ClientID, req.ClientSecret); err != nil {
		logger.WithError(err).Warn("save credentials failed")
		contracts.ResponseInternalError(c, err)
		return
	}
	// 若当前 deps 中的租户就是该租户，立即使内存 token 失效，触发下次调用刷新
	if h.deps != nil && h.deps.PowerXClient != nil && h.deps.Config != nil && h.deps.Config.GRPCUpstream != nil {
		if strings.EqualFold(h.deps.Config.GRPCUpstream.TenantUUID, tenantUUID) {
			h.deps.PowerXClient.InvalidateSTS()
		}
	}
	contracts.ResponseSuccess(c, &UpsertCredentialsResponse{PluginID: req.PluginID})
}
