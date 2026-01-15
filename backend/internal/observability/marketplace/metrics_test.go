package marketplace

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func resetMetrics() {
	mtx.Lock()
	defer mtx.Unlock()
	histograms = map[string]map[string][]float64{
		metricLicenseVerifyLatency: {},
	}
	counters = map[string]map[string]float64{
		metricUsageIngestTotal:  {},
		metricTaxProviderErrors: {},
	}
}

func TestRenderMetricsOutputsSamples(t *testing.T) {
	resetMetrics()

	ObserveLicenseVerification("success", "stripe_tax", "tenantA", 250*time.Millisecond)
	RecordUsageIngest("accepted", "tenantA", 3)
	IncrementTaxProviderError("stripe_tax", "429")

	var buf bytes.Buffer
	RenderMetrics(&buf)
	out := buf.String()

	if !strings.Contains(out, metricLicenseVerifyLatency) {
		t.Fatalf("expected license verify histogram in output, got %q", out)
	}
	if !strings.Contains(out, metricUsageIngestTotal) {
		t.Fatalf("expected usage ingest counter in output, got %q", out)
	}
	if !strings.Contains(out, metricTaxProviderErrors) {
		t.Fatalf("expected tax provider error counter in output, got %q", out)
	}
}
