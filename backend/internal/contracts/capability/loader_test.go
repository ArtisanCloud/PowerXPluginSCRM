package capability

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadCatalog(t *testing.T) {
	dir := t.TempDir()

	yamlContent := `id: crm.contact.create
type: API
version: 1.0.0
status: active
description: Test capability
rbac:
  resource: crm:contact
  actions: [create]
provides:
  - id: schema/output/crm.contact.create.v1.json
    path: schema/output/crm.contact.create.v1.json
    kind: output
    version: 1.0.0
consumes:
  - id: schema/input/crm.contact.create.v1.json
    path: schema/input/crm.contact.create.v1.json
    kind: input
    version: 1.0.0
`

	path := filepath.Join(dir, "crm.contact.create.yaml")
	if err := os.WriteFile(path, []byte(yamlContent), 0o644); err != nil {
		t.Fatalf("write capability: %v", err)
	}

	catalog, err := LoadCatalog(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	record, ok := catalog.Get("crm.contact.create")
	if !ok {
		t.Fatalf("expected capability to be loaded")
	}

	if record.Source != path {
		t.Fatalf("expected source %s, got %s", path, record.Source)
	}

	if len(record.Descriptor.Provides) != 1 {
		t.Fatalf("expected one provides schema, got %d", len(record.Descriptor.Provides))
	}
	if len(record.Descriptor.Consumes) != 1 {
		t.Fatalf("expected one consumes schema, got %d", len(record.Descriptor.Consumes))
	}

	if record.Descriptor.Provides[0].Kind != "output" {
		t.Fatalf("expected output kind, got %s", record.Descriptor.Provides[0].Kind)
	}
	if record.Descriptor.Consumes[0].Kind != "input" {
		t.Fatalf("expected input kind, got %s", record.Descriptor.Consumes[0].Kind)
	}
}
