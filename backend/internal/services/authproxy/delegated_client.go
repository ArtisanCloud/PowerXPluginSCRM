package authproxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	iamservice "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/iam"
)

// HTTPClient captures the subset of http.Client we rely on.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// DelegatedClient proxies auth calls to PowerX Core.
type DelegatedClient struct {
	apiBase      string
	httpClient   HTTPClient
	serviceToken string
	pluginID     string
}

// Option allows tweaking DelegatedClient construction.
type Option func(*DelegatedClient)

// WithHTTPClient overrides the default http.Client.
func WithHTTPClient(c HTTPClient) Option {
	return func(dc *DelegatedClient) {
		if c != nil {
			dc.httpClient = c
		}
	}
}

// WithPluginID sets the plugin identifier for logging/auditing headers.
func WithPluginID(id string) Option {
	return func(dc *DelegatedClient) {
		dc.pluginID = id
	}
}

// NewDelegatedClient creates a proxy with explicit endpoint/token.
func NewDelegatedClient(baseURL, token string, opts ...Option) (*DelegatedClient, error) {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		baseURL = strings.TrimSpace(os.Getenv("POWERX_CORE_ENDPOINT"))
	}
	if baseURL == "" {
		baseURL = "http://localhost:8077"
	}
	baseURL = strings.TrimRight(baseURL, "/")

	apiBase := ensureAPIBase(baseURL)

	client := &DelegatedClient{
		apiBase:      apiBase,
		httpClient:   &http.Client{Timeout: 10 * time.Second},
		serviceToken: strings.TrimSpace(token),
		pluginID:     strings.TrimSpace(os.Getenv("POWERX_PLUGIN_ID")),
	}
	if client.serviceToken == "" {
		client.serviceToken = strings.TrimSpace(os.Getenv("POWERX_AUTH_TOKEN"))
	}

	for _, opt := range opts {
		opt(client)
	}

	if client.httpClient == nil {
		client.httpClient = &http.Client{Timeout: 10 * time.Second}
	}

	return client, nil
}

func ensureAPIBase(base string) string {
	lower := strings.ToLower(base)
	if !strings.Contains(lower, "/api") {
		base = strings.TrimRight(base, "/") + "/api"
	}
	if !strings.HasSuffix(strings.ToLower(base), "/api/v1") {
		base = strings.TrimRight(base, "/") + "/v1"
	}
	return base
}

// Login delegates tenant login to PowerX Core.
func (c *DelegatedClient) Login(ctx context.Context, req iamservice.LoginRequest) (*iamservice.AuthTokens, error) {
	payload := map[string]any{
		"identifier": strings.TrimSpace(req.Identifier),
		"password":   req.Password,
	}
	if strings.TrimSpace(req.Tenant) != "" {
		payload["tenant"] = strings.TrimSpace(req.Tenant)
	}
	if req.Remember {
		payload["remember"] = true
	}

	var resp loginResponse
	if err := c.post(ctx, "/admin/user/auth/login", payload, &resp, nil); err != nil {
		return nil, err
	}
	return resp.toTokens(resp.RefreshToken), nil
}

// Refresh exchanges refresh_token for a fresh access token.
func (c *DelegatedClient) Refresh(ctx context.Context, refreshToken string) (*iamservice.AuthTokens, error) {
	refreshToken = strings.TrimSpace(refreshToken)
	if refreshToken == "" {
		return nil, fmt.Errorf("authproxy: refresh token required")
	}
	var resp loginResponse
	if err := c.post(ctx, "/admin/user/auth/refresh", map[string]string{
		"refresh_token": refreshToken,
	}, &resp, nil); err != nil {
		return nil, err
	}
	if resp.RefreshToken == "" {
		resp.RefreshToken = refreshToken
	}
	return resp.toTokens(resp.RefreshToken), nil
}

// Logout revokes the refresh token upstream.
func (c *DelegatedClient) Logout(ctx context.Context, refreshToken string) error {
	refreshToken = strings.TrimSpace(refreshToken)
	if refreshToken == "" {
		return fmt.Errorf("authproxy: refresh token required")
	}
	return c.post(ctx, "/admin/user/auth/logout", map[string]string{
		"refresh_token": refreshToken,
	}, nil, nil)
}

// MeContext fetches the current user context via the provided access token.
func (c *DelegatedClient) MeContext(ctx context.Context, accessToken string) (*MeContext, error) {
	accessToken = strings.TrimSpace(accessToken)
	if accessToken == "" {
		return nil, fmt.Errorf("authproxy: access token required")
	}
	headers := map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", accessToken),
	}
	var resp MeContext
	if err := c.get(ctx, "/auth/me/context", &resp, headers); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ProxyRequest forwards a generic JSON request to PowerX Core.
func (c *DelegatedClient) ProxyRequest(ctx context.Context, method, path string, payload any, out any, extraHeaders map[string]string) error {
	method = strings.ToUpper(strings.TrimSpace(method))
	if method == "" {
		method = http.MethodGet
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	var body io.Reader
	if payload != nil && method != http.MethodGet && method != http.MethodDelete {
		raw, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("authproxy: marshal payload: %w", err)
		}
		body = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.apiBase+path, body)
	if err != nil {
		return fmt.Errorf("authproxy: build request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	c.applyHeaders(req, extraHeaders)
	return c.do(req, out)
}

// ----- HTTP helpers -----

type loginResponse struct {
	TokenType     string `json:"token_type"`
	AccessToken   string `json:"access_token"`
	ExpiresIn     int    `json:"expires_in"`
	RefreshToken  string `json:"refresh_token,omitempty"`
	Scope         string `json:"scope"`
	PluginID      string `json:"plugin_id,omitempty"`
	PolicyVersion string `json:"policy_version,omitempty"`
}

func (lr loginResponse) toTokens(refresh string) *iamservice.AuthTokens {
	expires := time.Now().Add(time.Duration(lr.ExpiresIn) * time.Second)
	return &iamservice.AuthTokens{
		TokenType:     lr.TokenType,
		AccessToken:   lr.AccessToken,
		RefreshToken:  refresh,
		ExpiresIn:     int64(lr.ExpiresIn),
		Scope:         lr.Scope,
		ExpiresAt:     expires,
		PluginID:      lr.PluginID,
		PolicyVersion: lr.PolicyVersion,
	}
}

func (c *DelegatedClient) post(ctx context.Context, path string, payload any, out any, extraHeaders map[string]string) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("authproxy: marshal payload: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.apiBase+path, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("authproxy: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	c.applyHeaders(req, extraHeaders)
	return c.do(req, out)
}

func (c *DelegatedClient) get(ctx context.Context, path string, out any, extraHeaders map[string]string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.apiBase+path, nil)
	if err != nil {
		return fmt.Errorf("authproxy: build request: %w", err)
	}
	c.applyHeaders(req, extraHeaders)
	return c.do(req, out)
}

func (c *DelegatedClient) applyHeaders(req *http.Request, extra map[string]string) {
	if c.serviceToken != "" {
		if extra == nil || strings.TrimSpace(extra["Authorization"]) == "" {
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.serviceToken))
		}
		req.Header.Set("X-PowerX-Service-Token", c.serviceToken)
	}
	if c.pluginID != "" {
		req.Header.Set("X-PowerX-Plugin-ID", c.pluginID)
	}
	for k, v := range extra {
		req.Header.Set(k, v)
	}
}

func (c *DelegatedClient) do(req *http.Request, out any) error {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return iamservice.ErrAuthUnavailable
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return parseProxyError(resp)
	}

	if out == nil {
		io.Copy(io.Discard, resp.Body)
		return nil
	}

	var envelope hostSuccess
	envelope.Target = out
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return fmt.Errorf("authproxy: decode response: %w", err)
	}
	return nil
}

// hostSuccess decodes {code,message,data} envelopes.
type hostSuccess struct {
	Code      int             `json:"code"`
	Message   string          `json:"message"`
	Timestamp int64           `json:"timestamp"`
	RequestID string          `json:"request_id"`
	Target    any             `json:"-"`
	DataRaw   json.RawMessage `json:"data"`
}

func (h *hostSuccess) UnmarshalJSON(data []byte) error {
	type alias hostSuccess
	aux := struct {
		*alias
	}{alias: (*alias)(h)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if h.Target != nil && len(h.DataRaw) > 0 {
		if err := json.Unmarshal(h.DataRaw, h.Target); err != nil {
			return err
		}
	}
	return nil
}

// ProxyError captures upstream HTTP failures.
type ProxyError struct {
	Status  int
	Message string
}

func (e *ProxyError) Error() string {
	return fmt.Sprintf("authproxy: status %d: %s", e.Status, e.Message)
}

func parseProxyError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	var errResp struct {
		Code      int    `json:"code"`
		Message   string `json:"message"`
		Error     string `json:"error"`
		ErrorCode string `json:"error_code"`
	}
	msg := resp.Status
	if len(body) > 0 {
		if err := json.Unmarshal(body, &errResp); err == nil {
			switch {
			case errResp.Message != "":
				msg = errResp.Message
			case errResp.Error != "":
				msg = errResp.Error
			default:
				msg = string(body)
			}
		} else {
			msg = string(body)
		}
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return iamservice.ErrUnauthorized
	}
	return &ProxyError{Status: resp.StatusCode, Message: msg}
}

// MeContext mirrors PowerX Core /auth/me/context payload.
type MeContext struct {
	IsRoot            bool            `json:"is_root"`
	CurrentTenantUUID string          `json:"current_tenant_uuid"`
	CurrentMemberID   *uint64         `json:"current_member_id,omitempty"`
	Tenant            *MeTenantBrief  `json:"tenant,omitempty"`
	User              *MeUserBrief    `json:"user,omitempty"`
	Members           []MeMemberBrief `json:"members"`
	Roles             []string        `json:"roles"`
	Permissions       []string        `json:"permissions"`
	Capabilities      MeCapabilities  `json:"capabilities"`
	PolicyVersion     string          `json:"policy_version,omitempty"`
	PluginID          string          `json:"plugin_id,omitempty"`
}

type MeTenantBrief struct {
	UUID     string  `json:"uuid"`
	Key      string  `json:"key"`
	Name     string  `json:"name"`
	LegacyID *uint64 `json:"legacy_id,omitempty"`
}

type MeUserBrief struct {
	ID          uint64 `json:"id"`
	Username    string `json:"username"`
	Email       string `json:"email"`
	Phone       string `json:"phone"`
	DisplayName string `json:"display_name"`
	AvatarURL   string `json:"avatar_url"`
	Status      int16  `json:"status"`
	IsRoot      bool   `json:"is_root"`
	LastLoginAt *int64 `json:"last_login_at,omitempty"`
}

type MeMemberBrief struct {
	TenantUUID string `json:"tenant_uuid"`
	TenantName string `json:"tenant_name"`
	MemberID   uint64 `json:"member_id"`
	IsAdmin    bool   `json:"is_admin"`
}

type MeCapabilities struct {
	Templates *TemplateCapabilities `json:"templates,omitempty"`
}

type TemplateCapabilities struct {
	CanCreate bool `json:"can_create"`
	CanUpdate bool `json:"can_update"`
	CanDelete bool `json:"can_delete"`
}
