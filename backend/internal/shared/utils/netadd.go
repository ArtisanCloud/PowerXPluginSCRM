package utils

import (
	"os"
	"strings"
)

// ★ 解析 host-values.yml 里的 "__POWERX_DYNAMIC_PORT__" 占位符：
//   - 优先从 portEnv（HTTP 用 PORT，gRPC 用 POWERX_GRPC_PORT）取端口
//   - 如未设置则回退到 PORT，再不行用 "0"（让内核分配）
func ResolveDynamicAddr(raw, portEnv string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return s
	}
	if !strings.Contains(s, "__POWERX_DYNAMIC_PORT__") {
		return s
	}
	port := strings.TrimSpace(os.Getenv(portEnv))
	if port == "" {
		port = strings.TrimSpace(os.Getenv("PORT"))
	}
	if port == "" {
		port = "0"
	}
	if s == "__POWERX_DYNAMIC_PORT__" || s == ":__POWERX_DYNAMIC_PORT__" {
		return ":" + port
	}
	// 例如 "127.0.0.1:__POWERX_DYNAMIC_PORT__"
	return strings.ReplaceAll(s, "__POWERX_DYNAMIC_PORT__", port)
}
