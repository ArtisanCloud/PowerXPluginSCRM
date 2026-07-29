package runtime_ops

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/contracts"
	runtimeswitch "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/runtime/switches"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type createEventTopicRequest struct {
	TenantUUID        string `json:"tenant_uuid,omitempty"`
	Namespace         string `json:"namespace"`
	Name              string `json:"name"`
	PayloadFormat     string `json:"payload_format,omitempty"`
	VersioningMode    string `json:"versioning_mode,omitempty"`
	MaxRetry          int    `json:"max_retry,omitempty"`
	AckTimeoutSeconds int    `json:"ack_timeout_seconds,omitempty"`
}

type eventFabricUpstreamEnvelope struct {
	Success bool `json:"success"`
	Code    int  `json:"code"`
	Error   *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
	Message string `json:"message"`
}

func EventFabricCreateTopicHandler(deps *app.Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		if deps == nil || deps.Config == nil || deps.Config.Gateway == nil {
			contracts.ResponseServiceUnavailable(c, "gateway is not configured", nil)
			return
		}
		if !runtimeswitch.IsHostDriver(runtimeswitch.Resolve(deps.Config).EventTopic) {
			contracts.ResponseError(c, http.StatusBadRequest, "TOPIC_DRIVER_DISABLED", "event topic host proxy is disabled by runtime.drivers.event_topic")
			return
		}

		var req createEventTopicRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			contracts.ResponseBadRequest(c, "invalid payload")
			return
		}
		if strings.TrimSpace(req.Namespace) == "" || strings.TrimSpace(req.Name) == "" {
			contracts.ResponseError(c, http.StatusBadRequest, contracts.ErrCodeInvalidRequest, "namespace and name are required")
			return
		}

		tenantUUID, tenantMismatch := resolveGatewayTenantUUID(c, deps, req.TenantUUID)
		if tenantMismatch {
			contracts.ResponseError(c, http.StatusForbidden, contracts.ErrCodeTenantMismatch, "tenant mismatch")
			return
		}
		req.TenantUUID = tenantUUID
		outboundBearer := resolveGatewayBearerToken(c, deps)
		logGatewayAuthSelection(c, deps, outboundBearer, tenantUUID)

		bodyBytes, err := json.Marshal(req)
		if err != nil {
			contracts.ResponseError(c, http.StatusBadRequest, contracts.ErrCodeInvalidRequest, "failed to encode payload")
			return
		}

		endpoint := buildGatewayEndpoint(deps.Config.Gateway, "/event-fabric/topics")
		requestID := strings.TrimSpace(c.GetHeader("X-Request-ID"))
		if requestID == "" {
			requestID = uuid.NewString()
		}

		ctx := c.Request.Context()
		if ctx == nil {
			ctx = context.Background()
		}
		reqUpstream, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
		if err != nil {
			contracts.ResponseError(c, http.StatusBadRequest, contracts.ErrCodeInvalidRequest, "failed to build upstream request")
			return
		}
		reqUpstream.Header.Set("Content-Type", "application/json")
		reqUpstream.Header.Set("Accept", "application/json")
		reqUpstream.Header.Set("X-Request-ID", requestID)
		reqUpstream.Header.Set("Authorization", resolveGatewayAuthorization(deps, outboundBearer))
		if userAgent := strings.TrimSpace(deps.Config.Gateway.UserAgent); userAgent != "" {
			reqUpstream.Header.Set("User-Agent", userAgent)
		}

		client := &http.Client{Timeout: deps.Config.Gateway.Timeout}
		resp, err := client.Do(reqUpstream)
		if err != nil {
			contracts.ResponseError(c, http.StatusBadRequest, "TOPIC_UPSTREAM_FAILED", err.Error())
			return
		}
		defer resp.Body.Close()
		payload, err := io.ReadAll(resp.Body)
		if err != nil {
			contracts.ResponseError(c, http.StatusBadRequest, "TOPIC_UPSTREAM_FAILED", "failed to read upstream response")
			return
		}
		if resp.StatusCode >= http.StatusBadRequest {
			contracts.ResponseError(c, http.StatusBadRequest, "TOPIC_UPSTREAM_FAILED", extractEventFabricError(payload, resp.StatusCode))
			return
		}

		if !isEventFabricSuccess(payload) {
			contracts.ResponseError(c, http.StatusBadRequest, "TOPIC_UPSTREAM_FAILED", extractEventFabricError(payload, resp.StatusCode))
			return
		}
		contracts.ResponseSuccess(c, gin.H{"ok": true})
	}
}

func resolveGatewayAuthorization(deps *app.Deps, outboundBearer string) string {
	if strings.TrimSpace(outboundBearer) != "" {
		return "Bearer " + strings.TrimSpace(outboundBearer)
	}
	if deps == nil || deps.Config == nil || deps.Config.Gateway == nil {
		return ""
	}
	authScheme := strings.ToLower(strings.TrimSpace(deps.Config.Gateway.AuthScheme))
	apiKey := strings.TrimSpace(deps.Config.Gateway.APIKey)
	if authScheme == "apikey" || authScheme == "api_key" || authScheme == "api-key" || (authScheme == "" && apiKey != "") {
		if apiKey == "" {
			return ""
		}
		return "ApiKey " + apiKey
	}
	token, err := deps.HostBearerToken(context.Background())
	if err != nil || strings.TrimSpace(token) == "" {
		return ""
	}
	return "Bearer " + strings.TrimSpace(token)
}

func extractEventFabricError(payload []byte, statusCode int) string {
	message := fmt.Sprintf("topic create rejected with status %d", statusCode)
	if len(payload) == 0 {
		return message
	}
	var envelope eventFabricUpstreamEnvelope
	if err := json.Unmarshal(payload, &envelope); err == nil {
		if envelope.Error != nil && strings.TrimSpace(envelope.Error.Message) != "" {
			return strings.TrimSpace(envelope.Error.Message)
		}
		if strings.TrimSpace(envelope.Message) != "" {
			return strings.TrimSpace(envelope.Message)
		}
	}
	return message
}

func isEventFabricSuccess(payload []byte) bool {
	if len(payload) == 0 {
		return true
	}
	var envelope eventFabricUpstreamEnvelope
	if err := json.Unmarshal(payload, &envelope); err == nil {
		if envelope.Success {
			return true
		}
		if envelope.Code >= 200 && envelope.Code < 300 {
			return true
		}
		return false
	}
	return true
}
