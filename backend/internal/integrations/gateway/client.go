package gateway

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	frameworkgateway "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/gateway"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/config"
	powerxclient "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/grpc/client"
	skelLogger "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/logger"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

const (
	defaultRequestTimeout = 60 * time.Second
)

// InvokeParams 描述一次能力调用的输入。
type InvokeParams struct {
	CapabilityID      string
	Action            string
	PreferredProtocol string
	Payload           any
	Headers           map[string]string
	RequestID         string
	TenantUUID        string
	AuthRequired      bool
	TenantScoped      bool
}

// InvokeResult 描述能力调用的输出。
type InvokeResult struct {
	TraceID    string
	Status     string
	Data       map[string]any
	Raw        json.RawMessage
	Mock       bool
	MockReason string
}

// ListPlatformCapabilitiesOptions controls remote catalog listing.
type ListPlatformCapabilitiesOptions struct {
	Source   string
	Channel  string
	PageSize int
}

// PlatformCapabilityRecord mirrors PowerX capability registry DTO.
type PlatformCapabilityRecord struct {
	CapabilityID     string                       `json:"capability_id"`
	PluginID         string                       `json:"plugin_id"`
	PluginVersion    string                       `json:"plugin_version"`
	Title            string                       `json:"title"`
	Description      string                       `json:"description"`
	Source           string                       `json:"source"`
	Categories       []string                     `json:"categories"`
	Intents          []string                     `json:"intents"`
	ToolScope        []string                     `json:"tool_scope"`
	CapabilitiesHash string                       `json:"capabilities_hash"`
	ProtocolHash     string                       `json:"protocol_hash"`
	Status           string                       `json:"status"`
	ExecutionMode    string                       `json:"execution_mode"`
	Protocols        []PlatformCapabilityProtocol `json:"protocols"`
}

// PlatformCapabilityProtocol describes channel metadata.
type PlatformCapabilityProtocol struct {
	Channel   string `json:"channel"`
	Endpoint  string `json:"endpoint"`
	SchemaRef string `json:"schema_ref"`
	Method    string `json:"method"`
	RPC       string `json:"rpc"`
	ToolRef   string `json:"tool_ref"`
}

type gatewayInvoker interface {
	Invoke(ctx context.Context, req frameworkgateway.InvokeRequest) (*frameworkgateway.Response, error)
	Close() error
}

// Client 封装 Skeleton 环境的 Gateway 调用能力，支持 PX_USE_MOCK 与离线提示。
type Client struct {
	transport     gatewayInvoker
	logger        *logrus.Entry
	useMock       map[string]struct{}
	offlineReason string
	cfg           *config.Config
	refreshMu     sync.Mutex
	stsClient     *powerxclient.PowerXServiceClient
}

// NewClient 构造 Gateway Client；若凭证缺失，则进入离线模式。
func NewClient(cfg *config.Config, log *logrus.Entry) *Client {
	c := &Client{
		logger:  ensureLogger(log),
		useMock: make(map[string]struct{}),
		cfg:     cfg,
	}
	c.logger = c.logger.WithField("component", "skeleton.gateway.client")

	if cfg == nil || cfg.Gateway == nil {
		c.offlineReason = "未找到 gateway 配置，请配置 PX_GATEWAY_BASE_URL 与 STS 客户端凭证"
		return c
	}

	for _, module := range cfg.Gateway.UseMock {
		if normalized := normalizeModule(module); normalized != "" {
			c.useMock[normalized] = struct{}{}
		}
	}

	gcfg := cfg.Gateway
	baseURL := strings.TrimSpace(gcfg.BaseURL)
	if baseURL == "" || !hasGatewayCredential(cfg) {
		c.offlineReason = "PX_GATEWAY_BASE_URL 与鉴权凭证未配置（Bearer 模式需要 POWERX_STS_CLIENT_ID/SECRET）"
		return c
	}

	timeout := gcfg.Timeout
	if timeout <= 0 {
		timeout = defaultRequestTimeout
	}

	if err := c.reconnectTransport(); err != nil {
		c.offlineReason = fmt.Sprintf("初始化 Gateway Client 失败: %v", err)
		c.logger.WithError(err).Warn("无法初始化 Gateway Client，Skeleton 将保持离线状态")
		return c
	}

	c.logger.WithFields(logrus.Fields{
		"mockModules": c.mockModules(),
		"baseURL":     baseURL,
	}).Info("Gateway Client 初始化完成")
	return c
}

// Enabled 返回 Gateway Client 是否可用。
func (c *Client) Enabled() bool {
	return c.transport != nil
}

// Invoke 触发能力调用；若命中 PX_USE_MOCK，则直接返回 Mock 响应。
func (c *Client) Invoke(ctx context.Context, params InvokeParams) (*InvokeResult, error) {
	module, mocked := c.shouldMock(params.CapabilityID)
	if mocked {
		c.logger.WithFields(logrus.Fields{
			"capability": params.CapabilityID,
			"action":     params.Action,
			"module":     module,
		}).Info("PX_USE_MOCK 生效，返回 Mock 数据")
		return c.mockResult(module, params, "PX_USE_MOCK"), nil
	}

	if c.transport == nil {
		return nil, c.unavailableError(params.CapabilityID)
	}
	authRequired := params.AuthRequired
	tenantScoped := params.TenantScoped
	if !authRequired && !tenantScoped {
		// Backward-compatible defaults for legacy callers.
		authRequired = true
		tenantScoped = true
	}
	requestAuthHeader := headerValue(params.Headers, "Authorization")
	tokenSource := "none"
	if strings.TrimSpace(requestAuthHeader) != "" {
		tokenSource = "request"
	}
	tokenTID := ""
	if authRequired && strings.TrimSpace(requestAuthHeader) == "" {
		return nil, &PolicyError{
			Code:    "GW_POLICY_AUTH_REQUIRED",
			Message: "auth_required=true 时必须提供请求态 Authorization（Bearer STS token）",
		}
	}
	if tenantScoped {
		tid, ok := tenantUUIDFromAuthHeader(requestAuthHeader)
		if !ok || strings.TrimSpace(tid) == "" {
			return nil, &PolicyError{
				Code:    "GW_POLICY_TENANT_TOKEN_REQUIRED",
				Message: "tenant_scoped=true 时 Authorization 必须为包含 tid claim 的 Bearer token",
			}
		}
		if isZeroTenantUUID(tid) {
			return nil, &PolicyError{
				Code:    "GW_POLICY_ZERO_TENANT_FORBIDDEN",
				Message: "tenant_scoped=true 时不允许使用零租户 token（tid=00000000-...）",
			}
		}
		if wanted := strings.TrimSpace(params.TenantUUID); wanted != "" && !strings.EqualFold(wanted, tid) {
			return nil, &PolicyError{
				Code:    "GW_POLICY_TENANT_MISMATCH",
				Message: fmt.Sprintf("tenant token tid(%s) 与请求 tenant_uuid(%s) 不一致", tid, wanted),
			}
		}
		params.TenantUUID = tid
		tokenTID = tid
	}

	req := frameworkgateway.InvokeRequest{
		CapabilityID:      params.CapabilityID,
		Action:            params.Action,
		PreferredProtocol: params.PreferredProtocol,
		Payload:           params.Payload,
		RequestID:         params.RequestID,
		Headers:           copyHeaders(params.Headers),
		TenantUUID:        params.TenantUUID,
		DisableAuth:       !authRequired,
	}
	if c.logger != nil {
		c.logger.WithFields(logrus.Fields{
			"capability":         params.CapabilityID,
			"action":             params.Action,
			"request_id":         params.RequestID,
			"auth_required":      authRequired,
			"tenant_scoped":      tenantScoped,
			"token_source":       tokenSource,
			"token_tid":          maskTenantUUID(tokenTID),
			"preferred_protocol": params.PreferredProtocol,
		}).Info("gateway invoke dispatch")
	}
	resp, err := c.transport.Invoke(ctx, req)
	if err != nil {
		if retryResp, retryErr := c.handleInvokeError(ctx, req, params.CapabilityID, params.Action, err); retryErr == nil && retryResp != nil {
			resp = retryResp
		} else if retryErr != nil {
			return nil, fmt.Errorf("gateway invoke %s/%s 失败: %w", params.CapabilityID, params.Action, retryErr)
		} else {
			return nil, fmt.Errorf("gateway invoke %s/%s 失败: %w", params.CapabilityID, params.Action, err)
		}
	}
	return &InvokeResult{
		TraceID: resp.TraceID,
		Status:  resp.Status,
		Data:    resp.Data,
		Raw:     resp.RawData,
	}, nil
}

// ListPlatformCapabilities retrieves platform capability metadata via tenant API.
func (c *Client) ListPlatformCapabilities(ctx context.Context, opts ListPlatformCapabilitiesOptions) ([]PlatformCapabilityRecord, error) {
	if c.cfg == nil || c.cfg.Gateway == nil {
		return nil, fmt.Errorf("gateway config missing")
	}
	gcfg := c.cfg.Gateway
	baseURL := strings.TrimRight(strings.TrimSpace(gcfg.BaseURL), "/")
	if baseURL == "" {
		return nil, fmt.Errorf("PX_GATEWAY_BASE_URL 未配置")
	}
	apiKey := strings.TrimSpace(gcfg.APIKey)
	authScheme := resolveGatewayAuthScheme(gcfg)
	if authScheme == "apikey" && apiKey == "" {
		return nil, fmt.Errorf("PX_GATEWAY_API_KEY 未配置")
	}
	token := ""
	if authScheme != "apikey" {
		var err error
		token, err = c.gatewayBearerToken(ctx)
		if err != nil {
			return nil, err
		}
	}

	timeout := gcfg.Timeout
	if timeout <= 0 {
		timeout = defaultRequestTimeout
	}

	query := url.Values{}
	if source := strings.TrimSpace(opts.Source); source != "" {
		query.Set("source", source)
	} else {
		query.Set("source", "corex")
	}
	if channel := strings.TrimSpace(opts.Channel); channel != "" {
		query.Set("channel", channel)
	}
	pageSize := opts.PageSize
	if pageSize <= 0 || pageSize > 500 {
		pageSize = 200
	}
	query.Set("page_size", strconv.Itoa(pageSize))

	endpoint := buildGatewayEndpoint(gcfg, "/tenant/capabilities")
	if encoded := query.Encode(); encoded != "" {
		endpoint += "?" + encoded
	}

	ctxReq, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctxReq, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if authScheme == "apikey" {
		req.Header.Set("Authorization", "ApiKey "+apiKey)
	} else {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("X-Request-ID", uuid.NewString())

	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var payload platformCapabilityResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode capability list: %w", err)
	}
	if resp.StatusCode >= 400 || payload.Code >= 400 {
		return nil, fmt.Errorf("platform capability list failed: status=%d code=%d message=%s", resp.StatusCode, payload.Code, payload.Message)
	}
	return payload.Data.Items, nil
}

func (c *Client) handleInvokeError(ctx context.Context, req frameworkgateway.InvokeRequest, capabilityID, action string, invokeErr error) (*frameworkgateway.Response, error) {
	var gwErr *frameworkgateway.InvocationError
	if !errors.As(invokeErr, &gwErr) {
		return nil, invokeErr
	}
	if gwErr.StatusCode != http.StatusUnauthorized && gwErr.StatusCode != http.StatusForbidden {
		return nil, invokeErr
	}
	refreshed, err := c.refreshCredentials(ctx)
	if err != nil {
		c.logger.WithError(err).Warn("STS token refresh failed")
		return nil, invokeErr
	}
	if !refreshed {
		return nil, invokeErr
	}
	resp, err := c.transport.Invoke(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("gateway invoke %s/%s 失败: %w", capabilityID, action, err)
	}
	return resp, nil
}

// Close 释放底层连接。
func (c *Client) Close() error {
	if c.transport == nil {
		return nil
	}
	return c.transport.Close()
}

// UnavailableError 用于提示 Gateway 离线的原因。
type UnavailableError struct {
	Reason string
}

func (e *UnavailableError) Error() string {
	if e == nil {
		return ""
	}
	return "gateway 不可用: " + e.Reason
}

// PolicyError 表示调用策略不满足（例如缺少请求态鉴权信息）。
type PolicyError struct {
	Code    string
	Message string
}

func (e *PolicyError) Error() string {
	if e == nil {
		return ""
	}
	if strings.TrimSpace(e.Code) == "" {
		return strings.TrimSpace(e.Message)
	}
	return strings.TrimSpace(e.Code) + ": " + strings.TrimSpace(e.Message)
}

func (c *Client) unavailableError(capabilityID string) error {
	reason := c.offlineReason
	if reason == "" {
		reason = "Gateway 未初始化"
	}
	if capabilityID != "" {
		reason += " (capability: " + capabilityID + ")"
	}
	return &UnavailableError{Reason: reason}
}

func (c *Client) mockResult(module string, params InvokeParams, reason string) *InvokeResult {
	traceID := fmt.Sprintf("mock-%s-%d", module, time.Now().UnixNano())
	payloadEcho := params.Payload
	data := map[string]any{
		"mock":       true,
		"module":     module,
		"action":     params.Action,
		"capability": params.CapabilityID,
		"message":    fmt.Sprintf("Mock 模式生效：%s", reason),
	}
	if payloadEcho != nil {
		data["echoPayload"] = payloadEcho
	}
	return &InvokeResult{
		TraceID:    traceID,
		Status:     "mock",
		Data:       data,
		Mock:       true,
		MockReason: reason,
	}
}

func (c *Client) shouldMock(capabilityID string) (string, bool) {
	if len(c.useMock) == 0 {
		return "", false
	}
	module := moduleFromCapability(capabilityID)
	if module == "" {
		return "", false
	}
	_, ok := c.useMock[module]
	return module, ok
}

func (c *Client) mockModules() []string {
	if len(c.useMock) == 0 {
		return nil
	}
	result := make([]string, 0, len(c.useMock))
	for module := range c.useMock {
		result = append(result, module)
	}
	return result
}

func moduleFromCapability(capabilityID string) string {
	parts := strings.Split(strings.TrimSpace(capabilityID), ".")
	if len(parts) >= 3 {
		return normalizeModule(parts[2])
	}
	if len(parts) >= 2 {
		return normalizeModule(parts[1])
	}
	if len(parts) == 1 {
		return normalizeModule(parts[0])
	}
	return ""
}

func normalizeModule(module string) string {
	return strings.ToLower(strings.TrimSpace(module))
}

func copyHeaders(src map[string]string) map[string]string {
	if len(src) == 0 {
		return nil
	}
	dest := make(map[string]string, len(src))
	for k, v := range src {
		if strings.TrimSpace(k) == "" || strings.TrimSpace(v) == "" {
			continue
		}
		dest[k] = v
	}
	return dest
}

func headerValue(headers map[string]string, key string) string {
	if len(headers) == 0 {
		return ""
	}
	for k, v := range headers {
		if strings.EqualFold(strings.TrimSpace(k), key) {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func tenantUUIDFromAuthHeader(authHeader string) (string, bool) {
	header := strings.TrimSpace(authHeader)
	if header == "" {
		return "", false
	}
	if !strings.HasPrefix(strings.ToLower(header), "bearer ") {
		return "", false
	}
	token := strings.TrimSpace(header[len("Bearer "):])
	tid := tenantUUIDFromJWT(token)
	if strings.TrimSpace(tid) == "" {
		return "", false
	}
	return tid, true
}

func tenantUUIDFromJWT(token string) string {
	token = strings.TrimSpace(token)
	if token == "" {
		return ""
	}
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return ""
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return ""
	}
	data := map[string]any{}
	if err := json.Unmarshal(payload, &data); err != nil {
		return ""
	}
	for _, key := range []string{"tid", "tenant_uuid", "tenantUuid", "tenant"} {
		raw, ok := data[key]
		if !ok || raw == nil {
			continue
		}
		if value := strings.TrimSpace(fmt.Sprintf("%v", raw)); value != "" {
			return value
		}
	}
	return ""
}

func isZeroTenantUUID(value string) bool {
	return strings.EqualFold(strings.TrimSpace(value), "00000000-0000-0000-0000-000000000000")
}

func maskTenantUUID(value string) string {
	clean := strings.TrimSpace(value)
	if clean == "" {
		return ""
	}
	if len(clean) <= 8 {
		return clean
	}
	return clean[:8] + "..."
}

func ensureLogger(entry *logrus.Entry) *logrus.Entry {
	if entry != nil {
		return entry
	}
	return skelLogger.WithField("component", "skeleton.gateway.client")
}

// ValidateConfig 快速检查 Gateway 配置是否就绪（供启动前自检使用）。
func ValidateConfig(cfg *config.Config) error {
	if cfg == nil || cfg.Gateway == nil {
		return errors.New("gateway config missing")
	}
	base := strings.TrimSpace(cfg.Gateway.BaseURL)
	if base == "" || !hasGatewayCredential(cfg) {
		return errors.New("PX_GATEWAY_BASE_URL 与鉴权凭证未配置")
	}
	return nil
}

func (c *Client) refreshCredentials(ctx context.Context) (bool, error) {
	c.refreshMu.Lock()
	defer c.refreshMu.Unlock()

	if c.cfg == nil || c.cfg.Gateway == nil {
		return false, fmt.Errorf("gateway config missing")
	}
	if !gatewaySTSConfigured(c.cfg) {
		return false, fmt.Errorf("POWERX_STS_CLIENT_ID/SECRET 未配置")
	}
	if c.stsClient != nil {
		c.stsClient.InvalidateSTS()
	}
	if _, err := c.gatewayBearerToken(ctx); err != nil {
		return false, err
	}
	if err := c.reconnectTransport(); err != nil {
		return false, err
	}
	c.logger.Info("STS token 已刷新，准备重试 Gateway 调用")
	return true, nil
}

func (c *Client) reconnectTransport() error {
	if c.cfg == nil || c.cfg.Gateway == nil {
		return fmt.Errorf("gateway config missing")
	}
	gcfg := c.cfg.Gateway
	baseURL := strings.TrimSpace(gcfg.BaseURL)
	if baseURL == "" || !hasGatewayCredential(c.cfg) {
		return fmt.Errorf("PX_GATEWAY_BASE_URL 与鉴权凭证未配置")
	}

	timeout := gcfg.Timeout
	if timeout <= 0 {
		timeout = defaultRequestTimeout
	}

	bearerToken := ""
	if resolveGatewayAuthScheme(gcfg) != "apikey" {
		token, err := c.gatewayBearerToken(context.Background())
		if err != nil {
			return err
		}
		bearerToken = token
	}
	client, err := frameworkgateway.NewClient(frameworkgateway.Config{
		BaseURL:        baseURL,
		APIPrefix:      strings.TrimSpace(gcfg.APIPrefix),
		AuthScheme:     strings.TrimSpace(gcfg.AuthScheme),
		BearerToken:    bearerToken,
		APIKey:         strings.TrimSpace(gcfg.APIKey),
		TenantUUID:     strings.TrimSpace(gcfg.TenantUUID),
		RequestTimeout: timeout,
		UserAgent:      strings.TrimSpace(gcfg.UserAgent),
	})
	if err != nil {
		return err
	}
	if c.transport != nil {
		_ = c.transport.Close()
	}
	c.transport = client
	return nil
}

type platformCapabilityResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Items []PlatformCapabilityRecord `json:"items"`
	} `json:"data"`
}

func resolveGatewayAuthScheme(gcfg *config.GatewayConfig) string {
	if gcfg == nil {
		return "bearer"
	}
	scheme := strings.ToLower(strings.TrimSpace(gcfg.AuthScheme))
	switch scheme {
	case "apikey", "api_key", "api-key":
		return "apikey"
	case "bearer":
		return "bearer"
	}
	if strings.TrimSpace(gcfg.APIKey) != "" {
		return "apikey"
	}
	return "bearer"
}

func (c *Client) gatewayBearerToken(ctx context.Context) (string, error) {
	if c == nil || c.cfg == nil {
		return "", fmt.Errorf("gateway config missing")
	}
	if gatewaySTSConfigured(c.cfg) {
		if c.stsClient == nil {
			client, err := powerxclient.NewPowerXServiceClient(ctx, c.cfg.GRPCUpstream)
			if err != nil {
				return "", err
			}
			c.stsClient = client
		}
		token, err := c.stsClient.AccessToken(ctx)
		if err != nil {
			return "", err
		}
		if strings.TrimSpace(token) == "" {
			return "", fmt.Errorf("empty STS access token")
		}
		return strings.TrimSpace(token), nil
	}
	return "", fmt.Errorf("POWERX_STS_CLIENT_ID/SECRET 未配置")
}

func gatewaySTSConfigured(cfg *config.Config) bool {
	return cfg != nil &&
		cfg.GRPCUpstream != nil &&
		strings.TrimSpace(cfg.GRPCUpstream.STSClientID) != "" &&
		strings.TrimSpace(cfg.GRPCUpstream.STSClientSecret) != ""
}

func hasGatewayCredential(cfg *config.Config) bool {
	if cfg == nil || cfg.Gateway == nil {
		return false
	}
	gcfg := cfg.Gateway
	if resolveGatewayAuthScheme(gcfg) == "apikey" {
		return strings.TrimSpace(gcfg.APIKey) != ""
	}
	return gatewaySTSConfigured(cfg)
}

func normalizeGatewayAPIPrefix(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "/api/v1"
	}
	if !strings.HasPrefix(value, "/") {
		value = "/" + value
	}
	value = "/" + strings.Trim(strings.TrimSpace(value), "/")
	if value == "/" {
		return "/api/v1"
	}
	return value
}

func buildGatewayEndpoint(gcfg *config.GatewayConfig, routePath string) string {
	if gcfg == nil {
		return ""
	}
	base := strings.TrimRight(strings.TrimSpace(gcfg.BaseURL), "/")
	route := "/" + strings.TrimLeft(strings.TrimSpace(routePath), "/")
	prefix := normalizeGatewayAPIPrefix(strings.TrimSpace(gcfg.APIPrefix))
	if strings.HasSuffix(base, prefix) {
		return base + route
	}
	return base + prefix + route
}
