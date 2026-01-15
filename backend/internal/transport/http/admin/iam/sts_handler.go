package iam

import (
	"net/http"
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/contracts"
	authx "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/middleware"
	iamservice "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/iam"
	"github.com/gin-gonic/gin"
)

type STSHandler struct {
	mode iamservice.IAMMode
	svc  *iamservice.STSService
}

func NewSTSHandler(mode iamservice.IAMMode, svc *iamservice.STSService) *STSHandler {
	return &STSHandler{mode: mode, svc: svc}
}

func (h *STSHandler) Mint(c *gin.Context) {
	if h == nil || h.svc == nil || h.mode != iamservice.IAMModeLocal {
		contracts.ResponseServiceUnavailable(c, "当前路由仅在 Standalone 模式生效", nil)
		return
	}
	tc, ok := authx.GetTenantContext(c)
	if !ok || strings.TrimSpace(tc.TenantUUID) == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing tenant context"})
		return
	}
	token, err := h.svc.Mint(c.Request.Context(), tc)
	if err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, token)
}
