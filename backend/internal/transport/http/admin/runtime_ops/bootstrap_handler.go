package runtime_ops

import (
	"net/http"

	runtimeops "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/runtime_ops"
	"github.com/gin-gonic/gin"
)

// BootstrapHandler exposes runtime bootstrap endpoint.
type BootstrapHandler struct {
	svc *runtimeops.Service
}

// NewBootstrapHandler builds a runtime ops bootstrap handler.
func NewBootstrapHandler(svc *runtimeops.Service) *BootstrapHandler {
	if svc == nil {
		svc = runtimeops.NewService()
	}
	return &BootstrapHandler{svc: svc}
}

// Bootstrap is a placeholder handler for launching plugin runtime instances.
func (h *BootstrapHandler) Bootstrap(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, gin.H{"error": "runtime bootstrap endpoint not implemented"})
}
