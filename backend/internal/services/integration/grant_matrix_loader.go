package integration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	model "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/integration"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// GrantMatrixSource 描述策略来源。
const (
	GrantMatrixSourceStatic   = "STATIC"
	GrantMatrixSourceOverride = "OVERRIDE"
)

// GrantMatrixEntry 代表合并后的策略条目。
type GrantMatrixEntry struct {
	Scope       string                 `json:"scope" yaml:"scope"`
	Channel     string                 `json:"channel" yaml:"channel"`
	Resource    string                 `json:"resource" yaml:"resource"`
	Action      string                 `json:"action" yaml:"action"`
	Constraints map[string]any         `json:"constraints" yaml:"constraints"`
	Source      string                 `json:"source" yaml:"source"`
	Version     int                    `json:"version" yaml:"version"`
	ApprovedBy  string                 `json:"approved_by,omitempty" yaml:"approved_by,omitempty"`
	ApprovedAt  *time.Time             `json:"approved_at,omitempty" yaml:"approved_at,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

// LoaderOptions 自定义 Loader 行为。
type LoaderOptions struct {
	StaticFS   fs.FS
	StaticPath string
	CacheTTL   time.Duration
}

// GrantMatrixLoader 负责合并静态配置与数据库覆盖。
type GrantMatrixLoader struct {
	db        *gorm.DB
	logger    *logrus.Entry
	staticFS  fs.FS
	staticKey string
	cacheTTL  time.Duration

	mu     sync.RWMutex
	cache  []GrantMatrixEntry
	expiry time.Time
}

// NewGrantMatrixLoader 构造一个带缓存的 GrantMatrix 加载器。
func NewGrantMatrixLoader(db *gorm.DB, logger *logrus.Entry, opts LoaderOptions) *GrantMatrixLoader {
	staticFS := opts.StaticFS
	if staticFS == nil {
		basePath := resolveGrantMatrixBaseDir()
		if basePath != "" {
			staticFS = os.DirFS(basePath)
		}
	}

	staticPath := opts.StaticPath
	if strings.TrimSpace(staticPath) == "" {
		staticPath = "integration/grant_matrix.yaml"
	}

	cacheTTL := opts.CacheTTL
	if cacheTTL <= 0 {
		cacheTTL = time.Minute
	}

	l := &GrantMatrixLoader{
		db:        db,
		logger:    logger,
		staticFS:  staticFS,
		staticKey: staticPath,
		cacheTTL:  cacheTTL,
	}
	return l
}

// Load 返回合并后的 GrantMatrix 列表（带缓存）。
func (l *GrantMatrixLoader) Load(ctx context.Context) ([]GrantMatrixEntry, error) {
	l.mu.RLock()
	if l.cache != nil && time.Now().Before(l.expiry) {
		entries := cloneEntries(l.cache)
		l.mu.RUnlock()
		return entries, nil
	}
	l.mu.RUnlock()

	return l.reload(ctx)
}

// Invalidate 主动使缓存失效。
func (l *GrantMatrixLoader) Invalidate() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.cache = nil
	l.expiry = time.Time{}
}

// Warm 预热缓存，忽略错误仅记录日志。
func (l *GrantMatrixLoader) Warm(ctx context.Context) {
	if _, err := l.reload(ctx); err != nil && l.logger != nil {
		l.logger.WithError(err).Warn("failed to warm grant matrix cache")
	}
}

func (l *GrantMatrixLoader) reload(ctx context.Context) ([]GrantMatrixEntry, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// 双重检查，避免重复加载。
	if l.cache != nil && time.Now().Before(l.expiry) {
		return cloneEntries(l.cache), nil
	}

	staticEntries, err := l.loadStaticEntries()
	if err != nil {
		return nil, err
	}

	overrideEntries, err := l.loadOverrideEntries(ctx)
	if err != nil {
		return nil, err
	}

	combined := mergeGrantMatrix(staticEntries, overrideEntries)
	sort.Slice(combined, func(i, j int) bool {
		left, right := combined[i], combined[j]
		if !strings.EqualFold(left.Scope, right.Scope) {
			return strings.Compare(left.Scope, right.Scope) < 0
		}
		if !strings.EqualFold(left.Channel, right.Channel) {
			return strings.Compare(left.Channel, right.Channel) < 0
		}
		if left.Resource != right.Resource {
			return left.Resource < right.Resource
		}
		return left.Action < right.Action
	})

	l.cache = combined
	l.expiry = time.Now().Add(l.cacheTTL)
	return cloneEntries(combined), nil
}

func (l *GrantMatrixLoader) loadStaticEntries() ([]GrantMatrixEntry, error) {
	if l.staticFS == nil || strings.TrimSpace(l.staticKey) == "" {
		return nil, nil
	}
	data, err := fs.ReadFile(l.staticFS, l.staticKey)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read grant matrix yaml: %w", err)
	}

	entries, err := parseStaticEntries(data)
	if err != nil {
		return nil, err
	}

	for i := range entries {
		entries[i].Source = GrantMatrixSourceStatic
		if entries[i].Version == 0 {
			entries[i].Version = 1
		}
		if entries[i].Constraints == nil {
			entries[i].Constraints = map[string]any{}
		}
	}
	return entries, nil
}

func (l *GrantMatrixLoader) loadOverrideEntries(ctx context.Context) ([]GrantMatrixEntry, error) {
	if l.db == nil {
		return nil, nil
	}

	var rows []model.GrantMatrixOverride
	err := l.db.WithContext(ctx).
		Where("status = ?", "APPROVED").
		Find(&rows).
		Error
	if err != nil {
		return nil, fmt.Errorf("load grant matrix overrides: %w", err)
	}

	result := make([]GrantMatrixEntry, 0, len(rows))
	for _, row := range rows {
		entry := GrantMatrixEntry{
			Scope:      row.Scope,
			Channel:    strings.ToUpper(row.Channel),
			Resource:   row.Resource,
			Action:     strings.ToUpper(row.Action),
			Source:     GrantMatrixSourceOverride,
			Version:    row.Version,
			ApprovedBy: row.ApprovedBy,
			ApprovedAt: row.ApprovedAt,
		}

		if len(row.Constraints) > 0 {
			if constraints, err := decodeJSONMap(row.Constraints); err == nil {
				entry.Constraints = constraints
			} else if l.logger != nil {
				l.logger.WithError(err).WithFields(logrus.Fields{
					"scope":    row.Scope,
					"channel":  row.Channel,
					"resource": row.Resource,
					"action":   row.Action,
				}).Warn("invalid constraints payload on grant matrix override")
			}
		}

		if entry.Constraints == nil {
			entry.Constraints = map[string]any{}
		}

		result = append(result, entry)
	}
	return result, nil
}

func parseStaticEntries(data []byte) ([]GrantMatrixEntry, error) {
	var direct []GrantMatrixEntry
	if err := yaml.Unmarshal(data, &direct); err == nil && len(direct) > 0 {
		return direct, nil
	}

	var wrap struct {
		Entries []GrantMatrixEntry `yaml:"entries"`
	}
	if err := yaml.Unmarshal(data, &wrap); err != nil {
		return nil, fmt.Errorf("parse grant matrix yaml: %w", err)
	}
	return wrap.Entries, nil
}

func mergeGrantMatrix(staticEntries, overrides []GrantMatrixEntry) []GrantMatrixEntry {
	if len(staticEntries) == 0 && len(overrides) == 0 {
		return nil
	}

	combined := make(map[string]GrantMatrixEntry, len(staticEntries)+len(overrides))
	for _, entry := range staticEntries {
		key := entryKey(entry)
		combined[key] = entry
	}

	for _, entry := range overrides {
		key := entryKey(entry)
		combined[key] = entry
	}

	result := make([]GrantMatrixEntry, 0, len(combined))
	for _, entry := range combined {
		result = append(result, entry)
	}
	return result
}

func entryKey(entry GrantMatrixEntry) string {
	return strings.ToLower(strings.Join([]string{
		entry.Scope,
		entry.Channel,
		entry.Resource,
		entry.Action,
	}, "|"))
}

func cloneEntries(src []GrantMatrixEntry) []GrantMatrixEntry {
	if src == nil {
		return nil
	}
	out := make([]GrantMatrixEntry, len(src))
	for i, entry := range src {
		out[i] = GrantMatrixEntry{
			Scope:       entry.Scope,
			Channel:     entry.Channel,
			Resource:    entry.Resource,
			Action:      entry.Action,
			Source:      entry.Source,
			Version:     entry.Version,
			ApprovedBy:  entry.ApprovedBy,
			ApprovedAt:  entry.ApprovedAt,
			Constraints: cloneMap(entry.Constraints),
			Metadata:    cloneMap(entry.Metadata),
		}
	}
	return out
}

func cloneMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func decodeJSONMap(data datatypes.JSON) (map[string]any, error) {
	if len(data) == 0 {
		return map[string]any{}, nil
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	if out == nil {
		out = map[string]any{}
	}
	return out, nil
}

func resolveGrantMatrixBaseDir() string {
	if custom := strings.TrimSpace(os.Getenv("POWERX_GRANT_MATRIX_DIR")); custom != "" {
		if info, err := os.Stat(custom); err == nil && info.IsDir() {
			return custom
		}
	}
	candidates := []string{
		"backend/etc",
		"etc",
		"skeleton/backend/etc",
		"skeleton/etc",
	}
	for _, path := range candidates {
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			return path
		}
	}
	return ""
}
