package runtime_ops

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
	"github.com/ArtisanCloud/PowerXPlugin/framework/event"
)

type eventBridgeEmitRequest struct {
	TenantUUID string          `json:"tenant_uuid" binding:"required"`
	Topic      string          `json:"topic" binding:"required"`
	Payload    json.RawMessage `json:"payload"`
	RequestID  string          `json:"request_id"`
	TraceID    string          `json:"trace_id"`
}

func EventBridgeEmitHandler(deps *app.Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		if deps == nil || deps.EventEmitter == nil || deps.Config == nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "event bridge not configured"})
			return
		}

		var req eventBridgeEmitRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		topic := strings.TrimSpace(req.Topic)
		if topic == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "topic is required"})
			return
		}

		payload := req.Payload
		if len(payload) == 0 {
			payload = json.RawMessage(`{}`)
		}

		sourcePlugin := strings.TrimSpace(deps.Config.EventBridge.SourcePlugin)
		if sourcePlugin == "" {
			sourcePlugin = app.PluginID
		}
		payloadVersion := strings.TrimSpace(deps.Config.EventBridge.PayloadVersion)
		if payloadVersion == "" {
			payloadVersion = "v1"
		}

		metaBuilder := event.NewMetaBuilder(sourcePlugin, payloadVersion)
		meta, err := metaBuilder.Build(req.TenantUUID, req.RequestID, req.TraceID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		err = deps.EventEmitter.Emit(c.Request.Context(), event.Event{
			Topic:   event.Topic(topic),
			Meta:    meta,
			Payload: payload,
		})
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"ok":          false,
				"topic":       topic,
				"tenant_uuid": meta.TenantUUID,
				"trace_id":    meta.TraceID,
				"error":       err.Error(),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"ok":          true,
			"topic":       topic,
			"tenant_uuid": meta.TenantUUID,
			"trace_id":    meta.TraceID,
		})
	}
}
