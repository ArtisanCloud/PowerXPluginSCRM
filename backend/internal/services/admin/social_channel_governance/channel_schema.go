package social_channel_governance

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

type ChannelFieldSchema struct {
	Key         string `yaml:"key" json:"key"`
	Label       string `yaml:"label" json:"label"`
	Required    bool   `yaml:"required" json:"required"`
	Span        int    `yaml:"span,omitempty" json:"span,omitempty"`
	Placeholder string `yaml:"placeholder,omitempty" json:"placeholder,omitempty"`
	Hint        string `yaml:"hint,omitempty" json:"hint,omitempty"`
	InputType   string `yaml:"input_type,omitempty" json:"input_type,omitempty"`
	Hidden      bool   `yaml:"hidden,omitempty" json:"hidden,omitempty"`
	DerivedFrom string `yaml:"derived_from,omitempty" json:"derived_from,omitempty"`
	DefaultValue string `yaml:"default_value,omitempty" json:"default_value,omitempty"`
}

type ChannelAppTypeSchema struct {
	Code   string               `yaml:"code" json:"code"`
	Label  string               `yaml:"label" json:"label"`
	Fields []ChannelFieldSchema `yaml:"fields" json:"fields"`
}

type ChannelSchema struct {
	Code     string                 `yaml:"code" json:"code"`
	Label    string                 `yaml:"label" json:"label"`
	AppTypes []ChannelAppTypeSchema `yaml:"app_types" json:"app_types"`
}

type ChannelSchemaDocument struct {
	Version  int             `yaml:"version" json:"version"`
	Channels []ChannelSchema `yaml:"channels" json:"channels"`
}

type ChannelSchemaLoader struct {
	logger   *logrus.Entry
	cacheTTL time.Duration
	path     string

	mu     sync.RWMutex
	cache  *ChannelSchemaDocument
	expiry time.Time
}

type ChannelSchemaLoaderOptions struct {
	Path     string
	CacheTTL time.Duration
	Logger   *logrus.Entry
}

func NewChannelSchemaLoader(opts ChannelSchemaLoaderOptions) *ChannelSchemaLoader {
	cacheTTL := opts.CacheTTL
	if cacheTTL <= 0 {
		cacheTTL = time.Minute
	}
	return &ChannelSchemaLoader{
		logger:   opts.Logger,
		cacheTTL: cacheTTL,
		path:     strings.TrimSpace(opts.Path),
	}
}

func (l *ChannelSchemaLoader) Load(ctx context.Context) (*ChannelSchemaDocument, error) {
	_ = ctx
	l.mu.RLock()
	if l.cache != nil && time.Now().Before(l.expiry) {
		cached := cloneSchemaDocument(l.cache)
		l.mu.RUnlock()
		return cached, nil
	}
	l.mu.RUnlock()

	l.mu.Lock()
	defer l.mu.Unlock()
	if l.cache != nil && time.Now().Before(l.expiry) {
		return cloneSchemaDocument(l.cache), nil
	}

	path, fsys := resolveChannelSchemaSource(l.path)
	if fsys == nil {
		return nil, errors.New("channel schema not configured")
	}
	raw, err := fs.ReadFile(fsys, path)
	if err != nil {
		return nil, fmt.Errorf("read channel schema yaml: %w", err)
	}
	var doc ChannelSchemaDocument
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("parse channel schema yaml: %w", err)
	}
	normalizeSchema(&doc)
	l.cache = &doc
	l.expiry = time.Now().Add(l.cacheTTL)
	if l.logger != nil {
		l.logger.WithField("schema_path", path).Info("social channel schema loaded")
	}
	return cloneSchemaDocument(&doc), nil
}

func cloneSchemaDocument(doc *ChannelSchemaDocument) *ChannelSchemaDocument {
	if doc == nil {
		return nil
	}
	out := *doc
	out.Channels = make([]ChannelSchema, len(doc.Channels))
	for i, ch := range doc.Channels {
		out.Channels[i] = cloneChannelSchema(ch)
	}
	return &out
}

func cloneChannelSchema(ch ChannelSchema) ChannelSchema {
	out := ch
	out.AppTypes = make([]ChannelAppTypeSchema, len(ch.AppTypes))
	for i, app := range ch.AppTypes {
		out.AppTypes[i] = cloneAppTypeSchema(app)
	}
	return out
}

func cloneAppTypeSchema(app ChannelAppTypeSchema) ChannelAppTypeSchema {
	out := app
	out.Fields = make([]ChannelFieldSchema, len(app.Fields))
	copy(out.Fields, app.Fields)
	return out
}

func normalizeSchema(doc *ChannelSchemaDocument) {
	if doc == nil {
		return
	}
	for ci := range doc.Channels {
		doc.Channels[ci].Code = strings.ToLower(strings.TrimSpace(doc.Channels[ci].Code))
		for ai := range doc.Channels[ci].AppTypes {
			app := &doc.Channels[ci].AppTypes[ai]
			app.Code = strings.ToLower(strings.TrimSpace(app.Code))
			for fi := range app.Fields {
				field := &app.Fields[fi]
				field.Key = strings.ToLower(strings.TrimSpace(field.Key))
			}
		}
	}
}

func resolveChannelSchemaSource(explicitPath string) (string, fs.FS) {
	path := strings.TrimSpace(explicitPath)
	if path != "" {
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			return "social_channel_governance/channel_schema.yaml", os.DirFS(path)
		}
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			dir := filepath.Dir(path)
			return filepath.Base(path), os.DirFS(dir)
		}
	}

	candidates := []string{
		resolveEnvSchemaPath("POWERX_SOCIAL_CHANNEL_SCHEMA"),
		filepath.Join("config", "social_channel_governance", "channel_schema.yaml"),
		filepath.Join("etc", "social_channel_governance", "channel_schema.yaml"),
		filepath.Join("backend", "etc", "social_channel_governance", "channel_schema.yaml"),
		filepath.Join("..", "backend", "etc", "social_channel_governance", "channel_schema.yaml"),
		filepath.Join("..", "etc", "social_channel_governance", "channel_schema.yaml"),
		filepath.Join("skeleton", "backend", "etc", "social_channel_governance", "channel_schema.yaml"),
		filepath.Join("social_channel_governance", "channel_schema.yaml"),
	}
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		info, err := os.Stat(candidate)
		if err != nil || info.IsDir() {
			continue
		}
		dir := filepath.Dir(candidate)
		return filepath.Base(candidate), os.DirFS(dir)
	}
	return "", nil
}

func resolveEnvSchemaPath(key string) string {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return ""
	}
	return raw
}

func findAppSchema(doc *ChannelSchemaDocument, channel, appType string) (*ChannelSchema, *ChannelAppTypeSchema) {
	if doc == nil {
		return nil, nil
	}
	channel = strings.ToLower(strings.TrimSpace(channel))
	appType = strings.ToLower(strings.TrimSpace(appType))
	for i := range doc.Channels {
		ch := &doc.Channels[i]
		if ch.Code != channel {
			continue
		}
		for j := range ch.AppTypes {
			app := &ch.AppTypes[j]
			if app.Code == appType {
				return ch, app
			}
		}
		return ch, nil
	}
	return nil, nil
}

func applyDerivedFields(app *ChannelAppTypeSchema, values map[string]string) {
	if app == nil {
		return
	}
	for _, field := range app.Fields {
		if field.DerivedFrom == "" {
			continue
		}
		targetKey := field.Key
		if values[targetKey] != "" {
			continue
		}
		source := values[field.DerivedFrom]
		if source == "" {
			continue
		}
		values[targetKey] = source
	}
}

func validateRequiredFields(app *ChannelAppTypeSchema, values map[string]string) error {
	if app == nil {
		return errors.New("unsupported app_type for channel")
	}
	for _, field := range app.Fields {
		if !field.Required {
			continue
		}
		if strings.TrimSpace(values[field.Key]) == "" {
			if field.Key == "" {
				return errors.New("missing required field")
			}
			return fmt.Errorf("%s is required", field.Key)
		}
	}
	return nil
}
