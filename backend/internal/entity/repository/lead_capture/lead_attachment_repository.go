package lead_capture

import (
	"context"
	"errors"
	"fmt"
	"strings"

	model "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/lead_capture"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type LeadAttachmentAuditRepairResult struct {
	Scanned int `json:"scanned"`
	Missing int `json:"missing"`
	Created int `json:"created"`
}

var ErrLeadActivityNotFound = errors.New("lead activity not found")
var ErrLeadAttachmentNotFound = errors.New("lead attachment not found")
var ErrLeadAttachmentScopeInvalid = errors.New("lead attachment scope invalid")

type LeadAttachmentRepository struct {
	*repository.BaseRepository[model.LeadAttachment]
}

func NewLeadAttachmentRepository(db *gorm.DB) *LeadAttachmentRepository {
	return &LeadAttachmentRepository{BaseRepository: repository.NewBaseRepository[model.LeadAttachment](db)}
}

func (r *LeadAttachmentRepository) CreateForActivity(ctx context.Context, tenantUUID, leadUUID, activityUUID string, attachment *model.LeadAttachment, auditActivity *model.LeadActivity) error {
	if r == nil || r.DB == nil {
		return errors.New("repository database is not initialized")
	}
	if attachment == nil {
		return errors.New("lead attachment is required")
	}
	if auditActivity == nil {
		return errors.New("lead attachment audit activity is required")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	leadUUID = strings.ToLower(strings.TrimSpace(leadUUID))
	activityUUID = strings.ToLower(strings.TrimSpace(activityUUID))
	if tenantUUID == "" {
		return repository.ErrTenantUuidRequired
	}
	attachment.TenantUUID = tenantUUID
	attachment.LeadUUID = leadUUID
	attachment.ActivityUUID = &activityUUID
	return r.WithTenantTx(ctx, tenantUUID, func(tx *gorm.DB) error {
		if err := ensureLeadExists(ctx, tx, tenantUUID, leadUUID); err != nil {
			return err
		}
		var activity model.LeadActivity
		if err := tx.WithContext(ctx).
			Where("tenant_uuid = ? AND lead_uuid = ? AND activity_uuid = ?", tenantUUID, leadUUID, activityUUID).
			First(&activity).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrLeadActivityNotFound
			}
			return err
		}
		payload := map[string]any(activity.Payload)
		if attachment.StageKey == "" {
			attachment.StageKey = strings.ToLower(attachmentPayloadText(payload, "stage_key"))
		}
		if attachment.ActionKey == "" {
			attachment.ActionKey = attachmentPayloadText(payload, "action_key")
		}
		if !model.IsValidLeadStatus(attachment.StageKey) {
			return ErrLeadAttachmentScopeInvalid
		}
		if err := tx.WithContext(ctx).Create(attachment).Error; err != nil {
			return err
		}
		prepareAttachmentAuditActivity(auditActivity, tenantUUID, leadUUID, attachment, activityUUID)
		return tx.WithContext(ctx).Create(auditActivity).Error
	})
}

func (r *LeadAttachmentRepository) RepairMissingUploadAudits(ctx context.Context, tenantUUID string, apply bool) (*LeadAttachmentAuditRepairResult, error) {
	if r == nil || r.DB == nil {
		return nil, errors.New("repository database is not initialized")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	if tenantUUID == "" {
		return nil, repository.ErrTenantUuidRequired
	}
	result := &LeadAttachmentAuditRepairResult{}
	err := r.WithTenantTx(ctx, tenantUUID, func(tx *gorm.DB) error {
		var attachments []*model.LeadAttachment
		if err := tx.WithContext(ctx).Where("tenant_uuid = ?", tenantUUID).Order("created_at ASC").Find(&attachments).Error; err != nil {
			return err
		}
		result.Scanned = len(attachments)
		var audits []*model.LeadActivity
		if err := tx.WithContext(ctx).
			Where("tenant_uuid = ? AND activity_type = ?", tenantUUID, model.LeadActivityTypeAttachmentUploaded).
			Find(&audits).Error; err != nil {
			return err
		}
		existing := make(map[string]struct{}, len(audits))
		for _, audit := range audits {
			if audit == nil {
				continue
			}
			if attachmentUUID := strings.ToLower(attachmentPayloadText(map[string]any(audit.Payload), "attachment_uuid")); attachmentUUID != "" {
				existing[attachmentUUID] = struct{}{}
			}
		}
		for _, attachment := range attachments {
			if attachment == nil {
				continue
			}
			attachmentUUID := strings.ToLower(strings.TrimSpace(attachment.AttachmentUUID))
			if _, ok := existing[attachmentUUID]; ok {
				continue
			}
			result.Missing++
			if !apply {
				continue
			}
			audit := &model.LeadActivity{
				ActivityUUID: uuid.NewString(),
				TenantUUID:   tenantUUID,
				LeadUUID:     attachment.LeadUUID,
				ActivityType: model.LeadActivityTypeAttachmentUploaded,
				CreatedAt:    attachment.CreatedAt,
				UpdatedAt:    attachment.CreatedAt,
				Payload: datatypes.JSONMap{
					"actor_type":    model.LeadAuditActorTypeSystem,
					"repair_source": "historical_attachment",
				},
			}
			parentActivityUUID := ""
			if attachment.ActivityUUID != nil {
				parentActivityUUID = strings.TrimSpace(*attachment.ActivityUUID)
			}
			prepareAttachmentAuditActivity(audit, tenantUUID, attachment.LeadUUID, attachment, parentActivityUUID)
			if err := tx.WithContext(ctx).Create(audit).Error; err != nil {
				return err
			}
			existing[attachmentUUID] = struct{}{}
			result.Created++
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func attachmentPayloadText(payload map[string]any, key string) string {
	value, ok := payload[key]
	if !ok || value == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

func (r *LeadAttachmentRepository) CreateForNode(ctx context.Context, tenantUUID, leadUUID string, attachment *model.LeadAttachment, auditActivity *model.LeadActivity) error {
	if r == nil || r.DB == nil {
		return errors.New("repository database is not initialized")
	}
	if attachment == nil {
		return errors.New("lead attachment is required")
	}
	if auditActivity == nil {
		return errors.New("lead attachment audit activity is required")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	leadUUID = strings.ToLower(strings.TrimSpace(leadUUID))
	if tenantUUID == "" {
		return repository.ErrTenantUuidRequired
	}
	attachment.TenantUUID = tenantUUID
	attachment.LeadUUID = leadUUID
	attachment.ActivityUUID = nil
	return r.WithTenantTx(ctx, tenantUUID, func(tx *gorm.DB) error {
		if err := ensureLeadExists(ctx, tx, tenantUUID, leadUUID); err != nil {
			return err
		}
		if err := tx.WithContext(ctx).Create(attachment).Error; err != nil {
			return err
		}
		prepareAttachmentAuditActivity(auditActivity, tenantUUID, leadUUID, attachment, "")
		return tx.WithContext(ctx).Create(auditActivity).Error
	})
}

func prepareAttachmentAuditActivity(auditActivity *model.LeadActivity, tenantUUID, leadUUID string, attachment *model.LeadAttachment, parentActivityUUID string) {
	auditActivity.TenantUUID = tenantUUID
	auditActivity.LeadUUID = leadUUID
	if auditActivity.Payload == nil {
		auditActivity.Payload = map[string]any{}
	}
	auditActivity.Payload["attachment_uuid"] = attachment.AttachmentUUID
	auditActivity.Payload["stage_key"] = attachment.StageKey
	auditActivity.Payload["action_key"] = attachment.ActionKey
	auditActivity.Payload["file_name"] = attachment.FileName
	auditActivity.Payload["content_type"] = attachment.ContentType
	auditActivity.Payload["file_size"] = attachment.FileSize
	auditActivity.Payload["storage_provider"] = attachment.StorageProvider
	if parentActivityUUID != "" {
		auditActivity.Payload["parent_activity_uuid"] = parentActivityUUID
	}
}

func (r *LeadAttachmentRepository) DeleteWithAudit(ctx context.Context, tenantUUID, leadUUID, attachmentUUID string, auditActivity *model.LeadActivity) (*model.LeadAttachment, error) {
	if r == nil || r.DB == nil {
		return nil, errors.New("repository database is not initialized")
	}
	if auditActivity == nil {
		return nil, errors.New("lead attachment audit activity is required")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	leadUUID = strings.ToLower(strings.TrimSpace(leadUUID))
	attachmentUUID = strings.ToLower(strings.TrimSpace(attachmentUUID))
	if tenantUUID == "" {
		return nil, repository.ErrTenantUuidRequired
	}
	var attachment model.LeadAttachment
	err := r.WithTenantTx(ctx, tenantUUID, func(tx *gorm.DB) error {
		if err := ensureLeadExists(ctx, tx, tenantUUID, leadUUID); err != nil {
			return err
		}
		if err := tx.WithContext(ctx).
			Where("tenant_uuid = ? AND lead_uuid = ? AND attachment_uuid = ?", tenantUUID, leadUUID, attachmentUUID).
			First(&attachment).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrLeadAttachmentNotFound
			}
			return err
		}
		parentActivityUUID := ""
		if attachment.ActivityUUID != nil {
			parentActivityUUID = strings.TrimSpace(*attachment.ActivityUUID)
		}
		prepareAttachmentAuditActivity(auditActivity, tenantUUID, leadUUID, &attachment, parentActivityUUID)
		auditActivity.ActivityType = model.LeadActivityTypeAttachmentDeleted
		if err := tx.WithContext(ctx).Create(auditActivity).Error; err != nil {
			return err
		}
		return tx.WithContext(ctx).
			Where("tenant_uuid = ? AND lead_uuid = ? AND attachment_uuid = ?", tenantUUID, leadUUID, attachmentUUID).
			Delete(&model.LeadAttachment{}).Error
	})
	if err != nil {
		return nil, err
	}
	return &attachment, nil
}

func (r *LeadAttachmentRepository) ListForActivity(ctx context.Context, tenantUUID, leadUUID, activityUUID string) ([]*model.LeadAttachment, error) {
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	leadUUID = strings.ToLower(strings.TrimSpace(leadUUID))
	activityUUID = strings.ToLower(strings.TrimSpace(activityUUID))
	return r.list(ctx, tenantUUID, func(tx *gorm.DB) error {
		if err := ensureLeadExists(ctx, tx, tenantUUID, leadUUID); err != nil {
			return err
		}
		return ensureLeadActivityExists(ctx, tx, tenantUUID, leadUUID, activityUUID)
	}, func(tx *gorm.DB) *gorm.DB {
		return tx.Where("tenant_uuid = ? AND lead_uuid = ? AND activity_uuid = ?", tenantUUID, leadUUID, activityUUID)
	})
}

func (r *LeadAttachmentRepository) ListForNode(ctx context.Context, tenantUUID, leadUUID, stageKey, actionKey string) ([]*model.LeadAttachment, error) {
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	leadUUID = strings.ToLower(strings.TrimSpace(leadUUID))
	stageKey = strings.ToLower(strings.TrimSpace(stageKey))
	actionKey = strings.TrimSpace(actionKey)
	return r.list(ctx, tenantUUID, func(tx *gorm.DB) error {
		return ensureLeadExists(ctx, tx, tenantUUID, leadUUID)
	}, func(tx *gorm.DB) *gorm.DB {
		query := tx.Where("tenant_uuid = ? AND lead_uuid = ? AND activity_uuid IS NULL AND stage_key = ?", tenantUUID, leadUUID, stageKey)
		if actionKey != "" {
			query = query.Where("action_key = ?", actionKey)
		}
		return query
	})
}

func (r *LeadAttachmentRepository) Get(ctx context.Context, tenantUUID, leadUUID, attachmentUUID string) (*model.LeadAttachment, error) {
	if r == nil || r.DB == nil {
		return nil, errors.New("repository database is not initialized")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	leadUUID = strings.ToLower(strings.TrimSpace(leadUUID))
	attachmentUUID = strings.ToLower(strings.TrimSpace(attachmentUUID))
	if tenantUUID == "" {
		return nil, repository.ErrTenantUuidRequired
	}
	var out model.LeadAttachment
	err := r.WithTenantTx(ctx, tenantUUID, func(tx *gorm.DB) error {
		return tx.WithContext(ctx).
			Where("tenant_uuid = ? AND lead_uuid = ? AND attachment_uuid = ?", tenantUUID, leadUUID, attachmentUUID).
			First(&out).Error
	})
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrLeadAttachmentNotFound
	}
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (r *LeadAttachmentRepository) list(ctx context.Context, tenantUUID string, validate func(*gorm.DB) error, scope func(*gorm.DB) *gorm.DB) ([]*model.LeadAttachment, error) {
	if r == nil || r.DB == nil {
		return nil, errors.New("repository database is not initialized")
	}
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	if tenantUUID == "" {
		return nil, repository.ErrTenantUuidRequired
	}
	var out []*model.LeadAttachment
	err := r.WithTenantTx(ctx, tenantUUID, func(tx *gorm.DB) error {
		if validate != nil {
			if err := validate(tx); err != nil {
				return err
			}
		}
		query := scope(tx.WithContext(ctx)).
			Select("attachment_uuid", "tenant_uuid", "lead_uuid", "activity_uuid", "stage_key", "action_key", "file_name", "content_type", "file_size", "storage_provider", "created_at", "updated_at")
		return query.Order("created_at DESC").Find(&out).Error
	})
	return out, err
}

func ensureLeadActivityExists(ctx context.Context, tx *gorm.DB, tenantUUID, leadUUID, activityUUID string) error {
	var activity model.LeadActivity
	if err := tx.WithContext(ctx).
		Select("activity_uuid").
		Where("tenant_uuid = ? AND lead_uuid = ? AND activity_uuid = ?", tenantUUID, leadUUID, activityUUID).
		First(&activity).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrLeadActivityNotFound
		}
		return err
	}
	return nil
}

func ensureLeadExists(ctx context.Context, tx *gorm.DB, tenantUUID, leadUUID string) error {
	var lead model.Lead
	if err := tx.WithContext(ctx).
		Select("lead_uuid").
		Where("tenant_uuid = ? AND lead_uuid = ?", tenantUUID, leadUUID).
		First(&lead).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrLeadNotFound
		}
		return err
	}
	return nil
}
