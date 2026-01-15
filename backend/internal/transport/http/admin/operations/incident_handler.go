package operations

import (
	"fmt"
	"strings"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/contracts"
	operationsvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/operations"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
)

// IncidentHandler exposes incident lifecycle endpoints.
type IncidentHandler struct {
	svc *operationsvc.IncidentService
}

// NewIncidentHandler constructs a handler with dependencies.
func NewIncidentHandler(svc *operationsvc.IncidentService) *IncidentHandler {
	return &IncidentHandler{svc: svc}
}

type incidentCreateRequest struct {
	TenantUuid      *string         `json:"tenant_uuid"`
	Severity        string          `json:"severity" binding:"required"`
	DetectionSource string          `json:"detection_source" binding:"required"`
	Summary         string          `json:"summary" binding:"required"`
	Impact          map[string]any  `json:"impact"`
	Mitigation      string          `json:"mitigation"`
	Labels          map[string]bool `json:"labels"`
	Confidentiality string          `json:"confidentiality"`
	NextUpdateAt    *time.Time      `json:"next_update_at"`
}

type incidentListQuery struct {
	Severity []string   `form:"severity"`
	Status   []string   `form:"status"`
	Label    []string   `form:"label"`
	From     *time.Time `form:"from" time_format:"2006-01-02T15:04:05Z07:00"`
	To       *time.Time `form:"to" time_format:"2006-01-02T15:04:05Z07:00"`
}

type incidentUpdateRequest struct {
	Status          *string         `json:"status"`
	Mitigation      *string         `json:"mitigation"`
	RootCause       *string         `json:"root_cause"`
	NextUpdateAt    *time.Time      `json:"next_update_at"`
	Labels          map[string]bool `json:"labels"`
	Confidentiality *string         `json:"confidentiality"`
}

type timelineRequest struct {
	EntryType          string         `json:"entry_type" binding:"required"`
	Message            string         `json:"message" binding:"required"`
	StakeholderChannel string         `json:"stakeholder_channel"`
	AuthorRole         string         `json:"author_role"`
	Metadata           map[string]any `json:"metadata"`
}

// CreateIncident handles POST /incidents.
func (h *IncidentHandler) CreateIncident(c *gin.Context) {
	if h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "incident service unavailable", nil)
		return
	}
	var req incidentCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid body: "+err.Error())
		return
	}
	payload, err := h.svc.CreateIncident(c.Request.Context(), operationsvc.CreateIncidentRequest{
		TenantUuid:      req.TenantUuid,
		Severity:        req.Severity,
		DetectionSource: req.DetectionSource,
		Summary:         req.Summary,
		Impact:          req.Impact,
		Mitigation:      req.Mitigation,
		Labels:          req.Labels,
		Confidentiality: req.Confidentiality,
		NextUpdateAt:    req.NextUpdateAt,
	})
	if err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	view, err := h.svc.GetIncident(c.Request.Context(), payload.ID)
	if err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	basePath := strings.TrimSuffix(c.Request.URL.Path, "/")
	c.Header("Location", fmt.Sprintf("%s/%s", basePath, payload.ID))
	contracts.ResponseCreated(c, view)
}

// ListIncidents handles GET /incidents.
func (h *IncidentHandler) ListIncidents(c *gin.Context) {
	if h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "incident service unavailable", nil)
		return
	}
	var query incidentListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		contracts.ResponseBadRequest(c, "invalid query: "+err.Error())
		return
	}
	filter := operationsvc.IncidentFilter{
		Severities: normalizeStrings(query.Severity),
		Statuses:   normalizeStrings(query.Status),
		Labels:     normalizeStrings(query.Label),
		From:       query.From,
		To:         query.To,
	}
	records, err := h.svc.ListIncidents(c.Request.Context(), app.PluginID, filter)
	if err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, records)
}

// GetIncident handles GET /incidents/:id.
func (h *IncidentHandler) GetIncident(c *gin.Context) {
	if h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "incident service unavailable", nil)
		return
	}
	response, err := h.svc.GetIncident(c.Request.Context(), c.Param("incidentId"))
	if err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, response)
}

// UpdateIncident handles PATCH /incidents/:id.
func (h *IncidentHandler) UpdateIncident(c *gin.Context) {
	if h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "incident service unavailable", nil)
		return
	}
	var req incidentUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid body: "+err.Error())
		return
	}
	updated, err := h.svc.UpdateIncident(c.Request.Context(), c.Param("incidentId"), operationsvc.UpdateIncidentRequest(req))
	if err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	view, err := h.svc.GetIncident(c.Request.Context(), updated.ID)
	if err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, view)
}

// AppendTimeline handles POST /incidents/:id/timeline.
func (h *IncidentHandler) AppendTimeline(c *gin.Context) {
	if h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "incident service unavailable", nil)
		return
	}
	var req timelineRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid body: "+err.Error())
		return
	}
	entry, err := h.svc.AppendTimeline(c.Request.Context(), c.Param("incidentId"), operationsvc.TimelineEntryRequest(req))
	if err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccessWithMessage(c, entry, "timeline entry recorded")
}

func normalizeStrings(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, v := range values {
		n := strings.TrimSpace(strings.ToLower(v))
		if n == "" {
			continue
		}
		if _, ok := seen[n]; ok {
			continue
		}
		seen[n] = struct{}{}
		out = append(out, n)
	}
	return out
}
