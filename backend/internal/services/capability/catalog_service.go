package capability

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/capabilities"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/config"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/integrations/gateway"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/logger"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
	"gopkg.in/yaml.v3"
)

// CatalogService exposes read-only operations for capability catalog snapshots.
type CatalogService struct {
	manager            capabilities.Manager
	gateway            gatewayClient
	descriptorCache    map[string]*descriptorMetadata
	descriptorCacheMux sync.RWMutex
	cfg                *config.Config
	deps               *app.Deps
}

type gatewayClient interface {
	Enabled() bool
	Invoke(ctx context.Context, params gateway.InvokeParams) (*gateway.InvokeResult, error)
	ListPlatformCapabilities(ctx context.Context, opts gateway.ListPlatformCapabilitiesOptions) ([]gateway.PlatformCapabilityRecord, error)
	Close() error
}

// ListOptions controls how catalog entries are retrieved.
type ListOptions struct {
	Source string
}

type descriptorMetadata struct {
	Kind      string
	Protocols map[string]interface{}
}

// NewCatalogService builds a catalog service when a capabilities manager is available.
func NewCatalogService(deps *app.Deps) *CatalogService {
	if deps == nil {
		return nil
	}
	mgr := deps.CapabilitiesManager
	if mgr == nil {
		log := logger.WithField("component", "capability_catalog_service")
		mgr = capabilities.NewManager(deps.Config, log)
	}
	if mgr == nil {
		return nil
	}
	return &CatalogService{
		manager:         mgr,
		gateway:         deps.CapabilityGateway,
		cfg:             deps.Config,
		deps:            deps,
		descriptorCache: make(map[string]*descriptorMetadata),
	}
}

// List returns the normalized capability entries from the catalog snapshot.
func (s *CatalogService) List(ctx context.Context, opts ListOptions) ([]capabilities.CatalogEntry, error) {
	if s == nil || s.manager == nil {
		return nil, fmt.Errorf("capability catalog service not configured")
	}

	source := normalizeCatalogSource(opts.Source)

	if source == "corex" {
		if entries, err := s.listPlatformCatalog(ctx, opts); err == nil {
			return entries, nil
		} else {
			logger.WithError(err).WithField("component", "capability_catalog_service").
				Warn("failed to load platform capability catalog")
			return []capabilities.CatalogEntry{}, nil
		}
	}

	if source == "all" {
		platformEntries, platformErr := s.listPlatformCatalog(ctx, ListOptions{Source: "corex"})
		if platformErr != nil {
			logger.WithError(platformErr).WithField("component", "capability_catalog_service").
				Warn("failed to load platform capability catalog for source=all, falling back to local manifest")
		}
		localEntries, localErr := s.listLocalCatalog(ctx)
		if localErr != nil {
			return nil, localErr
		}
		return mergeCatalogEntries(platformEntries, localEntries), nil
	}

	return s.listLocalCatalog(ctx)
}

func normalizeCatalogSource(source string) string {
	normalized := strings.ToLower(strings.TrimSpace(source))
	switch normalized {
	case "", "all", "any":
		return "all"
	case "platform":
		return "corex"
	default:
		return normalized
	}
}

func mergeCatalogEntries(platformEntries, localEntries []capabilities.CatalogEntry) []capabilities.CatalogEntry {
	if len(platformEntries) == 0 {
		return localEntries
	}
	if len(localEntries) == 0 {
		return platformEntries
	}

	merged := make([]capabilities.CatalogEntry, 0, len(platformEntries)+len(localEntries))
	seen := make(map[string]struct{}, len(platformEntries)+len(localEntries))
	appendUnique := func(entries []capabilities.CatalogEntry) {
		for _, entry := range entries {
			id := strings.TrimSpace(entry.ID)
			if id == "" {
				continue
			}
			if _, exists := seen[id]; exists {
				continue
			}
			seen[id] = struct{}{}
			merged = append(merged, entry)
		}
	}
	appendUnique(platformEntries)
	appendUnique(localEntries)
	return merged
}

func (s *CatalogService) listLocalCatalog(ctx context.Context) ([]capabilities.CatalogEntry, error) {
	entries, err := s.manager.ListCapabilities(ctx)
	if err != nil {
		return nil, err
	}
	return s.decorate(entries), nil
}

func (s *CatalogService) listPlatformCatalog(ctx context.Context, opts ListOptions) ([]capabilities.CatalogEntry, error) {
	if s.gateway == nil || !s.gateway.Enabled() {
		return s.listPlatformCatalogViaAdminAPI(ctx)
	}
	records, err := s.gateway.ListPlatformCapabilities(ctx, gateway.ListPlatformCapabilitiesOptions{
		Source:  opts.Source,
		Channel: "",
	})
	if err == nil && len(records) > 0 {
		return s.fromPlatformRecords(records), nil
	}
	return s.listPlatformCatalogViaAdminAPI(ctx)
}

func (s *CatalogService) listPlatformCatalogViaAdminAPI(ctx context.Context) ([]capabilities.CatalogEntry, error) {
	if s.cfg == nil || s.cfg.Gateway == nil {
		return nil, errors.New("gateway config missing")
	}
	base := strings.TrimSpace(s.cfg.Gateway.BaseURL)
	if base == "" {
		return nil, errors.New("PX_GATEWAY_BASE_URL 未配置")
	}
	base = strings.TrimRight(base, "/")
	apiPrefix := normalizeGatewayAPIPrefix(s.cfg.Gateway.APIPrefix)
	apiKey := strings.TrimSpace(s.cfg.Gateway.APIKey)
	authScheme := strings.ToLower(strings.TrimSpace(s.cfg.Gateway.AuthScheme))
	if authScheme == "" {
		if apiKey != "" {
			authScheme = "apikey"
		} else {
			authScheme = "bearer"
		}
	}
	client := &http.Client{Timeout: 10 * time.Second}
	url := fmt.Sprintf("%s/admin/platform-capabilities?page=1&page_size=200", joinGatewayBaseAndPrefix(base, apiPrefix))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if authScheme == "apikey" && apiKey != "" {
		req.Header.Set("Authorization", "ApiKey "+apiKey)
	} else if s.deps != nil {
		token, err := s.deps.HostBearerToken(ctx)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("X-Request-ID", fmt.Sprintf("cap-catalog-%d", time.Now().UnixNano()))

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var payload platformCapabilitiesResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode platform capabilities: %w", err)
	}
	if resp.StatusCode >= 400 || payload.Code >= 400 {
		return nil, fmt.Errorf("platform capabilities request failed: status=%d code=%d message=%s", resp.StatusCode, payload.Code, payload.Message)
	}
	records := payload.toPlatformRecords()
	if len(records) == 0 {
		return nil, errors.New("platform capability catalog returned empty data")
	}
	return s.fromPlatformRecords(records), nil
}

func normalizeGatewayAPIPrefix(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "/api/v1"
	}
	if !strings.HasPrefix(value, "/") {
		value = "/" + value
	}
	value = "/" + strings.Trim(strings.TrimSpace(value), "/")
	if value == "/" {
		return "/api/v1"
	}
	return value
}

func joinGatewayBaseAndPrefix(base, apiPrefix string) string {
	normalizedBase := strings.TrimRight(strings.TrimSpace(base), "/")
	if normalizedBase == "" {
		return ""
	}
	normalizedPrefix := normalizeGatewayAPIPrefix(apiPrefix)
	if normalizedPrefix == "" || normalizedPrefix == "/" {
		return normalizedBase
	}
	if strings.HasSuffix(normalizedBase, normalizedPrefix) {
		return normalizedBase
	}
	return normalizedBase + normalizedPrefix
}

func (s *CatalogService) fromPlatformRecords(records []gateway.PlatformCapabilityRecord) []capabilities.CatalogEntry {
	result := make([]capabilities.CatalogEntry, 0, len(records))
	for _, record := range records {
		entry := capabilities.CatalogEntry{
			ID:        strings.TrimSpace(record.CapabilityID),
			Version:   strings.TrimSpace(record.PluginVersion),
			Tags:      append([]string{}, record.Categories...),
			Protocols: convertPlatformProtocols(record.Protocols),
			Execution: capabilities.ExecutionConfig{
				Mode: strings.ToLower(strings.TrimSpace(record.ExecutionMode)),
			},
			Module: deriveCapabilityModule(record.CapabilityID),
			Kind:   detectPlatformKind(record),
			Checksum: func() string {
				if strings.TrimSpace(record.CapabilitiesHash) != "" {
					return record.CapabilitiesHash
				}
				return record.ProtocolHash
			}(),
		}
		if strings.TrimSpace(entry.Execution.Mode) == "" {
			entry.Execution.Mode = "sync"
		}
		result = append(result, entry)
	}
	return result
}

func (s *CatalogService) decorate(entries []capabilities.CatalogEntry) []capabilities.CatalogEntry {
	for i := range entries {
		entries[i].Module = deriveCapabilityModule(entries[i].ID)
		meta := s.lookupDescriptorMeta(entries[i].Descriptor)
		if meta != nil {
			if meta.Kind != "" {
				entries[i].Kind = meta.Kind
			}
			if len(entries[i].Protocols) == 0 && len(meta.Protocols) > 0 {
				entries[i].Protocols = cloneProtocolMap(meta.Protocols)
			}
		}
	}
	return entries
}

func deriveCapabilityModule(id string) string {
	parts := strings.Split(strings.TrimSpace(id), ".")
	if len(parts) <= 1 {
		return strings.TrimSpace(id)
	}
	return strings.Join(parts[:len(parts)-1], ".")
}

func convertPlatformProtocols(protocols []gateway.PlatformCapabilityProtocol) map[string]interface{} {
	if len(protocols) == 0 {
		return nil
	}

	grouped := make(map[string][]map[string]interface{})
	for _, proto := range protocols {
		channel := strings.ToLower(strings.TrimSpace(proto.Channel))
		if channel == "" {
			channel = "rest"
		}

		payload := map[string]interface{}{}
		if v := strings.TrimSpace(proto.Endpoint); v != "" {
			payload["endpoint"] = v
			if channel == "rest" || channel == "http" {
				payload["path"] = v
			} else {
				payload["service"] = v
			}
		}
		if v := strings.TrimSpace(proto.Method); v != "" {
			payload["method"] = strings.ToUpper(v)
		}
		if v := strings.TrimSpace(proto.RPC); v != "" {
			payload["rpc"] = v
			if _, exists := payload["method"]; !exists {
				payload["method"] = v
			}
		}
		if v := strings.TrimSpace(proto.SchemaRef); v != "" {
			payload["schema_ref"] = v
		}
		if v := strings.TrimSpace(proto.ToolRef); v != "" {
			payload["tool_ref"] = v
		}
		if len(payload) == 0 {
			payload["defined"] = true
		}
		grouped[channel] = append(grouped[channel], payload)
	}

	result := make(map[string]interface{}, len(grouped))
	for channel, entries := range grouped {
		if len(entries) == 1 {
			result[channel] = entries[0]
			continue
		}
		result[channel] = entries
	}
	return result
}

func detectPlatformKind(record gateway.PlatformCapabilityRecord) string {
	if hasWorkflowChannel(record.Protocols) {
		return "Workflow"
	}
	return "Capability"
}

func hasWorkflowChannel(protocols []gateway.PlatformCapabilityProtocol) bool {
	for _, proto := range protocols {
		channel := strings.ToLower(strings.TrimSpace(proto.Channel))
		if strings.Contains(channel, "workflow") || strings.Contains(channel, "composite") {
			return true
		}
	}
	return false
}

type platformCapabilitiesResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		Modules []platformCapabilityModule `json:"modules"`
	} `json:"data"`
}

type platformCapabilityModule struct {
	Module       string                       `json:"module"`
	Capabilities []platformCapabilityEnvelope `json:"capabilities"`
}

type platformCapabilityEnvelope struct {
	CapabilityID  string                               `json:"capability_id"`
	PluginID      string                               `json:"plugin_id"`
	PluginVersion string                               `json:"plugin_version"`
	Source        string                               `json:"source"`
	Protocols     []platformCapabilityProtocolEnvelope `json:"protocols"`
}

type platformCapabilityProtocolEnvelope struct {
	Channel   string `json:"channel"`
	Endpoint  string `json:"endpoint"`
	SchemaRef string `json:"schema_ref"`
	Method    string `json:"method"`
	RPC       string `json:"rpc"`
	ToolRef   string `json:"tool_ref"`
}

func (resp platformCapabilitiesResponse) toPlatformRecords() []gateway.PlatformCapabilityRecord {
	var records []gateway.PlatformCapabilityRecord
	for _, module := range resp.Data.Modules {
		for _, cap := range module.Capabilities {
			record := gateway.PlatformCapabilityRecord{
				CapabilityID:     cap.CapabilityID,
				PluginID:         cap.PluginID,
				PluginVersion:    cap.PluginVersion,
				Source:           cap.Source,
				ExecutionMode:    "sync",
				Protocols:        make([]gateway.PlatformCapabilityProtocol, 0, len(cap.Protocols)),
				CapabilitiesHash: "",
				ProtocolHash:     "",
			}
			for _, proto := range cap.Protocols {
				record.Protocols = append(record.Protocols, gateway.PlatformCapabilityProtocol{
					Channel:   proto.Channel,
					Endpoint:  proto.Endpoint,
					SchemaRef: proto.SchemaRef,
					Method:    proto.Method,
					RPC:       proto.RPC,
					ToolRef:   proto.ToolRef,
				})
			}
			records = append(records, record)
		}
	}
	return records
}

var descriptorSearchBases = []string{
	".",
	"..",
	"../..",
	"../../..",
	"../../../..",
	"../../../../..",
	"../../../../../..",
}

func (s *CatalogService) lookupDescriptorMeta(path string) *descriptorMetadata {
	if s == nil {
		return nil
	}
	normalized := strings.TrimSpace(path)
	if normalized == "" {
		return nil
	}
	normalized = filepath.Clean(normalized)

	s.descriptorCacheMux.RLock()
	if v, ok := s.descriptorCache[normalized]; ok {
		s.descriptorCacheMux.RUnlock()
		return v
	}
	s.descriptorCacheMux.RUnlock()

	resolved := resolveDescriptorPath(normalized)
	if resolved == "" {
		logger.WithField("descriptor", normalized).Debug("capability descriptor not found for metadata inference")
		return nil
	}

	data, err := os.ReadFile(resolved)
	if err != nil {
		logger.WithError(err).WithField("descriptor", resolved).Debug("failed to read capability descriptor for metadata inference")
		return nil
	}
	var manifest struct {
		Type     string `yaml:"type"`
		Metadata struct {
			Protocols map[string]interface{} `yaml:"protocols"`
		} `yaml:"metadata"`
	}
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		logger.WithError(err).WithField("descriptor", resolved).Debug("failed to parse capability descriptor for metadata inference")
		return nil
	}

	meta := &descriptorMetadata{
		Kind:      classifyCapabilityKind(manifest.Type, manifest.Metadata.Protocols),
		Protocols: manifest.Metadata.Protocols,
	}

	s.descriptorCacheMux.Lock()
	s.descriptorCache[normalized] = meta
	if resolved != normalized {
		s.descriptorCache[resolved] = meta
	}
	s.descriptorCacheMux.Unlock()
	return meta
}

func resolveDescriptorPath(path string) string {
	if path == "" {
		return ""
	}
	if filepath.IsAbs(path) {
		if _, err := os.Stat(path); err == nil {
			return path
		}
		return ""
	}

	seen := map[string]struct{}{}
	candidates := []string{path}
	seen[path] = struct{}{}
	for _, base := range descriptorSearchBases {
		candidate := filepath.Clean(filepath.Join(base, path))
		if _, ok := seen[candidate]; ok {
			continue
		}
		candidates = append(candidates, candidate)
		seen[candidate] = struct{}{}
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			if abs, err := filepath.Abs(candidate); err == nil {
				return abs
			}
			return candidate
		}
	}

	return ""
}

func classifyCapabilityKind(declared string, protocols map[string]interface{}) string {
	if normalized := normalizeKindName(declared); normalized != "" {
		return normalized
	}

	if hasWorkflowProtocol(protocols) {
		return "Workflow"
	}
	return "Capability"
}

func normalizeKindName(raw string) string {
	value := strings.ToLower(strings.TrimSpace(raw))
	switch value {
	case "":
		return ""
	case "workflow":
		return "Workflow"
	case "tool":
		return "Tool"
	case "api", "capability":
		return "Capability"
	default:
		return strings.Title(value)
	}
}

func hasWorkflowProtocol(protocols map[string]interface{}) bool {
	if len(protocols) == 0 {
		return false
	}
	if v, ok := protocols["workflow"]; ok && v != nil {
		return true
	}
	return false
}

func cloneProtocolMap(input map[string]interface{}) map[string]interface{} {
	if len(input) == 0 {
		return nil
	}
	out := make(map[string]interface{}, len(input))
	for k, v := range input {
		out[k] = v
	}
	return out
}
