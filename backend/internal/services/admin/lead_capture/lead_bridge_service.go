package lead_capture

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	model "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/lead_capture"
	leadrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/lead_capture"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrInvalidLeadBridgePayload = errors.New("invalid lead bridge payload")
var ErrInvalidLeadBridgeField = errors.New("invalid lead bridge field")
var ErrLeadBridgeConflictOpen = errors.New("lead bridge conflict open")

type LeadBridgeService struct {
	db *gorm.DB
}

func NewLeadBridgeService(db *gorm.DB) *LeadBridgeService {
	return &LeadBridgeService{db: db}
}

type LeadBridgeIdentityUpdateRequest struct {
	LeadUUID         string            `json:"lead_uuid"`
	ExternalPluginID string            `json:"external_plugin_id"`
	ExternalLeadUUID string            `json:"external_lead_uuid"`
	Fields           map[string]string `json:"fields"`
	SourcePluginID   string            `json:"source_plugin_id"`
	EventUUID        string            `json:"event_uuid"`
}

type LeadBridgeMappingRequest struct {
	LeadUUID              string `json:"lead_uuid"`
	ExternalPluginID      string `json:"external_plugin_id"`
	ExternalLeadUUID      string `json:"external_lead_uuid"`
	ExternalLeadType      string `json:"external_lead_type"`
	RelationStatus        string `json:"relation_status"`
	LastExternalEventUUID string `json:"last_external_event_uuid"`
}

type LeadBridgeActivityRequest struct {
	LeadUUID         string         `json:"lead_uuid"`
	ExternalPluginID string         `json:"external_plugin_id"`
	ExternalLeadUUID string         `json:"external_lead_uuid"`
	EventUUID        string         `json:"event_uuid"`
	EventType        string         `json:"event_type"`
	ActorUUID        string         `json:"actor_uuid"`
	Payload          map[string]any `json:"payload"`
}

type LeadBridgeStatusRequest struct {
	LeadUUID         string `json:"lead_uuid"`
	ExternalPluginID string `json:"external_plugin_id"`
	ExternalLeadUUID string `json:"external_lead_uuid"`
}

type LeadBridgeConflictResolveRequest struct {
	ConflictUUID string `json:"conflict_uuid"`
	LeadUUID     string `json:"lead_uuid"`
	FieldName    string `json:"field_name"`
	Resolution   string `json:"resolution"`
	Value        string `json:"value"`
	ResolvedBy   string `json:"resolved_by"`
}

func (s *LeadBridgeService) UpdateIdentity(ctx context.Context, tenantUUID string, req LeadBridgeIdentityUpdateRequest) (map[string]any, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("lead bridge service unavailable")
	}
	tenantUUID = normalizeBridgeUUID(tenantUUID)
	leadUUID := normalizeBridgeUUID(req.LeadUUID)
	externalLeadUUID := normalizeBridgeUUID(req.ExternalLeadUUID)
	externalPluginID := strings.TrimSpace(req.ExternalPluginID)
	sourcePluginID := strings.TrimSpace(req.SourcePluginID)
	if sourcePluginID == "" {
		sourcePluginID = externalPluginID
	}
	if tenantUUID == "" || leadUUID == "" || externalPluginID == "" || externalLeadUUID == "" || len(req.Fields) == 0 {
		return nil, ErrInvalidLeadBridgePayload
	}
	for field := range req.Fields {
		if !isLeadBridgeIdentityField(field) {
			return nil, ErrInvalidLeadBridgeField
		}
	}

	result := map[string]any{"lead_uuid": leadUUID, "updated_fields": []string{}, "conflicts": []string{}}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var lead model.Lead
		if err := tx.Where("tenant_uuid = ? AND lead_uuid = ?", tenantUUID, leadUUID).First(&lead).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return leadrepo.ErrLeadNotFound
			}
			return err
		}
		updates := map[string]any{"updated_at": time.Now().UTC()}
		updatedFields := make([]string, 0, len(req.Fields))
		conflicts := make([]string, 0)
		for field, incoming := range req.Fields {
			incoming = strings.TrimSpace(incoming)
			local := leadBridgeLocalValue(&lead, field)
			if local != "" && incoming != "" && local != incoming {
				conflict := &model.LeadBridgeConflict{
					TenantUUID:       tenantUUID,
					LeadUUID:         leadUUID,
					ExternalPluginID: externalPluginID,
					ExternalLeadUUID: externalLeadUUID,
					FieldName:        field,
					LocalValue:       local,
					ExternalValue:    incoming,
					Status:           model.LeadBridgeConflictStatusOpen,
				}
				if err := tx.Create(conflict).Error; err != nil {
					return err
				}
				conflicts = append(conflicts, conflict.ConflictUUID)
				continue
			}
			updates[field] = incoming
			updatedFields = append(updatedFields, field)
			if err := upsertIdentitySyncState(ctx, tx, tenantUUID, leadUUID, externalPluginID, externalLeadUUID, field, incoming, sourcePluginID); err != nil {
				return err
			}
		}
		if len(updates) > 1 {
			if err := tx.Model(&model.Lead{}).
				Where("tenant_uuid = ? AND lead_uuid = ?", tenantUUID, leadUUID).
				Updates(updates).Error; err != nil {
				return err
			}
		}
		if err := recordExternalLeadActivity(ctx, tx, tenantUUID, leadUUID, model.LeadActivityTypeExternal, map[string]any{
			"event_uuid":          normalizeBridgeUUID(req.EventUUID),
			"event_type":          "identity.update",
			"source_plugin_id":    sourcePluginID,
			"external_plugin_id":  externalPluginID,
			"external_lead_uuid":  externalLeadUUID,
			"updated_fields":      updatedFields,
			"conflict_uuids":      conflicts,
			"object_sync_policy":  "identity_fields_only",
			"allowed_field_scope": []string{model.LeadBridgeFieldDisplayName, model.LeadBridgeFieldPhone, model.LeadBridgeFieldEmail},
		}); err != nil {
			return err
		}
		result["updated_fields"] = updatedFields
		result["conflicts"] = conflicts
		if len(conflicts) > 0 {
			result["status"] = "conflict"
			return ErrLeadBridgeConflictOpen
		}
		result["status"] = "accepted"
		return nil
	})
	if errors.Is(err, ErrLeadBridgeConflictOpen) {
		return result, nil
	}
	return result, err
}

func (s *LeadBridgeService) RecordActivity(ctx context.Context, tenantUUID string, req LeadBridgeActivityRequest) (map[string]any, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("lead bridge service unavailable")
	}
	tenantUUID = normalizeBridgeUUID(tenantUUID)
	leadUUID := normalizeBridgeUUID(req.LeadUUID)
	externalLeadUUID := normalizeBridgeUUID(req.ExternalLeadUUID)
	externalPluginID := strings.TrimSpace(req.ExternalPluginID)
	eventType := strings.TrimSpace(req.EventType)
	if tenantUUID == "" || leadUUID == "" || externalLeadUUID == "" || externalPluginID == "" || eventType == "" {
		return nil, ErrInvalidLeadBridgePayload
	}
	var activityUUID string
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := ensureBridgeLeadExists(ctx, tx, tenantUUID, leadUUID); err != nil {
			return err
		}
		payload := datatypes.JSONMap{}
		for key, value := range req.Payload {
			payload[key] = value
		}
		payload["event_uuid"] = normalizeBridgeUUID(req.EventUUID)
		payload["event_type"] = eventType
		payload["actor_uuid"] = normalizeBridgeUUID(req.ActorUUID)
		payload["external_plugin_id"] = externalPluginID
		payload["external_lead_uuid"] = externalLeadUUID
		payload["object_sync_policy"] = "notify_only"
		activity := &model.LeadActivity{
			LeadUUID:     leadUUID,
			TenantUUID:   tenantUUID,
			ActivityType: model.LeadActivityTypeExternal,
			Payload:      payload,
		}
		if err := tx.Create(activity).Error; err != nil {
			return err
		}
		activityUUID = activity.ActivityUUID
		return nil
	})
	if err != nil {
		return nil, err
	}
	return map[string]any{"status": "accepted", "activity_uuid": activityUUID}, nil
}

func (s *LeadBridgeService) UpsertMapping(ctx context.Context, tenantUUID string, req LeadBridgeMappingRequest) (*model.LeadBridgeMapping, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("lead bridge service unavailable")
	}
	tenantUUID = normalizeBridgeUUID(tenantUUID)
	leadUUID := normalizeBridgeUUID(req.LeadUUID)
	externalLeadUUID := normalizeBridgeUUID(req.ExternalLeadUUID)
	externalPluginID := strings.TrimSpace(req.ExternalPluginID)
	if tenantUUID == "" || leadUUID == "" || externalPluginID == "" || externalLeadUUID == "" {
		return nil, ErrInvalidLeadBridgePayload
	}
	leadType := strings.TrimSpace(req.ExternalLeadType)
	if leadType == "" {
		leadType = "sales_lead"
	}
	relationStatus := strings.TrimSpace(req.RelationStatus)
	if relationStatus == "" {
		relationStatus = "active"
	}
	mapping := &model.LeadBridgeMapping{
		TenantUUID:            tenantUUID,
		LeadUUID:              leadUUID,
		ExternalPluginID:      externalPluginID,
		ExternalLeadUUID:      externalLeadUUID,
		ExternalLeadType:      leadType,
		RelationStatus:        relationStatus,
		LastExternalEventUUID: normalizeBridgeUUID(req.LastExternalEventUUID),
	}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := ensureBridgeLeadExists(ctx, tx, tenantUUID, leadUUID); err != nil {
			return err
		}
		return tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "tenant_uuid"}, {Name: "external_plugin_id"}, {Name: "external_lead_uuid"}},
			DoUpdates: clause.Assignments(map[string]any{
				"lead_uuid":                leadUUID,
				"external_lead_type":       leadType,
				"relation_status":          relationStatus,
				"last_external_event_uuid": mapping.LastExternalEventUUID,
				"updated_at":               time.Now().UTC(),
			}),
		}).Create(mapping).Error
	})
	if err != nil {
		return nil, err
	}
	return mapping, nil
}

func (s *LeadBridgeService) GetStatus(ctx context.Context, tenantUUID string, req LeadBridgeStatusRequest) (map[string]any, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("lead bridge service unavailable")
	}
	tenantUUID = normalizeBridgeUUID(tenantUUID)
	leadUUID := normalizeBridgeUUID(req.LeadUUID)
	if tenantUUID == "" || leadUUID == "" {
		return nil, ErrInvalidLeadBridgePayload
	}
	var lead model.Lead
	if err := s.db.WithContext(ctx).Where("tenant_uuid = ? AND lead_uuid = ?", tenantUUID, leadUUID).First(&lead).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, leadrepo.ErrLeadNotFound
		}
		return nil, err
	}
	var mappings []model.LeadBridgeMapping
	q := s.db.WithContext(ctx).Where("tenant_uuid = ? AND lead_uuid = ?", tenantUUID, leadUUID)
	if pluginID := strings.TrimSpace(req.ExternalPluginID); pluginID != "" {
		q = q.Where("external_plugin_id = ?", pluginID)
	}
	if externalLeadUUID := normalizeBridgeUUID(req.ExternalLeadUUID); externalLeadUUID != "" {
		q = q.Where("external_lead_uuid = ?", externalLeadUUID)
	}
	if err := q.Order("updated_at DESC").Find(&mappings).Error; err != nil {
		return nil, err
	}
	var openConflictCount int64
	if err := s.db.WithContext(ctx).Model(&model.LeadBridgeConflict{}).
		Where("tenant_uuid = ? AND lead_uuid = ? AND status = ?", tenantUUID, leadUUID, model.LeadBridgeConflictStatusOpen).
		Count(&openConflictCount).Error; err != nil {
		return nil, err
	}
	return map[string]any{
		"lead_uuid":            leadUUID,
		"status":               lead.Status,
		"owner_user_uuid":      lead.OwnerUserUUID,
		"mappings":             mappings,
		"open_conflict_count":  openConflictCount,
		"funnel":               "private_domain_operations",
		"identity_sync_fields": []string{model.LeadBridgeFieldDisplayName, model.LeadBridgeFieldPhone, model.LeadBridgeFieldEmail},
	}, nil
}

func (s *LeadBridgeService) ResolveConflict(ctx context.Context, tenantUUID string, req LeadBridgeConflictResolveRequest) (map[string]any, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("lead bridge service unavailable")
	}
	tenantUUID = normalizeBridgeUUID(tenantUUID)
	conflictUUID := normalizeBridgeUUID(req.ConflictUUID)
	if tenantUUID == "" || conflictUUID == "" {
		return nil, ErrInvalidLeadBridgePayload
	}
	resolution := strings.TrimSpace(req.Resolution)
	if resolution == "" {
		return nil, ErrInvalidLeadBridgePayload
	}
	var out map[string]any
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var conflict model.LeadBridgeConflict
		if err := tx.Where("tenant_uuid = ? AND conflict_uuid = ? AND status = ?", tenantUUID, conflictUUID, model.LeadBridgeConflictStatusOpen).First(&conflict).Error; err != nil {
			return err
		}
		if !isLeadBridgeIdentityField(conflict.FieldName) {
			return ErrInvalidLeadBridgeField
		}
		value := strings.TrimSpace(req.Value)
		switch resolution {
		case "use_local":
			value = conflict.LocalValue
		case "use_external":
			value = conflict.ExternalValue
		case "custom":
			if value == "" {
				return ErrInvalidLeadBridgePayload
			}
		default:
			return ErrInvalidLeadBridgePayload
		}
		if resolution != "use_local" {
			if err := tx.Model(&model.Lead{}).
				Where("tenant_uuid = ? AND lead_uuid = ?", tenantUUID, conflict.LeadUUID).
				Updates(map[string]any{conflict.FieldName: value, "updated_at": time.Now().UTC()}).Error; err != nil {
				return err
			}
		}
		resolvedAt := time.Now().UTC()
		resolutionPayload := datatypes.JSONMap{
			"resolution":  resolution,
			"resolved_by": normalizeBridgeUUID(req.ResolvedBy),
			"value_hash":  hashBridgeValue(value),
		}
		if err := tx.Model(&model.LeadBridgeConflict{}).
			Where("tenant_uuid = ? AND conflict_uuid = ?", tenantUUID, conflictUUID).
			Updates(map[string]any{
				"status":      model.LeadBridgeConflictStatusResolved,
				"resolution":  resolutionPayload,
				"resolved_at": resolvedAt,
				"updated_at":  resolvedAt,
			}).Error; err != nil {
			return err
		}
		if err := upsertIdentitySyncState(ctx, tx, tenantUUID, conflict.LeadUUID, conflict.ExternalPluginID, conflict.ExternalLeadUUID, conflict.FieldName, value, "com.powerx.plugins.scrm"); err != nil {
			return err
		}
		if err := recordExternalLeadActivity(ctx, tx, tenantUUID, conflict.LeadUUID, model.LeadActivityTypeExternal, map[string]any{
			"event_type":         "identity.conflict.resolved",
			"conflict_uuid":      conflictUUID,
			"field_name":         conflict.FieldName,
			"external_plugin_id": conflict.ExternalPluginID,
			"external_lead_uuid": conflict.ExternalLeadUUID,
			"resolution":         resolution,
		}); err != nil {
			return err
		}
		out = map[string]any{"status": "resolved", "conflict_uuid": conflictUUID, "lead_uuid": conflict.LeadUUID}
		return nil
	})
	return out, err
}

func ensureBridgeLeadExists(ctx context.Context, tx *gorm.DB, tenantUUID, leadUUID string) error {
	var count int64
	if err := tx.WithContext(ctx).Model(&model.Lead{}).Where("tenant_uuid = ? AND lead_uuid = ?", tenantUUID, leadUUID).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return leadrepo.ErrLeadNotFound
	}
	return nil
}

func recordExternalLeadActivity(ctx context.Context, tx *gorm.DB, tenantUUID, leadUUID, activityType string, payload map[string]any) error {
	item := &model.LeadActivity{
		LeadUUID:     leadUUID,
		TenantUUID:   tenantUUID,
		ActivityType: activityType,
		Payload:      datatypes.JSONMap(payload),
	}
	return tx.WithContext(ctx).Create(item).Error
}

func upsertIdentitySyncState(ctx context.Context, tx *gorm.DB, tenantUUID, leadUUID, pluginID, externalLeadUUID, fieldName, value, sourcePluginID string) error {
	row := &model.LeadIdentitySyncState{
		TenantUUID:          tenantUUID,
		LeadUUID:            leadUUID,
		ExternalPluginID:    pluginID,
		ExternalLeadUUID:    externalLeadUUID,
		FieldName:           fieldName,
		LastSyncedValueHash: hashBridgeValue(value),
		LastSourcePluginID:  sourcePluginID,
		LastSyncedAt:        time.Now().UTC(),
	}
	return tx.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "tenant_uuid"}, {Name: "lead_uuid"}, {Name: "external_plugin_id"}, {Name: "external_lead_uuid"}, {Name: "field_name"}},
		DoUpdates: clause.Assignments(map[string]any{
			"last_synced_value_hash": row.LastSyncedValueHash,
			"last_source_plugin_id":  row.LastSourcePluginID,
			"last_synced_at":         row.LastSyncedAt,
			"updated_at":             time.Now().UTC(),
		}),
	}).Create(row).Error
}

func isLeadBridgeIdentityField(field string) bool {
	switch strings.TrimSpace(field) {
	case model.LeadBridgeFieldDisplayName, model.LeadBridgeFieldPhone, model.LeadBridgeFieldEmail:
		return true
	default:
		return false
	}
}

func leadBridgeLocalValue(lead *model.Lead, field string) string {
	if lead == nil {
		return ""
	}
	switch strings.TrimSpace(field) {
	case model.LeadBridgeFieldDisplayName:
		return strings.TrimSpace(lead.DisplayName)
	case model.LeadBridgeFieldPhone:
		return strings.TrimSpace(lead.Phone)
	case model.LeadBridgeFieldEmail:
		return strings.TrimSpace(lead.Email)
	default:
		return ""
	}
}

func normalizeBridgeUUID(value string) string {
	parsed, err := uuid.Parse(strings.TrimSpace(value))
	if err != nil || parsed == uuid.Nil {
		return ""
	}
	return strings.ToLower(parsed.String())
}

func hashBridgeValue(value string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(value)))
	return hex.EncodeToString(sum[:])
}
