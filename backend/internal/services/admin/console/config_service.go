package console

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"strings"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/config"
	model "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/admin_console"
	consolerepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/admin_console"
	adminmetrics "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/observability/admin_console"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

var (
	ErrServiceUnavailable = errors.New("config service unavailable")
	ErrUnknownSection     = errors.New("unknown config section")
)

type validationError struct {
	Field   string
	Message string
}

func (v validationError) Error() string {
	return fmt.Sprintf("%s: %s", v.Field, v.Message)
}

// ConfigService orchestrates admin console configuration flows.
type ConfigService struct {
	cfg      *config.Config
	store    *consolerepo.Store
	changes  *consolerepo.ConfigChangeRepository
	audits   *consolerepo.AuditRepository
	metrics  *adminmetrics.Metrics
	sections map[string]SectionDefinition
	ordered  []SectionDefinition
}

// NewConfigService constructs a ConfigService from shared deps.
func NewConfigService(deps *app.Deps) *ConfigService {
	if deps == nil || deps.DB == nil {
		return &ConfigService{}
	}
	metrics := deps.AdminConsoleMetrics
	if metrics == nil {
		metrics = adminmetrics.NewMetrics()
	}
	defs := DefaultSections(deps.Config)
	return &ConfigService{
		cfg:      deps.Config,
		store:    consolerepo.NewStore(deps.DB),
		changes:  consolerepo.NewConfigChangeRepository(deps.DB),
		audits:   consolerepo.NewAuditRepository(deps.DB),
		metrics:  metrics,
		sections: ToMap(defs),
		ordered:  defs,
	}
}

// Actor carries request actor metadata.
type Actor struct {
	ID             string
	Name           string
	Email          string
	PermissionCode string
}

// ConfigSection encapsulates configuration section payload for API consumers.
type ConfigSection struct {
	Key            string                    `json:"key"`
	Title          string                    `json:"title"`
	Description    string                    `json:"description,omitempty"`
	Fields         []FieldDefinition         `json:"fields"`
	CurrentValues  map[string]any            `json:"current_values"`
	LastModifiedAt *time.Time                `json:"last_modified_at,omitempty"`
	LastModifiedBy string                    `json:"last_modified_by,omitempty"`
	Validation     map[string]map[string]any `json:"validation_rules,omitempty"`
}

// ListSections returns configured sections merged with stored values.
func (s *ConfigService) ListSections(ctx context.Context, tenantID *string) ([]ConfigSection, error) {
	if s == nil || s.changes == nil {
		return nil, ErrServiceUnavailable
	}
	result := make([]ConfigSection, 0, len(s.ordered))
	for _, def := range s.ordered {
		section := ConfigSection{
			Key:           def.Key,
			Title:         def.Title,
			Description:   def.Description,
			Fields:        def.Fields,
			CurrentValues: def.DefaultValues(s.cfg),
		}
		section.Validation = collectValidation(def.Fields)
		change, err := s.changes.LatestBySection(ctx, app.PluginID, tenantID, def.Key)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				result = append(result, section)
				continue
			}
			return nil, err
		}
		if change != nil {
			if len(change.NextSnapshot) > 0 {
				var next map[string]any
				if err := json.Unmarshal(change.NextSnapshot, &next); err == nil {
					section.CurrentValues = MergeValues(section.CurrentValues, next)
				}
			}
			section.LastModifiedAt = &change.AppliedAt
			if change.AuditEvent != nil {
				section.LastModifiedBy = actorLabel(change.AuditEvent)
			}
		}
		result = append(result, section)
	}
	return result, nil
}

// UpdateSectionInput describes update payload.
type UpdateSectionInput struct {
	TenantUuid *string
	SectionKey string
	Values     map[string]any
	Comment    string
	Actor      Actor
}

// UpdateSection applies user-submitted configuration changes.
func (s *ConfigService) UpdateSection(ctx context.Context, input UpdateSectionInput) (*ConfigSection, error) {
	if s == nil || s.changes == nil || s.audits == nil {
		return nil, ErrServiceUnavailable
	}
	def, ok := s.sections[input.SectionKey]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnknownSection, input.SectionKey)
	}
	if input.Actor.ID == "" {
		return nil, validationError{Field: "actor", Message: "actor id required"}
	}
	sanitized, err := sanitizeValues(def, input.Values)
	if err != nil {
		return nil, err
	}
	defaults := def.DefaultValues(s.cfg)
	tenantKey := normalizeTenantKey(input.TenantUuid)

	var latest *model.ConfigChange
	latest, err = s.changes.LatestBySection(ctx, app.PluginID, input.TenantUuid, def.Key)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	var previous map[string]any
	if latest != nil {
		previous = defaults
		if len(latest.NextSnapshot) > 0 {
			var prev map[string]any
			_ = json.Unmarshal(latest.NextSnapshot, &prev)
			previous = MergeValues(defaults, prev)
		}
	} else {
		previous = defaults
	}

	changeType := "create"
	if latest != nil {
		changeType = "update"
	}

	err = s.withTransaction(ctx, input.TenantUuid, func(tx *gorm.DB) error {
		auditRepo := consolerepo.NewAuditRepository(tx)
		changeRepo := consolerepo.NewConfigChangeRepository(tx)

		diff := map[string]any{"previous": previous, "next": sanitized}
		diffJSON, _ := json.Marshal(diff)

		var actorName *string
		if input.Actor.Name != "" {
			actorName = &input.Actor.Name
		}
		var actorEmail *string
		if input.Actor.Email != "" {
			actorEmail = &input.Actor.Email
		}
		audit := &model.AuditEvent{
			PluginID:       app.PluginID,
			TenantUuid:     input.TenantUuid,
			ActorID:        input.Actor.ID,
			ActorName:      actorName,
			ActorEmail:     actorEmail,
			PermissionCode: input.Actor.PermissionCode,
			Action:         fmt.Sprintf("config.section.%s", changeType),
			ResourceType:   "config.section",
			ResourceRef:    &def.Key,
			Summary:        strPtr(fmt.Sprintf("Updated %s", def.Title)),
			Diff:           datatypes.JSON(diffJSON),
		}
		audit.ID = uuid.NewString()
		if err := auditRepo.Create(ctx, audit); err != nil {
			return err
		}

		change := &model.ConfigChange{
			PluginID:         app.PluginID,
			TenantUuid:       tenantKey,
			SectionKey:       def.Key,
			ChangeType:       changeType,
			PreviousSnapshot: mapToJSON(previous),
			NextSnapshot:     mapToJSON(sanitized),
			AuditEventID:     audit.ID,
		}
		change.ID = uuid.NewString()
		if err := changeRepo.Create(ctx, change); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	sections, err := s.ListSections(ctx, input.TenantUuid)
	if err != nil {
		return nil, err
	}
	for _, sec := range sections {
		if sec.Key == def.Key {
			return &sec, nil
		}
	}
	return nil, errors.New("updated section not found")
}

func (s *ConfigService) withTransaction(ctx context.Context, tenantID *string, fn func(tx *gorm.DB) error) error {
	if tenantID != nil && s.store != nil {
		return s.store.WithTenant(ctx, *tenantID, fn)
	}
	return s.changes.DB.WithContext(ctx).Transaction(fn)
}

func sanitizeValues(def SectionDefinition, values map[string]any) (map[string]any, error) {
	cleaned := map[string]any{}
	expected := map[string]FieldDefinition{}
	for _, field := range def.Fields {
		expected[field.Name] = field
	}
	for name, field := range expected {
		raw, ok := values[name]
		if !ok {
			if field.Required {
				return nil, newValidationError(name, "is required")
			}
			continue
		}
		val, err := coerceFieldValue(field, raw)
		if err != nil {
			return nil, wrapFieldError(name, err)
		}
		cleaned[name] = val
	}
	return cleaned, nil
}

func coerceFieldValue(field FieldDefinition, raw any) (any, error) {
	switch field.Type {
	case "number":
		num, err := normalizeNumber(raw)
		if err != nil {
			return nil, err
		}
		if bounds := field.Validation; len(bounds) > 0 {
			if min, ok := bounds["min"].(int); ok && num < float64(min) {
				return nil, fmt.Errorf("value must be >= %d", min)
			}
			if max, ok := bounds["max"].(int); ok && num > float64(max) {
				return nil, fmt.Errorf("value must be <= %d", max)
			}
		}
		return int(num), nil
	case "select":
		str, err := normalizeString(raw)
		if err != nil {
			return nil, err
		}
		if len(field.Options) > 0 {
			allowed := false
			for _, opt := range field.Options {
				if opt.Value == str {
					allowed = true
					break
				}
			}
			if !allowed {
				return nil, fmt.Errorf("value must be one of: %s", joinOptions(field.Options))
			}
		}
		return str, nil
	case "boolean":
		return normalizeBool(raw)
	default:
		str, err := normalizeString(raw)
		if err != nil {
			return nil, err
		}
		return str, nil
	}
}

func normalizeNumber(v any) (float64, error) {
	switch n := v.(type) {
	case int:
		return float64(n), nil
	case int8:
		return float64(n), nil
	case int16:
		return float64(n), nil
	case int32:
		return float64(n), nil
	case int64:
		return float64(n), nil
	case float32:
		return float64(n), nil
	case float64:
		return n, nil
	case json.Number:
		f, err := n.Float64()
		if err != nil {
			return 0, err
		}
		return f, nil
	default:
		return 0, fmt.Errorf("expected number, got %T", v)
	}
}

func normalizeString(v any) (string, error) {
	switch s := v.(type) {
	case string:
		if strings.TrimSpace(s) == "" {
			return "", errors.New("value cannot be empty")
		}
		return strings.TrimSpace(s), nil
	default:
		return "", fmt.Errorf("expected string, got %T", v)
	}
}

func normalizeBool(v any) (bool, error) {
	switch b := v.(type) {
	case bool:
		return b, nil
	case string:
		cleaned := strings.ToLower(strings.TrimSpace(b))
		if cleaned == "true" || cleaned == "1" {
			return true, nil
		}
		if cleaned == "false" || cleaned == "0" {
			return false, nil
		}
		return false, fmt.Errorf("invalid boolean value: %s", b)
	default:
		return false, fmt.Errorf("expected boolean, got %T", v)
	}
}

func collectValidation(fields []FieldDefinition) map[string]map[string]any {
	rules := make(map[string]map[string]any)
	for _, f := range fields {
		if len(f.Validation) == 0 {
			continue
		}
		rules[f.Name] = maps.Clone(f.Validation)
	}
	return rules
}

func actorLabel(evt *model.AuditEvent) string {
	if evt == nil {
		return ""
	}
	if evt.ActorName != nil && *evt.ActorName != "" {
		return *evt.ActorName
	}
	if evt.ActorID != "" {
		return evt.ActorID
	}
	return ""
}

func strPtr(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}

func joinOptions(opts []FieldOption) string {
	values := make([]string, len(opts))
	for i, opt := range opts {
		values[i] = opt.Value
	}
	return strings.Join(values, ", ")
}

func normalizeTenantKey(tenantID *string) *string {
	if tenantID == nil {
		return nil
	}
	cleaned := strings.TrimSpace(*tenantID)
	if cleaned == "" {
		return nil
	}
	return &cleaned
}

func newValidationError(field, message string) error {
	return validationError{Field: field, Message: message}
}

func wrapFieldError(field string, err error) error {
	if err == nil {
		return nil
	}
	var v validationError
	if errors.As(err, &v) {
		if v.Field == "" {
			v.Field = field
		}
		return v
	}
	return validationError{Field: field, Message: err.Error()}
}

// IsValidationError reports validation metadata if present.
func IsValidationError(err error) (field, message string, ok bool) {
	var v validationError
	if errors.As(err, &v) {
		return v.Field, v.Message, true
	}
	return "", "", false
}

func mapToJSON(m map[string]any) datatypes.JSON {
	if len(m) == 0 {
		return datatypes.JSON([]byte("{}"))
	}
	b, err := json.Marshal(m)
	if err != nil {
		return datatypes.JSON([]byte("{}"))
	}
	return datatypes.JSON(b)
}
