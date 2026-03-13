package capability

import (
	"net/http"
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/contracts"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/logger"
	capservice "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/capability"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
)

// CatalogHandler exposes read-only capability catalog endpoints.
type CatalogHandler struct {
	service *capservice.CatalogService
}

type sourceOption struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
}

// NewCatalogHandler wires catalog handler when the service is available.
func NewCatalogHandler(deps *app.Deps) *CatalogHandler {
	svc := capservice.NewCatalogService(deps)
	if svc == nil {
		return nil
	}
	return &CatalogHandler{service: svc}
}

// List returns all capabilities collected in the catalog snapshot.
func (h *CatalogHandler) List(c *gin.Context) {
	source := strings.TrimSpace(c.Query("source"))
	entries, err := h.service.List(c.Request.Context(), capservice.ListOptions{Source: source})
	if err != nil {
		logger.WithError(err).
			WithField("component", "capability_catalog_handler").
			Error("failed to load capability catalog")
		contracts.ResponseErrorWithDetails(
			c,
			http.StatusInternalServerError,
			contracts.ErrCodeInternalError,
			"failed to load capability catalog",
			gin.H{"error": err.Error()},
		)
		return
	}
	logger.WithFields(logger.Fields{
		"component":    "capability_catalog_handler",
		"entry_count":  len(entries),
		"request_path": c.FullPath(),
	}).Info("capability catalog request handled")
	contracts.ResponseSuccess(c, entries)
}

// Sources returns capability source enums and alias mapping.
func (h *CatalogHandler) Sources(c *gin.Context) {
	contracts.ResponseSuccess(c, gin.H{
		"default": "all",
		"aliases": gin.H{
			"all":      "all",
			"any":      "all",
			"platform": "corex",
		},
		"sources": []sourceOption{
			{ID: "all", Label: "all", Description: "查询全部来源（不传 source 或 source=all）"},
			{ID: "corex", Label: "corex", Description: "PowerX 底座能力"},
			{ID: "plugin", Label: "plugin", Description: "插件/租户注册能力"},
		},
	})
}
