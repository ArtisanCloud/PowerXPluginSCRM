package tool_grant_verifier

import (
	"context"
	"net/http"
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/config"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/middleware"
	toolgrantservice "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/agent/tool_grant"
	"github.com/gin-gonic/gin"
)

func BenchmarkMiddleware(b *testing.B) {
	gin.SetMode(gin.TestMode)
	cfg := &config.Config{Security: &config.SecurityConfig{ToolGrantSecret: "bench-secret"}}
	svc := toolgrantservice.NewService(nil, cfg, nil, []byte("bench-secret"))
	handler := Middleware(svc, func(c *gin.Context) string { return c.GetHeader("X-ToolGrant") })
	c, _ := gin.CreateTestContext(nil)
	token, _ := svc.Issue(context.Background(), "tenant", "tool", "agent", nil, "test")
	req, _ := http.NewRequest("GET", "/", nil)
	req = req.WithContext(middleware.ContextWithTenantUUID(context.Background(), "tenant-bench-uuid"))
	req.Header.Set("X-ToolGrant", token)
	c.Request = req

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		handler(c)
	}
}
