package middleware

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type JWTAuthConfig struct {
	Issuer             string   `yaml:"issuer" json:"issuer"`
	AcceptAudiences    []string `yaml:"accept_audiences" json:"accept_audiences"`
	HMACSecret         string   `yaml:"hmac_secret" json:"hmac_secret"`
	ClockSkewSeconds   int      `yaml:"clock_skew_seconds" json:"clock_skew_seconds"`
	Optional           bool     `yaml:"optional" json:"optional"`
	AllowSignedContext bool     `yaml:"allow_signed_context" json:"allow_signed_context"`
	ContextHMACSecret  string   `yaml:"context_hmac_secret" json:"context_hmac_secret"`
	MaxCtxAgeSeconds   int64    `yaml:"max_ctx_age_seconds" json:"max_ctx_age_seconds"`
}

type PowerXClaims struct {
	TenantUUID    TenantClaim `json:"tid"`
	UserID        int64       `json:"uid"`
	Roles         []string    `json:"roles"`
	Permissions   []string    `json:"perms"`
	PolicyVersion string      `json:"policy_version"`
	PluginID      string      `json:"plugin_id,omitempty"`
	jwt.RegisteredClaims
}

type TenantClaim string

func (t *TenantClaim) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || bytes.Equal(data, []byte("null")) {
		*t = ""
		return nil
	}
	if data[0] == '"' {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
		*t = TenantClaim(strings.TrimSpace(s))
		return nil
	}
	var num json.Number
	if err := json.Unmarshal(data, &num); err != nil {
		return err
	}
	*t = TenantClaim(strings.TrimSpace(num.String()))
	return nil
}

func (t TenantClaim) String() string {
	return string(t)
}

func ParseFromHeaders(h func(string) string, cfg JWTAuthConfig) (tc TenantContext, rawBearer string, ok bool) {
	// 1) Authorization: Bearer
	authz := h("Authorization")
	if strings.HasPrefix(strings.ToLower(authz), "bearer ") {
		raw := strings.TrimSpace(authz[7:])
		if raw != "" && cfg.HMACSecret != "" {
			if t, err := parseHS256(raw, cfg); err == nil {
				return t, raw, true
			}
		}
	}
	// 2) 回退 Signed-Context
	if cfg.AllowSignedContext && cfg.ContextHMACSecret != "" {
		if t, ok := tryLoadSignedContext(h, cfg.ContextHMACSecret, cfg.MaxCtxAgeSeconds); ok {
			return t, "", true
		}
	}
	return TenantContext{}, "", false
}

func parseHS256(raw string, cfg JWTAuthConfig) (TenantContext, error) {
	leeway := time.Duration(cfg.ClockSkewSeconds)
	if leeway <= 0 {
		leeway = 60
	}
	claims := &PowerXClaims{}
	token, err := jwt.ParseWithClaims(raw, claims, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, errors.New("unexpected sign method")
		}
		return []byte(cfg.HMACSecret), nil
	}, jwt.WithIssuer(cfg.Issuer), jwt.WithAudience(cfg.AcceptAudiences...), jwt.WithLeeway(leeway*time.Second))
	if err != nil || token == nil || !token.Valid {
		return TenantContext{}, errors.New("invalid token")
	}
	return TenantContext{
		TenantUUID:    strings.TrimSpace(claims.TenantUUID.String()),
		UserID:        claims.UserID,
		Roles:         claims.Roles,
		Permissions:   claims.Permissions,
		PolicyVersion: claims.PolicyVersion,
		PluginID:      strings.TrimSpace(claims.PluginID),
	}, nil
}

type signedCtx struct {
	TenantUUID    string   `json:"tid"`
	UserID        int64    `json:"uid"`
	Roles         []string `json:"roles"`
	Permissions   []string `json:"perms"`
	PolicyVersion string   `json:"policy_version"`
	PluginID      string   `json:"plugin_id,omitempty"`
	TS            int64    `json:"ts"`
}

func tryLoadSignedContext(h func(string) string, secret string, maxAgeSec int64) (TenantContext, bool) {
	ctxB64 := h("X-PowerX-CTX")
	if ctxB4 := ctxB64; ctxB4 == "" {
		return TenantContext{}, false
	}
	sigHex := h("X-PowerX-CTX-SIG")
	if sigHex == "" {
		return TenantContext{}, false
	}
	raw, err := base64.StdEncoding.DecodeString(ctxB64)
	if err != nil {
		return TenantContext{}, false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(ctxB64))
	if !hmac.Equal([]byte(hex.EncodeToString(mac.Sum(nil))), []byte(sigHex)) {
		return TenantContext{}, false
	}
	var sc signedCtx
	if err := json.Unmarshal(raw, &sc); err != nil {
		return TenantContext{}, false
	}
	if maxAgeSec > 0 && (time.Now().Unix()-sc.TS) > maxAgeSec {
		return TenantContext{}, false
	}
	return TenantContext{TenantUUID: strings.TrimSpace(sc.TenantUUID), UserID: sc.UserID, Roles: sc.Roles,
		Permissions: sc.Permissions, PolicyVersion: sc.PolicyVersion, PluginID: strings.TrimSpace(sc.PluginID)}, true
}

// 供客户端出站兜底：把 TenantContext 签成 X-PowerX-CTX / SIG
func SignContext(tc TenantContext, secret string) (ctxB64, sigHex string, ts int64, err error) {
	sc := signedCtx{TenantUUID: strings.TrimSpace(tc.TenantUUID), UserID: tc.UserID, Roles: tc.Roles,
		Permissions: tc.Permissions, PolicyVersion: tc.PolicyVersion, PluginID: tc.PluginID, TS: time.Now().Unix()}
	b, e := json.Marshal(&sc)
	if e != nil {
		return "", "", 0, e
	}
	ctxB64 = base64.StdEncoding.EncodeToString(b)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(ctxB64))
	sigHex = hex.EncodeToString(mac.Sum(nil))
	return ctxB64, sigHex, sc.TS, nil
}
