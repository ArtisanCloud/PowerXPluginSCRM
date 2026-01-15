package capability

import (
	"net/http"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/contracts"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/capability"
	reviewsvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/capability_review"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
)

// RegisterHandler exposes capability registration endpoints.
type RegisterHandler struct {
	service *capability.RegisterService
	review  *reviewsvc.WorkflowService
}

// NewRegisterHandler builds a handler instance.
func NewRegisterHandler(deps *app.Deps) *RegisterHandler {
	if deps == nil {
		return nil
	}
	return &RegisterHandler{
		service: capability.NewRegisterService(deps),
		review:  reviewsvc.SharedWorkflowService(deps),
	}
}

// GetTemplate returns the form template metadata.
func (h *RegisterHandler) GetTemplate(c *gin.Context) {
	if h == nil || h.service == nil {
		contracts.ResponseServiceUnavailable(c, "capability register service not available", nil)
		return
	}
	tpl := h.service.Template(c.Request.Context())
	contracts.ResponseSuccess(c, tpl)
}

// ValidateDraft validates a submission without persisting.
func (h *RegisterHandler) ValidateDraft(c *gin.Context) {
	if h == nil || h.service == nil {
		contracts.ResponseServiceUnavailable(c, "capability register service not available", nil)
		return
	}
	var payload capability.RegisterInput
	if err := c.ShouldBindJSON(&payload); err != nil {
		contracts.ResponseBadRequest(c, "invalid payload: "+err.Error())
		return
	}
	result, err := h.service.Validate(c.Request.Context(), &payload)
	if err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	if result.Valid {
		contracts.ResponseSuccess(c, result)
		return
	}
	contracts.ResponseErrorWithDetails(c, http.StatusUnprocessableEntity, contracts.ErrCodeValidationFailed, "capability validation failed", result)
}

// Submit persists a capability draft or submission.
func (h *RegisterHandler) Submit(c *gin.Context) {
	if h == nil || h.service == nil {
		contracts.ResponseServiceUnavailable(c, "capability register service not available", nil)
		return
	}
	var payload capability.RegisterInput
	if err := c.ShouldBindJSON(&payload); err != nil {
		contracts.ResponseBadRequest(c, "invalid payload: "+err.Error())
		return
	}
	record, validation, err := h.service.Submit(c.Request.Context(), &payload)
	if err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	if validation != nil && !validation.Valid {
		contracts.ResponseErrorWithDetails(c, http.StatusUnprocessableEntity, contracts.ErrCodeValidationFailed, "capability validation failed", validation)
		return
	}
	if h.review != nil && record != nil && !record.Draft {
		if _, err := h.review.EnsureTasks(c.Request.Context(), record); err != nil {
			contracts.ResponseInternalError(c, err)
			return
		}
	}
	contracts.ResponseSuccess(c, record)
}
