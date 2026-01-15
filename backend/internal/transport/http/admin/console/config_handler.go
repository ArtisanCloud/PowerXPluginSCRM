package console

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/contracts"
	authx "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/middleware"
	consolesvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/console"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
)

// ConfigHandler exposes configuration endpoints for the Dev Console.
type ConfigHandler struct {
	svc *consolesvc.ConfigService
}

// NewConfigHandler wires a ConfigHandler if dependencies are available.
func NewConfigHandler(deps *app.Deps) *ConfigHandler {
	if deps == nil || deps.DB == nil {
		return &ConfigHandler{}
	}
	return &ConfigHandler{svc: consolesvc.NewConfigService(deps)}
}

type configSectionsQuery struct {
	TenantUuid string `form:"tenant_uuid"`
}

// ListSections returns configuration schema merged with current values.
func (h *ConfigHandler) ListSections(c *gin.Context) {
	if h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "config service unavailable", nil)
		return
	}
	var query configSectionsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		contracts.ResponseBadRequest(c, "invalid query parameters: "+err.Error())
		return
	}
	var tenantPtr *string
	if strings.TrimSpace(query.TenantUuid) != "" {
		clean := strings.TrimSpace(query.TenantUuid)
		tenantPtr = &clean
	}
	sections, err := h.svc.ListSections(c.Request.Context(), tenantPtr)
	if err != nil {
		if errors.Is(err, consolesvc.ErrServiceUnavailable) {
			contracts.ResponseServiceUnavailable(c, err.Error(), nil)
			return
		}
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, gin.H{"sections": sections})
}

type updateSectionBody struct {
	TenantUuid *string        `json:"tenant_uuid"`
	Values     map[string]any `json:"values" binding:"required"`
	Comment    string         `json:"comment"`
}

// UpdateSection persists changes to a configuration section.
func (h *ConfigHandler) UpdateSection(c *gin.Context) {
	if h.svc == nil {
		contracts.ResponseServiceUnavailable(c, "config service unavailable", nil)
		return
	}
	sectionKey := strings.TrimSpace(c.Param("sectionKey"))
	if sectionKey == "" {
		contracts.ResponseBadRequest(c, "section key is required")
		return
	}
	var body updateSectionBody
	if err := c.ShouldBindJSON(&body); err != nil {
		contracts.ResponseBadRequest(c, "invalid request body: "+err.Error())
		return
	}
	actor := resolveActor(c)
	if actor.PermissionCode == "" {
		actor.PermissionCode = "operations.plugin.admin"
	}
	input := consolesvc.UpdateSectionInput{
		TenantUuid: body.TenantUuid,
		SectionKey: sectionKey,
		Values:     body.Values,
		Comment:    body.Comment,
		Actor:      actor,
	}
	updated, err := h.svc.UpdateSection(c.Request.Context(), input)
	if err != nil {
		if field, msg, ok := consolesvc.IsValidationError(err); ok {
			details := gin.H{"field": field}
			contracts.ResponseErrorWithDetails(c, http.StatusBadRequest, contracts.ErrCodeInvalidRequest, msg, details)
			return
		}
		if errors.Is(err, consolesvc.ErrUnknownSection) {
			contracts.ResponseNotFound(c, err.Error())
			return
		}
		if errors.Is(err, consolesvc.ErrServiceUnavailable) {
			contracts.ResponseServiceUnavailable(c, err.Error(), nil)
			return
		}
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, updated)
}

func resolveActor(c *gin.Context) consolesvc.Actor {
	tc, _ := authx.GetTenantContext(c)
	actorID := "system"
	if tc.UserID > 0 {
		actorID = fmt.Sprintf("user:%d", tc.UserID)
	}
	name := strings.TrimSpace(c.GetHeader("X-User-Name"))
	email := strings.TrimSpace(c.GetHeader("X-User-Email"))
	return consolesvc.Actor{
		ID:             actorID,
		Name:           name,
		Email:          email,
		PermissionCode: "operations.plugin.admin",
	}
}
