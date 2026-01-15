package runtime_ops

import (
	"net/http"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/config"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/logger"
	runtimeops "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/runtime_ops"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// QuotaHandler exposes quota management endpoints.
type QuotaHandler struct {
	svc  *runtimeops.QuotaService
	deps *app.Deps
}

// NewQuotaHandler constructs handler with service initialized from dependencies.
func NewQuotaHandler(deps *app.Deps, defaults *config.RuntimeOpsDefaults) *QuotaHandler {
	var (
		db       *gorm.DB
		runtimeD *config.RuntimeOpsDefaults
	)
	if deps != nil {
		db = deps.DB
		runtimeD = deps.RuntimeDefaults()
	}
	if defaults != nil {
		runtimeD = defaults
	}
	handler := &QuotaHandler{
		svc:  runtimeops.NewQuotaService(db, runtimeD),
		deps: deps,
	}
	return handler
}

// GetStatus returns current quota utilization for a tenant scope.
func (h *QuotaHandler) GetStatus(c *gin.Context) {
	pluginID := c.DefaultQuery("plugin_id", app.PluginID)
	tenantID := c.Query("tenant_uuid")

	defaults := h.svc.Defaults()
	window := 5 * time.Minute
	if defaults != nil && defaults.QuotaWindowMinutes > 0 {
		window = time.Duration(defaults.QuotaWindowMinutes) * time.Minute
	}

	start := time.Now().Add(-window)
	entries, err := h.svc.ListUsage(c.Request.Context(), "tenant", tenantID, start, time.Now())
	if err != nil {
		h.log(c, logrus.ErrorLevel, "failed to list quota ledger", logger.Fields{
			"tenant_uuid": tenantID,
			"plugin_id":   pluginID,
			"error":       err.Error(),
		})
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.log(c, logrus.InfoLevel, "quota ledger retrieved", logger.Fields{
		"tenant_uuid": tenantID,
		"plugin_id":   pluginID,
		"entries":     len(entries),
	})

	c.JSON(http.StatusOK, gin.H{
		"plugin_id":   pluginID,
		"tenant_uuid": tenantID,
		"ledger":      entries,
		"window": gin.H{
			"minutes": int(window.Minutes()),
			"start":   start.UTC(),
			"end":     time.Now().UTC(),
		},
	})
}

// SetOverride accepts manual quota overrides (placeholder).
func (h *QuotaHandler) SetOverride(c *gin.Context) {
	var req struct {
		PluginID   string `json:"plugin_id"`
		TenantUuid string `json:"tenant_uuid"`
		Capability string `json:"capability"`
		Action     string `json:"action"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		h.log(c, logrus.WarnLevel, "invalid quota override payload", logger.Fields{
			"error": err.Error(),
		})
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.svc.HandleBreach(c.Request.Context(), req.PluginID, req.TenantUuid, req.Capability, req.Action)
	h.log(c, logrus.InfoLevel, "manual quota override accepted", logger.Fields{
		"plugin_id":   req.PluginID,
		"tenant_uuid": req.TenantUuid,
		"action":      req.Action,
		"path":        c.FullPath(),
	})

	c.JSON(http.StatusAccepted, gin.H{"status": "override accepted"})
}

func (h *QuotaHandler) log(c *gin.Context, level logrus.Level, msg string, fields logger.Fields) {
	if h.deps == nil {
		return
	}
	if fields == nil {
		fields = logger.Fields{}
	}
	if c != nil {
		if reqID := c.GetString("request_id"); reqID != "" {
			fields["request_id"] = reqID
		}
	}
	entry := h.deps.RuntimeLogger(c.Request.Context(), "admin.runtime.quota", fields)
	switch level {
	case logrus.ErrorLevel:
		entry.Error(msg)
	case logrus.WarnLevel:
		entry.Warn(msg)
	default:
		entry.Info(msg)
	}
}
