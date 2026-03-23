package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/contracts/capability"
	yaml "gopkg.in/yaml.v3"
)

func main() {
	pluginPath := flag.String("plugin", "plugin.yaml", "Path to plugin.yaml")
	manifestPath := flag.String("manifest", "docs/lifecycle/examples/manifest.yaml", "Path to manifest.yaml")
	schemaPath := flag.String("schema", "docs/lifecycle/contracts/manifest.schema.json", "Path to manifest JSON schema (for documentation reference)")
	capabilitiesOnly := flag.Bool("capabilities-only", false, "Only run capability validation (skip manifest shape checks)")
	eventFabricPath := flag.String("event-fabric", "", "Path to event fabric yaml for topic parity checks")
	pluginOnly := flag.Bool("plugin-only", false, "Validate plugin.yaml only (skip plugin-manifest parity checks)")
	syncCatalogs := flag.Bool("sync-catalogs", false, "Sync plugin.d capability/exposure catalogs from contracts/capabilities")
	syncOnly := flag.Bool("sync-only", false, "Sync catalogs then exit")
	flag.Parse()

	if err := run(*pluginPath, *manifestPath, *schemaPath, *eventFabricPath, *capabilitiesOnly, *pluginOnly, *syncCatalogs, *syncOnly); err != nil {
		fmt.Fprintf(os.Stderr, "manifestcheck: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("manifestcheck: plugin and manifest metadata validated successfully")
}

func run(pluginPath, manifestPath, schemaPath, eventFabricPath string, capabilitiesOnly, pluginOnly, syncCatalogs, syncOnly bool) error {
	if syncCatalogs {
		if err := syncPluginCatalogs(pluginPath); err != nil {
			return err
		}
		if syncOnly {
			return nil
		}
	}

	pluginMap, err := loadYAMLFile(pluginPath)
	if err != nil {
		return fmt.Errorf("load plugin: %w", err)
	}
	if err := mergeCatalogReferences(pluginPath, pluginMap); err != nil {
		return fmt.Errorf("merge plugin catalogs: %w", err)
	}

	var manifestMap map[string]interface{}
	if manifestPath != "" {
		manifestMap, err = loadYAMLFile(manifestPath)
		if err != nil {
			return fmt.Errorf("load manifest: %w", err)
		}
	}

	if !capabilitiesOnly && !pluginOnly {
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

	if strings.TrimSpace(eventFabricPath) != "" {
		eventFabricMap, err := loadYAMLFile(eventFabricPath)
		if err != nil {
			return fmt.Errorf("load event fabric: %w", err)
		}
		if err := validateEventTopics(pluginMap, eventFabricMap); err != nil {
			return err
		}
	}

	return nil
}

func mergeCatalogReferences(pluginPath string, plugin map[string]interface{}) error {
	catalogsValue, ok := plugin["catalogs"]
	if !ok || catalogsValue == nil {
		return nil
	}
	catalogs, ok := catalogsValue.(map[string]interface{})
	if !ok {
		return errors.New("catalogs must be an object")
	}

	pluginDir := filepath.Dir(pluginPath)
	loadCatalog := func(key string) (map[string]interface{}, error) {
		rawPath := strings.TrimSpace(stringValue(catalogs, key))
		if rawPath == "" {
			return nil, nil
		}
		filePath := rawPath
		if !filepath.IsAbs(filePath) {
			filePath = filepath.Join(pluginDir, filepath.FromSlash(rawPath))
		}
		doc, err := loadYAMLFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("load catalogs.%s (%s): %w", key, rawPath, err)
		}
		return doc, nil
	}

	if doc, err := loadCatalog("capabilities"); err != nil {
		return err
	} else if doc != nil {
		if section, ok := doc["capabilities"]; ok {
			plugin["capabilities"] = section
		}
	}
	if doc, err := loadCatalog("events"); err != nil {
		return err
	} else if doc != nil {
		if section, ok := doc["events"]; ok {
			plugin["events"] = section
		}
	}
	if doc, err := loadCatalog("agent_tools"); err != nil {
		return err
	} else if doc != nil {
		if section, ok := doc["agent_tools"]; ok {
			plugin["agent_tools"] = section
		}
	}
	if doc, err := loadCatalog("exposure"); err != nil {
		return err
	} else if doc != nil {
		if section, ok := doc["exposure"]; ok {
			plugin["exposure"] = section
		}
	}
	if doc, err := loadCatalog("rbac"); err != nil {
		return err
	} else if doc != nil {
		if section, ok := doc["rbac"]; ok {
			plugin["rbac"] = section
		}
		if section, ok := doc["permissions"]; ok {
			plugin["permissions"] = section
		}
		if section, ok := doc["routes"]; ok {
			plugin["routes"] = section
		}
	}

	return nil
}

func validateEventTopics(pluginData, eventFabricData map[string]interface{}) error {
	pluginTopics, err := collectTopics(pluginData, []string{"events", "topics"}, []string{"key", "name", "topic"})
	if err != nil {
		return err
	}
	fabricTopics, err := collectTopics(eventFabricData, []string{"topics"}, []string{"topic", "key", "name"})
	if err != nil {
		return err
	}

	pluginSet := make(map[string]struct{}, len(pluginTopics))
	for _, topic := range pluginTopics {
		pluginSet[topic] = struct{}{}
	}
	fabricSet := make(map[string]struct{}, len(fabricTopics))
	for _, topic := range fabricTopics {
		fabricSet[topic] = struct{}{}
	}

	var missingInFabric []string
	for _, topic := range pluginTopics {
		if _, ok := fabricSet[topic]; !ok {
			missingInFabric = append(missingInFabric, topic)
		}
	}
	var missingInPlugin []string
	for _, topic := range fabricTopics {
		if _, ok := pluginSet[topic]; !ok {
			missingInPlugin = append(missingInPlugin, topic)
		}
	}

	if len(missingInFabric) == 0 && len(missingInPlugin) == 0 {
		return nil
	}

	sort.Strings(missingInFabric)
	sort.Strings(missingInPlugin)
	var issues []string
	if len(missingInFabric) > 0 {
		issues = append(issues, fmt.Sprintf("plugin.yaml.events.topics 缺失执行层映射: %s", strings.Join(missingInFabric, ", ")))
	}
	if len(missingInPlugin) > 0 {
		issues = append(issues, fmt.Sprintf("config/event_fabric.yaml.topics 存在未声明 topic: %s", strings.Join(missingInPlugin, ", ")))
	}

	return errors.New(strings.Join(issues, "; "))
}

func collectTopics(data map[string]interface{}, path []string, keys []string) ([]string, error) {
	value, err := lookup(data, path...)
	if err != nil {
		return nil, fmt.Errorf("missing required field: %s", strings.Join(path, "."))
	}

	items, ok := value.([]interface{})
	if !ok {
		return nil, fmt.Errorf("%s must be an array", strings.Join(path, "."))
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("%s must contain at least one item", strings.Join(path, "."))
	}

	seen := make(map[string]struct{}, len(items))
	topics := make([]string, 0, len(items))
	for idx, item := range items {
		obj, ok := item.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("%s[%d] must be an object", strings.Join(path, "."), idx)
		}
		var topic string
		for _, key := range keys {
			topic = strings.TrimSpace(stringValue(obj, key))
			if topic != "" {
				break
			}
		}
		if topic == "" {
			return nil, fmt.Errorf("%s[%d] missing topic key (supported: %s)", strings.Join(path, "."), idx, strings.Join(keys, ", "))
		}
		if _, exists := seen[topic]; exists {
			continue
		}
		seen[topic] = struct{}{}
		topics = append(topics, topic)
	}

	return topics, nil
}

func stringValue(data map[string]interface{}, key string) string {
	value, ok := data[key]
	if !ok || value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	default:
		return fmt.Sprintf("%v", typed)
	}
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

const (
	apiPrefix    = "/api/v1"
	httpBaseExpr = "${POWERX_PLUGIN_HTTP_BASE:-/api/v1}"
)

type catalogRestEntry struct {
	Method     string
	Entrypoint string
}

type catalogDescriptor struct {
	ID            string
	Version       string
	Descriptor    string
	SchemasInput  []string
	SchemasOutput []string
	RestEntries   []catalogRestEntry
	RBACResource  string
	RBACActions   []string
	RBAC          string
}

func syncPluginCatalogs(pluginPath string) error {
	rootDir, err := filepath.Abs(filepath.Dir(pluginPath))
	if err != nil {
		return fmt.Errorf("resolve plugin root: %w", err)
	}

	descriptors, err := collectCatalogDescriptors(rootDir)
	if err != nil {
		return err
	}
	if len(descriptors) == 0 {
		return fmt.Errorf("未找到 contracts/capabilities/*.yaml，无法生成 plugin.d")
	}

	capFile := filepath.Join(rootDir, "plugin.d", "capabilities.yaml")
	expFile := filepath.Join(rootDir, "plugin.d", "exposure.yaml")
	rbacFile := filepath.Join(rootDir, "plugin.d", "rbac.yaml")

	existingCap, err := loadYAMLFileOptional(capFile)
	if err != nil {
		return fmt.Errorf("load existing capabilities catalog: %w", err)
	}
	existingExp, err := loadYAMLFileOptional(expFile)
	if err != nil {
		return fmt.Errorf("load existing exposure catalog: %w", err)
	}
	existingRBAC, err := loadYAMLFileOptional(rbacFile)
	if err != nil {
		return fmt.Errorf("load existing rbac catalog: %w", err)
	}

	targetCap := buildCapabilitiesCatalog(existingCap, descriptors)
	targetExp := buildExposureCatalog(existingExp, descriptors)
	targetRBAC := buildRBACCatalog(existingRBAC, descriptors)

	if err := writeYAMLFile(capFile, targetCap); err != nil {
		return fmt.Errorf("write %s: %w", capFile, err)
	}
	if err := writeYAMLFile(expFile, targetExp); err != nil {
		return fmt.Errorf("write %s: %w", expFile, err)
	}
	if err := writeYAMLFile(rbacFile, targetRBAC); err != nil {
		return fmt.Errorf("write %s: %w", rbacFile, err)
	}

	fmt.Println("✅ 已同步 plugin catalog:")
	fmt.Printf("   - %s\n", capFile)
	fmt.Printf("   - %s\n", expFile)
	fmt.Printf("   - %s\n", rbacFile)
	return nil
}

func collectCatalogDescriptors(rootDir string) ([]catalogDescriptor, error) {
	glob := filepath.Join(rootDir, "contracts", "capabilities", "*.yaml")
	files, err := filepath.Glob(glob)
	if err != nil {
		return nil, err
	}
	sort.Strings(files)

	descriptors := make([]catalogDescriptor, 0, len(files))
	for _, file := range files {
		doc, err := loadYAMLFile(file)
		if err != nil {
			return nil, fmt.Errorf("load capability descriptor %s: %w", file, err)
		}
		capID := strings.TrimSpace(stringFromAny(doc["id"]))
		if capID == "" {
			continue
		}
		version := strings.TrimSpace(stringFromAny(doc["version"]))
		if version == "" {
			version = "1.0.0"
		}

		schemasInput := make([]string, 0)
		schemasOutput := make([]string, 0)
		for _, field := range []string{"provides", "consumes"} {
			rawList, ok := doc[field].([]interface{})
			if !ok {
				continue
			}
			for _, item := range rawList {
				entry, ok := item.(map[string]interface{})
				if !ok {
					continue
				}
				kind := strings.ToLower(strings.TrimSpace(stringFromAny(entry["kind"])))
				schemaPath := normalizeContractPath(stringFromAny(entry["path"]))
				if schemaPath == "" {
					continue
				}
				switch kind {
				case "input":
					schemasInput = append(schemasInput, schemaPath)
				case "output":
					schemasOutput = append(schemasOutput, schemaPath)
				}
			}
		}

		restEntries := make([]catalogRestEntry, 0)
		if metadata, ok := doc["metadata"].(map[string]interface{}); ok {
			if protocols, ok := metadata["protocols"].(map[string]interface{}); ok {
				if restValue, exists := protocols["rest"]; exists {
					switch typed := restValue.(type) {
					case map[string]interface{}:
						method := strings.ToUpper(strings.TrimSpace(stringFromAny(typed["method"])))
						if method == "" {
							method = "GET"
						}
						entrypoint := toEntrypoint(stringFromAny(typed["path"]))
						if entrypoint != "" {
							restEntries = append(restEntries, catalogRestEntry{Method: method, Entrypoint: entrypoint})
						}
					case []interface{}:
						for _, item := range typed {
							spec, ok := item.(map[string]interface{})
							if !ok {
								continue
							}
							method := strings.ToUpper(strings.TrimSpace(stringFromAny(spec["method"])))
							if method == "" {
								method = "GET"
							}
							entrypoint := toEntrypoint(stringFromAny(spec["path"]))
							if entrypoint != "" {
								restEntries = append(restEntries, catalogRestEntry{Method: method, Entrypoint: entrypoint})
							}
						}
					}
				}
			}
		}

		rbacResource := ""
		rbacActions := make([]string, 0)
		rbacCode := ""
		if rbac, ok := doc["rbac"].(map[string]interface{}); ok {
			rbacResource = strings.TrimSpace(stringFromAny(rbac["resource"]))
			rbacActions = uniqueSortedStrings(actionListFromAny(rbac["actions"]))
			if rbacResource != "" && len(rbacActions) > 0 {
				rbacCode = rbacResource + ":" + rbacActions[0]
			}
		}
		schemasInput = uniqueSortedStrings(schemasInput)
		schemasOutput = uniqueSortedStrings(schemasOutput)
		descriptors = append(descriptors, catalogDescriptor{
			ID:            capID,
			Version:       version,
			Descriptor:    filepath.ToSlash(filepath.Join("contracts", "capabilities", filepath.Base(file))),
			SchemasInput:  schemasInput,
			SchemasOutput: schemasOutput,
			RestEntries:   restEntries,
			RBACResource:  rbacResource,
			RBACActions:   rbacActions,
			RBAC:          rbacCode,
		})
	}

	sort.Slice(descriptors, func(i, j int) bool {
		return descriptors[i].ID < descriptors[j].ID
	})
	return descriptors, nil
}

func buildCapabilitiesCatalog(existing map[string]interface{}, descriptors []catalogDescriptor) map[string]interface{} {
	required := make([]string, 0)
	if caps, ok := existing["capabilities"].(map[string]interface{}); ok {
		if reqRaw, ok := caps["required"].([]interface{}); ok {
			for _, item := range reqRaw {
				value := strings.TrimSpace(stringFromAny(item))
				if value != "" {
					required = append(required, value)
				}
			}
		}
	}
	required = uniqueSortedStrings(required)

	provides := make([]map[string]interface{}, 0, len(descriptors))
	imports := make([]string, 0, len(descriptors))
	for _, descriptor := range descriptors {
		schemas := map[string]interface{}{"input": "", "output": ""}
		if len(descriptor.SchemasInput) > 0 {
			schemas["input"] = descriptor.SchemasInput[0]
		}
		if len(descriptor.SchemasOutput) > 0 {
			schemas["output"] = descriptor.SchemasOutput[0]
		}
		provides = append(provides, map[string]interface{}{
			"id":         descriptor.ID,
			"version":    descriptor.Version,
			"descriptor": descriptor.Descriptor,
			"schemas":    schemas,
		})
		imports = append(imports, descriptor.Descriptor)
	}

	return map[string]interface{}{
		"capabilities": map[string]interface{}{
			"required": required,
			"provides": provides,
			"imports":  imports,
		},
	}
}

func buildExposureCatalog(existing map[string]interface{}, descriptors []catalogDescriptor) map[string]interface{} {
	keepNonRest := make([]map[string]interface{}, 0)
	if exposure, ok := existing["exposure"].(map[string]interface{}); ok {
		if channels, ok := exposure["channels"].([]interface{}); ok {
			for _, item := range channels {
				channel, ok := item.(map[string]interface{})
				if !ok {
					continue
				}
				channelType := strings.ToLower(strings.TrimSpace(stringFromAny(channel["type"])))
				if channelType == "rest" {
					continue
				}
				keepNonRest = append(keepNonRest, channel)
			}
		}
	}

	seen := make(map[string]struct{})
	restChannels := make([]map[string]interface{}, 0)
	for _, descriptor := range descriptors {
		for _, rest := range descriptor.RestEntries {
			key := descriptor.ID + "|" + rest.Method + "|" + rest.Entrypoint
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			channel := map[string]interface{}{
				"type":       "rest",
				"capability": descriptor.ID,
				"entrypoint": rest.Entrypoint,
				"method":     rest.Method,
				"auth":       "jwt",
			}
			if descriptor.RBAC != "" {
				channel["rbac"] = descriptor.RBAC
			}
			restChannels = append(restChannels, channel)
		}
	}

	channels := make([]map[string]interface{}, 0, len(restChannels)+len(keepNonRest))
	channels = append(channels, restChannels...)
	channels = append(channels, keepNonRest...)

	return map[string]interface{}{
		"exposure": map[string]interface{}{
			"channels": channels,
		},
	}
}

func buildRBACCatalog(existing map[string]interface{}, descriptors []catalogDescriptor) map[string]interface{} {
	resourceActions := make(map[string]map[string]bool)

	mergeResourceActions := func(items []interface{}) {
		for _, item := range items {
			entry, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			resource := strings.TrimSpace(stringFromAny(entry["resource"]))
			if resource == "" {
				continue
			}
			if _, ok := resourceActions[resource]; !ok {
				resourceActions[resource] = make(map[string]bool)
			}
			for _, action := range actionListFromAny(entry["actions"]) {
				action = strings.TrimSpace(action)
				if action == "" {
					continue
				}
				resourceActions[resource][action] = true
			}
		}
	}

	if rbac, ok := existing["rbac"].(map[string]interface{}); ok {
		if raw, ok := rbac["resources"].([]interface{}); ok {
			mergeResourceActions(raw)
		}
	}
	if raw, ok := existing["permissions"].([]interface{}); ok {
		mergeResourceActions(raw)
	}

	for _, descriptor := range descriptors {
		resource := strings.TrimSpace(descriptor.RBACResource)
		if resource == "" {
			continue
		}
		if _, ok := resourceActions[resource]; !ok {
			resourceActions[resource] = make(map[string]bool)
		}
		for _, action := range descriptor.RBACActions {
			action = strings.TrimSpace(action)
			if action == "" {
				continue
			}
			resourceActions[resource][action] = true
		}
	}

	resourceNames := make([]string, 0, len(resourceActions))
	for resource := range resourceActions {
		resourceNames = append(resourceNames, resource)
	}
	sort.Strings(resourceNames)

	resources := make([]map[string]interface{}, 0, len(resourceNames))
	permissions := make([]map[string]interface{}, 0, len(resourceNames))
	for _, resource := range resourceNames {
		actions := make([]string, 0, len(resourceActions[resource]))
		for action := range resourceActions[resource] {
			actions = append(actions, action)
		}
		sort.Strings(actions)
		resources = append(resources, map[string]interface{}{
			"resource": resource,
			"actions":  actions,
		})
		permissions = append(permissions, map[string]interface{}{
			"resource": resource,
			"actions":  actions,
		})
	}

	routes := map[string]interface{}{
		"basePath":      "/api/v1",
		"adminManifest": "/api/v1/admin/manifest",
		"rbac":          "/api/v1/admin/rbac",
	}
	if raw, ok := existing["routes"].(map[string]interface{}); ok {
		for key, value := range raw {
			routes[key] = value
		}
	}

	return map[string]interface{}{
		"rbac": map[string]interface{}{
			"resources": resources,
		},
		"permissions": permissions,
		"routes":      routes,
	}
}

func loadYAMLFileOptional(path string) (map[string]interface{}, error) {
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return map[string]interface{}{}, nil
		}
		return nil, err
	}
	return loadYAMLFile(path)
}

func writeYAMLFile(path string, data interface{}) error {
	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)
	if err := encoder.Encode(data); err != nil {
		return err
	}
	if err := encoder.Close(); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0o644)
}

func firstAction(value interface{}) string {
	switch actions := value.(type) {
	case []interface{}:
		for _, action := range actions {
			raw := strings.TrimSpace(stringFromAny(action))
			if raw != "" {
				return raw
			}
		}
	case string:
		return strings.TrimSpace(actions)
	default:
		return strings.TrimSpace(stringFromAny(actions))
	}
	return ""
}

func actionListFromAny(value interface{}) []string {
	switch actions := value.(type) {
	case []interface{}:
		out := make([]string, 0, len(actions))
		for _, action := range actions {
			raw := strings.TrimSpace(stringFromAny(action))
			if raw != "" {
				out = append(out, raw)
			}
		}
		return out
	case string:
		raw := strings.TrimSpace(actions)
		if raw == "" {
			return nil
		}
		return []string{raw}
	default:
		raw := strings.TrimSpace(stringFromAny(actions))
		if raw == "" {
			return nil
		}
		return []string{raw}
	}
}

func normalizeContractPath(path string) string {
	candidate := strings.TrimSpace(path)
	candidate = strings.TrimPrefix(candidate, "./")
	if candidate == "" {
		return ""
	}
	if strings.HasPrefix(candidate, "contracts/") {
		return candidate
	}
	return "contracts/" + candidate
}

func toEntrypoint(pathValue string) string {
	raw := strings.TrimSpace(pathValue)
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(raw, "${POWERX_PLUGIN_HTTP_BASE") {
		return raw
	}
	if strings.HasPrefix(raw, apiPrefix) {
		suffix := strings.TrimPrefix(raw, apiPrefix)
		if suffix == "" {
			return httpBaseExpr
		}
		if !strings.HasPrefix(suffix, "/") {
			suffix = "/" + suffix
		}
		return httpBaseExpr + suffix
	}
	if strings.HasPrefix(raw, "/") {
		return httpBaseExpr + raw
	}
	return httpBaseExpr + "/" + raw
}

func uniqueSortedStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, item := range values {
		value := strings.TrimSpace(item)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func stringFromAny(value interface{}) string {
	if value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	default:
		return fmt.Sprintf("%v", typed)
	}
}
