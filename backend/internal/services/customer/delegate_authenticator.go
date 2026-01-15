package customer

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/config"
	customerdomain "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/customer"
	"github.com/sirupsen/logrus"
)

var (
	ErrCustomerDelegateUnavailable = errors.New("customer delegate unavailable")
)

type DelegateAuthenticator struct {
	endpoint string
	timeout  time.Duration
	ttl      time.Duration
	client   *http.Client
	logger   *logrus.Entry

	now func() time.Time

	mu    sync.RWMutex
	cache map[string]delegateCacheEntry
}

type delegateCacheEntry struct {
	ctx *customerdomain.CustomerContext
	exp time.Time
}

func NewDelegateAuthenticator(cfg *config.Config, client *http.Client, logger *logrus.Entry) *DelegateAuthenticator {
	endpoint := ""
	timeout := 3 * time.Second
	ttl := time.Duration(0)

	if cfg != nil && cfg.CustomerAuth != nil {
		endpoint = strings.TrimSpace(cfg.CustomerAuth.DelegateEndpoint)
		if raw := strings.TrimSpace(cfg.CustomerAuth.DelegateTimeout); raw != "" {
			if d, err := time.ParseDuration(raw); err == nil && d > 0 {
				timeout = d
			}
		}
		if cfg.CustomerAuth.CacheTTLSeconds > 0 {
			ttl = time.Duration(cfg.CustomerAuth.CacheTTLSeconds) * time.Second
		}
	}

	if client == nil {
		client = &http.Client{Timeout: timeout + 500*time.Millisecond}
	}
	if logger == nil {
		logger = logrus.WithField("component", "customer.delegate_authenticator")
	}

	return &DelegateAuthenticator{
		endpoint: endpoint,
		timeout:  timeout,
		ttl:      ttl,
		client:   client,
		logger:   logger,
		now:      time.Now,
		cache:    map[string]delegateCacheEntry{},
	}
}

// NowForTest overrides time source for deterministic tests.
func (a *DelegateAuthenticator) NowForTest(now func() time.Time) {
	if a == nil || now == nil {
		return
	}
	a.now = now
}

func (a *DelegateAuthenticator) Authenticate(ctx context.Context, requestTenantUUID string, token string) (*customerdomain.CustomerContext, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, ErrCustomerTokenInvalid
	}
	if strings.TrimSpace(a.endpoint) == "" {
		return nil, ErrCustomerAuthNotImplemented
	}

	requestTenantUUID = strings.ToLower(strings.TrimSpace(requestTenantUUID))
	now := a.now()
	if a.ttl > 0 {
		if cached, ok := a.getCached(token, now); ok {
			return cached, nil
		}
	}

	cc, exp, err := a.validate(ctx, requestTenantUUID, token)
	if err != nil {
		return nil, err
	}
	if cc == nil {
		return nil, ErrCustomerTokenInvalid
	}

	if a.ttl > 0 {
		a.setCached(token, cc, exp, now)
	}
	return cc, nil
}

func (a *DelegateAuthenticator) getCached(token string, now time.Time) (*customerdomain.CustomerContext, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	entry, ok := a.cache[token]
	if !ok || entry.ctx == nil {
		return nil, false
	}
	if !entry.exp.IsZero() && now.After(entry.exp) {
		return nil, false
	}
	return entry.ctx, true
}

func (a *DelegateAuthenticator) setCached(token string, cc *customerdomain.CustomerContext, tokenExp time.Time, now time.Time) {
	exp := now.Add(a.ttl)
	if !tokenExp.IsZero() && tokenExp.Before(exp) {
		exp = tokenExp
	}
	a.mu.Lock()
	a.cache[token] = delegateCacheEntry{ctx: cc, exp: exp}
	a.mu.Unlock()
}

func (a *DelegateAuthenticator) validate(ctx context.Context, requestTenantUUID string, token string) (*customerdomain.CustomerContext, time.Time, error) {
	reqBody := map[string]any{
		"token":       token,
		"tenant_uuid": requestTenantUUID,
	}
	payload, _ := json.Marshal(reqBody)

	doOnce := func() (*http.Response, []byte, error) {
		ctx, cancel := context.WithTimeout(ctx, a.timeout)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.endpoint, bytes.NewReader(payload))
		if err != nil {
			return nil, nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)
		if requestTenantUUID != "" {
			req.Header.Set("X-Tenant-UUID", requestTenantUUID)
		}

		resp, err := a.client.Do(req)
		if err != nil {
			return nil, nil, err
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		return resp, body, nil
	}

	// minimal retry: network errors / 5xx
	var lastErr error
	var resp *http.Response
	var body []byte
	for attempt := 0; attempt < 2; attempt++ {
		r, b, err := doOnce()
		if err != nil {
			lastErr = err
		} else {
			resp, body = r, b
			if resp.StatusCode < 500 {
				lastErr = nil
				break
			}
			lastErr = errors.New("upstream 5xx")
		}
		if attempt == 0 {
			select {
			case <-time.After(60 * time.Millisecond):
			case <-ctx.Done():
				break
			}
		}
	}
	if lastErr != nil && resp == nil {
		return nil, time.Time{}, ErrCustomerDelegateUnavailable
	}
	if resp == nil {
		return nil, time.Time{}, ErrCustomerDelegateUnavailable
	}

	switch resp.StatusCode {
	case http.StatusOK:
		cc, exp, err := parseDelegateResponse(body, a.now())
		if err != nil {
			a.logger.WithError(err).WithField("status", resp.StatusCode).Debug("delegate response parse failed")
			return nil, time.Time{}, ErrCustomerTokenInvalid
		}
		cc.SourceMode = customerdomain.CustomerAuthModeDelegate
		cc.Authenticated = true
		return cc, exp, nil
	case http.StatusUnauthorized, http.StatusForbidden:
		return nil, time.Time{}, ErrCustomerTokenInvalid
	default:
		a.logger.WithFields(logrus.Fields{
			"status":  resp.StatusCode,
			"payload": string(body),
		}).Warn("delegate validation failed")
		return nil, time.Time{}, ErrCustomerDelegateUnavailable
	}
}

func parseDelegateResponse(body []byte, now time.Time) (*customerdomain.CustomerContext, time.Time, error) {
	if len(body) == 0 {
		return nil, time.Time{}, errors.New("empty body")
	}

	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, time.Time{}, err
	}

	// Support both envelope and raw payload.
	payload := raw
	if v, ok := raw["success"]; ok {
		success, _ := v.(bool)
		if !success {
			return nil, time.Time{}, errors.New("upstream not success")
		}
		if data, ok := raw["data"].(map[string]any); ok {
			payload = data
		} else {
			return nil, time.Time{}, errors.New("missing data")
		}
	}

	tenantUUID, _ := payload["tenant_uuid"].(string)
	customerUUID, _ := payload["customer_uuid"].(string)
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	customerUUID = strings.ToLower(strings.TrimSpace(customerUUID))

	roles := []string{}
	if rv, ok := payload["roles"]; ok {
		switch v := rv.(type) {
		case []any:
			for _, it := range v {
				if s, ok := it.(string); ok && strings.TrimSpace(s) != "" {
					roles = append(roles, strings.TrimSpace(s))
				}
			}
		case []string:
			for _, s := range v {
				if strings.TrimSpace(s) != "" {
					roles = append(roles, strings.TrimSpace(s))
				}
			}
		}
	}

	exp := parseExpiry(payload, now)
	return &customerdomain.CustomerContext{
		TenantUUID:   tenantUUID,
		CustomerUUID: customerUUID,
		Roles:        roles,
		Attributes:   payload,
		RawClaims:    payload,
	}, exp, nil
}

func parseExpiry(payload map[string]any, now time.Time) time.Time {
	// Accept exp (unix seconds), expires_at (RFC3339), expires_in (seconds)
	if v, ok := payload["exp"]; ok {
		switch n := v.(type) {
		case float64:
			if n > 0 {
				return time.Unix(int64(n), 0)
			}
		case int64:
			if n > 0 {
				return time.Unix(n, 0)
			}
		case json.Number:
			if x, err := n.Int64(); err == nil && x > 0 {
				return time.Unix(x, 0)
			}
		}
	}
	if v, ok := payload["expires_at"].(string); ok {
		if t, err := time.Parse(time.RFC3339, strings.TrimSpace(v)); err == nil {
			return t
		}
	}
	if v, ok := payload["expires_in"]; ok {
		switch n := v.(type) {
		case float64:
			if n > 0 {
				return now.Add(time.Duration(int64(n)) * time.Second)
			}
		case int64:
			if n > 0 {
				return now.Add(time.Duration(n) * time.Second)
			}
		case json.Number:
			if x, err := n.Int64(); err == nil && x > 0 {
				return now.Add(time.Duration(x) * time.Second)
			}
		}
	}
	return time.Time{}
}
