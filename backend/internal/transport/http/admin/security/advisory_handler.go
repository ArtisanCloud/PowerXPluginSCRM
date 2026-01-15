package security

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	adminsec "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/security"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// AdvisoryHandler exposes admin endpoints for vulnerability advisories.
type AdvisoryHandler struct {
	service *adminsec.AdvisoryService
}

func NewAdvisoryHandler(deps *app.Deps) *AdvisoryHandler {
	logger := deps.RuntimeLogger(deps.Ctx, "admin_security_advisories", nil)
	service := adminsec.NewAdvisoryService(deps.DB, logger)
	return &AdvisoryHandler{service: service}
}

type createAdvisoryRequest struct {
	Reference        string   `json:"reference" binding:"required"`
	Severity         string   `json:"severity" binding:"required"`
	Summary          string   `json:"summary" binding:"required"`
	DetailsMarkdown  string   `json:"details_markdown"`
	AffectedVersions []string `json:"affected_versions"`
	SlaDeadline      string   `json:"sla_deadline"`
}

type publishAdvisoryRequest struct {
	PatchedInVersion string                 `json:"patched_in_version" binding:"required"`
	NotifyChannels   []string               `json:"notify_channels"`
	Metadata         map[string]interface{} `json:"metadata"`
}

func (h *AdvisoryHandler) Create(c *gin.Context) {
	var req createAdvisoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid advisory payload"})
		return
	}
	var deadline *time.Time
	if req.SlaDeadline != "" {
		parsed, err := time.Parse(time.RFC3339, req.SlaDeadline)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "sla_deadline must be RFC3339"})
			return
		}
		deadline = &parsed
	}
	advisory, err := h.service.CreateAdvisory(c.Request.Context(), adminsec.CreateAdvisoryParams{
		Reference:        req.Reference,
		Severity:         req.Severity,
		Summary:          req.Summary,
		DetailsMarkdown:  req.DetailsMarkdown,
		AffectedVersions: req.AffectedVersions,
		SlaDeadline:      deadline,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, NewAdvisoryResponse(advisory))
}

func (h *AdvisoryHandler) List(c *gin.Context) {
	severity := c.QueryArray("severity")
	if len(severity) == 0 {
		if v := c.Query("severity"); v != "" {
			severity = []string{v}
		}
	}
	statuses := c.QueryArray("status")
	if len(statuses) == 0 {
		if v := c.Query("status"); v != "" {
			statuses = []string{v}
		}
	}
	limit := 0
	if raw := c.Query("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			limit = parsed
		}
	}
	advisories, err := h.service.ListAdvisories(c.Request.Context(), severity, statuses, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, NewAdvisoryListResponse(advisories))
}

func (h *AdvisoryHandler) Publish(c *gin.Context) {
	advisoryID := c.Param("advisoryId")
	if advisoryID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "advisoryId is required"})
		return
	}
	var req publishAdvisoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid publish payload"})
		return
	}
	var metadata datatypes.JSONMap
	if req.Metadata != nil {
		metadata = datatypes.JSONMap(req.Metadata)
	}
	advisory, _, err := h.service.PublishAdvisory(c.Request.Context(), adminsec.PublishAdvisoryParams{
		AdvisoryID:       advisoryID,
		PatchedInVersion: req.PatchedInVersion,
		NotifyChannels:   req.NotifyChannels,
		Metadata:         metadata,
	})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "advisory not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, NewAdvisoryResponse(advisory))
}
