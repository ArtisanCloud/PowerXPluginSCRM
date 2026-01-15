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
