package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/config"
)

// RefreshToolToken 调用 PowerX Auth 接口刷新 Tool Token。
func RefreshToolToken(ctx context.Context, cfg *config.Config) (string, string, error) {
	if cfg == nil || cfg.Gateway == nil {
		return "", "", fmt.Errorf("gateway config missing")
	}
	refreshToken := strings.TrimSpace(cfg.Gateway.RefreshToken)
	if refreshToken == "" {
		return "", "", fmt.Errorf("PX_TOOL_REFRESH_TOKEN 未配置")
	}

	base := strings.TrimSpace(cfg.Gateway.AuthBaseURL)
	if base == "" {
		base = cfg.Gateway.BaseURL
	}
	if base == "" {
		return "", "", fmt.Errorf("PX_GATEWAY_BASE_URL 未配置")
	}
	endpoint := strings.TrimRight(base, "/") + "/admin/user/auth/refresh"

	payload := map[string]string{"refresh_token": refreshToken}
	body := &bytes.Buffer{}
	if err := json.NewEncoder(body).Encode(payload); err != nil {
		return "", "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, body)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Content-Type", "application/json")

	httpClient := &http.Client{Timeout: 15 * time.Second}
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	var result struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
			TokenType    string `json:"token_type"`
			ExpiresIn    int64  `json:"expires_in"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", "", err
	}
	if resp.StatusCode >= 400 || result.Code >= 400 {
		return "", "", fmt.Errorf("refresh failed: status=%d code=%d message=%s", resp.StatusCode, result.Code, result.Message)
	}
	newToken := strings.TrimSpace(result.Data.AccessToken)
	if newToken == "" {
		return "", "", fmt.Errorf("refresh response missing access_token")
	}
	newRefresh := strings.TrimSpace(result.Data.RefreshToken)
	if newRefresh == "" {
		newRefresh = refreshToken
	}

	cfg.Gateway.ToolToken = newToken
	cfg.Gateway.RefreshToken = newRefresh
	os.Setenv("PX_TOOL_TOKEN", newToken)
	os.Setenv("PX_TOOL_REFRESH_TOKEN", newRefresh)

	return newToken, newRefresh, nil
}
