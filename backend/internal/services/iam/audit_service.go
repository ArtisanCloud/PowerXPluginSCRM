package iam

import (
	"context"
	"strings"

	iamm "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/iam"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// AuditEntry captures the minimal information for IAM audit trails.
type AuditEntry struct {
	TenantUUID    string
	ActorMemberID *uint64
	Action        string
	Resource      string
	Diff          map[string]any
}

type AuditService struct {
	db *gorm.DB
}

// AuditListFilter constrains audit log queries.
type AuditListFilter struct {
	TenantUUID string
	Resource   string
	Action     string
	Limit      int
	AfterID    uint64
}

func NewAuditService(db *gorm.DB) *AuditService {
	return &AuditService{db: db}
}

func (s *AuditService) Record(ctx context.Context, entry AuditEntry) error {
	if s == nil || s.db == nil {
		return nil
	}
	if stringsTrim(entry.TenantUUID) == "" || stringsTrim(entry.Action) == "" || stringsTrim(entry.Resource) == "" {
		// 不记录空租户/动作
		return nil
	}
	record := &iamm.AuditLog{}
	record.TenantUuid = stringsTrim(entry.TenantUUID)
	record.Action = stringsTrim(entry.Action)
	record.Resource = stringsTrim(entry.Resource)
	if entry.ActorMemberID != nil {
		record.ActorUserID = entry.ActorMemberID
	}
	if entry.Diff != nil {
		record.Diff = datatypes.JSONMap(entry.Diff)
	}
	return s.db.WithContext(ctx).Create(record).Error
}

// List returns audit logs filtered by tenant/resource/action with optional pagination.
func (s *AuditService) List(ctx context.Context, filter AuditListFilter) ([]*iamm.AuditLog, error) {
	if s == nil || s.db == nil {
		return nil, nil
	}
	limit := filter.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	query := s.db.WithContext(ctx).Model(&iamm.AuditLog{}).Order("id DESC").Limit(limit)
	if filter.AfterID > 0 {
		query = query.Where("id < ?", filter.AfterID)
	}
	if tenant := stringsTrim(filter.TenantUUID); tenant != "" {
		query = query.Where("tenant_uuid = ?", tenant)
	}
	if res := stringsTrim(filter.Resource); res != "" {
		query = query.Where("resource = ?", res)
	}
	if act := stringsTrim(filter.Action); act != "" {
		query = query.Where("action = ?", act)
	}
	var records []*iamm.AuditLog
	if err := query.Find(&records).Error; err != nil {
		return nil, err
	}
	return records, nil
}

func stringsTrim(value string) string {
	return strings.TrimSpace(value)
}
