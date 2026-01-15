package capability

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidate_Success(t *testing.T) {
	root := t.TempDir()
	setupContracts(t, root)

	pluginData := map[string]interface{}{
		"capabilities": map[string]interface{}{
			"provides": []interface{}{
				map[string]interface{}{
					"id":         "base.template.create",
					"version":    "1.0.0",
					"descriptor": "contracts/capabilities/base.template.create.yaml",
					"schemas": map[string]interface{}{
						"input":  []interface{}{"schema/input/base.template.create.v1.json"},
						"output": []interface{}{"schema/output/base.template.create.v1.json"},
					},
				},
			},
		},
		"rbac": map[string]interface{}{
			"resources": []interface{}{
				map[string]interface{}{
					"resource": "base:template",
					"actions":  []interface{}{"create"},
				},
			},
		},
	}

	manifestData := map[string]interface{}{
		"capabilities": map[string]interface{}{
			"provides": []interface{}{
				map[string]interface{}{
					"id":         "base.template.create",
					"descriptor": "contracts/capabilities/base.template.create.yaml",
				},
			},
		},
	}

	err := Validate(ValidateOptions{
		RootDir:      root,
		PluginData:   pluginData,
		ManifestData: manifestData,
	})
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
}

func TestValidate_MissingSchema(t *testing.T) {
	root := t.TempDir()
	setupContracts(t, root)

	// Remove one schema to trigger validation error.
	if err := os.Remove(filepath.Join(root, "contracts", "schema", "output", "base.template.create.v1.json")); err != nil {
		t.Fatalf("remove schema: %v", err)
	}

	pluginData := map[string]interface{}{
		"capabilities": map[string]interface{}{
			"provides": []interface{}{
				map[string]interface{}{
					"id":         "base.template.create",
					"descriptor": "contracts/capabilities/base.template.create.yaml",
					"schemas": map[string]interface{}{
						"input":  []interface{}{"schema/input/base.template.create.v1.json"},
						"output": []interface{}{"schema/output/base.template.create.v1.json"},
					},
				},
			},
		},
		"rbac": map[string]interface{}{
			"resources": []interface{}{
				map[string]interface{}{
					"resource": "base:template",
					"actions":  []interface{}{"create"},
				},
			},
		},
	}

	err := Validate(ValidateOptions{
		RootDir:    root,
		PluginData: pluginData,
	})
	if err == nil {
		t.Fatalf("expected error due to missing schema")
	}
	if msg := err.Error(); !strings.Contains(msg, "capability \"base.template.create\" output schema not found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidate_RBACMismatch(t *testing.T) {
	root := t.TempDir()
	setupContracts(t, root)

	pluginData := map[string]interface{}{
		"capabilities": map[string]interface{}{
			"provides": []interface{}{
				map[string]interface{}{
					"id":         "base.template.create",
					"descriptor": "contracts/capabilities/base.template.create.yaml",
				},
			},
		},
		"rbac": map[string]interface{}{
			"resources": []interface{}{
				map[string]interface{}{
					"resource": "base:template",
					"actions":  []interface{}{"read"},
				},
			},
		},
	}

	err := Validate(ValidateOptions{
		RootDir:    root,
		PluginData: pluginData,
	})
	if err == nil {
		t.Fatalf("expected RBAC parity error")
	}
	if msg := err.Error(); !strings.Contains(msg, "action \"create\" not present") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func setupContracts(t *testing.T, root string) {
	t.Helper()
	contractsDir := filepath.Join(root, "contracts")
	if err := os.MkdirAll(filepath.Join(contractsDir, "capabilities"), 0o755); err != nil {
		t.Fatalf("mkdir capabilities: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(contractsDir, "schema", "input"), 0o755); err != nil {
		t.Fatalf("mkdir schema input: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(contractsDir, "schema", "output"), 0o755); err != nil {
		t.Fatalf("mkdir schema output: %v", err)
	}

	descriptor := `id: base.template.create
type: Tool
version: 1.0.0
status: active
rbac:
  resource: base:template
  actions: [create]
provides:
  - id: schema/output/base.template.create.v1.json
    path: schema/output/base.template.create.v1.json
consumes:
  - id: schema/input/base.template.create.v1.json
    path: schema/input/base.template.create.v1.json
`
	if err := os.WriteFile(filepath.Join(contractsDir, "capabilities", "base.template.create.yaml"), []byte(descriptor), 0o644); err != nil {
		t.Fatalf("write descriptor: %v", err)
	}

	if err := os.WriteFile(filepath.Join(contractsDir, "schema", "input", "base.template.create.v1.json"), []byte("{}"), 0o644); err != nil {
		t.Fatalf("write input schema: %v", err)
	}
	if err := os.WriteFile(filepath.Join(contractsDir, "schema", "output", "base.template.create.v1.json"), []byte("{}"), 0o644); err != nil {
		t.Fatalf("write output schema: %v", err)
	}
}
