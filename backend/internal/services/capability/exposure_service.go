package capability

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/logger"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

const defaultExposureFile = "../../contracts/exposure/exposure-packages.json"

var (
	errCapabilityRequired = errors.New("capability id required")
	errChannelRequired    = errors.New("至少需要启用一个通道")
)

// ExposureService stores channel configuration and tenant quotas for capabilities.
type ExposureService struct {
	logger      *logrus.Entry
	mu          sync.RWMutex
	packages    map[string]*ExposurePackage
	storagePath string
}

// ExposurePackage defines multi-protocol exposure settings for a capability.
type ExposurePackage struct {
	CapabilityID string            `json:"capability_id"`
	Channels     []ExposureChannel `json:"channels"`
	Auth         AuthConfig        `json:"auth"`
	RateLimit    RateLimitConfig   `json:"rate_limit"`
	Tenants      []TenantQuota     `json:"tenants"`
	SyncStatus   string            `json:"sync_status"`
	DocsVersion  string            `json:"docs_version"`
	SDKVersion   string            `json:"sdk_version"`
	UpdatedBy    string            `json:"updated_by"`
	UpdatedAt    time.Time         `json:"updated_at"`
}

// ExposureChannel records protocol specific configuration.
type ExposureChannel struct {
	Type        string   `json:"type"`
	Name        string   `json:"name"`
	Method      string   `json:"method,omitempty"`
	Path        string   `json:"path,omitempty"`
	Target      string   `json:"target,omitempty"`
	Description string   `json:"description,omitempty"`
	Enabled     bool     `json:"enabled"`
	Scopes      []string `json:"scopes,omitempty"`
}

// AuthConfig represents the exposure authentication plan.
type AuthConfig struct {
	Strategy string   `json:"strategy"`
	Audience string   `json:"audience,omitempty"`
	Scopes   []string `json:"scopes,omitempty"`
}

// RateLimitConfig controls throttling behaviour.
type RateLimitConfig struct {
	RequestsPerMinute int `json:"requests_per_minute"`
	Burst             int `json:"burst"`
	Concurrency       int `json:"concurrency"`
}

// TenantQuota stores per-tenant authorization data.
type TenantQuota struct {
	ID        string `json:"tenant_id"`
	Name      string `json:"tenant_name"`
	Quota     int    `json:"quota"`
	Used      int    `json:"used"`
	Status    string `json:"status"`
	UpdatedAt string `json:"updated_at"`
	Notes     string `json:"notes,omitempty"`
}

// ExposureTemplate provides defaults for the web form.
type ExposureTemplate struct {
	ChannelTypes   []string        `json:"channel_types"`
	AuthStrategies []string        `json:"auth_strategies"`
	DefaultRate    RateLimitConfig `json:"default_rate"`
}

// ExposureInput is accepted from HTTP handlers.
type ExposureInput struct {
	CapabilityID string            `json:"capability_id"`
	Channels     []ExposureChannel `json:"channels"`
	Auth         AuthConfig        `json:"auth"`
	RateLimit    RateLimitConfig   `json:"rate_limit"`
	Tenants      []TenantQuota     `json:"tenants"`
	DocsVersion  string            `json:"docs_version"`
	SDKVersion   string            `json:"sdk_version"`
	Actor        string            `json:"actor"`
}

// NewExposureService builds a service instance backed by JSON storage.
func NewExposureService(deps *app.Deps) *ExposureService {
	var log *logrus.Entry
	if deps != nil {
		log = deps.RuntimeLogger(deps.Ctx, "capability_exposure_service", nil)
	}
	if log == nil {
		log = logger.WithRuntimeFields(app.PluginID, "", "", "capability_exposure_service", nil)
	}
	svc := &ExposureService{
		logger:      log,
		packages:    map[string]*ExposurePackage{},
		storagePath: resolveExposureStorage(),
	}
	svc.loadFromDisk()
	return svc
}

// Template returns default selectable options.
func (s *ExposureService) Template() *ExposureTemplate {
	return &ExposureTemplate{
		ChannelTypes: []string{"rest", "graphql", "grpc", "webhook", "workflow", "agent", "agent_sse", "sdk"},
		AuthStrategies: []string{
			"powerx_session",
			"oauth_client_credentials",
			"api_key",
		},
		DefaultRate: RateLimitConfig{
			RequestsPerMinute: 600,
			Burst:             120,
			Concurrency:       10,
		},
	}
}

// Upsert persists an exposure package.
func (s *ExposureService) Upsert(ctx context.Context, input *ExposureInput) (*ExposurePackage, error) {
	if input == nil || strings.TrimSpace(input.CapabilityID) == "" {
		return nil, errCapabilityRequired
	}
	channels := normalizeChannels(input.Channels)
	if len(channels) == 0 {
		return nil, errChannelRequired
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	pkg := &ExposurePackage{
		CapabilityID: strings.TrimSpace(input.CapabilityID),
		Channels:     channels,
		Auth:         normalizeAuth(input.Auth),
		RateLimit:    normalizeRateLimit(input.RateLimit),
		Tenants:      normalizeTenants(input.Tenants),
		SyncStatus:   "pending",
		DocsVersion:  fallbackVersion(input.DocsVersion),
		SDKVersion:   fallbackVersion(input.SDKVersion),
		UpdatedBy:    strings.TrimSpace(input.Actor),
		UpdatedAt:    time.Now().UTC(),
	}
	s.packages[pkg.CapabilityID] = pkg
	if err := s.persistLocked(); err != nil {
		return nil, err
	}

	s.logger.WithFields(logrus.Fields{
		"capability_id": pkg.CapabilityID,
		"channels":      len(pkg.Channels),
	}).Info("capability exposure package updated")

	return cloneExposure(pkg), nil
}

// Get returns the exposure package for the capability.
func (s *ExposureService) Get(_ context.Context, capabilityID string) (*ExposurePackage, error) {
	if strings.TrimSpace(capabilityID) == "" {
		return nil, errCapabilityRequired
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if pkg, ok := s.packages[capabilityID]; ok {
		return cloneExposure(pkg), nil
	}
	return nil, nil
}

// UpdateQuota adjusts a tenant quota entry.
func (s *ExposureService) UpdateQuota(ctx context.Context, capabilityID string, quota TenantQuota) (*ExposurePackage, error) {
	if strings.TrimSpace(capabilityID) == "" {
		return nil, errCapabilityRequired
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	pkg, ok := s.packages[capabilityID]
	if !ok {
		return nil, errors.New("exposure package not found")
	}
	quota.ID = strings.TrimSpace(quota.ID)
	if quota.ID == "" {
		return nil, errors.New("tenant_id required")
	}
	quota.Status = normalizeStatus(quota.Status)
	quota.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	replaced := false
	for idx, existing := range pkg.Tenants {
		if strings.EqualFold(existing.ID, quota.ID) {
			pkg.Tenants[idx] = quota
			replaced = true
			break
		}
	}
	if !replaced {
		pkg.Tenants = append(pkg.Tenants, quota)
	}
	pkg.UpdatedAt = time.Now().UTC()
	if err := s.persistLocked(); err != nil {
		return nil, err
	}

	s.logger.WithFields(logrus.Fields{
		"capability_id": capabilityID,
		"tenant_id":     quota.ID,
		"quota":         quota.Quota,
	}).Info("capability quota updated")

	return cloneExposure(pkg), nil
}

// ListQuotas returns tenant quota entries.
func (s *ExposureService) ListQuotas(capabilityID string) []TenantQuota {
	s.mu.RLock()
	defer s.mu.RUnlock()
	pkg, ok := s.packages[capabilityID]
	if !ok {
		return nil
	}
	out := make([]TenantQuota, len(pkg.Tenants))
	copy(out, pkg.Tenants)
	return out
}

func (s *ExposureService) loadFromDisk() {
	data, err := os.ReadFile(s.storagePath)
	if err != nil {
		if !os.IsNotExist(err) {
			s.logger.WithError(err).Warn("failed to read exposure packages")
		}
		return
	}
	var snapshot exposureSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		s.logger.WithError(err).Warn("failed to parse exposure packages")
		return
	}
	for _, pkg := range snapshot.Packages {
		if pkg == nil || strings.TrimSpace(pkg.CapabilityID) == "" {
			continue
		}
		s.packages[pkg.CapabilityID] = pkg
	}
}

func (s *ExposureService) persistLocked() error {
	if len(s.packages) == 0 {
		return nil
	}
	pkgs := make([]*ExposurePackage, 0, len(s.packages))
	for _, pkg := range s.packages {
		if pkg == nil {
			continue
		}
		clone := *pkg
		pkgs = append(pkgs, &clone)
	}
	sort.Slice(pkgs, func(i, j int) bool {
		return strings.Compare(pkgs[i].CapabilityID, pkgs[j].CapabilityID) < 0
	})
	snapshot := exposureSnapshot{
		Packages:  pkgs,
		UpdatedAt: time.Now().UTC(),
	}
	if err := os.MkdirAll(filepath.Dir(s.storagePath), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.storagePath, data, 0o644)
}

func normalizeChannels(channels []ExposureChannel) []ExposureChannel {
	var out []ExposureChannel
	seen := map[string]struct{}{}
	for _, ch := range channels {
		if !ch.Enabled {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(ch.Type))
		if key == "" {
			continue
		}
		name := strings.TrimSpace(ch.Name)
		if name == "" {
			name = strings.ToUpper(key)
		}
		channel := ExposureChannel{
			Type:        key,
			Name:        name,
			Method:      strings.ToUpper(strings.TrimSpace(ch.Method)),
			Path:        strings.TrimSpace(ch.Path),
			Target:      strings.TrimSpace(ch.Target),
			Description: strings.TrimSpace(ch.Description),
			Enabled:     true,
			Scopes:      trimAll(ch.Scopes),
		}
		if _, exists := seen[key+"|"+channel.Method+channel.Path]; exists {
			continue
		}
		seen[key+"|"+channel.Method+channel.Path] = struct{}{}
		out = append(out, channel)
	}
	return out
}

func normalizeAuth(config AuthConfig) AuthConfig {
	strategy := strings.TrimSpace(config.Strategy)
	if strategy == "" {
		strategy = "powerx_session"
	}
	return AuthConfig{
		Strategy: strings.ToLower(strategy),
		Audience: strings.TrimSpace(config.Audience),
		Scopes:   trimAll(config.Scopes),
	}
}

func normalizeRateLimit(limit RateLimitConfig) RateLimitConfig {
	if limit.RequestsPerMinute <= 0 {
		limit.RequestsPerMinute = 600
	}
	if limit.Burst <= 0 {
		limit.Burst = 120
	}
	if limit.Concurrency <= 0 {
		limit.Concurrency = 10
	}
	return limit
}

func normalizeTenants(tenants []TenantQuota) []TenantQuota {
	if len(tenants) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]TenantQuota, 0, len(tenants))
	for _, t := range tenants {
		id := strings.TrimSpace(t.ID)
		if id == "" {
			id = strings.TrimSpace(t.Name)
		}
		if id == "" {
			id = uuid.NewString()
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		t.ID = id
		t.Name = strings.TrimSpace(t.Name)
		if t.Quota < 0 {
			t.Quota = 0
		}
		if t.Used < 0 {
			t.Used = 0
		}
		t.Status = normalizeStatus(t.Status)
		if strings.TrimSpace(t.UpdatedAt) == "" {
			t.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
		}
		out = append(out, t)
	}
	return out
}

func normalizeStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "active", "":
		return "active"
	case "suspended":
		return "suspended"
	default:
		return "active"
	}
}

func fallbackVersion(version string) string {
	ver := strings.TrimSpace(version)
	if ver == "" {
		return "1.0.0"
	}
	return ver
}

func trimAll(items []string) []string {
	var out []string
	seen := map[string]struct{}{}
	for _, item := range items {
		s := strings.TrimSpace(item)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

func cloneExposure(pkg *ExposurePackage) *ExposurePackage {
	if pkg == nil {
		return nil
	}
	cp := *pkg
	if pkg.Channels != nil {
		cp.Channels = make([]ExposureChannel, len(pkg.Channels))
		copy(cp.Channels, pkg.Channels)
	}
	if pkg.Tenants != nil {
		cp.Tenants = make([]TenantQuota, len(pkg.Tenants))
		copy(cp.Tenants, pkg.Tenants)
	}
	return &cp
}

type exposureSnapshot struct {
	Packages  []*ExposurePackage `json:"packages"`
	UpdatedAt time.Time          `json:"updated_at"`
}

func resolveExposureStorage() string {
	workDir, err := os.Getwd()
	if err != nil {
		return defaultExposureFile
	}
	candidate := filepath.Clean(filepath.Join(workDir, defaultExposureFile))
	return candidate
}
