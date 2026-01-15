package iam

import (
	"context"
	"errors"
	"fmt"
	"strings"

	iamm "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/iam"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TenantService struct {
	db    *gorm.DB
	audit *AuditService
}

func NewTenantService(db *gorm.DB, audit *AuditService) *TenantService {
	return &TenantService{db: db, audit: audit}
}

type TenantListFilter struct {
	Status   string
	Query    string
	Page     int
	PageSize int
}

type TenantListResult struct {
	Items []iamm.Tenant
	Total int64
}

func (s *TenantService) List(ctx context.Context) ([]iamm.Tenant, error) {
	result, err := s.ListWithFilter(ctx, TenantListFilter{})
	if err != nil {
		return nil, err
	}
	return result.Items, nil
}

func (s *TenantService) ListWithFilter(ctx context.Context, filter TenantListFilter) (*TenantListResult, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("iam: tenant service unavailable")
	}
	query := s.db.WithContext(ctx).Model(&iamm.Tenant{})
	if status := normalizeTenantStatus(filter.Status); status != "" {
		query = query.Where("status = ?", status)
	}
	if q := strings.TrimSpace(strings.ToLower(filter.Query)); q != "" {
		like := "%" + q + "%"
		query = query.Where("lower(name) LIKE ? OR lower(key) LIKE ?", like, like)
	}
	page := filter.Page
	if page <= 0 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	countQuery := query.Session(&gorm.Session{})
	var total int64
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, err
	}
	var tenants []iamm.Tenant
	if err := query.Order("created_at ASC").
		Limit(pageSize).
		Offset((page - 1) * pageSize).
		Find(&tenants).Error; err != nil {
		return nil, err
	}
	return &TenantListResult{Items: tenants, Total: total}, nil
}

type CreateTenantInput struct {
	Key     string
	Name    string
	Plan    string
	Status  string
	ActorID *uint64
}

func (s *TenantService) Create(ctx context.Context, input CreateTenantInput) (*iamm.Tenant, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("iam: tenant service unavailable")
	}
	key, err := normalizeTenantKey(input.Key)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, errors.New("tenant name required")
	}
	status := normalizeTenantStatus(input.Status)
	if status == "" {
		status = iamm.StatusActive
	}
	plan := strings.TrimSpace(input.Plan)
	if plan == "" {
		plan = "free"
	}
	var exists int64
	if err := s.db.WithContext(ctx).Model(&iamm.Tenant{}).
		Where("lower(key) = ?", key).
		Count(&exists).Error; err != nil {
		return nil, err
	}
	if exists > 0 {
		return nil, fmt.Errorf("tenant key %s already exists", key)
	}
	tenant := &iamm.Tenant{
		UUID:   uuid.NewString(),
		Key:    key,
		Name:   name,
		Status: status,
		Plan:   plan,
	}
	if err := s.db.WithContext(ctx).Create(tenant).Error; err != nil {
		return nil, err
	}
	if s.audit != nil {
		_ = s.audit.Record(ctx, AuditEntry{
			TenantUUID:    tenantUUIDValue(tenant),
			ActorMemberID: input.ActorID,
			Action:        "create",
			Resource:      "iam.tenant",
			Diff: map[string]any{
				"name":   tenant.Name,
				"status": tenant.Status,
				"plan":   tenant.Plan,
			},
		})
	}
	return tenant, nil
}

type UpdateTenantInput struct {
	Name    string
	Status  string
	Plan    string
	ActorID *uint64
}

func (s *TenantService) Update(ctx context.Context, id uint64, input UpdateTenantInput) (*iamm.Tenant, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("iam: tenant service unavailable")
	}
	var tenant iamm.Tenant
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&tenant).Error; err != nil {
		return nil, err
	}

	changes := map[string]any{}
	if name := strings.TrimSpace(input.Name); name != "" && name != tenant.Name {
		changes["name"] = name
	}
	if status := normalizeTenantStatus(input.Status); status != "" && status != tenant.Status {
		changes["status"] = status
	}
	if plan := strings.TrimSpace(input.Plan); plan != "" && plan != tenant.Plan {
		changes["plan"] = plan
	}
	if len(changes) == 0 {
		return &tenant, nil
	}
	tenantUUID := tenantUUIDValue(&tenant)
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if status, ok := changes["status"]; ok {
			nextStatus := status.(string)
			if nextStatus == iamm.StatusSuspended && tenant.Status != iamm.StatusSuspended {
				hasOther, err := s.hasAnotherActiveTenant(ctx, tx, tenant.ID)
				if err != nil {
					return err
				}
				if !hasOther {
					return fmt.Errorf("at least one active tenant must remain enabled")
				}
				if err := s.disableTenantMembers(ctx, tx, tenantUUID); err != nil {
					return err
				}
			}
		}
		if err := tx.Model(&tenant).Updates(changes).Error; err != nil {
			return err
		}
		return tx.Where("id = ?", tenant.ID).First(&tenant).Error
	})
	if err != nil {
		return nil, err
	}
	if s.audit != nil {
		_ = s.audit.Record(ctx, AuditEntry{
			TenantUUID:    tenantUUID,
			ActorMemberID: input.ActorID,
			Action:        "update",
			Resource:      "iam.tenant",
			Diff:          changes,
		})
	}
	return &tenant, nil
}

func normalizeTenantStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "", iamm.StatusActive:
		return iamm.StatusActive
	case iamm.StatusSuspended:
		return iamm.StatusSuspended
	default:
		return ""
	}
}

func normalizeTenantKey(key string) (string, error) {
	k := strings.TrimSpace(strings.ToLower(key))
	if k == "" {
		return "", errors.New("tenant key required")
	}
	if len(k) < 6 {
		return "", fmt.Errorf("tenant key %s is too short", k)
	}
	return k, nil
}

func (s *TenantService) hasAnotherActiveTenant(ctx context.Context, tx *gorm.DB, currentID uint64) (bool, error) {
	var count int64
	if err := tx.WithContext(ctx).Model(&iamm.Tenant{}).
		Where("status = ?", iamm.StatusActive).
		Where("id <> ?", currentID).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func tenantUUIDValue(t *iamm.Tenant) string {
	if t == nil {
		return ""
	}
	if v := strings.TrimSpace(t.UUID); v != "" {
		return strings.ToLower(v)
	}
	return strings.ToLower(strings.TrimSpace(t.Key))
}

func (s *TenantService) disableTenantMembers(ctx context.Context, tx *gorm.DB, tenantUUID string) error {
	tenantUUID = strings.TrimSpace(strings.ToLower(tenantUUID))
	if tenantUUID == "" {
		return nil
	}
	if err := tx.WithContext(ctx).
		Model(&iamm.Member{}).
		Where("tenant_uuid = ?", tenantUUID).
		Updates(map[string]any{"status": iamm.StatusDisabled}).Error; err != nil {
		return err
	}
	return revokeTenantSessions(ctx, tx, tenantUUID)
}
