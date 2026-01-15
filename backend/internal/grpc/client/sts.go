package client

import (
	"context"
	"errors"
	"sync"
	"time"
)

// STSExchangeRequest/Response 与 PowerX STS 约定保持一致
type STSExchangeRequest struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	Audience     string `json:"audience"`
	Scope        string `json:"scope"`
	TTL          int32  `json:"ttl"`
}

type STSExchangeResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int32  `json:"expires_in"`
}

// TokenManager 负责在内存中维护 STS 短期令牌
type TokenManager struct {
	// static
	clientID     string
	clientSecret string
	audience     string
	scope        string
	ttl          time.Duration

	// runtime
	mu     sync.RWMutex
	token  string
	expiry time.Time

	exchange func(ctx context.Context, req *STSExchangeRequest) (*STSExchangeResponse, error)
}

func NewTokenManager(clientID, clientSecret, audience, scope string, ttl time.Duration, exchanger func(context.Context, *STSExchangeRequest) (*STSExchangeResponse, error)) *TokenManager {
	return &TokenManager{
		clientID:     clientID,
		clientSecret: clientSecret,
		audience:     audience,
		scope:        scope,
		ttl:          ttl,
		exchange:     exchanger,
	}
}

// GetToken 返回一个可用的 token；临期时自动刷新
func (m *TokenManager) GetToken(ctx context.Context) (string, error) {
	// fast path
	if tok, ok := m.peek(); ok {
		return tok, nil
	}
	tok, _, err := m.refresh(ctx)
	return tok, err
}

func (m *TokenManager) peek() (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.token == "" {
		return "", false
	}
	// 预留 60s 刷新窗口
	if time.Now().After(m.expiry.Add(-60 * time.Second)) {
		return "", false
	}
	return m.token, true
}

func (m *TokenManager) refresh(ctx context.Context) (string, time.Time, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	// double-check after acquiring lock
	if m.token != "" && time.Now().Before(m.expiry.Add(-60*time.Second)) {
		return m.token, m.expiry, nil
	}
	if m.exchange == nil {
		return "", time.Time{}, errors.New("sts exchanger not configured")
	}
	req := &STSExchangeRequest{
		ClientID:     m.clientID,
		ClientSecret: m.clientSecret,
		Audience:     m.audience,
		Scope:        m.scope,
		TTL:          int32(m.ttl / time.Second),
	}
	resp, err := m.exchange(ctx, req)
	if err != nil {
		return "", time.Time{}, err
	}
	if resp == nil || resp.AccessToken == "" {
		return "", time.Time{}, errors.New("empty access token from STS")
	}
	ttl := time.Duration(resp.ExpiresIn) * time.Second
	if ttl <= 0 {
		ttl = m.ttl
	}
	m.token = resp.AccessToken
	m.expiry = time.Now().Add(ttl)
	return m.token, m.expiry, nil
}

// Invalidate 使当前 token 失效，用于 401/403 兜底重试
func (m *TokenManager) Invalidate() {
	m.mu.Lock()
	m.token = ""
	m.expiry = time.Time{}
	m.mu.Unlock()
}

// HasValid 在不触发刷新情况下判断是否有可用 token
func (m *TokenManager) HasValid() bool {
	_, ok := m.peek()
	return ok
}

// ExchangeNow 立即向 STS 发起交换，返回 token 与过期时间
func (m *TokenManager) ExchangeNow(ctx context.Context) (string, time.Time, error) {
	return m.refresh(ctx)
}
