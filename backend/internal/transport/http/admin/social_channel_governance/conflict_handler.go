package social_channel_governance

import (
	"errors"
	"net/http"
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/contracts"
	socialrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/social_channel_governance"
	socialsvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/social_channel_governance"
	httpmw "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/middleware"
	"github.com/gin-gonic/gin"
)

type ConflictHandler struct {
	conflictSvc *socialsvc.ConflictResolutionService
	deadSvc     *socialsvc.RetryDeadletterService
}

func NewConflictHandler(conflictSvc *socialsvc.ConflictResolutionService, deadSvc *socialsvc.RetryDeadletterService) *ConflictHandler {
	return &ConflictHandler{conflictSvc: conflictSvc, deadSvc: deadSvc}
}

func (h *ConflictHandler) List(c *gin.Context) {
	tenantUUID, ok := httpmw.TenantUUIDFromContext(c)
	if !ok || strings.TrimSpace(tenantUUID) == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	if h == nil || h.conflictSvc == nil {
		contracts.ResponseServiceUnavailable(c, "conflict service unavailable", nil)
		return
	}
	domain := strings.TrimSpace(c.Query("domain"))
	status := strings.TrimSpace(c.Query("status"))
	limit := parseLimit(c.Query("limit"))
	items, err := h.conflictSvc.List(c.Request.Context(), tenantUUID, domain, status, limit)
	if err != nil {
		contracts.ResponseError(c, http.StatusBadRequest, contracts.ErrCodeInvalidRequest, err.Error())
		return
	}
	contracts.ResponseSuccess(c, gin.H{"items": items})
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
	if h == nil || h.conflictSvc == nil {
		contracts.ResponseServiceUnavailable(c, "conflict service unavailable", nil)
		return
	}
	var req struct {
		ResolvedBy string `json:"resolved_by"`
	}
	_ = c.ShouldBindJSON(&req)
	item, err := h.conflictSvc.Replay(c.Request.Context(), tenantUUID, conflictUUID, req.ResolvedBy)
	if err != nil {
		if errors.Is(err, socialrepo.ErrSyncConflictNotFound) {
			contracts.ResponseNotFound(c, err.Error())
			return
		}
		contracts.ResponseError(c, http.StatusBadRequest, contracts.ErrCodeInvalidRequest, err.Error())
		return
	}
	contracts.ResponseSuccess(c, item)
}

func (h *ConflictHandler) ListDeadLetters(c *gin.Context) {
	tenantUUID, ok := httpmw.TenantUUIDFromContext(c)
	if !ok || strings.TrimSpace(tenantUUID) == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}
	if h == nil || h.deadSvc == nil {
		contracts.ResponseServiceUnavailable(c, "dead letter service unavailable", nil)
		return
	}
	items, err := h.deadSvc.List(
		c.Request.Context(),
		tenantUUID,
		strings.TrimSpace(c.Query("domain")),
		strings.TrimSpace(c.Query("replay_status")),
		parseLimit(c.Query("limit")),
	)
	if err != nil {
		contracts.ResponseError(c, http.StatusBadRequest, contracts.ErrCodeInvalidRequest, err.Error())
		return
	}
	contracts.ResponseSuccess(c, gin.H{"items": items})
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
	if h == nil || h.deadSvc == nil {
		contracts.ResponseServiceUnavailable(c, "dead letter service unavailable", nil)
		return
	}
	var req struct {
		ReplayedBy string `json:"replayed_by"`
	}
	_ = c.ShouldBindJSON(&req)
	item, err := h.deadSvc.Replay(c.Request.Context(), tenantUUID, deadLetterUUID, req.ReplayedBy)
	if err != nil {
		if errors.Is(err, socialrepo.ErrSyncJobNotFound) {
			contracts.ResponseNotFound(c, "dead letter not found")
			return
		}
		contracts.ResponseError(c, http.StatusBadRequest, contracts.ErrCodeInvalidRequest, err.Error())
		return
	}
	contracts.ResponseSuccess(c, item)
}

func (h *ConflictHandler) NotSupported(c *gin.Context) {
	contracts.ResponseError(c, http.StatusNotImplemented, "NOT_SUPPORTED", "not supported in current channel capability")
}

func parseLimit(raw string) int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 50
	}
	n := 0
	for _, ch := range raw {
		if ch < '0' || ch > '9' {
			return 50
		}
		n = n*10 + int(ch-'0')
	}
	if n <= 0 || n > 200 {
		return 50
	}
	return n
}
