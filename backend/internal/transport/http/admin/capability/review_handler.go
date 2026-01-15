package capability

import (
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/contracts"
	reviewsvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/capability_review"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
)

// ReviewHandler manages capability review endpoints.
type ReviewHandler struct {
	service *reviewsvc.WorkflowService
}

// NewReviewHandler creates a handler using shared dependencies.
func NewReviewHandler(deps *app.Deps) *ReviewHandler {
	if deps == nil {
		return nil
	}
	return &ReviewHandler{service: reviewsvc.SharedWorkflowService(deps)}
}

// List returns review tasks for a capability.
func (h *ReviewHandler) List(c *gin.Context) {
	if h == nil || h.service == nil {
		contracts.ResponseServiceUnavailable(c, "capability review service not available", nil)
		return
	}
	capabilityID := strings.TrimSpace(c.Param("capabilityID"))
	if capabilityID == "" {
		contracts.ResponseBadRequest(c, "capability id required")
		return
	}
	tasks := h.service.ListTasks(capabilityID)
	contracts.ResponseSuccess(c, gin.H{
		"capability_id": capabilityID,
		"tasks":         tasks,
	})
}

// Resubmit reopens review tasks after remediation.
func (h *ReviewHandler) Resubmit(c *gin.Context) {
	if h == nil || h.service == nil {
		contracts.ResponseServiceUnavailable(c, "capability review service not available", nil)
		return
	}
	capabilityID := strings.TrimSpace(c.Param("capabilityID"))
	if capabilityID == "" {
		contracts.ResponseBadRequest(c, "capability id required")
		return
	}
	var req struct {
		Actor       string                      `json:"actor"`
		Note        string                      `json:"note"`
		Attachments []reviewsvc.AttachmentInput `json:"attachments"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid payload: "+err.Error())
		return
	}
	tasks, err := h.service.Resubmit(c.Request.Context(), capabilityID, reviewsvc.ResubmitInput{
		Actor:       req.Actor,
		Note:        req.Note,
		Attachments: req.Attachments,
	})
	if err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, gin.H{
		"capability_id": capabilityID,
		"tasks":         tasks,
	})
}

// AddComment adds a reviewer comment.
func (h *ReviewHandler) AddComment(c *gin.Context) {
	if h == nil || h.service == nil {
		contracts.ResponseServiceUnavailable(c, "capability review service not available", nil)
		return
	}
	taskID := strings.TrimSpace(c.Param("taskID"))
	if taskID == "" {
		contracts.ResponseBadRequest(c, "task id required")
		return
	}
	var req struct {
		Author      string                      `json:"author"`
		Message     string                      `json:"message"`
		Attachments []reviewsvc.AttachmentInput `json:"attachments"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid payload: "+err.Error())
		return
	}
	task, err := h.service.AddComment(c.Request.Context(), taskID, reviewsvc.CommentInput{
		Author:      req.Author,
		Attachments: req.Attachments,
		Message:     req.Message,
	})
	if err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, task)
}

// Decide records a reviewer decision for a specific task.
func (h *ReviewHandler) Decide(c *gin.Context) {
	if h == nil || h.service == nil {
		contracts.ResponseServiceUnavailable(c, "capability review service not available", nil)
		return
	}
	taskID := strings.TrimSpace(c.Param("taskID"))
	if taskID == "" {
		contracts.ResponseBadRequest(c, "task id required")
		return
	}
	var req struct {
		Actor       string                      `json:"actor"`
		Decision    string                      `json:"decision"`
		Note        string                      `json:"note"`
		Attachments []reviewsvc.AttachmentInput `json:"attachments"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid payload: "+err.Error())
		return
	}
	task, err := h.service.Resolve(c.Request.Context(), taskID, reviewsvc.DecisionInput{
		Actor:       req.Actor,
		Decision:    req.Decision,
		Note:        req.Note,
		Attachments: req.Attachments,
	})
	if err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, task)
}
