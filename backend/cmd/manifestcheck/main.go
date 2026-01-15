package main

import (
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/contracts/capability"
	yaml "gopkg.in/yaml.v3"
)

func main() {
	pluginPath := flag.String("plugin", "plugin.yaml", "Path to plugin.yaml")
	manifestPath := flag.String("manifest", "docs/lifecycle/examples/manifest.yaml", "Path to manifest.yaml")
	schemaPath := flag.String("schema", "docs/lifecycle/contracts/manifest.schema.json", "Path to manifest JSON schema (for documentation reference)")
	capabilitiesOnly := flag.Bool("capabilities-only", false, "Only run capability validation (skip manifest shape checks)")
	flag.Parse()

	if err := run(*pluginPath, *manifestPath, *schemaPath, *capabilitiesOnly); err != nil {
		fmt.Fprintf(os.Stderr, "manifestcheck: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("manifestcheck: plugin and manifest metadata validated successfully")
}

func run(pluginPath, manifestPath, schemaPath string, capabilitiesOnly bool) error {
	pluginMap, err := loadYAMLFile(pluginPath)
	if err != nil {
		return fmt.Errorf("load plugin: %w", err)
	}

	var manifestMap map[string]interface{}
	if manifestPath != "" {
		manifestMap, err = loadYAMLFile(manifestPath)
		if err != nil {
			return fmt.Errorf("load manifest: %w", err)
		}
	}

	if !capabilitiesOnly {
		if manifestMap == nil {
			return errors.New("manifest path must be provided when capabilities-only is false")
		}
		if err := validateManifestShape(manifestMap); err != nil {
			return err
		}

		if err := ensureSchemaExists(schemaPath); err != nil {
			return err
		}

		if err := comparePluginAndManifest(pluginMap, manifestMap); err != nil {
			return err
		}
	}

	root, err := filepath.Abs(filepath.Dir(pluginPath))
	if err != nil {
		return fmt.Errorf("resolve plugin root: %w", err)
	}

	if err := capability.Validate(capability.ValidateOptions{
		RootDir:      root,
		PluginData:   pluginMap,
		ManifestData: manifestMap,
	}); err != nil {
		return err
	}

	return nil
}

func ensureSchemaExists(path string) error {
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("schema file not found: %s", path)
		}
		return fmt.Errorf("schema file error: %w", err)
	}
	return nil
}

func loadYAMLFile(path string) (map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var out map[string]interface{}
	if err := yaml.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("decode yaml: %w", err)
	}
	return out, nil
}

func lookupString(m map[string]interface{}, path ...string) (string, error) {
	v, err := lookup(m, path...)
	if err != nil {
		return "", err
	}
	switch val := v.(type) {
	case string:
		if strings.TrimSpace(val) == "" {
			return "", fmt.Errorf("%s is empty", strings.Join(path, "."))
		}
		return val, nil
	case fmt.Stringer:
		return val.String(), nil
	default:
		if val == nil {
			return "", fmt.Errorf("%s is nil", strings.Join(path, "."))
		}
		// numbers from YAML may become int/float; convert to string
		return fmt.Sprintf("%v", val), nil
	}
}

func lookupArray(m map[string]interface{}, path ...string) ([]interface{}, error) {
	v, err := lookup(m, path...)
	if err != nil {
		return nil, err
	}
	arr, ok := v.([]interface{})
	if !ok {
		return nil, fmt.Errorf("%s is not an array", strings.Join(path, "."))
	}
	if len(arr) == 0 {
		return nil, fmt.Errorf("%s must contain at least one item", strings.Join(path, "."))
	}
	return arr, nil
}

func lookup(m map[string]interface{}, path ...string) (interface{}, error) {
	current := interface{}(m)
	for _, key := range path {
		obj, ok := current.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("%s is not an object", strings.Join(path, "."))
		}
		next, exists := obj[key]
		if !exists {
			return nil, fmt.Errorf("missing required field: %s", strings.Join(path, "."))
		}
		current = next
	}
	return current, nil
}

func validateManifestShape(manifest map[string]interface{}) error {
	requiredStrings := [][]string{
		{"id"},
		{"name"},
		{"version"},
		{"channel"},
		{"min_core"},
		{"runtime", "backend", "entrypoint"},
		{"runtime", "backend", "sha256"},
		{"lifecycle", "status"},
		{"signature", "algorithm"},
		{"signature", "signed_by"},
		{"build", "commit"},
		{"build", "builder"},
		{"build", "timestamp"},
	}
	for _, p := range requiredStrings {
		if _, err := lookupString(manifest, p...); err != nil {
			return err
		}
	}

	channel, _ := lookupString(manifest, "channel")
	switch channel {
	case "stable", "beta", "alpha", "dev":
	default:
		return fmt.Errorf("channel must be one of stable|beta|alpha|dev, got %q", channel)
	}

	frontends, err := lookupArray(manifest, "frontends")
	if err != nil {
		return err
	}
	for i, item := range frontends {
		obj, ok := item.(map[string]interface{})
		if !ok {
			return fmt.Errorf("frontends[%d] is not an object", i)
		}
		if _, err := lookupString(obj, "name"); err != nil {
			return err
		}
		if _, err := lookupString(obj, "path"); err != nil {
			return err
		}
		if _, err := lookupString(obj, "sha256"); err != nil {
			return err
		}
	}

	hashes, err := lookupArray(manifest, "signature", "hashes")
	if err != nil {
		return err
	}
	for i, item := range hashes {
		obj, ok := item.(map[string]interface{})
		if !ok {
			return fmt.Errorf("signature.hashes[%d] is not an object", i)
		}
		if _, err := lookupString(obj, "path"); err != nil {
			return err
		}
		if _, err := lookupString(obj, "sha256"); err != nil {
			return err
		}
	}

	if _, err := lookupArray(manifest, "migrations"); err != nil {
		return err
	}
	if _, err := lookupArray(manifest, "rbac"); err != nil {
		return err
	}

	return nil
}

func comparePluginAndManifest(plugin, manifest map[string]interface{}) error {
	pluginID, err := lookupString(plugin, "id")
	if err != nil {
		return err
	}
	manifestID, err := lookupString(manifest, "id")
	if err != nil {
		return err
	}
	if pluginID != manifestID {
		return fmt.Errorf("id mismatch: plugin=%s manifest=%s", pluginID, manifestID)
	}

	pluginName, err := lookupString(plugin, "name")
	if err != nil {
		return err
	}
	manifestName, err := lookupString(manifest, "name")
	if err != nil {
		return err
	}
	if pluginName != manifestName {
		return fmt.Errorf("name mismatch: plugin=%s manifest=%s", pluginName, manifestName)
	}

	pluginVersion, err := lookupString(plugin, "version")
	if err != nil {
		return err
	}
	manifestVersion, err := lookupString(manifest, "version")
	if err != nil {
		return err
	}
	if pluginVersion != manifestVersion {
		return fmt.Errorf("version mismatch: plugin=%s manifest=%s", pluginVersion, manifestVersion)
	}

	pluginRuntimeEntry, err := lookupString(plugin, "runtime", "entry")
	if err == nil {
		manifestEntry, err := lookupString(manifest, "runtime", "backend", "entrypoint")
		if err != nil {
			return err
		}
		if pluginRuntimeEntry != manifestEntry {
			return fmt.Errorf("runtime entry mismatch: plugin=%s manifest=%s", pluginRuntimeEntry, manifestEntry)
		}
	}

	// optional: ensure manifest references plugin.yaml for parity
	manifestContracts, err := lookup(manifest, "contracts")
	if err != nil {
		return err
	}
	if _, ok := manifestContracts.(map[string]interface{}); !ok {
		return fmt.Errorf("contracts must be an object")
	}

	return nil
}
