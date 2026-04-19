package integration

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	frameworkgateway "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/gateway"
	capgateway "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/integrations/gateway"
	authx "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/middleware"
	iamservice "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/iam"
	integrationService "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/integration"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// Handler 提供 integration HTTP API 的入口。
type Handler struct {
	deps     *app.Deps
	dispatch *integrationService.DispatchService
	logger   *logrus.Entry
	sts      *iamservice.STSService
}

type capabilityInvokeRequest struct {
	CapabilityID      string                 `json:"capabilityId"`
	Action            string                 `json:"action"`
	PreferredProtocol string                 `json:"preferredProtocol"`
	Payload           map[string]any         `json:"payload"`
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
}

type gatewayAuthPolicy struct {
	AuthRequired bool
	TenantScoped bool
}

// NewHandler 构造新的 Handler。
func NewHandler(deps *app.Deps) *Handler {
	var logger *logrus.Entry
	if deps != nil {
		logger = deps.RuntimeLogger(deps.Ctx, "integration_http", nil)
	}
	h := &Handler{
		deps:   deps,
		logger: logger,
	}
	if deps != nil {
		h.sts = iamservice.NewSTSService(deps.Config, nil, app.PluginID, "")
	}
	h.dispatch = h.buildDispatchService()
	return h
}

func (h *Handler) buildDispatchService() *integrationService.DispatchService {
	if h.deps == nil {
		return nil
	}

	logger := h.logger
	if logger == nil {
		logger = logrus.WithField("component", "integration_http")
	}
	service := integrationService.BuildDispatchService(h.deps, logger)
	if service == nil {
		return nil
	}
	return service
}

// ListGrantMatrix 返回当前 GrantMatrix 视图。
func (h *Handler) ListGrantMatrix(c *gin.Context) {
	respondPlaceholder(c, http.StatusOK, "grant matrix listing not implemented")
}

// SubmitGrantMatrix 接收数据库覆盖项。
func (h *Handler) SubmitGrantMatrix(c *gin.Context) {
	respondPlaceholder(c, http.StatusAccepted, "grant matrix override submission pending approval workflow")
}

// CreateSubscription 注册 webhook 订阅。
func (h *Handler) CreateSubscription(c *gin.Context) {
	respondPlaceholder(c, http.StatusCreated, "webhook subscription endpoint not implemented")
}

// ListSubscriptions 查询 webhook 订阅。
func (h *Handler) ListSubscriptions(c *gin.Context) {
	respondPlaceholder(c, http.StatusOK, "webhook subscription list not implemented")
}

// ReplayDLQ 触发 DLQ 补发。
func (h *Handler) ReplayDLQ(c *gin.Context) {
	respondPlaceholder(c, http.StatusAccepted, "webhook DLQ replay not implemented")
}

// CreateSecret 注册外部凭证。
func (h *Handler) CreateSecret(c *gin.Context) {
	respondPlaceholder(c, http.StatusCreated, "secret lifecycle endpoint not implemented")
}

// RotateSecret 触发凭证轮换。
func (h *Handler) RotateSecret(c *gin.Context) {
	respondPlaceholder(c, http.StatusAccepted, "secret rotation workflow not implemented")
}

// InvokeCapability 代理插件侧前端对 PowerX Gateway 的能力调用。
func (h *Handler) InvokeCapability(c *gin.Context) {
	if h == nil || h.deps == nil || h.deps.CapabilityGateway == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "capability gateway unavailable"})
		return
	}

	var req capabilityInvokeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid capability payload"})
		return
	}

	capabilityID := strings.TrimSpace(req.CapabilityID)
	action := strings.TrimSpace(req.Action)
	if capabilityID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "capabilityId is required"})
		return
	}

	payload := ensurePayload(req.Payload)
	preferredProtocol := strings.TrimSpace(req.PreferredProtocol)
	if preferredProtocol == "" {
		preferredProtocol = inferPreferredProtocol(payload)
	}
	if preferredProtocol == "" {
		preferredProtocol = "rest"
	}
	if strings.EqualFold(preferredProtocol, "rest") {
		if err := validateRestPayload(payload); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}

	headers := collectCapabilityHeaders(c)
	policy := resolveGatewayAuthPolicy(req.Metadata, payload)
	headers, err := h.resolveGatewayAuthHeaders(c, headers, policy)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "GW_POLICY_AUTH_REQUIRED",
				"message": err.Error(),
			},
		})
		return
	}
	warnings := collectCapabilityWarnings(headers)
	requestID := strings.TrimSpace(c.GetHeader("X-Request-ID"))
	tenantUUID := ""
	if tc, ok := authx.GetTenantContext(c); ok {
		tenantUUID = strings.TrimSpace(tc.TenantUUID)
	}

	result, err := h.deps.CapabilityGateway.Invoke(c.Request.Context(), capgateway.InvokeParams{
		CapabilityID:      capabilityID,
		Action:            action,
		PreferredProtocol: preferredProtocol,
		Payload:           payload,
		Headers:           headers,
		RequestID:         requestID,
		TenantUUID:        tenantUUID,
		AuthRequired:      policy.AuthRequired,
		TenantScoped:      policy.TenantScoped,
	})
	if err != nil {
		h.writeCapabilityError(c, err, warnings)
		return
	}

	if result == nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "capability invoke returned no result"})
		return
	}

	if result.TraceID != "" {
		c.Header("X-Trace-Id", result.TraceID)
	}
	if result.Mock {
		msg := result.MockReason
		if strings.TrimSpace(msg) == "" {
			msg = "PX_USE_MOCK 生效，返回模拟数据"
		}
		warnings = append(warnings, msg)
	}

	response := gin.H{
		"traceId": result.TraceID,
		"status":  result.Status,
		"data":    result.Data,
	}
	if len(result.Raw) > 0 {
		response["raw"] = json.RawMessage(result.Raw)
	}
	if len(warnings) > 0 {
		response["warnings"] = warnings
	}

	c.JSON(http.StatusOK, response)
}

func respondPlaceholder(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{
		"status":  "pending",
		"message": message,
	})
}

func collectCapabilityHeaders(c *gin.Context) map[string]string {
	if c == nil {
		return nil
	}
	forward := []string{"X-PX-Use-Mock", "Authorization"}
	headers := make(map[string]string)
	for _, name := range forward {
		if value := strings.TrimSpace(c.GetHeader(name)); value != "" {
			if strings.EqualFold(name, "Authorization") {
				switch authSchemeFromHeader(value) {
				case "bearer", "apikey":
					headers[name] = value
				}
				continue
			}
			headers[name] = value
		}
	}
	if len(headers) == 0 {
		return nil
	}
	return headers
}

func collectCapabilityWarnings(headers map[string]string) []string {
	if len(headers) == 0 {
		return nil
	}
	module := strings.TrimSpace(headers["X-PX-Use-Mock"])
	if module == "" {
		return nil
	}
	return []string{fmt.Sprintf("通过 X-PX-Use-Mock 请求 Mock 模块: %s", module)}
}

func ensurePayload(value map[string]any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	return value
}

func inferPreferredProtocol(payload map[string]any) string {
	if len(payload) == 0 {
		return ""
	}
	if method := strings.TrimSpace(strings.ToUpper(fmt.Sprint(payload["method"]))); method != "" {
		return "rest"
	}
	if proto := strings.TrimSpace(fmt.Sprint(payload["preferred_protocol"])); proto != "" {
		return strings.ToLower(proto)
	}
	if proto := strings.TrimSpace(fmt.Sprint(payload["preferredProtocol"])); proto != "" {
		return strings.ToLower(proto)
	}
	return ""
}

func validateRestPayload(payload map[string]any) error {
	method := strings.TrimSpace(strings.ToUpper(fmt.Sprint(payload["method"])))
	endpoint := strings.TrimSpace(fmt.Sprint(payload["endpoint"]))
	if method == "" {
		return errors.New("REST 协议需要提供 method 字段")
	}
	if endpoint == "" {
		return errors.New("REST 协议需要提供 endpoint 字段")
	}
	return nil
}

func (h *Handler) writeCapabilityError(c *gin.Context, err error, warnings []string) {
	if c == nil {
		return
	}

	var unavailable *capgateway.UnavailableError
	if errors.As(err, &unavailable) {
		payload := gin.H{
			"error":   unavailable.Error(),
			"traceId": "",
		}
		if len(warnings) > 0 {
			payload["warnings"] = warnings
		}
		c.JSON(http.StatusServiceUnavailable, payload)
		return
	}
	var policyErr *capgateway.PolicyError
	if errors.As(err, &policyErr) {
		payload := gin.H{
			"error": gin.H{
				"code":    strings.TrimSpace(policyErr.Code),
				"message": strings.TrimSpace(policyErr.Message),
			},
			"traceId": "",
		}
		if len(warnings) > 0 {
			payload["warnings"] = warnings
		}
		c.JSON(http.StatusBadRequest, payload)
		return
	}

	var invocationErr *frameworkgateway.InvocationError
	if errors.As(err, &invocationErr) {
		if trace := strings.TrimSpace(invocationErr.TraceID); trace != "" {
			c.Header("X-Trace-Id", trace)
		}
		status := invocationErr.StatusCode
		if status <= 0 {
			status = http.StatusBadGateway
		}
		errPayload := gin.H{
			"code":    firstGatewayCode(invocationErr),
			"message": firstGatewayMessage(invocationErr),
			"type":    categorizeGatewayError(status),
		}
		if body := parseGatewayBody(invocationErr.Body); body != nil {
			errPayload["details"] = body
		}
		payload := gin.H{
			"error":   errPayload,
			"traceId": invocationErr.TraceID,
		}
		if len(warnings) > 0 {
			payload["warnings"] = warnings
		}
		c.JSON(status, payload)
		return
	}

	payload := gin.H{
		"error":   err.Error(),
		"traceId": "",
	}
	if len(warnings) > 0 {
		payload["warnings"] = warnings
	}
	c.JSON(http.StatusBadGateway, payload)
}

func firstGatewayCode(err *frameworkgateway.InvocationError) string {
	if err == nil {
		return ""
	}
	if len(err.Errors) == 0 {
		return ""
	}
	return err.Errors[0].Code
}

func firstGatewayMessage(err *frameworkgateway.InvocationError) string {
	if err == nil {
		return ""
	}
	if len(err.Errors) == 0 || strings.TrimSpace(err.Errors[0].Message) == "" {
		text := strings.TrimSpace(string(err.Body))
		if text != "" {
			return text
		}
		return err.Error()
	}
	return err.Errors[0].Message
}

func categorizeGatewayError(status int) string {
	switch status {
	case http.StatusBadRequest:
		return "validation"
	case http.StatusUnauthorized, http.StatusForbidden:
		return "unauthorized"
	case http.StatusTooManyRequests:
		return "rate_limited"
	default:
		return "upstream"
	}
}

func parseGatewayBody(body []byte) any {
	if len(body) == 0 {
		return nil
	}
	text := strings.TrimSpace(string(body))
	if text == "" {
		return nil
	}
	var decoded any
	if json.Unmarshal(body, &decoded) == nil {
		return decoded
	}
	return text
}

func (h *Handler) resolveGatewayAuthHeaders(c *gin.Context, headers map[string]string, policy gatewayAuthPolicy) (map[string]string, error) {
	if !policy.AuthRequired {
		if headers != nil {
			delete(headers, "Authorization")
		}
		return headers, nil
	}
	if headers == nil {
		headers = make(map[string]string)
	}
	rawBearer, _ := authx.GetRawBearerToken(c)
	tc, hasTC := authx.GetTenantContext(c)
	if strings.TrimSpace(rawBearer) != "" && hasTC && strings.TrimSpace(tc.TenantUUID) != "" {
		token, err := h.mintRequestSTSToken(c, tc)
		if err != nil {
			return nil, fmt.Errorf("请求态 STS exchange 失败: %w", err)
		}
		headers["Authorization"] = "Bearer " + token
		return headers, nil
	}
	if policy.TenantScoped {
		return nil, errors.New("tenant_scoped=true 需要有效请求上下文（tenant/user）用于 STS exchange")
	}
	if strings.TrimSpace(headers["Authorization"]) == "" {
		return nil, errors.New("auth_required=true 需要请求态 Authorization")
	}
	return headers, nil
}

func (h *Handler) mintRequestSTSToken(c *gin.Context, tc authx.TenantContext) (string, error) {
	if h == nil {
		return "", errors.New("handler unavailable")
	}
	if h.sts == nil {
		if h.deps == nil {
			return "", errors.New("sts service unavailable")
		}
		h.sts = iamservice.NewSTSService(h.deps.Config, nil, app.PluginID, "")
	}
	token, err := h.sts.Mint(c.Request.Context(), tc)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(token.AccessToken), nil
}

func resolveGatewayAuthPolicy(metadata map[string]interface{}, payload map[string]any) gatewayAuthPolicy {
	policy := gatewayAuthPolicy{
		AuthRequired: true,
		TenantScoped: true,
	}
	readBool := func(m map[string]interface{}, key string) (bool, bool) {
		if len(m) == 0 {
			return false, false
		}
		raw, ok := m[key]
		if !ok || raw == nil {
			return false, false
		}
		switch v := raw.(type) {
		case bool:
			return v, true
		case string:
			s := strings.TrimSpace(strings.ToLower(v))
			switch s {
			case "1", "true", "yes", "on", "y":
				return true, true
			case "0", "false", "no", "off", "n":
				return false, true
			}
		case float64:
			if v == 1 {
				return true, true
			}
			if v == 0 {
				return false, true
			}
		}
		return false, false
	}
	if v, ok := readBool(metadata, "auth_required"); ok {
		policy.AuthRequired = v
	} else if len(payload) > 0 {
		if raw, exists := payload["auth_required"]; exists {
			if vv, ok := raw.(bool); ok {
				policy.AuthRequired = vv
			}
		}
	}
	if v, ok := readBool(metadata, "tenant_scoped"); ok {
		policy.TenantScoped = v
	} else if len(payload) > 0 {
		if raw, exists := payload["tenant_scoped"]; exists {
			if vv, ok := raw.(bool); ok {
				policy.TenantScoped = vv
			}
		}
	}
	return policy
}

func authSchemeFromHeader(authHeader string) string {
	auth := strings.TrimSpace(strings.ToLower(authHeader))
	switch {
	case strings.HasPrefix(auth, "bearer "):
		return "bearer"
	case strings.HasPrefix(auth, "apikey "), strings.HasPrefix(auth, "api_key "), strings.HasPrefix(auth, "api-key "):
		return "apikey"
	default:
		return "none"
	}
}
