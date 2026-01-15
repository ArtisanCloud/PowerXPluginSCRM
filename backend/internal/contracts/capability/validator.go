package capability

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ValidateOptions controls capability validation behaviour.
type ValidateOptions struct {
	RootDir      string
	PluginData   map[string]interface{}
	ManifestData map[string]interface{}
}

// Validate verifies capability descriptors, schema references, and RBAC parity.
func Validate(opts ValidateOptions) error {
	if opts.RootDir == "" {
		return fmt.Errorf("validate: root directory is required")
	}
	contractsDir := filepath.Join(opts.RootDir, "contracts")
	capabilitiesDir := filepath.Join(contractsDir, "capabilities")

	catalog, err := LoadCatalog(capabilitiesDir)
	if err != nil {
		return err
	}

	issues := newCollector()

	provides := extractCapabilityRefs(opts.PluginData, issues, "plugin")
	if len(provides) == 0 {
		issues.add("plugin.capabilities.provides must declare at least one capability")
	}

	manifestRefs := extractCapabilityRefs(opts.ManifestData, issues, "manifest")
	rbacMap := extractRBACResources(opts.PluginData, issues)

	seenIDs := make(map[string]struct{})

	for _, ref := range provides {
		if _, exists := seenIDs[ref.ID]; exists {
			issues.add(fmt.Sprintf("plugin.capabilities.provides contains duplicate capability id %q", ref.ID))
			continue
		}
		seenIDs[ref.ID] = struct{}{}

		record, ok := catalog.Get(ref.ID)
		if !ok {
			issues.add(fmt.Sprintf("capability descriptor for id %q not found under %s", ref.ID, capabilitiesDir))
			continue
		}

		expectedDescriptor := filepath.Clean(filepath.Join(opts.RootDir, ref.Descriptor))
		if expectedDescriptor != filepath.Clean(record.Source) {
			issues.add(fmt.Sprintf("capability %q descriptor mismatch: plugin points to %s but loaded file is %s", ref.ID, expectedDescriptor, record.Source))
		}

		if ref.Version != "" && ref.Version != record.Descriptor.Version {
			issues.add(fmt.Sprintf("capability %q version mismatch: plugin=%s descriptor=%s", ref.ID, ref.Version, record.Descriptor.Version))
		}

		ensureSchemaFiles(ref, contractsDir, issues)
		ensureDescriptorSchemas(record, contractsDir, issues)
		verifyRBAC(record.Descriptor, rbacMap, issues)
	}

	if len(manifestRefs) > 0 {
		// ensure manifest covers the same capability IDs.
		pluginIDs := make(map[string]struct{}, len(provides))
		for _, ref := range provides {
			pluginIDs[ref.ID] = struct{}{}
		}
		for _, ref := range manifestRefs {
			if _, exists := pluginIDs[ref.ID]; !exists {
				issues.add(fmt.Sprintf("manifest.capabilities.provides contains capability %q not declared in plugin.yaml", ref.ID))
			}
		}
		for id := range pluginIDs {
			if !containsCapability(manifestRefs, id) {
				issues.add(fmt.Sprintf("manifest.capabilities.provides missing capability %q listed in plugin.yaml", id))
			}
		}
	}

	if issues.hasAny() {
		return issues.toError()
	}
	return nil
}

type capabilityRef struct {
	ID         string
	Version    string
	Descriptor string
	Schemas    struct {
		Input  []string
		Output []string
	}
}

func extractCapabilityRefs(data map[string]interface{}, issues *collector, source string) []capabilityRef {
	if data == nil {
		return nil
	}
	value, ok := data["capabilities"]
	if !ok {
		return nil
	}
	capabilities, ok := value.(map[string]interface{})
	if !ok {
		issues.add(fmt.Sprintf("%s.capabilities must be an object", source))
		return nil
	}
	providesValue, ok := capabilities["provides"]
	if !ok {
		return nil
	}
	items, ok := providesValue.([]interface{})
	if !ok {
		issues.add(fmt.Sprintf("%s.capabilities.provides must be an array", source))
		return nil
	}

	var refs []capabilityRef
	for idx, item := range items {
		obj, ok := item.(map[string]interface{})
		if !ok {
			issues.add(fmt.Sprintf("%s.capabilities.provides[%d] must be an object", source, idx))
			continue
		}
		ref := capabilityRef{
			ID:         stringValue(obj, "id"),
			Version:    stringValue(obj, "version"),
			Descriptor: stringValue(obj, "descriptor"),
		}
		if ref.ID == "" {
			issues.add(fmt.Sprintf("%s.capabilities.provides[%d] must include id", source, idx))
		}
		if ref.Descriptor == "" {
			issues.add(fmt.Sprintf("%s.capabilities.provides[%d] must include descriptor", source, idx))
		}
		if schemasValue, ok := obj["schemas"].(map[string]interface{}); ok {
			ref.Schemas.Input = stringSlice(schemasValue, "input")
			ref.Schemas.Output = stringSlice(schemasValue, "output")
		}
		refs = append(refs, ref)
	}
	return refs
}

func extractRBACResources(data map[string]interface{}, issues *collector) map[string]map[string]struct{} {
	result := make(map[string]map[string]struct{})
	if data == nil {
		return result
	}

	rbacValue, ok := data["rbac"]
	if !ok {
		return result
	}
	rbac, ok := rbacValue.(map[string]interface{})
	if !ok {
		issues.add("plugin.rbac must be an object")
		return result
	}
	resourcesValue, ok := rbac["resources"]
	if !ok {
		return result
	}
	resources, ok := resourcesValue.([]interface{})
	if !ok {
		issues.add("plugin.rbac.resources must be an array")
		return result
	}

	for idx, item := range resources {
		obj, ok := item.(map[string]interface{})
		if !ok {
			issues.add(fmt.Sprintf("plugin.rbac.resources[%d] must be an object", idx))
			continue
		}
		resource := stringValue(obj, "resource")
		if resource == "" {
			issues.add(fmt.Sprintf("plugin.rbac.resources[%d] must include resource", idx))
			continue
		}
		actions := stringSlice(obj, "actions")
		if len(actions) == 0 {
			issues.add(fmt.Sprintf("plugin.rbac.resources[%d] must declare at least one action", idx))
			continue
		}
		set := make(map[string]struct{}, len(actions))
		for _, action := range actions {
			set[action] = struct{}{}
		}
		result[resource] = set
	}
	return result
}

func ensureSchemaFiles(ref capabilityRef, contractsDir string, issues *collector) {
	for _, path := range ref.Schemas.Input {
		fullPath := filepath.Join(contractsDir, path)
		if !fileExists(fullPath) {
			issues.add(fmt.Sprintf("capability %q input schema not found: %s", ref.ID, fullPath))
		}
	}
	for _, path := range ref.Schemas.Output {
		fullPath := filepath.Join(contractsDir, path)
		if !fileExists(fullPath) {
			issues.add(fmt.Sprintf("capability %q output schema not found: %s", ref.ID, fullPath))
		}
	}
}

func ensureDescriptorSchemas(record Record, contractsDir string, issues *collector) {
	for _, schema := range record.Descriptor.Provides {
		fullPath := filepath.Join(contractsDir, schema.Path)
		if schema.Path == "" {
			issues.add(fmt.Sprintf("capability %q provides schema missing path", record.Descriptor.ID))
			continue
		}
		if !fileExists(fullPath) {
			issues.add(fmt.Sprintf("capability %q provides schema not found: %s", record.Descriptor.ID, fullPath))
		}
	}
	for _, schema := range record.Descriptor.Consumes {
		fullPath := filepath.Join(contractsDir, schema.Path)
		if schema.Path == "" {
			issues.add(fmt.Sprintf("capability %q consumes schema missing path", record.Descriptor.ID))
			continue
		}
		if !fileExists(fullPath) {
			issues.add(fmt.Sprintf("capability %q consumes schema not found: %s", record.Descriptor.ID, fullPath))
		}
	}
}

func verifyRBAC(descriptor Descriptor, rbac map[string]map[string]struct{}, issues *collector) {
	if descriptor.RBAC.Resource == "" {
		issues.add(fmt.Sprintf("capability %q missing rbac.resource", descriptor.ID))
		return
	}
	actions := descriptor.RBAC.Actions
	if len(actions) == 0 {
		issues.add(fmt.Sprintf("capability %q rbac.actions must declare at least one action", descriptor.ID))
	}
	resourceActions, ok := rbac[descriptor.RBAC.Resource]
	if !ok {
		issues.add(fmt.Sprintf("capability %q rbac.resource %q not declared in plugin.rbac.resources", descriptor.ID, descriptor.RBAC.Resource))
		return
	}
	for _, action := range actions {
		if _, exists := resourceActions[action]; !exists {
			issues.add(fmt.Sprintf("capability %q action %q not present in plugin.rbac.resources[%s]", descriptor.ID, action, descriptor.RBAC.Resource))
		}
	}
}

func containsCapability(refs []capabilityRef, id string) bool {
	for _, ref := range refs {
		if ref.ID == id {
			return true
		}
	}
	return false
}

type collector struct {
	issues []string
}

func newCollector() *collector {
	return &collector{issues: []string{}}
}

func (c *collector) add(msg string) {
	if strings.TrimSpace(msg) == "" {
		return
	}
	c.issues = append(c.issues, msg)
}

func (c *collector) hasAny() bool {
	return len(c.issues) > 0
}

func (c *collector) toError() error {
	return fmt.Errorf("%s", strings.Join(c.issues, "\n"))
}

func stringValue(m map[string]interface{}, key string) string {
	if m == nil {
		return ""
	}
	if val, ok := m[key]; ok {
		if s, ok := val.(string); ok {
			return strings.TrimSpace(s)
		}
	}
	return ""
}

func stringSlice(m map[string]interface{}, key string) []string {
	if m == nil {
		return nil
	}
	val, ok := m[key]
	if !ok {
		return nil
	}
	switch slice := val.(type) {
	case []interface{}:
		var out []string
		for _, item := range slice {
			if s, ok := item.(string); ok {
				out = append(out, strings.TrimSpace(s))
			}
		}
		return out
	case []string:
		return slice
	default:
		return nil
	}
}

func fileExists(path string) bool {
	if path == "" {
		return false
	}
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}
