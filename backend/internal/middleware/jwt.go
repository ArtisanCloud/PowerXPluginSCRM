package middleware

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type JWTAuthConfig struct {
	Issuer           string   `yaml:"issuer" json:"issuer"`
	AcceptAudiences  []string `yaml:"accept_audiences" json:"accept_audiences"`
	HMACSecret       string   `yaml:"hmac_secret" json:"hmac_secret"`
	ClockSkewSeconds int      `yaml:"clock_skew_seconds" json:"clock_skew_seconds"`
	Optional         bool     `yaml:"optional" json:"optional"`
}

type PowerXClaims struct {
	TenantUUID    TenantClaim `json:"tid"`
	UserID        Int64Claim  `json:"uid"`
	UserUUID      string      `json:"user_uuid,omitempty"`
	ActorUUID     string      `json:"actor_uuid,omitempty"`
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

type Int64Claim int64

func (i *Int64Claim) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || bytes.Equal(data, []byte("null")) {
		*i = 0
		return nil
	}
	if data[0] == '"' {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
		parsed, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
		if err != nil {
			return err
		}
		*i = Int64Claim(parsed)
		return nil
	}
	var num json.Number
	if err := json.Unmarshal(data, &num); err != nil {
		return err
	}
	parsed, err := strconv.ParseInt(strings.TrimSpace(num.String()), 10, 64)
	if err != nil {
		return err
	}
	*i = Int64Claim(parsed)
	return nil
}

func (i Int64Claim) Int64() int64 {
	return int64(i)
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
	return TenantContext{}, "", false
}

func parseHS256(raw string, cfg JWTAuthConfig) (TenantContext, error) {
	leeway := time.Duration(cfg.ClockSkewSeconds)
	if leeway <= 0 {
		leeway = 60
	}
	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(raw, claims, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, errors.New("unexpected sign method")
		}
		return []byte(cfg.HMACSecret), nil
	}, jwt.WithIssuer(cfg.Issuer), jwt.WithAudience(cfg.AcceptAudiences...), jwt.WithLeeway(leeway*time.Second))
	if err != nil || token == nil || !token.Valid {
		return TenantContext{}, errors.New("invalid token")
	}
	tenantUUID, userID, userUUID, roles, permissions, policyVersion, pluginID := normalizeClaims(nil, claims)
	if tenantUUID == "" {
		return TenantContext{}, errors.New("tenant claim missing")
	}
	return TenantContext{
		TenantUUID:    tenantUUID,
		UserID:        userID,
		UserUUID:      userUUID,
		Roles:         roles,
		Permissions:   permissions,
		PolicyVersion: policyVersion,
		PluginID:      pluginID,
	}, nil
}

func normalizeClaims(claims *PowerXClaims, raw jwt.Claims) (tenantUUID string, userID int64, userUUID string, roles []string, permissions []string, policyVersion string, pluginID string) {
	if claims != nil {
		tenantUUID = strings.TrimSpace(claims.TenantUUID.String())
		userID = claims.UserID.Int64()
		userUUID = firstValidUUID(strings.TrimSpace(claims.UserUUID), strings.TrimSpace(claims.ActorUUID))
		roles = claims.Roles
		permissions = claims.Permissions
		policyVersion = strings.TrimSpace(claims.PolicyVersion)
		pluginID = strings.TrimSpace(claims.PluginID)
	}
	mapClaims, ok := raw.(jwt.MapClaims)
	if !ok {
		return
	}
	tenantUUID = firstNonEmpty(tenantUUID, claimString(mapClaims, "tid", "tenant_uuid", "tenantUuid", "tenant_id", "tenantId"))
	userID = firstNonZeroInt64(userID, claimInt64(mapClaims, "uid", "user_id", "userId", "member_id", "memberId"))
	userUUID = firstNonEmpty(userUUID, firstValidUUID(
		claimString(mapClaims, "actor_uuid", "actorUserUUID", "actor_user_uuid"),
		claimString(mapClaims, "user_uuid", "userUuid"),
		claimString(mapClaims, "member_uuid", "memberUuid"),
		claimString(mapClaims, "sub"),
	))
	roles = firstNonEmptySlice(roles, claimStringSlice(mapClaims, "roles", "role_codes"))
	permissions = firstNonEmptySlice(permissions, claimStringSlice(mapClaims, "perms", "permissions", "permission_codes"))
	policyVersion = firstNonEmpty(policyVersion, claimString(mapClaims, "policy_version", "policyVersion"))
	pluginID = firstNonEmpty(pluginID, claimString(mapClaims, "plugin_id", "pluginId"))
	return
}

func claimString(claims jwt.MapClaims, keys ...string) string {
	for _, key := range keys {
		value, ok := claims[key]
		if !ok || value == nil {
			continue
		}
		switch v := value.(type) {
		case string:
			if strings.TrimSpace(v) != "" {
				return strings.TrimSpace(v)
			}
		case fmt.Stringer:
			if strings.TrimSpace(v.String()) != "" {
				return strings.TrimSpace(v.String())
			}
		case json.Number:
			if strings.TrimSpace(v.String()) != "" {
				return strings.TrimSpace(v.String())
			}
		case float64:
			if v != 0 {
				return strconv.FormatInt(int64(v), 10)
			}
		}
	}
	return ""
}

func claimInt64(claims jwt.MapClaims, keys ...string) int64 {
	for _, key := range keys {
		value, ok := claims[key]
		if !ok || value == nil {
			continue
		}
		switch v := value.(type) {
		case int64:
			if v != 0 {
				return v
			}
		case int:
			if v != 0 {
				return int64(v)
			}
		case float64:
			if v != 0 {
				return int64(v)
			}
		case json.Number:
			if parsed, err := strconv.ParseInt(strings.TrimSpace(v.String()), 10, 64); err == nil && parsed != 0 {
				return parsed
			}
		case string:
			if parsed, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64); err == nil && parsed != 0 {
				return parsed
			}
		}
	}
	return 0
}

func claimStringSlice(claims jwt.MapClaims, keys ...string) []string {
	for _, key := range keys {
		value, ok := claims[key]
		if !ok || value == nil {
			continue
		}
		out := make([]string, 0)
		switch v := value.(type) {
		case []string:
			out = append(out, v...)
		case []any:
			for _, item := range v {
				if text := strings.TrimSpace(fmt.Sprint(item)); text != "" {
					out = append(out, text)
				}
			}
		case string:
			for _, item := range strings.Split(v, ",") {
				if text := strings.TrimSpace(item); text != "" {
					out = append(out, text)
				}
			}
		}
		if len(out) > 0 {
			return out
		}
	}
	return nil
}

func firstNonZeroInt64(values ...int64) int64 {
	for _, value := range values {
		if value != 0 {
			return value
		}
	}
	return 0
}

func firstNonEmptySlice(values ...[]string) []string {
	for _, value := range values {
		if len(value) > 0 {
			return value
		}
	}
	return nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func firstValidUUID(values ...string) string {
	for _, value := range values {
		parsed, err := uuid.Parse(strings.TrimSpace(value))
		if err == nil && parsed != uuid.Nil {
			return strings.ToLower(parsed.String())
		}
	}
	return ""
}
