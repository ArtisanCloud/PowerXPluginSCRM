package console

import (
	"encoding/base64"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/contracts"
	consolesvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/console"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
)

// AuditHandler exposes audit history endpoints.
type AuditHandler struct {
	svc *consolesvc.AuditService
}

// NewAuditHandler constructs handler with shared deps.
func NewAuditHandler(deps *app.Deps) *AuditHandler {
	if deps == nil || deps.DB == nil {
		return &AuditHandler{}
	}
	return &AuditHandler{svc: consolesvc.NewAuditService(deps)}
}

type listAuditQuery struct {
	TenantUuid     string `form:"tenant_uuid"`
	ActorID        string `form:"actor_id"`
	Action         string `form:"action"`
	PermissionCode string `form:"permission_code"`
	OccurredAfter  string `form:"occurred_after"`
	OccurredBefore string `form:"occurred_before"`
	Cursor         string `form:"cursor"`
	Limit          string `form:"limit"`
}

// ListEvents returns audit entries according to filters.
func (h *AuditHandler) ListEvents(c *gin.Context) {
	if h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "audit service unavailable", nil)
		return
	}
	var query listAuditQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		contracts.ResponseBadRequest(c, "invalid query parameters: "+err.Error())
		return
	}
	input := consolesvc.ListAuditInput{
		ActorID:        strings.TrimSpace(query.ActorID),
		Action:         strings.TrimSpace(query.Action),
		PermissionCode: strings.TrimSpace(query.PermissionCode),
		Cursor:         strings.TrimSpace(query.Cursor),
	}
	if strings.TrimSpace(query.TenantUuid) != "" {
		clean := strings.TrimSpace(query.TenantUuid)
		input.TenantUuid = &clean
	}
	if strings.TrimSpace(query.Limit) != "" {
		if n, err := strconv.Atoi(query.Limit); err == nil {
			input.Limit = n
		}
	}
	if query.OccurredAfter != "" {
		if ts, err := time.Parse(time.RFC3339, query.OccurredAfter); err == nil {
			input.OccurredAfter = &ts
		}
	}
	if query.OccurredBefore != "" {
		if ts, err := time.Parse(time.RFC3339, query.OccurredBefore); err == nil {
			input.OccurredBefore = &ts
		}
	}
	result, err := h.svc.ListEvents(c.Request.Context(), input)
	if err != nil {
		if errors.Is(err, consolesvc.ErrServiceUnavailable) {
			contracts.ResponseServiceUnavailable(c, err.Error(), nil)
			return
		}
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, result)
}

type exportAuditQuery struct {
	TenantUuid     string `form:"tenant_uuid"`
	ActorID        string `form:"actor_id"`
	Action         string `form:"action"`
	PermissionCode string `form:"permission_code"`
	OccurredAfter  string `form:"occurred_after"`
	OccurredBefore string `form:"occurred_before"`
	Format         string `form:"format"`
}

// ExportEvents streams audit events in CSV or JSON.
func (h *AuditHandler) ExportEvents(c *gin.Context) {
	if h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "audit service unavailable", nil)
		return
	}
	var query exportAuditQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		contracts.ResponseBadRequest(c, "invalid query parameters: "+err.Error())
		return
	}
	input := consolesvc.ExportAuditInput{
		ActorID:        strings.TrimSpace(query.ActorID),
		Action:         strings.TrimSpace(query.Action),
		PermissionCode: strings.TrimSpace(query.PermissionCode),
		Format:         strings.TrimSpace(query.Format),
	}
	if strings.TrimSpace(query.TenantUuid) != "" {
		clean := strings.TrimSpace(query.TenantUuid)
		input.TenantUuid = &clean
	}
	if query.OccurredAfter != "" {
		if ts, err := time.Parse(time.RFC3339, query.OccurredAfter); err == nil {
			input.OccurredAfter = &ts
		}
	}
	if query.OccurredBefore != "" {
		if ts, err := time.Parse(time.RFC3339, query.OccurredBefore); err == nil {
			input.OccurredBefore = &ts
		}
	}
	result, err := h.svc.ExportEvents(c.Request.Context(), input)
	if err != nil {
		if field, msg, ok := consolesvc.IsValidationError(err); ok {
			details := gin.H{"field": field}
			contracts.ResponseErrorWithDetails(c, http.StatusBadRequest, contracts.ErrCodeInvalidRequest, msg, details)
			return
		}
		if errors.Is(err, consolesvc.ErrServiceUnavailable) {
			contracts.ResponseServiceUnavailable(c, err.Error(), nil)
			return
		}
		contracts.ResponseInternalError(c, err)
		return
	}
	encoded := base64.StdEncoding.EncodeToString(result.Content)
	c.Header("Content-Type", "application/json")
	contracts.ResponseSuccess(c, gin.H{
		"filename":       result.Filename,
		"content_type":   result.ContentType,
		"content_base64": encoded,
	})
}
