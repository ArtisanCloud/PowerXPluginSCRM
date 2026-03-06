package gateway

import (
	"context"
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
		c.offlineReason = "未找到 gateway 配置，请执行 `px-plugin login --manifest ./skeleton/plugin.yaml` 或在 .env.local 写入 PX_GATEWAY_*"
		return c
	}

	for _, module := range cfg.Gateway.UseMock {
		if normalized := normalizeModule(module); normalized != "" {
			c.useMock[normalized] = struct{}{}
		}
	}

	gcfg := cfg.Gateway
	baseURL := strings.TrimSpace(gcfg.BaseURL)
	if baseURL == "" || !hasGatewayCredential(gcfg) {
		c.offlineReason = "PX_GATEWAY_BASE_URL 与鉴权凭证未配置（bearer 需要 PX_TOOL_TOKEN，apikey 需要 PX_GATEWAY_API_KEY）"
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

	req := frameworkgateway.InvokeRequest{
		CapabilityID:      params.CapabilityID,
		Action:            params.Action,
		PreferredProtocol: params.PreferredProtocol,
		Payload:           params.Payload,
		RequestID:         params.RequestID,
		Headers:           copyHeaders(params.Headers),
		TenantUUID:        params.TenantUUID,
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
	token := strings.TrimSpace(gcfg.ToolToken)
	apiKey := strings.TrimSpace(gcfg.APIKey)
	authScheme := resolveGatewayAuthScheme(gcfg)
	if authScheme == "apikey" && apiKey == "" {
		return nil, fmt.Errorf("PX_GATEWAY_API_KEY 未配置")
	}
	if authScheme != "apikey" && token == "" {
		return nil, fmt.Errorf("PX_TOOL_TOKEN 未配置")
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
		c.logger.WithError(err).Warn("PX_TOOL_TOKEN 自动刷新失败")
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
	if base == "" || !hasGatewayCredential(cfg.Gateway) {
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
	if strings.TrimSpace(c.cfg.Gateway.RefreshToken) == "" {
		return false, fmt.Errorf("PX_TOOL_REFRESH_TOKEN 未配置")
	}
	c.logger.Info("检测到 Gateway 凭证失败，尝试自动刷新 PX_TOOL_TOKEN")
	if _, _, err := RefreshToolToken(ctx, c.cfg); err != nil {
		return false, err
	}
	if err := c.reconnectTransport(); err != nil {
		return false, err
	}
	c.logger.Info("PX_TOOL_TOKEN 已刷新，准备重试 Gateway 调用")
	return true, nil
}

func (c *Client) reconnectTransport() error {
	if c.cfg == nil || c.cfg.Gateway == nil {
		return fmt.Errorf("gateway config missing")
	}
	gcfg := c.cfg.Gateway
	baseURL := strings.TrimSpace(gcfg.BaseURL)
	if baseURL == "" || !hasGatewayCredential(gcfg) {
		return fmt.Errorf("PX_GATEWAY_BASE_URL 与鉴权凭证未配置")
	}

	timeout := gcfg.Timeout
	if timeout <= 0 {
		timeout = defaultRequestTimeout
	}

	client, err := frameworkgateway.NewClient(frameworkgateway.Config{
		BaseURL:        baseURL,
		APIPrefix:      strings.TrimSpace(gcfg.APIPrefix),
		AuthScheme:     strings.TrimSpace(gcfg.AuthScheme),
		ToolToken:      strings.TrimSpace(gcfg.ToolToken),
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

func hasGatewayCredential(gcfg *config.GatewayConfig) bool {
	if gcfg == nil {
		return false
	}
	if resolveGatewayAuthScheme(gcfg) == "apikey" {
		return strings.TrimSpace(gcfg.APIKey) != ""
	}
	return strings.TrimSpace(gcfg.ToolToken) != ""
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
