package security

import (
	"net/http"
	"strconv"
	"time"

	toolgrantservice "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/agent/tool_grant"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
)

type ToolGrantHandler struct {
	service *toolgrantservice.Service
}

func NewToolGrantHandler(deps *app.Deps) *ToolGrantHandler {
	signingKey := []byte(deps.Config.Security.ToolGrantSecret)
	logger := deps.RuntimeLogger(deps.Ctx, "admin_toolgrant", nil)
	svc := toolgrantservice.NewService(deps.DB, deps.Config, logger, signingKey)
	return &ToolGrantHandler{service: svc}
}

func (h *ToolGrantHandler) Revoke(c *gin.Context) {
	var payload struct {
		TenantUuid  string `json:"tenant_uuid" binding:"required"`
		ToolGrantID string `json:"toolgrant_id" binding:"required"`
		Reason      string `json:"reason"`
		RequestedBy string `json:"requested_by"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request payload"})
		return
	}
	if payload.RequestedBy == "" {
		payload.RequestedBy = "admin"
	}
	ttl := time.Now().UTC()
	if err := h.service.Revoke(c.Request.Context(), payload.TenantUuid, payload.ToolGrantID, payload.Reason, payload.RequestedBy, ttl); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *ToolGrantHandler) ListRevocations(c *gin.Context) {
	tenantID := c.Query("tenant_uuid")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_uuid is required"})
		return
	}
	limit := 0
	if v := c.Query("limit"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			limit = parsed
		}
	}
	records, err := h.service.RevocationHistory(c.Request.Context(), tenantID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": records})
}

func (h *ToolGrantHandler) ListUsageEvents(c *gin.Context) {
	tenantID := c.Query("tenant_uuid")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_uuid is required"})
		return
	}
	toolGrantID := c.Query("toolgrant_id")
	limit := 0
	if v := c.Query("limit"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			limit = parsed
		}
	}
	rows, err := h.service.UsageHistory(c.Request.Context(), tenantID, toolGrantID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": rows})
}
