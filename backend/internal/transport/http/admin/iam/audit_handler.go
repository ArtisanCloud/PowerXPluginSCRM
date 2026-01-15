package iam

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/contracts"
	authmw "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/middleware"
	srviam "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/iam"
	"github.com/gin-gonic/gin"
)

type AuditHandler struct {
	svc *srviam.AuditService
}

func NewAuditHandler(svc *srviam.AuditService) *AuditHandler {
	return &AuditHandler{svc: svc}
}

func (h *AuditHandler) List(c *gin.Context) {
	if h == nil || h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "审计日志服务未初始化", nil)
		return
	}
	tc, ok := authmw.GetTenantContext(c)
	if !ok || strings.TrimSpace(tc.TenantUUID) == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing tenant context"})
		return
	}
	reqTenant := strings.TrimSpace(c.Query("tenant_uuid"))
	systemAdmin := isSystemAdmin(tc)
	if !systemAdmin {
		if reqTenant != "" && !strings.EqualFold(reqTenant, tc.TenantUUID) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "跨租户访问被拒绝"})
			return
		}
		reqTenant = tc.TenantUUID
	}
	limit := clampLimit(c.DefaultQuery("limit", "50"))
	afterID := parseUintQuery(c.Query("after_id"))
	filter := srviam.AuditListFilter{
		TenantUUID: reqTenant,
		Resource:   strings.TrimSpace(c.Query("resource")),
		Action:     strings.TrimSpace(c.Query("action")),
		Limit:      limit,
		AfterID:    afterID,
	}
	records, err := h.svc.List(c.Request.Context(), filter)
	if err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	items := make([]gin.H, 0, len(records))
	for _, rec := range records {
		items = append(items, gin.H{
			"id":           rec.ID,
			"tenant_uuid":  rec.TenantUuid,
			"actor_member": rec.ActorUserID,
			"action":       rec.Action,
			"resource":     rec.Resource,
			"diff":         rec.Diff,
			"created_at":   rec.CreatedAt,
		})
	}
	contracts.ResponseSuccess(c, gin.H{"items": items})
}

func isSystemAdmin(tc authmw.TenantContext) bool {
	for _, role := range tc.Roles {
		switch strings.ToLower(strings.TrimSpace(role)) {
		case "superadmin", "system.admin":
			return true
		}
	}
	for _, perm := range tc.Permissions {
		candidate := strings.ToLower(strings.TrimSpace(perm))
		if candidate == "*" || candidate == "*:*" {
			return true
		}
	}
	return false
}

func clampLimit(raw string) int {
	if raw == "" {
		return 50
	}
	if v, err := strconv.Atoi(raw); err == nil {
		if v < 1 {
			return 1
		}
		if v > 200 {
			return 200
		}
		return v
	}
	return 50
}

func parseUintQuery(raw string) uint64 {
	if strings.TrimSpace(raw) == "" {
		return 0
	}
	v, err := strconv.ParseUint(strings.TrimSpace(raw), 10, 64)
	if err != nil {
		return 0
	}
	return v
}
