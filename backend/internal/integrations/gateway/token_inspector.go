package gateway

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"
)

// ParseTokenExpiry 尝试从 Tool Token 中解析过期时间（支持 JWT exp 或 expires_at 字段）。
func ParseTokenExpiry(token string) (time.Time, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return time.Time{}, errors.New("token is empty")
	}
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return time.Time{}, errors.New("token is not a JWT")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return time.Time{}, err
	}
	data := map[string]any{}
	if err := json.Unmarshal(payload, &data); err != nil {
		return time.Time{}, err
	}

	if exp, ok := parseNumericField(data["exp"]); ok {
		return exp, nil
	}
	if exp, ok := parseNumericField(data["expires"]); ok {
		return exp, nil
	}
	if exp, ok := parseStringTimeField(data["expires_at"]); ok {
		return exp, nil
	}
	if exp, ok := parseStringTimeField(data["expiresAt"]); ok {
		return exp, nil
	}
	return time.Time{}, errors.New("exp/expires_at not found")
}

func parseNumericField(value any) (time.Time, bool) {
	if value == nil {
		return time.Time{}, false
	}
	switch v := value.(type) {
	case float64:
		return unixToTime(int64(v)), true
	case json.Number:
		if n, err := v.Int64(); err == nil {
			return unixToTime(n), true
		}
	case string:
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			return unixToTime(n), true
		}
	}
	return time.Time{}, false
}

func parseStringTimeField(value any) (time.Time, bool) {
	if value == nil {
		return time.Time{}, false
	}
	switch v := value.(type) {
	case string:
		v = strings.TrimSpace(v)
		if v == "" {
			return time.Time{}, false
		}
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			return t.UTC(), true
		}
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			// treat >= 13 digits as milliseconds
			if len(v) >= 13 {
				return time.Unix(0, n*int64(time.Millisecond)).UTC(), true
			}
			return time.Unix(n, 0).UTC(), true
		}
	}
	return time.Time{}, false
}

func unixToTime(n int64) time.Time {
	if n <= 0 {
		return time.Time{}
	}
	return time.Unix(n, 0).UTC()
}
