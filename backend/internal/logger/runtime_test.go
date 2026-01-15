package logger

import "testing"

func TestWithRuntimeFields(t *testing.T) {
	entry := WithRuntimeFields("plugin.demo", "tenant-1", "trace-abc", "component", Fields{"foo": "bar"})
	fields := entry.Data
	if fields["plugin_id"] != "plugin.demo" {
		t.Fatalf("plugin_id not set")
	}
	if fields["tenant_uuid"] != "tenant-1" {
		t.Fatalf("tenant_uuid not set")
	}
	if fields["trace_id"] != "trace-abc" {
		t.Fatalf("trace_id not set")
	}
	if fields["component"] != "component" {
		t.Fatalf("component not set")
	}
	if fields["foo"] != "bar" {
		t.Fatalf("extra field missing")
	}
}
