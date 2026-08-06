package integration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	domain "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/integration"
	dbtemplate "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/template"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/mcp/stream"
	authx "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/middleware"
	leadsvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/lead_capture"
	srvtemplates "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/templates"
	"github.com/sirupsen/logrus"
)

const (
	defaultListPageSize = 10
	maxListPageSize     = 100
)

// CapabilityHandler 表示单个能力的执行体。
type CapabilityHandler interface {
	CapabilityID() string
	Handle(ctx context.Context, envelope *domain.IntegrationEnvelope) (*HostInvocationResult, error)
}

// HandlerRegistry 管理 capability → handler 以及 scope → capability 的映射。
type HandlerRegistry struct {
	byCapability map[string]CapabilityHandler
	scopeAlias   map[string]string
}

func NewHandlerRegistry() *HandlerRegistry {
	return &HandlerRegistry{
		byCapability: make(map[string]CapabilityHandler),
		scopeAlias:   make(map[string]string),
	}
}

func (r *HandlerRegistry) Register(handler CapabilityHandler, scopes ...string) {
	if handler == nil {
		return
	}
	capabilityID := strings.TrimSpace(handler.CapabilityID())
	if capabilityID == "" {
		return
	}
	r.byCapability[capabilityID] = handler
	for _, scope := range scopes {
		if s := strings.TrimSpace(strings.ToLower(scope)); s != "" {
			r.scopeAlias[s] = capabilityID
		}
	}
}

func (r *HandlerRegistry) HandlerByCapability(capabilityID string) (CapabilityHandler, bool) {
	handler, ok := r.byCapability[capabilityID]
	return handler, ok
}

func (r *HandlerRegistry) CapabilityByScope(scope string) string {
	if r == nil {
		return ""
	}
	return r.scopeAlias[strings.TrimSpace(strings.ToLower(scope))]
}

// CapabilityInvoker 根据 registry 调度不同能力的 handler。
type CapabilityInvoker struct {
	registry *HandlerRegistry
	fallback HostInvoker
	logger   *logrus.Entry
}

// NewCapabilityInvoker wires business handlers with HostInvoker contract.
func NewCapabilityInvoker(templateService *srvtemplates.TemplateService, leadBridgeService *leadsvc.LeadBridgeService, broker *stream.Broker, logger *logrus.Entry, fallback HostInvoker) HostInvoker {
	registry := NewHandlerRegistry()
	registry.Register(&templateListHandler{svc: templateService}, "agent.template.list")
	registry.Register(&templateReadHandler{svc: templateService}, "agent.template.read")
	registry.Register(&templateCreateHandler{svc: templateService}, "agent.template.create")
	registry.Register(&templateUpdateHandler{svc: templateService}, "agent.template.update")
	registry.Register(&templateDeleteHandler{svc: templateService}, "agent.template.delete")
	registry.Register(&templateComposeHandler{svc: templateService, broker: broker, logger: logger}, "agent.template.compose")
	registry.Register(&templateAuditHandler{svc: templateService, broker: broker, logger: logger}, "agent.template.audit")
	registry.Register(&templateQualityHandler{svc: templateService, broker: broker, logger: logger}, "agent.template.quality_distribute")
	registry.Register(&leadIdentityUpdateHandler{svc: leadBridgeService}, "scrm.lead.identity.update")
	registry.Register(&leadActivityRecordHandler{svc: leadBridgeService}, "scrm.lead.activity.record")
	registry.Register(&leadMappingUpsertHandler{svc: leadBridgeService}, "scrm.lead.mapping.upsert")
	registry.Register(&leadStatusGetHandler{svc: leadBridgeService}, "scrm.lead.status.get")
	registry.Register(&leadConflictResolveHandler{svc: leadBridgeService}, "scrm.lead.conflict.resolve")

	return &CapabilityInvoker{
		registry: registry,
		fallback: fallback,
		logger:   logger,
	}
}

// Invoke satisfies HostInvoker by dispatching to registered handlers.
func (i *CapabilityInvoker) Invoke(ctx context.Context, envelope *domain.IntegrationEnvelope) (*HostInvocationResult, error) {
	if envelope == nil {
		return nil, errors.New("integration envelope is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	ctx = authx.ContextWithTenantUUID(ctx, envelope.TenantUuid)

	capabilityID := i.capabilityFromEnvelope(envelope)
	if handler, ok := i.registry.HandlerByCapability(capabilityID); ok {
		return handler.Handle(ctx, envelope)
	}
	if i.logger != nil {
		i.logger.WithField("capability_id", capabilityID).Debug("delegating capability to fallback invoker")
	}
	if i.fallback != nil {
		return i.fallback.Invoke(ctx, envelope)
	}
	return nil, fmt.Errorf("unsupported capability: %s", capabilityID)
}

func (i *CapabilityInvoker) capabilityFromEnvelope(envelope *domain.IntegrationEnvelope) string {
	if envelope == nil {
		return ""
	}
	if envelope.Metadata != nil {
		if raw, ok := envelope.Metadata["capability_id"]; ok {
			if s := normalizeString(raw); s != "" {
				return s
			}
		}
	}
	if alias := i.registry.CapabilityByScope(envelope.ToolScope); alias != "" {
		return alias
	}
	return ""
}

// ---- handler 实现 ----

type templateListHandler struct {
	svc *srvtemplates.TemplateService
}

func (h *templateListHandler) Handle(ctx context.Context, envelope *domain.IntegrationEnvelope) (*HostInvocationResult, error) {
	if h == nil || h.svc == nil {
		return nil, errors.New("template service unavailable")
	}
	var payload templateListPayload
	if err := decodeInlinePayload(envelope.PayloadRef, &payload); err != nil {
		return nil, err
	}
	page := payload.Page
	if page <= 0 {
		page = 1
	}
	pageSize := payload.PageSize
	if pageSize <= 0 {
		pageSize = defaultListPageSize
	} else if pageSize > maxListPageSize {
		pageSize = maxListPageSize
	}
	result, err := h.svc.List(ctx, payload.Q, page, pageSize)
	if err != nil {
		return nil, err
	}
	payloadBytes, _ := json.Marshal(result)
	return &HostInvocationResult{Status: "accepted", Payload: payloadBytes}, nil
}

func (h *templateListHandler) CapabilityID() string {
	return "com.powerx.plugins.scrm.template.list"
}

type templateReadHandler struct {
	svc *srvtemplates.TemplateService
}

func (h *templateReadHandler) Handle(ctx context.Context, envelope *domain.IntegrationEnvelope) (*HostInvocationResult, error) {
	if h == nil || h.svc == nil {
		return nil, errors.New("template service unavailable")
	}
	var payload templateReadPayload
	if err := decodeInlinePayload(envelope.PayloadRef, &payload); err != nil {
		return nil, err
	}
	if payload.TemplateID == 0 {
		return nil, errors.New("template_id is required")
	}
	tpl, err := h.svc.GetByID(ctx, payload.TemplateID)
	if err != nil {
		return nil, err
	}
	payloadBytes, _ := json.Marshal(tpl)
	return &HostInvocationResult{Status: "accepted", Payload: payloadBytes}, nil
}

func (h *templateReadHandler) CapabilityID() string {
	return "com.powerx.plugins.scrm.template.read"
}

type templateCreateHandler struct {
	svc *srvtemplates.TemplateService
}

func (h *templateCreateHandler) Handle(ctx context.Context, envelope *domain.IntegrationEnvelope) (*HostInvocationResult, error) {
	if h == nil || h.svc == nil {
		return nil, errors.New("template service unavailable")
	}
	var payload templateCreatePayload
	if err := decodeInlinePayload(envelope.PayloadRef, &payload); err != nil {
		return nil, err
	}
	name := strings.TrimSpace(payload.Name)
	if name == "" {
		return nil, errors.New("name is required")
	}
	content := strings.TrimSpace(payload.Content)
	if content == "" {
		content = fmt.Sprintf("Generated at %s", time.Now().UTC().Format(time.RFC3339))
	}
	tpl, err := h.svc.Create(ctx, name, payload.Description, content)
	if err != nil {
		return nil, err
	}
	payloadBytes, _ := json.Marshal(tpl)
	return &HostInvocationResult{Status: "accepted", Payload: payloadBytes}, nil
}

func (h *templateCreateHandler) CapabilityID() string {
	return "com.powerx.plugins.scrm.template.create"
}

type templateUpdateHandler struct {
	svc *srvtemplates.TemplateService
}

func (h *templateUpdateHandler) Handle(ctx context.Context, envelope *domain.IntegrationEnvelope) (*HostInvocationResult, error) {
	if h == nil || h.svc == nil {
		return nil, errors.New("template service unavailable")
	}
	var payload templateUpdatePayload
	if err := decodeInlinePayload(envelope.PayloadRef, &payload); err != nil {
		return nil, err
	}
	if payload.TemplateID == 0 {
		return nil, errors.New("template_id is required")
	}
	current, err := h.svc.GetByID(ctx, payload.TemplateID)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(payload.Name)
	if name == "" {
		name = current.Name
	}
	description := strings.TrimSpace(payload.Description)
	if description == "" {
		description = current.Description
	}
	content := strings.TrimSpace(payload.Content)
	if content == "" {
		content = current.Content
	}
	updated, err := h.svc.Update(ctx, payload.TemplateID, name, description, content)
	if err != nil {
		return nil, err
	}
	payloadBytes, _ := json.Marshal(updated)
	return &HostInvocationResult{Status: "accepted", Payload: payloadBytes}, nil
}

func (h *templateUpdateHandler) CapabilityID() string {
	return "com.powerx.plugins.scrm.template.update"
}

type templateDeleteHandler struct {
	svc *srvtemplates.TemplateService
}

func (h *templateDeleteHandler) Handle(ctx context.Context, envelope *domain.IntegrationEnvelope) (*HostInvocationResult, error) {
	if h == nil || h.svc == nil {
		return nil, errors.New("template service unavailable")
	}
	var payload templateDeletePayload
	if err := decodeInlinePayload(envelope.PayloadRef, &payload); err != nil {
		return nil, err
	}
	if payload.TemplateID == 0 {
		return nil, errors.New("template_id is required")
	}
	if err := h.svc.Delete(ctx, payload.TemplateID); err != nil {
		return nil, err
	}
	resp := map[string]any{
		"deleted":     true,
		"template_id": payload.TemplateID,
	}
	payloadBytes, _ := json.Marshal(resp)
	return &HostInvocationResult{Status: "accepted", Payload: payloadBytes}, nil
}

func (h *templateDeleteHandler) CapabilityID() string {
	return "com.powerx.plugins.scrm.template.delete"
}

type templateComposeHandler struct {
	svc    *srvtemplates.TemplateService
	broker *stream.Broker
	logger *logrus.Entry
}

func (h *templateComposeHandler) Handle(ctx context.Context, envelope *domain.IntegrationEnvelope) (*HostInvocationResult, error) {
	if h == nil || h.svc == nil {
		return nil, errors.New("template service unavailable")
	}
	var payload templateComposePayload
	if err := decodeInlinePayload(envelope.PayloadRef, &payload); err != nil {
		return nil, err
	}
	if strings.TrimSpace(payload.Draft.Name) == "" {
		return nil, errors.New("draft.name is required")
	}
	content := strings.TrimSpace(payload.Draft.Content)
	if content == "" {
		content = fmt.Sprintf("Generated at %s", time.Now().UTC().Format(time.RFC3339))
	}
	tpl, err := h.svc.Create(ctx, payload.Draft.Name, payload.Draft.Description, content)
	if err != nil {
		return nil, err
	}
	lifecycle := make([]map[string]any, 0, 4)
	lifecycle = append(lifecycle, map[string]any{
		"stage":           "draft",
		"status":          tpl.Status,
		"review_status":   tpl.ReviewStatus,
		"template_id":     tpl.ID,
		"publish_channel": tpl.PublishChannel,
	})
	emitEvent(h.broker, envelope, "draft.created", tpl)

	reviewer := strings.TrimSpace(payload.Review.Reviewer)
	if reviewer == "" {
		reviewer = "system"
	}
	reviewed, err := h.svc.MarkReviewed(ctx, tpl.ID, reviewer, payload.Review.Comment, true)
	if err != nil {
		return nil, err
	}
	lifecycle = append(lifecycle, map[string]any{
		"stage":         "review",
		"status":        reviewed.Status,
		"review_status": reviewed.ReviewStatus,
		"reviewed_by":   reviewed.ReviewedBy,
		"reviewed_at":   reviewed.ReviewedAt,
	})
	emitEvent(h.broker, envelope, "template.review.completed", map[string]any{
		"template_id":    reviewed.ID,
		"review_status":  reviewed.ReviewStatus,
		"reviewed_by":    reviewed.ReviewedBy,
		"reviewed_at":    reviewed.ReviewedAt,
		"review_comment": reviewed.ReviewComment,
	})

	published, err := h.svc.Publish(ctx, tpl.ID, payload.PublishChannel)
	if err != nil {
		return nil, err
	}
	publishStatusValue := srvtemplates.TemplateStatusPublished
	if ch := strings.TrimSpace(published.PublishChannel); ch != "" {
		publishStatusValue = fmt.Sprintf("published:%s", ch)
	}
	publishPayload := map[string]any{
		"template_id":     published.ID,
		"status":          published.Status,
		"publish_channel": published.PublishChannel,
		"published_at":    published.PublishedAt,
	}
	lifecycle = append(lifecycle, map[string]any{
		"stage":           "publish",
		"status":          published.Status,
		"publish_channel": published.PublishChannel,
		"published_at":    published.PublishedAt,
	})
	emitEvent(h.broker, envelope, "publish.status", publishPayload)
	emitEvent(h.broker, envelope, "template.publish.completed", publishPayload)

	cleanupReason := strings.TrimSpace(payload.Cleanup.Reason)
	if cleanupReason == "" {
		cleanupReason = "auto-clean"
	}
	cleaned, err := h.svc.Cleanup(ctx, tpl.ID, cleanupReason)
	if err != nil {
		return nil, err
	}
	lifecycle = append(lifecycle, map[string]any{
		"stage":          "cleanup",
		"status":         cleaned.Status,
		"cleanup_reason": cleaned.CleanupReason,
		"cleaned_at":     cleaned.CleanedAt,
	})
	emitEvent(h.broker, envelope, "template.cleanup.completed", map[string]any{
		"template_id":    cleaned.ID,
		"status":         cleaned.Status,
		"cleanup_reason": cleaned.CleanupReason,
		"cleaned_at":     cleaned.CleanedAt,
	})

	resp := map[string]any{
		"template_id":     strconv.FormatUint(tpl.ID, 10),
		"draft_id":        strconv.FormatUint(tpl.ID, 10),
		"status":          cleaned.Status,
		"review_status":   cleaned.ReviewStatus,
		"publish_status":  publishStatusValue,
		"publish_channel": published.PublishChannel,
		"cleanup_reason":  cleaned.CleanupReason,
		"cleaned_at":      cleaned.CleanedAt,
		"lifecycle":       lifecycle,
	}
	payloadBytes, _ := json.Marshal(resp)

	return &HostInvocationResult{
		Status:  "accepted",
		Payload: payloadBytes,
		Metadata: map[string]any{
			"template_id": tpl.ID,
		},
	}, nil
}

func (h *templateComposeHandler) CapabilityID() string {
	return "com.powerx.plugins.scrm.template.compose"
}

type templateAuditHandler struct {
	svc    *srvtemplates.TemplateService
	broker *stream.Broker
	logger *logrus.Entry
}

func (h *templateAuditHandler) Handle(ctx context.Context, envelope *domain.IntegrationEnvelope) (*HostInvocationResult, error) {
	if h == nil || h.svc == nil {
		return nil, errors.New("template service unavailable")
	}
	var payload templateAuditPayload
	if err := decodeInlinePayload(envelope.PayloadRef, &payload); err != nil {
		return nil, err
	}
	if strings.TrimSpace(payload.UpdatePayload.Content) == "" {
		return nil, errors.New("update_payload.content is required")
	}
	page := payload.Filters.Page
	if page <= 0 {
		page = 1
	}
	pageSize := payload.Filters.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	query := strings.TrimSpace(payload.Filters.Status)
	list, err := h.svc.List(ctx, query, page, pageSize)
	if err != nil {
		return nil, err
	}
	scannedTotal := int(list.Total)
	var selected *dbtemplate.Template
	if list.List != nil && len(list.List) > 0 {
		selected = list.List[0]
	}
	response := map[string]any{
		"scanned_total": scannedTotal,
		"updated":       false,
		"events":        []map[string]any{},
	}
	if selected != nil {
		desc := payload.UpdatePayload.Description
		if strings.TrimSpace(desc) == "" {
			desc = selected.Description
		}
		updated, err := h.svc.Update(ctx, selected.ID, selected.Name, desc, payload.UpdatePayload.Content)
		if err != nil {
			return nil, err
		}
		response["selected_template_id"] = strconv.FormatUint(updated.ID, 10)
		response["updated"] = true
		evtPayload := map[string]any{
			"template_id": updated.ID,
			"description": updated.Description,
		}
		events := []map[string]any{
			{
				"name":    "audit.template.updated",
				"payload": evtPayload,
			},
		}
		response["events"] = events
		emitEvent(h.broker, envelope, "audit.template.updated", evtPayload)
	}
	payloadBytes, _ := json.Marshal(response)
	return &HostInvocationResult{Status: "accepted", Payload: payloadBytes}, nil
}

func (h *templateAuditHandler) CapabilityID() string { return "com.powerx.plugins.scrm.template.audit" }

type templateQualityHandler struct {
	svc    *srvtemplates.TemplateService
	broker *stream.Broker
	logger *logrus.Entry
}

func (h *templateQualityHandler) Handle(ctx context.Context, envelope *domain.IntegrationEnvelope) (*HostInvocationResult, error) {
	if h == nil || h.svc == nil {
		return nil, errors.New("template service unavailable")
	}
	var payload templateQualityPayload
	if err := decodeInlinePayload(envelope.PayloadRef, &payload); err != nil {
		return nil, err
	}
	if strings.TrimSpace(payload.UpdatePayload.Content) == "" {
		return nil, errors.New("update_payload.content is required")
	}
	page := payload.ScanFilter.Page
	if page <= 0 {
		page = 1
	}
	pageSize := payload.ScanFilter.PageSize
	if pageSize <= 0 {
		pageSize = 5
	}
	if pageSize > 50 {
		pageSize = 50
	}
	list, err := h.svc.List(ctx, payload.ScanFilter.Q, page, pageSize)
	if err != nil {
		return nil, err
	}
	response := map[string]any{
		"scanned_total": int(list.Total),
	}
	if list.List == nil || len(list.List) == 0 {
		payloadBytes, _ := json.Marshal(response)
		return &HostInvocationResult{Status: "accepted", Payload: payloadBytes}, nil
	}
	primary := list.List[0]
	if validation, err := h.svc.Validate(ctx, primary.ID, payload.ValidateRules, false); err == nil {
		response["validated_template_id"] = strconv.FormatUint(primary.ID, 10)
		response["validation_passed"] = validation.Valid
		emitEvent(h.broker, envelope, "template.validate.completed", map[string]any{
			"template_id": primary.ID,
			"valid":       validation.Valid,
			"violations":  validation.Violations,
		})
	}
	sourceIDs := make([]uint64, len(list.List))
	for idx, tpl := range list.List {
		sourceIDs[idx] = tpl.ID
	}
	copies := payload.Clone.Copies
	if copies <= 0 {
		copies = 1
	}
	if copies > 20 {
		copies = 20
	}
	cloneResult, err := h.svc.BatchClone(ctx, sourceIDs, copies, srvtemplates.BatchCloneOptions{
		NamePrefix:        payload.Clone.NamePrefix,
		DescriptionPrefix: payload.Clone.DescriptionPrefix,
	})
	if err != nil {
		return nil, err
	}
	response["created_template_ids"] = cloneResult.CreatedIDs
	emitEvent(h.broker, envelope, "template.batch_clone.completed", map[string]any{
		"source_ids":  sourceIDs,
		"created_ids": cloneResult.CreatedIDs,
	})
	targetID := primary.ID
	if len(cloneResult.CreatedIDs) > 0 {
		targetID = cloneResult.CreatedIDs[0]
	}
	target, err := h.svc.GetByID(ctx, targetID)
	if err != nil {
		return nil, err
	}
	desc := strings.TrimSpace(payload.UpdatePayload.Description)
	if desc == "" {
		desc = target.Description
	}
	updated, err := h.svc.Update(ctx, targetID, target.Name, desc, payload.UpdatePayload.Content)
	if err != nil {
		return nil, err
	}
	response["updated_template_id"] = strconv.FormatUint(updated.ID, 10)
	emitEvent(h.broker, envelope, "template.update.completed", map[string]any{
		"template_id": updated.ID,
		"description": updated.Description,
	})
	payloadBytes, _ := json.Marshal(response)
	return &HostInvocationResult{Status: "accepted", Payload: payloadBytes}, nil
}

func (h *templateQualityHandler) CapabilityID() string {
	return "com.powerx.plugins.scrm.template.quality_distribute"
}

type leadIdentityUpdateHandler struct {
	svc *leadsvc.LeadBridgeService
}

func (h *leadIdentityUpdateHandler) CapabilityID() string {
	return "com.powerx.plugins.scrm.lead.identity.update"
}

func (h *leadIdentityUpdateHandler) Handle(ctx context.Context, envelope *domain.IntegrationEnvelope) (*HostInvocationResult, error) {
	if h == nil || h.svc == nil {
		return nil, errors.New("lead bridge service unavailable")
	}
	var payload leadsvc.LeadBridgeIdentityUpdateRequest
	if err := decodeInlinePayload(envelope.PayloadRef, &payload); err != nil {
		return nil, err
	}
	result, err := h.svc.UpdateIdentity(ctx, envelope.TenantUuid, payload)
	if err != nil {
		return nil, err
	}
	return jsonInvocationResult("accepted", result)
}

type leadActivityRecordHandler struct {
	svc *leadsvc.LeadBridgeService
}

func (h *leadActivityRecordHandler) CapabilityID() string {
	return "com.powerx.plugins.scrm.lead.activity.record"
}

func (h *leadActivityRecordHandler) Handle(ctx context.Context, envelope *domain.IntegrationEnvelope) (*HostInvocationResult, error) {
	if h == nil || h.svc == nil {
		return nil, errors.New("lead bridge service unavailable")
	}
	var payload leadsvc.LeadBridgeActivityRequest
	if err := decodeInlinePayload(envelope.PayloadRef, &payload); err != nil {
		return nil, err
	}
	result, err := h.svc.RecordActivity(ctx, envelope.TenantUuid, payload)
	if err != nil {
		return nil, err
	}
	return jsonInvocationResult("accepted", result)
}

type leadMappingUpsertHandler struct {
	svc *leadsvc.LeadBridgeService
}

func (h *leadMappingUpsertHandler) CapabilityID() string {
	return "com.powerx.plugins.scrm.lead.mapping.upsert"
}

func (h *leadMappingUpsertHandler) Handle(ctx context.Context, envelope *domain.IntegrationEnvelope) (*HostInvocationResult, error) {
	if h == nil || h.svc == nil {
		return nil, errors.New("lead bridge service unavailable")
	}
	var payload leadsvc.LeadBridgeMappingRequest
	if err := decodeInlinePayload(envelope.PayloadRef, &payload); err != nil {
		return nil, err
	}
	result, err := h.svc.UpsertMapping(ctx, envelope.TenantUuid, payload)
	if err != nil {
		return nil, err
	}
	return jsonInvocationResult("accepted", result)
}

type leadStatusGetHandler struct {
	svc *leadsvc.LeadBridgeService
}

func (h *leadStatusGetHandler) CapabilityID() string {
	return "com.powerx.plugins.scrm.lead.status.get"
}

func (h *leadStatusGetHandler) Handle(ctx context.Context, envelope *domain.IntegrationEnvelope) (*HostInvocationResult, error) {
	if h == nil || h.svc == nil {
		return nil, errors.New("lead bridge service unavailable")
	}
	var payload leadsvc.LeadBridgeStatusRequest
	if err := decodeInlinePayload(envelope.PayloadRef, &payload); err != nil {
		return nil, err
	}
	result, err := h.svc.GetStatus(ctx, envelope.TenantUuid, payload)
	if err != nil {
		return nil, err
	}
	return jsonInvocationResult("accepted", result)
}

type leadConflictResolveHandler struct {
	svc *leadsvc.LeadBridgeService
}

func (h *leadConflictResolveHandler) CapabilityID() string {
	return "com.powerx.plugins.scrm.lead.conflict.resolve"
}

func (h *leadConflictResolveHandler) Handle(ctx context.Context, envelope *domain.IntegrationEnvelope) (*HostInvocationResult, error) {
	if h == nil || h.svc == nil {
		return nil, errors.New("lead bridge service unavailable")
	}
	var payload leadsvc.LeadBridgeConflictResolveRequest
	if err := decodeInlinePayload(envelope.PayloadRef, &payload); err != nil {
		return nil, err
	}
	result, err := h.svc.ResolveConflict(ctx, envelope.TenantUuid, payload)
	if err != nil {
		return nil, err
	}
	return jsonInvocationResult("accepted", result)
}

func emitEvent(broker *stream.Broker, envelope *domain.IntegrationEnvelope, eventType string, payload interface{}) {
	if broker == nil {
		return
	}
	sessionID := sessionIDFromEnvelope(envelope)
	if sessionID == "" {
		return
	}
	broker.Publish(stream.Event{
		SessionID: sessionID,
		Type:      eventType,
		Payload:   payload,
		Timestamp: time.Now().UTC(),
	})
}

func jsonInvocationResult(status string, payload interface{}) (*HostInvocationResult, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return &HostInvocationResult{Status: status, Payload: payloadBytes}, nil
}

func sessionIDFromEnvelope(envelope *domain.IntegrationEnvelope) string {
	if envelope == nil || envelope.Metadata == nil {
		return ""
	}
	return normalizeString(envelope.Metadata["session_id"])
}

func decodeInlinePayload(payloadRef string, dest interface{}) error {
	trimmed := strings.TrimSpace(payloadRef)
	if trimmed == "" {
		return errors.New("payload_ref is empty")
	}
	if !strings.HasPrefix(trimmed, "{") && !strings.HasPrefix(trimmed, "[") {
		return errors.New("payload_ref must be inline JSON for this capability")
	}
	if err := json.Unmarshal([]byte(trimmed), dest); err != nil {
		return fmt.Errorf("invalid payload_ref: %w", err)
	}
	return nil
}

func normalizeString(v interface{}) string {
	switch val := v.(type) {
	case string:
		return strings.TrimSpace(val)
	case fmt.Stringer:
		return strings.TrimSpace(val.String())
	case json.Number:
		return strings.TrimSpace(val.String())
	default:
		if val == nil {
			return ""
		}
		return strings.TrimSpace(fmt.Sprint(val))
	}
}

type templateListPayload struct {
	Q        string `json:"q"`
	Page     int    `json:"page"`
	PageSize int    `json:"page_size"`
}

type templateReadPayload struct {
	TemplateID uint64 `json:"template_id"`
}

type templateCreatePayload struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Content     string `json:"content"`
}

type templateUpdatePayload struct {
	TemplateID  uint64 `json:"template_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Content     string `json:"content"`
}

type templateDeletePayload struct {
	TemplateID uint64 `json:"template_id"`
}

type templateComposePayload struct {
	Draft struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Content     string `json:"content"`
	} `json:"draft"`
	PublishChannel string `json:"publish_channel"`
	Review         struct {
		Reviewer string `json:"reviewer"`
		Comment  string `json:"comment"`
	} `json:"review"`
	Cleanup struct {
		Reason string `json:"reason"`
	} `json:"cleanup"`
}

type templateAuditPayload struct {
	Filters struct {
		Status   string   `json:"status"`
		Tags     []string `json:"tags"`
		Page     int      `json:"page"`
		PageSize int      `json:"page_size"`
	} `json:"filters"`
	UpdatePayload struct {
		Description string         `json:"description"`
		Content     string         `json:"content"`
		Metadata    map[string]any `json:"metadata"`
	} `json:"update_payload"`
}

type templateQualityPayload struct {
	ScanFilter struct {
		Q        string `json:"q"`
		Page     int    `json:"page"`
		PageSize int    `json:"page_size"`
	} `json:"scan_filter"`
	ValidateRules []string `json:"validate_rules"`
	Clone         struct {
		Copies            int    `json:"copies"`
		NamePrefix        string `json:"name_prefix"`
		DescriptionPrefix string `json:"description_prefix"`
	} `json:"clone"`
	UpdatePayload struct {
		Description string         `json:"description"`
		Content     string         `json:"content"`
		Metadata    map[string]any `json:"metadata"`
	} `json:"update_payload"`
	PublishChannel string `json:"publish_channel"`
}
