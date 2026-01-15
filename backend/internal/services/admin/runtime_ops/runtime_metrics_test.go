package runtime_ops

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestMetricsHTTPHandler_ExposesCoreSeries(t *testing.T) {
	resetMetrics()

	IncRequest("plugin.demo", "bootstrap", 120*time.Millisecond, nil)
	SetQuotaUsage("plugin.demo", "tenant", "tenant-1", 0.75)
	AddCost("plugin.demo", "tenant-1", 3.5)
	SetMCPSessions("plugin.demo", 2)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/metrics", nil)
	MetricsHTTPHandler().ServeHTTP(rec, req)

	if got := rec.Header().Get("Content-Type"); !strings.Contains(got, "text/plain") {
		t.Fatalf("expected text/plain content type, got %q", got)
	}

	body := rec.Body.String()

	expectedSnippets := []string{
		`powerx_plugin_request_total{capability="bootstrap",plugin_id="plugin.demo"} 1`,
		`powerx_plugin_quota_usage{plugin_id="plugin.demo",scope="tenant",scope_ref="tenant-1"} 0.75`,
		`powerx_plugin_cost_total{plugin_id="plugin.demo",tenant_uuid="tenant-1"} 3.5`,
		`powerx_mcp_sessions_total{plugin_id="plugin.demo"} 2`,
	}

	for _, snippet := range expectedSnippets {
		if !strings.Contains(body, snippet) {
			t.Fatalf("expected metrics output to contain %q\nbody:\n%s", snippet, body)
		}
	}
}
