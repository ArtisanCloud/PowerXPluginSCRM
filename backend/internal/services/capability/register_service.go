package capability

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/capabilities"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/logger"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

var (
	sensitivityOptions = []string{"low", "medium", "high"}
	asyncModes         = []string{"sync", "async"}
	tagSuggestions     = []string{"integration", "workflow", "agent", "draft"}
	fieldHints         = map[string]string{
		"name.zh":        "必填：展示给国内租户的能力名称",
		"name.en":        "必填：用于全球站点的英文名称",
		"summary.zh":     "一句话描述能力价值，最多 120 字",
		"summary.en":     "One-line summary visible to global tenants",
		"schemas.input":  "引用 contracts/schema/input/*.json",
		"schemas.output": "引用 contracts/schema/output/*.json",
	}
	fileSafePattern = regexp.MustCompile(`[^a-z0-9._-]+`)
	namespaceRule   = regexp.MustCompile(`^[a-z][a-z0-9.]{2,120}$`)
)

// RegisterService governs capability registration, validation, and draft storage.
type RegisterService struct {
	logger  *logrus.Entry
	manager capabilities.Manager

	mu      sync.RWMutex
	records map[string]*CapabilityRecord
}

// CapabilityRecord captures a submitted or drafted capability.
type CapabilityRecord struct {
	ID          string            `json:"capability_id"`
	Name        LocalizedField    `json:"name"`
	Summary     LocalizedField    `json:"summary"`
	Description LocalizedField    `json:"description"`
	Scenario    string            `json:"scenario"`
	Sensitivity string            `json:"sensitivity"`
	Tags        []string          `json:"tags"`
	TenantScope string            `json:"tenant_scope"`
	Schemas     SchemaPair        `json:"schemas"`
	Protocols   ProtocolMatrix    `json:"protocols"`
	Samples     SampleBundle      `json:"samples"`
	Demo        DemoInfo          `json:"demo"`
	Owner       ContactInfo       `json:"owner"`
	AsyncMode   string            `json:"async_mode"`
	AsyncConfig AsyncConfig       `json:"async_config"`
	Status      string            `json:"status"`
	Draft       bool              `json:"draft"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	AuditID     string            `json:"audit_id,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// LocalizedField stores zh/en text.
type LocalizedField struct {
	Zh string `json:"zh"`
	En string `json:"en"`
}

// SchemaPair references JSON schema paths.
type SchemaPair struct {
	Input  string `json:"input"`
	Output string `json:"output"`
}

// ProtocolMatrix records protocol descriptors (REST/gRPC/workflow etc).
type ProtocolMatrix map[string]any

// SampleBundle contains sample payloads and error codes.
type SampleBundle struct {
	Request  map[string]any `json:"request"`
	Response map[string]any `json:"response"`
	Errors   []SampleError  `json:"errors"`
}

// SampleError captures sample error codes.
type SampleError struct {
	Code     string `json:"code"`
	Message  string `json:"message"`
	Solution string `json:"solution,omitempty"`
}

// DemoInfo records demo URLs or credentials.
type DemoInfo struct {
	URL            string `json:"url"`
	CredentialHint string `json:"credential_hint"`
}

// ContactInfo stores owner contact.
type ContactInfo struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Slack string `json:"slack,omitempty"`
}

// AsyncConfig stores async callback metadata.
type AsyncConfig struct {
	CallbackURL    string `json:"callback_url,omitempty"`
	SSEChannel     string `json:"sse_channel,omitempty"`
	StatusEndpoint string `json:"status_endpoint,omitempty"`
}

// RegisterTemplate is returned to the web-admin for defaults.
type RegisterTemplate struct {
	Namespace          string            `json:"namespace"`
	SensitivityOptions []string          `json:"sensitivity_options"`
	AsyncModes         []string          `json:"async_modes"`
	TagSuggestions     []string          `json:"tag_suggestions"`
	FieldHints         map[string]string `json:"field_hints"`
	SchemaPlaceholders SchemaPair        `json:"schema_placeholders"`
	ProtocolSamples    map[string]string `json:"protocol_samples"`
	IdentifierExample  string            `json:"identifier_example"`
}

// RegisterInput mirrors the HTTP payload.
type RegisterInput struct {
	Namespace   string            `json:"namespace"`
	Resource    string            `json:"resource"`
	Action      string            `json:"action"`
	Name        LocalizedField    `json:"name"`
	Summary     LocalizedField    `json:"summary"`
	Description LocalizedField    `json:"description"`
	Scenario    string            `json:"scenario"`
	Sensitivity string            `json:"sensitivity"`
	Tags        []string          `json:"tags"`
	TenantScope string            `json:"tenant_scope"`
	Schemas     SchemaPair        `json:"schemas"`
	Protocols   ProtocolMatrix    `json:"protocols"`
	Samples     SampleBundle      `json:"samples"`
	Demo        DemoInfo          `json:"demo"`
	Owner       ContactInfo       `json:"owner"`
	AsyncMode   string            `json:"async_mode"`
	AsyncConfig AsyncConfig       `json:"async_config"`
	Draft       bool              `json:"draft"`
	Metadata    map[string]string `json:"metadata"`
}

// ValidationResult summarizes validation outcome.
type ValidationResult struct {
	CapabilityID string       `json:"capability_id"`
	Valid        bool         `json:"valid"`
	Errors       []FieldError `json:"errors"`
}

// FieldError pinpoints invalid fields.
type FieldError struct {
	Field      string `json:"field"`
	Message    string `json:"message"`
	Suggestion string `json:"suggestion,omitempty"`
}

// NewRegisterService builds a new service instance.
func NewRegisterService(deps *app.Deps) *RegisterService {
	var log *logrus.Entry
	if deps != nil {
		log = deps.RuntimeLogger(deps.Ctx, "capability_register_service", nil)
	}
	if log == nil {
		log = logger.WithRuntimeFields(app.PluginID, "", "", "capability_register_service", nil)
	}
	var mgr capabilities.Manager
	if deps != nil {
		mgr = deps.CapabilitiesManager
	}
	return &RegisterService{
		logger:  log,
		manager: mgr,
		records: map[string]*CapabilityRecord{},
	}
}

// Template returns default form metadata.
func (s *RegisterService) Template(ctx context.Context) *RegisterTemplate {
	namespace := app.PluginID
	return &RegisterTemplate{
		Namespace:          namespace,
		SensitivityOptions: append([]string{}, sensitivityOptions...),
		AsyncModes:         append([]string{}, asyncModes...),
		TagSuggestions:     append([]string{}, tagSuggestions...),
		FieldHints:         fieldHints,
		SchemaPlaceholders: SchemaPair{
			Input:  "contracts/schema/input/" + namespace + ".sample.json",
			Output: "contracts/schema/output/" + namespace + ".sample.json",
		},
		ProtocolSamples: map[string]string{
			"rest_path":         "/api/v1/templates",
			"grpc_service":      "powerx.template.TemplateService/Create",
			"workflow_template": "contracts/exposure/workflow/template-sample.json",
		},
		IdentifierExample: namespace + ".template.create",
	}
}

// Validate checks structural and semantic rules.
func (s *RegisterService) Validate(ctx context.Context, input *RegisterInput) (*ValidationResult, error) {
	if input == nil {
		return nil, errors.New("input is required")
	}
	id := generateCapabilityID(input.Namespace, input.Resource, input.Action)
	result := &ValidationResult{CapabilityID: id, Valid: true}

	addErr := func(field, msg, suggestion string) {
		result.Valid = false
		result.Errors = append(result.Errors, FieldError{
			Field:      field,
			Message:    msg,
			Suggestion: suggestion,
		})
	}

	if id == "" {
		addErr("capability_id", "无法生成合法的能力 ID，请填写 namespace/resource/action", "示例：namespace=com.powerx.demo, resource=template, action=create")
	}

	ns := sanitizeNamespace(input.Namespace)
	if ns == "" || !namespaceRule.MatchString(ns) {
		addErr("namespace", "Namespace 需使用反向域名，并以字母开头", "例如 com.powerx.demo")
	}

	if !input.Draft {
		if strings.TrimSpace(input.Name.Zh) == "" {
			addErr("name.zh", "中文名称必填", "")
		}
		if strings.TrimSpace(input.Name.En) == "" {
			addErr("name.en", "英文名称必填", "")
		}
		if strings.TrimSpace(input.Summary.Zh) == "" {
			addErr("summary.zh", "需要提供简要摘要（中文）", "")
		}
		if strings.TrimSpace(input.Summary.En) == "" {
			addErr("summary.en", "需要提供简要摘要（英文）", "")
		}
		if strings.TrimSpace(input.Schemas.Input) == "" {
			addErr("schemas.input", "请填写输入 Schema 路径", "contracts/schema/input/"+fileSafe(id)+".json")
		} else if !s.schemaExists(input.Schemas.Input) {
			addErr("schemas.input", "引用的输入 Schema 文件不存在", "")
		}
		if strings.TrimSpace(input.Schemas.Output) == "" {
			addErr("schemas.output", "请填写输出 Schema 路径", "contracts/schema/output/"+fileSafe(id)+".json")
		} else if !s.schemaExists(input.Schemas.Output) {
			addErr("schemas.output", "引用的输出 Schema 文件不存在", "")
		}
		if strings.TrimSpace(input.Owner.Email) == "" {
			addErr("owner.email", "请提供负责人邮箱，便于审核沟通", "")
		}
		if _, ok := ensureProtocol(input.Protocols, "rest"); !ok {
			addErr("protocols.rest", "至少需声明一个 REST 协议入口", "示例：{ \"path\": \"/api/v1/templates\", \"method\": \"POST\" }")
		}
	}

	if !containsString(sensitivityOptions, strings.ToLower(strings.TrimSpace(input.Sensitivity))) {
		addErr("sensitivity", "敏感度仅支持 low/medium/high", "")
	}

	asyncMode := normalizeAsyncMode(input.AsyncMode)
	if asyncMode == "async" {
		if strings.TrimSpace(input.AsyncConfig.CallbackURL) == "" && strings.TrimSpace(input.AsyncConfig.SSEChannel) == "" {
			addErr("async_config.callback_url", "异步能力需要配置 callback_url 或 SSE 通道", "")
		}
		if strings.TrimSpace(input.AsyncConfig.StatusEndpoint) == "" {
			addErr("async_config.status_endpoint", "异步能力需要提供状态查询接口", "")
		}
	}

	if rec, ok := s.localRecord(id); ok {
		if !(rec.Draft || input.Draft) {
			addErr("capability_id", "能力 ID 已存在，请修改 namespace/resource/action", "")
		}
	} else if s.idExistsInCatalog(ctx, id) {
		addErr("capability_id", "能力 ID 已存在，请修改 namespace/resource/action", "")
	}

	return result, nil
}

// Submit validates and records a capability entry.
func (s *RegisterService) Submit(ctx context.Context, input *RegisterInput) (*CapabilityRecord, *ValidationResult, error) {
	result, err := s.Validate(ctx, input)
	if err != nil {
		return nil, nil, err
	}
	if !result.Valid {
		return nil, result, nil
	}
	capabilityID := result.CapabilityID
	result = nil
	now := time.Now().UTC()
	record := &CapabilityRecord{
		ID:          capabilityID,
		Name:        input.Name,
		Summary:     input.Summary,
		Description: input.Description,
		Scenario:    input.Scenario,
		Sensitivity: strings.ToLower(strings.TrimSpace(input.Sensitivity)),
		Tags:        normalizeTags(input.Tags),
		TenantScope: strings.TrimSpace(input.TenantScope),
		Schemas:     input.Schemas,
		Protocols:   cloneProtocols(input.Protocols),
		Samples:     normalizeSamples(input.Samples),
		Demo:        input.Demo,
		Owner:       input.Owner,
		AsyncMode:   normalizeAsyncMode(input.AsyncMode),
		AsyncConfig: input.AsyncConfig,
		Status:      "under_review",
		Draft:       input.Draft,
		CreatedAt:   now,
		UpdatedAt:   now,
		Metadata:    input.Metadata,
	}
	if input.Draft {
		record.Status = "draft"
	}
	if record.Demo.URL == "" && record.Demo.CredentialHint == "" {
		record.Demo = DemoInfo{}
	}
	if record.AsyncMode == "sync" {
		record.AsyncConfig = AsyncConfig{}
	}
	if record.Owner.Email != "" {
		record.Owner.Email = strings.TrimSpace(record.Owner.Email)
	}
	record.AuditID = uuid.NewString()

	s.mu.Lock()
	s.records[record.ID] = record
	s.mu.Unlock()

	if s.logger != nil {
		s.logger.WithFields(logrus.Fields{
			"capability_id": record.ID,
			"draft":         record.Draft,
			"status":        record.Status,
		}).Info("capability registry submission stored")
	}

	return record, result, nil
}

func (s *RegisterService) localRecord(id string) (*CapabilityRecord, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rec, ok := s.records[id]
	return rec, ok
}

func (s *RegisterService) idExistsInCatalog(ctx context.Context, id string) bool {
	if id == "" || s.manager == nil {
		return false
	}
	entries, err := s.manager.ListCapabilities(ctx)
	if err != nil {
		if s.logger != nil {
			s.logger.WithError(err).Warn("failed to list capabilities when checking duplicates")
		}
		return false
	}
	for _, entry := range entries {
		if strings.EqualFold(entry.ID, id) {
			return true
		}
	}
	return false
}

func (s *RegisterService) schemaExists(path string) bool {
	if path == "" {
		return false
	}
	if filepath.IsAbs(path) {
		_, err := os.Stat(path)
		return err == nil
	}
	basePaths := []string{".", "..", "../..", "../../..", "../../../..", "../../../../.."}
	for _, base := range basePaths {
		candidate := filepath.Join(base, path)
		if _, err := os.Stat(candidate); err == nil {
			return true
		}
	}
	return false
}

func sanitizeNamespace(ns string) string {
	ns = strings.TrimSpace(strings.ToLower(ns))
	if ns == "" {
		return app.PluginID
	}
	return ns
}

func normalizeAsyncMode(mode string) string {
	mode = strings.ToLower(strings.TrimSpace(mode))
	if !containsString(asyncModes, mode) {
		return "sync"
	}
	return mode
}

func normalizeTags(tags []string) []string {
	if len(tags) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	var out []string
	for _, t := range tags {
		trimmed := strings.ToLower(strings.TrimSpace(t))
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}

func normalizeSamples(samples SampleBundle) SampleBundle {
	out := SampleBundle{
		Request:  cloneJSON(samples.Request),
		Response: cloneJSON(samples.Response),
	}
	for _, err := range samples.Errors {
		if strings.TrimSpace(err.Code) == "" && strings.TrimSpace(err.Message) == "" {
			continue
		}
		out.Errors = append(out.Errors, SampleError{
			Code:     strings.TrimSpace(err.Code),
			Message:  strings.TrimSpace(err.Message),
			Solution: strings.TrimSpace(err.Solution),
		})
	}
	return out
}

func ensureProtocol(matrix ProtocolMatrix, key string) (any, bool) {
	if matrix == nil {
		return nil, false
	}
	value, ok := matrix[key]
	if !ok {
		return nil, false
	}
	return value, true
}

func generateCapabilityID(namespace, resource, action string) string {
	ns := sanitizeNamespace(namespace)
	res := sanitizeSegment(resource)
	act := sanitizeSegment(action)

	var parts []string
	if ns != "" {
		parts = append(parts, ns)
	}
	if res != "" {
		parts = append(parts, res)
	}
	if act != "" {
		parts = append(parts, act)
	}
	return strings.Join(parts, ".")
}

func sanitizeSegment(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return ""
	}
	value = fileSafePattern.ReplaceAllString(value, "-")
	value = strings.Trim(value, ".-")
	return value
}

func fileSafe(id string) string {
	if id == "" {
		return ""
	}
	id = strings.ToLower(strings.TrimSpace(id))
	id = strings.ReplaceAll(id, string(filepath.Separator), ".")
	return strings.ReplaceAll(id, "..", ".")
}

func containsString(list []string, target string) bool {
	for _, item := range list {
		if item == target {
			return true
		}
	}
	return false
}

func cloneProtocols(src ProtocolMatrix) ProtocolMatrix {
	if src == nil {
		return nil
	}
	dst := make(ProtocolMatrix, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func cloneJSON(src map[string]any) map[string]any {
	if src == nil {
		return nil
	}
	dst := make(map[string]any, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
