package capabilities

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/config"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/logger"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

const (
	defaultCatalogPath  = "capabilities/catalog.json"
	catalogEnvKey       = "POWERX_CAPABILITY_CATALOG"
	exposureDir         = "contracts/exposure"
	agentSDKDir         = "dist/agent-sdk"
	protocolOpenAPIType = "openapi"
	protocolProtoType   = "proto"
	protocolWorkflow    = "workflow"
	protocolMCP         = "mcp_manifest"
	protocolAgentStream = "agent_stream"
)

// Manager exposes high-level capability operations for runtime bootstrap and host sync.
type Manager interface {
	ListCapabilities(ctx context.Context) ([]CatalogEntry, error)
	ExportProtocols(ctx context.Context) ([]ProtocolAsset, error)
	RegisterWithHost(ctx context.Context, client HostSyncClient) error
}

// HostSyncClient represents the client used to register capability catalogs with PowerX.
type HostSyncClient interface {
	RegisterCatalog(ctx context.Context, catalog *CatalogSnapshot, assets []ProtocolAsset) error
}

// CatalogLoader resolves catalog snapshots from manifest outputs.
type CatalogLoader interface {
	LoadCatalog(ctx context.Context) (*CatalogSnapshot, error)
}

// AssetExporter enumerates protocol assets (OpenAPI, Proto, Workflow, MCP manifests, etc.).
type AssetExporter interface {
	Export(ctx context.Context, catalog *CatalogSnapshot) ([]ProtocolAsset, error)
}

// CatalogSnapshot is persisted JSON emitted by scripts/capabilities/catalog-parser.ts.
type CatalogSnapshot struct {
	PluginID        string         `json:"plugin_id"`
	ManifestVersion string         `json:"manifest_version"`
	GeneratedAt     string         `json:"generated_at"`
	Imports         []string       `json:"imports"`
	Entries         []CatalogEntry `json:"entries"`
}

// CatalogEntry describes a single capability definition.
type CatalogEntry struct {
	ID         string                 `json:"id"`
	Version    string                 `json:"version"`
	Descriptor string                 `json:"descriptor"`
	Schemas    map[string]string      `json:"schemas"`
	Protocols  map[string]interface{} `json:"protocols"`
	Tags       []string               `json:"tags"`
	Execution  ExecutionConfig        `json:"execution"`
	Checksum   string                 `json:"checksum"`
	Module     string                 `json:"module,omitempty"`
	Kind       string                 `json:"kind,omitempty"`
}

// ExecutionConfig controls sync/async semantics of a capability.
type ExecutionConfig struct {
	Mode           string `json:"mode"`
	CallbackURL    string `json:"callback_url,omitempty"`
	SSEChannel     string `json:"sse_channel,omitempty"`
	StatusEndpoint string `json:"status_endpoint,omitempty"`
	TimeoutSeconds int    `json:"timeout_seconds,omitempty"`
}

// ProtocolAsset captures generated exposure artefacts.
type ProtocolAsset struct {
	Type string `json:"type"`
	Path string `json:"path"`
}

type manager struct {
	cfg           *config.Config
	logger        *logrus.Entry
	catalogLoader CatalogLoader
	assetExporter AssetExporter
}

// NewManager builds a filesystem-based manager using repo defaults.
func NewManager(cfg *config.Config, log *logrus.Entry) Manager {
	catalogPath := resolveCatalogPath(os.Getenv(catalogEnvKey))
	if strings.TrimSpace(catalogPath) == "" {
		catalogPath = defaultCatalogPath
	}

	loader := &fileSystemCatalogLoader{path: catalogPath}
	exporter := &fileSystemAssetExporter{
		roots: []assetRoot{
			{dir: exposureDir, protocolType: protocolOpenAPIType, matcher: func(path string) bool {
				return strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml")
			}},
			{dir: filepath.Join(exposureDir, "proto"), protocolType: protocolProtoType, matcher: func(path string) bool {
				return strings.HasSuffix(path, ".proto")
			}},
			{dir: filepath.Join(exposureDir, "workflow"), protocolType: protocolWorkflow, matcher: nil},
			{dir: filepath.Join(exposureDir, "mcp-tools.json"), protocolType: protocolMCP, matcher: nil},
			{dir: filepath.Join(exposureDir, "agent-streams"), protocolType: protocolAgentStream, matcher: nil},
			{dir: agentSDKDir, protocolType: "sdk", matcher: nil},
		},
	}

	return &manager{
		cfg:           cfg,
		logger:        log,
		catalogLoader: loader,
		assetExporter: exporter,
	}
}

func (m *manager) ListCapabilities(ctx context.Context) ([]CatalogEntry, error) {
	catalog, err := m.catalogLoader.LoadCatalog(ctx)
	if err != nil {
		return nil, err
	}
	if err := normalizeEntries(catalog.Entries); err != nil {
		return nil, err
	}
	return catalog.Entries, nil
}

func (m *manager) ExportProtocols(ctx context.Context) ([]ProtocolAsset, error) {
	catalog, err := m.catalogLoader.LoadCatalog(ctx)
	if err != nil {
		return nil, err
	}
	return m.assetExporter.Export(ctx, catalog)
}

func (m *manager) RegisterWithHost(ctx context.Context, client HostSyncClient) error {
	if client == nil {
		return errors.New("host sync client is nil")
	}
	catalog, err := m.catalogLoader.LoadCatalog(ctx)
	if err != nil {
		return err
	}
	if err := normalizeEntries(catalog.Entries); err != nil {
		return err
	}
	assets, err := m.assetExporter.Export(ctx, catalog)
	if err != nil {
		return err
	}
	return client.RegisterCatalog(ctx, catalog, assets)
}

// fileSystemCatalogLoader loads catalog snapshots from disk.
type fileSystemCatalogLoader struct {
	path string
}

func (l *fileSystemCatalogLoader) LoadCatalog(ctx context.Context) (*CatalogSnapshot, error) {
	log := logger.WithField("component", "capability_catalog_loader")
	p := strings.TrimSpace(l.path)
	if p == "" {
		p = defaultCatalogPath
	}
	if !filepath.IsAbs(p) {
		cwd, _ := os.Getwd()
		p = filepath.Join(cwd, p)
	}
	log = log.WithField("catalog_path", p)
	log.Debug("loading capability catalog from disk")
	data, err := os.ReadFile(p)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			log.WithError(err).Warn("capability catalog not found, falling back to plugin.yaml capabilities")
			snapshot, fallbackErr := loadCatalogFromManifest(log)
			if fallbackErr != nil {
				log.WithError(fallbackErr).Warn("failed to load capability catalog from plugin.yaml, returning empty snapshot")
				return &CatalogSnapshot{GeneratedAt: time.Now().UTC().Format(time.RFC3339), Entries: []CatalogEntry{}}, nil
			}
			return snapshot, nil
		}
		log.WithError(err).Error("failed to read capability catalog")
		return nil, fmt.Errorf("capabilities: failed to read catalog %s: %w", p, err)
	}
	var snapshot CatalogSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		log.WithError(err).Error("failed to parse capability catalog JSON")
		return nil, fmt.Errorf("capabilities: invalid catalog JSON %s: %w", p, err)
	}
	if strings.TrimSpace(snapshot.GeneratedAt) == "" {
		snapshot.GeneratedAt = time.Now().UTC().Format(time.RFC3339)
	}
	if snapshot.Entries == nil {
		snapshot.Entries = []CatalogEntry{}
	}
	log.WithFields(logrus.Fields{
		"entry_count":  len(snapshot.Entries),
		"import_count": len(snapshot.Imports),
		"generated_at": snapshot.GeneratedAt,
		"plugin_id":    snapshot.PluginID,
		"manifest_ver": snapshot.ManifestVersion,
	}).Info("capability catalog loaded")
	return &snapshot, nil
}

type manifestCapability struct {
	ID         string            `yaml:"id"`
	Version    string            `yaml:"version"`
	Descriptor string            `yaml:"descriptor"`
	Schemas    map[string]string `yaml:"schemas"`
	Protocols  map[string]any    `yaml:"protocols"`
	Tags       []string          `yaml:"tags"`
	Execution  struct {
		Mode           string `yaml:"mode"`
		CallbackURL    string `yaml:"callback_url"`
		SSEChannel     string `yaml:"sse_channel"`
		StatusEndpoint string `yaml:"status_endpoint"`
		TimeoutSeconds int    `yaml:"timeout_seconds"`
	} `yaml:"execution"`
}

type manifestDoc struct {
	ID           string `yaml:"id"`
	Version      string `yaml:"version"`
	Capabilities struct {
		Imports  []string            `yaml:"imports"`
		Provides []manifestCapability `yaml:"provides"`
	} `yaml:"capabilities"`
}

func loadCatalogFromManifest(log *logrus.Entry) (*CatalogSnapshot, error) {
	manifestPath := resolveManifestPath()
	if manifestPath == "" {
		return nil, errors.New("manifest path not found")
	}
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("read manifest %s: %w", manifestPath, err)
	}
	var doc manifestDoc
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("parse manifest %s: %w", manifestPath, err)
	}
	manifestDir := filepath.Dir(manifestPath)
	now := time.Now().UTC().Format(time.RFC3339)
	snapshot := &CatalogSnapshot{
		PluginID:        strings.TrimSpace(doc.ID),
		ManifestVersion: strings.TrimSpace(doc.Version),
		GeneratedAt:     now,
		Imports:         doc.Capabilities.Imports,
		Entries:         []CatalogEntry{},
	}

	// Prefer manifest.capabilities.provides for out-of-the-box dev experience.
	for _, cap := range doc.Capabilities.Provides {
		if strings.TrimSpace(cap.ID) == "" {
			continue
		}
		version := strings.TrimSpace(cap.Version)
		if version == "" {
			version = snapshot.ManifestVersion
		}
		execMode := strings.ToLower(strings.TrimSpace(cap.Execution.Mode))
		if execMode == "" {
			execMode = "sync"
		}

		payload, _ := yaml.Marshal(cap)
		checksum := fmt.Sprintf("%x", sha256.Sum256(payload))

		entry := CatalogEntry{
			ID:         cap.ID,
			Version:    version,
			Descriptor: strings.TrimSpace(cap.Descriptor),
			Schemas:    cap.Schemas,
			Protocols:  cap.Protocols,
			Tags:       cap.Tags,
			Execution: ExecutionConfig{
				Mode:           execMode,
				CallbackURL:    cap.Execution.CallbackURL,
				SSEChannel:     cap.Execution.SSEChannel,
				StatusEndpoint: cap.Execution.StatusEndpoint,
				TimeoutSeconds: cap.Execution.TimeoutSeconds,
			},
			Checksum: checksum,
		}
		if entry.Schemas == nil {
			entry.Schemas = map[string]string{}
		}
		if entry.Protocols == nil {
			entry.Protocols = map[string]any{}
		}
		if entry.Tags == nil {
			entry.Tags = []string{}
		}
		snapshot.Entries = append(snapshot.Entries, entry)
	}

	// If provides are empty but imports exist, load descriptors from the filesystem like scripts/capabilities/catalog-parser.ts.
	if len(snapshot.Entries) == 0 && len(snapshot.Imports) > 0 {
		for _, importPath := range snapshot.Imports {
			rel := strings.TrimSpace(importPath)
			if rel == "" {
				continue
			}
			abs := rel
			if !filepath.IsAbs(abs) {
				abs = filepath.Join(manifestDir, filepath.FromSlash(rel))
			}
			rawDesc, err := os.ReadFile(abs)
			if err != nil {
				if log != nil {
					log.WithError(err).Warnf("capability import missing: %s", rel)
				}
				continue
			}
			var desc manifestCapability
			if err := yaml.Unmarshal(rawDesc, &desc); err != nil {
				if log != nil {
					log.WithError(err).Warnf("capability import invalid yaml: %s", rel)
				}
				continue
			}
			if strings.TrimSpace(desc.ID) == "" {
				continue
			}
			version := strings.TrimSpace(desc.Version)
			if version == "" {
				version = snapshot.ManifestVersion
			}
			execMode := strings.ToLower(strings.TrimSpace(desc.Execution.Mode))
			if execMode == "" {
				execMode = "sync"
			}
			checksum := fmt.Sprintf("%x", sha256.Sum256(rawDesc))
			entry := CatalogEntry{
				ID:         desc.ID,
				Version:    version,
				Descriptor: rel,
				Schemas:    desc.Schemas,
				Protocols:  desc.Protocols,
				Tags:       desc.Tags,
				Execution: ExecutionConfig{
					Mode:           execMode,
					CallbackURL:    desc.Execution.CallbackURL,
					SSEChannel:     desc.Execution.SSEChannel,
					StatusEndpoint: desc.Execution.StatusEndpoint,
					TimeoutSeconds: desc.Execution.TimeoutSeconds,
				},
				Checksum: checksum,
			}
			if entry.Schemas == nil {
				entry.Schemas = map[string]string{}
			}
			if entry.Protocols == nil {
				entry.Protocols = map[string]any{}
			}
			if entry.Tags == nil {
				entry.Tags = []string{}
			}
			snapshot.Entries = append(snapshot.Entries, entry)
		}
	}

	if log != nil {
		log.WithFields(logrus.Fields{
			"manifest_path": manifestPath,
			"entry_count":   len(snapshot.Entries),
			"plugin_id":     snapshot.PluginID,
			"manifest_ver":  snapshot.ManifestVersion,
		}).Info("capability catalog loaded from plugin.yaml fallback")
	}
	return snapshot, nil
}

func resolveManifestPath() string {
	if p := strings.TrimSpace(os.Getenv("POWERX_PLUGIN_MANIFEST")); p != "" {
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			return p
		}
	}
	cwd, _ := os.Getwd()
	candidates := []string{
		filepath.Join(cwd, "plugin.yaml"),
		filepath.Join(cwd, "skeleton", "plugin.yaml"),
		filepath.Join(cwd, "..", "plugin.yaml"),
		filepath.Join(cwd, "..", "..", "plugin.yaml"),
	}
	for _, cand := range candidates {
		if info, err := os.Stat(cand); err == nil && !info.IsDir() {
			return cand
		}
	}
	return ""
}

// fileSystemAssetExporter walks known roots and emits protocol assets.
type fileSystemAssetExporter struct {
	roots []assetRoot
}

type assetRoot struct {
	dir          string
	protocolType string
	matcher      func(path string) bool
}

func (e *fileSystemAssetExporter) Export(ctx context.Context, catalog *CatalogSnapshot) ([]ProtocolAsset, error) {
	var assets []ProtocolAsset
	for _, root := range e.roots {
		if strings.TrimSpace(root.dir) == "" {
			continue
		}
		files, err := walkFiles(root.dir, root.matcher)
		if err != nil {
			return nil, err
		}
		for _, f := range files {
			assets = append(assets, ProtocolAsset{Type: root.protocolType, Path: f})
		}
	}
	return assets, nil
}

func walkFiles(dir string, matcher func(string) bool) ([]string, error) {
	var results []string
	root := filepath.Clean(dir)
	cwd, _ := os.Getwd()
	absRoot := root
	if !filepath.IsAbs(absRoot) && cwd != "" {
		absRoot = filepath.Join(cwd, root)
	}
	err := filepath.WalkDir(absRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel := path
		if cwd != "" {
			if r, err := filepath.Rel(cwd, path); err == nil {
				rel = r
			}
		}
		rel = filepath.ToSlash(rel)
		if matcher != nil && !matcher(rel) {
			return nil
		}
		results = append(results, rel)
		return nil
	})
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("capabilities: walk %s failed: %w", dir, err)
	}
	return results, nil
}

// ValidateExecution enforces sync/async constraints and populates defaults.
func ValidateExecution(entries []CatalogEntry) error {
	return normalizeEntries(entries)
}

func normalizeEntries(entries []CatalogEntry) error {
	for i := range entries {
		mode := strings.TrimSpace(strings.ToLower(entries[i].Execution.Mode))
		if mode == "" {
			entries[i].Execution.Mode = "sync"
			continue
		}
		switch mode {
		case "sync":
			entries[i].Execution.Mode = "sync"
		case "async":
			if strings.TrimSpace(entries[i].Execution.CallbackURL) == "" && strings.TrimSpace(entries[i].Execution.SSEChannel) == "" {
				return fmt.Errorf("capabilities: async capability %s missing callback or SSE channel", entries[i].ID)
			}
			if strings.TrimSpace(entries[i].Execution.StatusEndpoint) == "" {
				return fmt.Errorf("capabilities: async capability %s missing status_endpoint", entries[i].ID)
			}
			entries[i].Execution.Mode = "async"
		default:
			return fmt.Errorf("capabilities: capability %s has invalid execution mode %q", entries[i].ID, entries[i].Execution.Mode)
		}
	}
	return nil
}

// EnsureManager warms up capability catalog during startup and logs the result.
func EnsureManager(ctx context.Context, mgr Manager, log *logrus.Entry) error {
	if mgr == nil {
		return nil
	}
	if _, err := mgr.ListCapabilities(ctx); err != nil {
		return err
	}
	if _, err := mgr.ExportProtocols(ctx); err != nil {
		return err
	}
	if log != nil {
		log.Debug("capabilities: catalog verified")
	}
	return nil
}

func managerLogger(component string) *logrus.Entry {
	return logger.WithField("component", component)
}

func resolveCatalogPath(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	if strings.Contains(trimmed, "$(pwd)") {
		if cwd, err := os.Getwd(); err == nil {
			trimmed = strings.ReplaceAll(trimmed, "$(pwd)", cwd)
		}
	}
	trimmed = os.ExpandEnv(trimmed)
	return strings.TrimSpace(trimmed)
}
