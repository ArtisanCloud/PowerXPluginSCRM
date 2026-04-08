package social_channel_governance

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/contracts"
	model "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/social_channel_governance"
	socialrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/social_channel_governance"
	socialobs "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/observability/social_channel_governance"
	httpmw "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/middleware"
	"github.com/gin-gonic/gin"
)

type MetricsHandler struct {
	repo *socialrepo.SyncFoundationRepository
}

func NewMetricsHandler(repo *socialrepo.SyncFoundationRepository) *MetricsHandler {
	return &MetricsHandler{repo: repo}
}

func (h *MetricsHandler) GetSyncMetrics(c *gin.Context) {
	if h == nil || h.repo == nil {
		contracts.ResponseServiceUnavailable(c, "metrics service unavailable", nil)
		return
	}
	tenantUUID, ok := httpmw.TenantUUIDFromContext(c)
	if !ok || strings.TrimSpace(tenantUUID) == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}

	windowRaw := strings.TrimSpace(c.DefaultQuery("window", "7d"))
	topRaw := strings.TrimSpace(c.DefaultQuery("top", "5"))
	topN, _ := strconv.Atoi(topRaw)
	if topN <= 0 {
		topN = 5
	}
	if topN > 20 {
		topN = 20
	}
	windowDur, err := parseMetricsWindow(windowRaw)
	if err != nil {
		contracts.ResponseError(c, http.StatusBadRequest, contracts.ErrCodeInvalidRequest, err.Error())
		return
	}
	cutoff := time.Now().UTC().Add(-windowDur)

	jobs, err := h.repo.ListJobs(c.Request.Context(), tenantUUID, "", "", 200)
	if err != nil {
		contracts.ResponseError(c, http.StatusBadRequest, contracts.ErrCodeInvalidRequest, err.Error())
		return
	}
	conflicts, err := h.repo.ListConflicts(c.Request.Context(), tenantUUID, "", "", 200)
	if err != nil {
		contracts.ResponseError(c, http.StatusBadRequest, contracts.ErrCodeInvalidRequest, err.Error())
		return
	}
	deadLetters, err := h.repo.ListDeadLetters(c.Request.Context(), tenantUUID, "", "", 200)
	if err != nil {
		contracts.ResponseError(c, http.StatusBadRequest, contracts.ErrCodeInvalidRequest, err.Error())
		return
	}

	filteredJobs := make([]model.SyncJob, 0, len(jobs))
	for _, item := range jobs {
		if item.CreatedAt.Before(cutoff) {
			continue
		}
		filteredJobs = append(filteredJobs, item)
	}
	filteredConflicts := make([]model.SyncConflict, 0, len(conflicts))
	for _, item := range conflicts {
		if item.CreatedAt.Before(cutoff) {
			continue
		}
		filteredConflicts = append(filteredConflicts, item)
	}
	filteredDeadLetters := make([]model.SyncDeadLetterItem, 0, len(deadLetters))
	for _, item := range deadLetters {
		if item.CreatedAt.Before(cutoff) {
			continue
		}
		filteredDeadLetters = append(filteredDeadLetters, item)
	}

	snapshot := socialobs.BuildSyncMetricsSnapshot(
		windowRaw,
		filteredJobs,
		filteredConflicts,
		filteredDeadLetters,
		topN,
	)
	contracts.ResponseSuccess(c, snapshot)
}

func parseMetricsWindow(value string) (time.Duration, error) {
	v := strings.TrimSpace(strings.ToLower(value))
	switch v {
	case "", "24h", "1d":
		return 24 * time.Hour, nil
	case "7d":
		return 7 * 24 * time.Hour, nil
	case "30d":
		return 30 * 24 * time.Hour, nil
	default:
		return 0, errors.New("window only supports 24h/7d/30d")
	}
}
