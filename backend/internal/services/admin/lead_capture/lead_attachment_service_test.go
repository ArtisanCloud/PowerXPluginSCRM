package lead_capture

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	basemodels "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	leadmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/lead_capture"
	leadrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/lead_capture"
	authx "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/middleware"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestLeadAttachmentService_SeparatesNodeAndActivityScopes(t *testing.T) {
	db := openLeadAttachmentServiceDB(t, "lead_attachment_scope")
	service := NewLeadService(leadrepo.NewLeadRepository(db))
	tenantUUID := "00000000-0000-4000-8000-000000000001"
	otherTenantUUID := "00000000-0000-4000-8000-000000000002"
	leadUUID := "10000000-0000-4000-8000-000000000001"
	activityUUID := "20000000-0000-4000-8000-000000000001"
	operatorMemberUUID := "30000000-0000-4000-8000-000000000001"
	requestContext := authx.ContextWithMemberUUID(context.Background(), operatorMemberUUID)

	require.NoError(t, db.Create(&leadmodel.Lead{
		LeadUUID: leadUUID, TenantUUID: tenantUUID, DisplayName: "lead", Status: leadmodel.LeadStatusEngaging,
	}).Error)
	require.NoError(t, db.Create(&leadmodel.LeadActivity{
		ActivityUUID: activityUUID,
		TenantUUID:   tenantUUID,
		LeadUUID:     leadUUID,
		ActivityType: leadmodel.LeadActivityTypeManual,
		Payload: datatypes.JSONMap{
			"stage_key":  leadmodel.LeadStatusEngaging,
			"action_key": "activity",
		},
	}).Error)

	nodeAttachment, err := service.UploadNodeAttachment(requestContext, tenantUUID, leadUUID, LeadAttachmentUploadRequest{
		StageKey:    leadmodel.LeadStatusEngaging,
		ActionKey:   "engagement",
		FileName:    "node.txt",
		ContentType: "text/plain",
		FileSize:    4,
		Content:     bytes.NewBufferString("node"),
	})
	require.NoError(t, err)
	require.Nil(t, nodeAttachment.ActivityUUID)

	activityAttachment, err := service.UploadActivityAttachment(requestContext, tenantUUID, leadUUID, LeadAttachmentUploadRequest{
		ActivityUUID: activityUUID,
		FileName:     "activity.txt",
		ContentType:  "text/plain",
		FileSize:     8,
		Content:      bytes.NewBufferString("activity"),
	})
	require.NoError(t, err)
	require.NotNil(t, activityAttachment.ActivityUUID)
	require.Equal(t, leadmodel.LeadStatusEngaging, activityAttachment.StageKey)
	require.Equal(t, "activity", activityAttachment.ActionKey)

	nodeItems, err := service.ListNodeAttachments(context.Background(), tenantUUID, leadUUID, leadmodel.LeadStatusEngaging, "engagement")
	require.NoError(t, err)
	require.Len(t, nodeItems, 1)
	require.Equal(t, nodeAttachment.AttachmentUUID, nodeItems[0].AttachmentUUID)

	activityItems, err := service.ListActivityAttachments(context.Background(), tenantUUID, leadUUID, activityUUID)
	require.NoError(t, err)
	require.Len(t, activityItems, 1)
	require.Equal(t, activityAttachment.AttachmentUUID, activityItems[0].AttachmentUUID)

	var auditActivities []leadmodel.LeadActivity
	require.NoError(t, db.Where("tenant_uuid = ? AND lead_uuid = ? AND activity_type = ?", tenantUUID, leadUUID, leadmodel.LeadActivityTypeAttachmentUploaded).
		Order("created_at ASC").Find(&auditActivities).Error)
	require.Len(t, auditActivities, 2)
	require.Equal(t, nodeAttachment.AttachmentUUID, auditActivities[0].Payload["attachment_uuid"])
	require.Equal(t, leadmodel.LeadStatusEngaging, auditActivities[0].Payload["stage_key"])
	require.Equal(t, "engagement", auditActivities[0].Payload["action_key"])
	require.Equal(t, operatorMemberUUID, auditActivities[0].Payload["operator_member_uuid"])
	require.Equal(t, activityAttachment.AttachmentUUID, auditActivities[1].Payload["attachment_uuid"])
	require.Equal(t, activityUUID, auditActivities[1].Payload["parent_activity_uuid"])
	require.Equal(t, "activity", auditActivities[1].Payload["action_key"])

	_, err = service.GetAttachment(context.Background(), otherTenantUUID, leadUUID, nodeAttachment.AttachmentUUID)
	require.ErrorIs(t, err, leadrepo.ErrLeadAttachmentNotFound)

	_, err = service.ListNodeAttachments(context.Background(), otherTenantUUID, leadUUID, leadmodel.LeadStatusEngaging, "engagement")
	require.ErrorIs(t, err, leadrepo.ErrLeadNotFound)

	_, err = service.DeleteAttachment(requestContext, otherTenantUUID, leadUUID, nodeAttachment.AttachmentUUID)
	require.ErrorIs(t, err, leadrepo.ErrLeadNotFound)

	deletedAttachment, err := service.DeleteAttachment(requestContext, tenantUUID, leadUUID, nodeAttachment.AttachmentUUID)
	require.NoError(t, err)
	require.Equal(t, nodeAttachment.AttachmentUUID, deletedAttachment.AttachmentUUID)
	_, err = service.GetAttachment(context.Background(), tenantUUID, leadUUID, nodeAttachment.AttachmentUUID)
	require.ErrorIs(t, err, leadrepo.ErrLeadAttachmentNotFound)

	var deletedAudit leadmodel.LeadActivity
	require.NoError(t, db.Where("tenant_uuid = ? AND lead_uuid = ? AND activity_type = ?", tenantUUID, leadUUID, leadmodel.LeadActivityTypeAttachmentDeleted).First(&deletedAudit).Error)
	require.Equal(t, nodeAttachment.AttachmentUUID, deletedAudit.Payload["attachment_uuid"])
	require.Equal(t, "node.txt", deletedAudit.Payload["file_name"])
	require.Equal(t, operatorMemberUUID, deletedAudit.Payload["operator_member_uuid"])
}

func TestLeadService_RecordActivityPersistsOperatorMemberUUID(t *testing.T) {
	db := openLeadAttachmentServiceDB(t, "lead_manual_activity_operator")
	service := NewLeadService(leadrepo.NewLeadRepository(db))
	tenantUUID := "00000000-0000-4000-8000-000000000001"
	leadUUID := "10000000-0000-4000-8000-000000000001"
	operatorMemberUUID := "30000000-0000-4000-8000-000000000001"
	requestContext := authx.ContextWithMemberUUID(context.Background(), operatorMemberUUID)

	require.NoError(t, db.Create(&leadmodel.Lead{
		LeadUUID: leadUUID, TenantUUID: tenantUUID, DisplayName: "lead", Status: leadmodel.LeadStatusEngaging,
	}).Error)

	activity, err := service.RecordActivity(requestContext, tenantUUID, leadUUID, LeadActivityCreateRequest{
		Method:    "wechat",
		Content:   "确认需求",
		StageKey:  leadmodel.LeadStatusEngaging,
		ActionKey: "engagement",
	})
	require.NoError(t, err)
	require.Equal(t, operatorMemberUUID, activity.Payload["operator_member_uuid"])
	require.Equal(t, leadmodel.LeadAuditActorTypeMember, activity.Payload["actor_type"])
}

func TestLeadAttachmentService_RollsBackAttachmentWhenAuditWriteFails(t *testing.T) {
	db := openLeadAttachmentServiceDB(t, "lead_attachment_audit_rollback")
	service := NewLeadService(leadrepo.NewLeadRepository(db))
	tenantUUID := "00000000-0000-4000-8000-000000000001"
	leadUUID := "10000000-0000-4000-8000-000000000001"
	require.NoError(t, db.Create(&leadmodel.Lead{
		LeadUUID: leadUUID, TenantUUID: tenantUUID, DisplayName: "lead", Status: leadmodel.LeadStatusCaptured,
	}).Error)
	require.NoError(t, db.Exec(`CREATE TRIGGER reject_attachment_audit
		BEFORE INSERT ON lead_capture_activities
		WHEN NEW.activity_type = 'attachment_uploaded'
		BEGIN SELECT RAISE(ABORT, 'attachment audit rejected'); END;`).Error)

	_, err := service.UploadNodeAttachment(context.Background(), tenantUUID, leadUUID, LeadAttachmentUploadRequest{
		StageKey:  leadmodel.LeadStatusCaptured,
		ActionKey: "capture",
		FileName:  "rollback.txt",
		Content:   bytes.NewBufferString("rollback"),
	})
	require.Error(t, err)

	var attachmentCount int64
	require.NoError(t, db.Model(&leadmodel.LeadAttachment{}).Where("tenant_uuid = ? AND lead_uuid = ?", tenantUUID, leadUUID).Count(&attachmentCount).Error)
	require.Zero(t, attachmentCount)
}

func TestLeadAttachmentService_RollsBackDeleteWhenAuditWriteFails(t *testing.T) {
	db := openLeadAttachmentServiceDB(t, "lead_attachment_delete_audit_rollback")
	service := NewLeadService(leadrepo.NewLeadRepository(db))
	tenantUUID := "00000000-0000-4000-8000-000000000001"
	leadUUID := "10000000-0000-4000-8000-000000000001"
	attachmentUUID := "40000000-0000-4000-8000-000000000001"
	require.NoError(t, db.Create(&leadmodel.Lead{
		LeadUUID: leadUUID, TenantUUID: tenantUUID, DisplayName: "lead", Status: leadmodel.LeadStatusCaptured,
	}).Error)
	require.NoError(t, db.Create(&leadmodel.LeadAttachment{
		AttachmentUUID: attachmentUUID, TenantUUID: tenantUUID, LeadUUID: leadUUID,
		StageKey: leadmodel.LeadStatusCaptured, ActionKey: "capture", FileName: "keep.txt",
		FileSize: 4, StorageProvider: leadmodel.LeadAttachmentStorageProviderDatabase, Content: []byte("keep"),
	}).Error)
	require.NoError(t, db.Exec(`CREATE TRIGGER reject_attachment_delete_audit
		BEFORE INSERT ON lead_capture_activities
		WHEN NEW.activity_type = 'attachment_deleted'
		BEGIN SELECT RAISE(ABORT, 'attachment delete audit rejected'); END;`).Error)

	_, err := service.DeleteAttachment(context.Background(), tenantUUID, leadUUID, attachmentUUID)
	require.Error(t, err)

	kept, err := service.GetAttachment(context.Background(), tenantUUID, leadUUID, attachmentUUID)
	require.NoError(t, err)
	require.Equal(t, "keep.txt", kept.FileName)
}

func TestLeadAttachmentService_RejectsInvalidStageAndOversizedFile(t *testing.T) {
	db := openLeadAttachmentServiceDB(t, "lead_attachment_validation")
	service := NewLeadService(leadrepo.NewLeadRepository(db))
	tenantUUID := "00000000-0000-4000-8000-000000000001"
	leadUUID := "10000000-0000-4000-8000-000000000001"
	require.NoError(t, db.Create(&leadmodel.Lead{
		LeadUUID: leadUUID, TenantUUID: tenantUUID, DisplayName: "lead", Status: leadmodel.LeadStatusCaptured,
	}).Error)

	_, err := service.UploadNodeAttachment(context.Background(), tenantUUID, leadUUID, LeadAttachmentUploadRequest{
		StageKey: "sales_won",
		FileName: "invalid.txt",
		FileSize: 1,
		Content:  bytes.NewBufferString("x"),
	})
	require.ErrorIs(t, err, ErrInvalidLeadPayload)

	_, err = service.UploadNodeAttachment(context.Background(), tenantUUID, leadUUID, LeadAttachmentUploadRequest{
		StageKey: leadmodel.LeadStatusCaptured,
		FileName: "large.bin",
		FileSize: maxLeadAttachmentBytes + 1,
		Content:  bytes.NewBufferString("x"),
	})
	require.ErrorIs(t, err, ErrLeadAttachmentTooLarge)

	_, err = service.UploadActivityAttachment(context.Background(), tenantUUID, leadUUID, LeadAttachmentUploadRequest{
		ActivityUUID: "20000000-0000-4000-8000-000000000099",
		FileName:     "missing.txt",
		FileSize:     1,
		Content:      bytes.NewBufferString("x"),
	})
	require.True(t, errors.Is(err, ErrInvalidLeadPayload))
}

func TestLeadAttachmentAuditRepair_DryRunApplyAndIdempotency(t *testing.T) {
	db := openLeadAttachmentServiceDB(t, "lead_attachment_audit_repair")
	service := NewLeadService(leadrepo.NewLeadRepository(db))
	tenantUUID := "00000000-0000-4000-8000-000000000001"
	otherTenantUUID := "00000000-0000-4000-8000-000000000002"
	leadUUID := "10000000-0000-4000-8000-000000000001"
	otherLeadUUID := "10000000-0000-4000-8000-000000000002"
	attachmentUUID := "40000000-0000-4000-8000-000000000001"
	createdAt := time.Date(2026, 8, 6, 22, 7, 0, 0, time.UTC)

	for _, lead := range []*leadmodel.Lead{
		{LeadUUID: leadUUID, TenantUUID: tenantUUID, DisplayName: "lead", Status: leadmodel.LeadStatusCaptured},
		{LeadUUID: otherLeadUUID, TenantUUID: otherTenantUUID, DisplayName: "other", Status: leadmodel.LeadStatusCaptured},
	} {
		require.NoError(t, db.Create(lead).Error)
	}
	for _, attachment := range []*leadmodel.LeadAttachment{
		{
			AttachmentUUID: attachmentUUID, TenantUUID: tenantUUID, LeadUUID: leadUUID,
			StageKey: leadmodel.LeadStatusCaptured, ActionKey: "capture", FileName: "historical.xlsx",
			FileSize: 7, StorageProvider: leadmodel.LeadAttachmentStorageProviderDatabase, Content: []byte("history"), CreatedAt: createdAt,
		},
		{
			AttachmentUUID: "40000000-0000-4000-8000-000000000002", TenantUUID: otherTenantUUID, LeadUUID: otherLeadUUID,
			StageKey: leadmodel.LeadStatusCaptured, ActionKey: "capture", FileName: "other.xlsx",
			FileSize: 5, StorageProvider: leadmodel.LeadAttachmentStorageProviderDatabase, Content: []byte("other"), CreatedAt: createdAt,
		},
	} {
		require.NoError(t, db.Create(attachment).Error)
	}

	dryRun, err := service.RepairHistoricalAttachmentAudits(context.Background(), tenantUUID, false)
	require.NoError(t, err)
	require.Equal(t, &leadrepo.LeadAttachmentAuditRepairResult{Scanned: 1, Missing: 1, Created: 0}, dryRun)

	var auditCount int64
	require.NoError(t, db.Model(&leadmodel.LeadActivity{}).Count(&auditCount).Error)
	require.Zero(t, auditCount)

	applied, err := service.RepairHistoricalAttachmentAudits(context.Background(), tenantUUID, true)
	require.NoError(t, err)
	require.Equal(t, &leadrepo.LeadAttachmentAuditRepairResult{Scanned: 1, Missing: 1, Created: 1}, applied)

	var audit leadmodel.LeadActivity
	require.NoError(t, db.Where("tenant_uuid = ? AND activity_type = ?", tenantUUID, leadmodel.LeadActivityTypeAttachmentUploaded).First(&audit).Error)
	require.Equal(t, attachmentUUID, audit.Payload["attachment_uuid"])
	require.Equal(t, leadmodel.LeadAuditActorTypeSystem, audit.Payload["actor_type"])
	require.Equal(t, "historical_attachment", audit.Payload["repair_source"])
	require.WithinDuration(t, createdAt, audit.CreatedAt, time.Second)

	repeated, err := service.RepairHistoricalAttachmentAudits(context.Background(), tenantUUID, true)
	require.NoError(t, err)
	require.Equal(t, &leadrepo.LeadAttachmentAuditRepairResult{Scanned: 1, Missing: 0, Created: 0}, repeated)
	require.NoError(t, db.Model(&leadmodel.LeadActivity{}).Count(&auditCount).Error)
	require.EqualValues(t, 1, auditCount)
}

func openLeadAttachmentServiceDB(t *testing.T, name string) *gorm.DB {
	t.Helper()
	basemodels.ForceSchemaForTests("")
	db, err := gorm.Open(sqlite.Open("file:"+name+"?mode=memory&cache=shared"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	require.NoError(t, err)

	statements := []string{
		`CREATE TABLE lead_capture_leads (
			lead_uuid TEXT PRIMARY KEY, tenant_uuid TEXT NOT NULL, display_name TEXT, phone TEXT, email TEXT,
			status TEXT NOT NULL, owner_user_uuid TEXT, source_channel TEXT, source_app_type TEXT,
			source_account_uuid TEXT, created_at DATETIME, updated_at DATETIME
		);`,
		`CREATE TABLE lead_capture_activities (
			activity_uuid TEXT PRIMARY KEY, lead_uuid TEXT NOT NULL, tenant_uuid TEXT NOT NULL,
			activity_type TEXT NOT NULL, payload TEXT, created_at DATETIME, updated_at DATETIME
		);`,
		`CREATE TABLE lead_capture_attachments (
			attachment_uuid TEXT PRIMARY KEY, tenant_uuid TEXT NOT NULL, lead_uuid TEXT NOT NULL,
			activity_uuid TEXT NULL, stage_key TEXT, action_key TEXT, file_name TEXT NOT NULL,
			content_type TEXT, file_size INTEGER NOT NULL DEFAULT 0, storage_provider TEXT NOT NULL,
			content BLOB NOT NULL, created_at DATETIME, updated_at DATETIME
		);`,
	}
	for _, statement := range statements {
		require.NoError(t, db.Exec(statement).Error)
	}
	return db
}
