package social_channel_governance

import (
	"net/http"
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/contracts"
	httpmw "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/middleware"
	"github.com/gin-gonic/gin"
)

// ConflictHandler currently provides foundational endpoints as route placeholders.
// Domain-specific replay logic is handled by dedicated services in later phases.
type ConflictHandler struct{}

func NewConflictHandler() *ConflictHandler {
	return &ConflictHandler{}
}

func (h *ConflictHandler) List(c *gin.Context) {
	tenantUUID, ok := httpmw.TenantUUIDFromContext(c)
	if !ok || strings.TrimSpace(tenantUUID) == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	contracts.ResponseSuccess(c, gin.H{
		"items": []any{},
		"meta":  gin.H{"tenant_uuid": tenantUUID, "status": strings.TrimSpace(c.Query("status"))},
	})
}

func (h *ConflictHandler) Replay(c *gin.Context) {
	tenantUUID, ok := httpmw.TenantUUIDFromContext(c)
	if !ok || strings.TrimSpace(tenantUUID) == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	conflictUUID := strings.TrimSpace(c.Param("conflict_uuid"))
	if conflictUUID == "" {
		contracts.ResponseBadRequest(c, "conflict_uuid is required")
		return
	}
	contracts.ResponseSuccess(c, gin.H{
		"tenant_uuid":   tenantUUID,
		"conflict_uuid": conflictUUID,
		"status":        "accepted",
	})
}

func (h *ConflictHandler) ListDeadLetters(c *gin.Context) {
	tenantUUID, ok := httpmw.TenantUUIDFromContext(c)
	if !ok || strings.TrimSpace(tenantUUID) == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	contracts.ResponseSuccess(c, gin.H{
		"items": []any{},
		"meta":  gin.H{"tenant_uuid": tenantUUID},
	})
}

func (h *ConflictHandler) ReplayDeadLetter(c *gin.Context) {
	tenantUUID, ok := httpmw.TenantUUIDFromContext(c)
	if !ok || strings.TrimSpace(tenantUUID) == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	deadLetterUUID := strings.TrimSpace(c.Param("dead_letter_uuid"))
	if deadLetterUUID == "" {
		contracts.ResponseBadRequest(c, "dead_letter_uuid is required")
		return
	}
	contracts.ResponseSuccess(c, gin.H{
		"tenant_uuid":      tenantUUID,
		"dead_letter_uuid": deadLetterUUID,
		"status":           "accepted",
	})
}

func (h *ConflictHandler) NotSupported(c *gin.Context) {
	contracts.ResponseError(c, http.StatusNotImplemented, "NOT_SUPPORTED", "not supported in current channel capability")
}
