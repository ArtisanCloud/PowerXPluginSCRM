package security

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"

	"github.com/ArtisanCloud/PowerXPlugin/framework/event"
	fweventbridge "github.com/ArtisanCloud/PowerXPlugin/framework/eventbridge"
)

var ErrEventPermissionDenied = errors.New("event permission denied")

type EventPermissions struct {
	enforced  bool
	publish   map[string]struct{}
	subscribe map[string]struct{}
}

func (p EventPermissions) Enforced() bool { return p.enforced }

func (p EventPermissions) CanPublish(topic string) bool {
	if !p.enforced {
		return true
	}
	_, ok := p.publish[strings.TrimSpace(topic)]
	return ok
}

func (p EventPermissions) CanSubscribe(topic string) bool {
	if !p.enforced {
		return true
	}
	_, ok := p.subscribe[strings.TrimSpace(topic)]
	return ok
}

type pluginManifest struct {
	Events *struct {
		Publish   []string `yaml:"publish"`
		Subscribe []string `yaml:"subscribe"`
	} `yaml:"events"`
}

func LoadEventPermissionsFromManifest(manifestPath string, logger *logrus.Entry) (EventPermissions, error) {
	manifestPath = strings.TrimSpace(manifestPath)
	if manifestPath == "" {
		manifestPath = defaultManifestPath()
	}

	content, err := os.ReadFile(manifestPath)
	if err != nil {
		if logger != nil {
			logger.WithError(err).Warn("event permissions manifest not found; permissions enforcement disabled")
		}
		return EventPermissions{enforced: false}, nil
	}

	var m pluginManifest
	if err := yaml.Unmarshal(content, &m); err != nil {
		return EventPermissions{}, fmt.Errorf("parse manifest yaml failed: %w", err)
	}

	if m.Events == nil {
		if logger != nil {
			logger.Warn("manifest.events not found; permissions enforcement disabled")
		}
		return EventPermissions{enforced: false}, nil
	}

	perms := EventPermissions{
		enforced:  true,
		publish:   map[string]struct{}{},
		subscribe: map[string]struct{}{},
	}

	for _, t := range m.Events.Publish {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		perms.publish[t] = struct{}{}
	}
	for _, t := range m.Events.Subscribe {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		perms.subscribe[t] = struct{}{}
	}

	return perms, nil
}

func defaultManifestPath() string {
	if p := strings.TrimSpace(os.Getenv("POWERX_PLUGIN_MANIFEST_PATH")); p != "" {
		return p
	}
	if _, err := os.Stat("plugin.yaml"); err == nil {
		return "plugin.yaml"
	}
	if _, err := os.Stat(filepath.Join("skeleton", "plugin.yaml")); err == nil {
		return filepath.Join("skeleton", "plugin.yaml")
	}
	return "plugin.yaml"
}

type PermissionedEmitter struct {
	inner  fweventbridge.Emitter
	perms  EventPermissions
	logger *logrus.Entry
}

func NewPermissionedEmitter(inner fweventbridge.Emitter, perms EventPermissions, logger *logrus.Entry) fweventbridge.Emitter {
	if inner == nil {
		return nil
	}
	if logger == nil {
		logger = logrus.NewEntry(logrus.StandardLogger())
	}
	return &PermissionedEmitter{
		inner:  inner,
		perms:  perms,
		logger: logger,
	}
}

func (e *PermissionedEmitter) Emit(ctx context.Context, ev event.Event) error {
	if e == nil || e.inner == nil {
		return errors.New("permissioned emitter not configured")
	}
	topic := strings.TrimSpace(string(ev.Topic))
	if !e.perms.CanPublish(topic) {
		e.logger.WithFields(logrus.Fields{
			"topic":       topic,
			"tenant_uuid": ev.Meta.TenantUUID,
			"trace_id":    ev.Meta.TraceID,
		}).Warn("event publish denied by manifest permissions")
		return ErrEventPermissionDenied
	}
	return e.inner.Emit(ctx, ev)
}
