package iam

import (
	"context"
	"errors"
	"fmt"
	"strings"

	iamm "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/iam"
	"gorm.io/gorm"
)

type DepartmentService struct {
	db    *gorm.DB
	audit *AuditService
}

func NewDepartmentService(db *gorm.DB, audit *AuditService) *DepartmentService {
	return &DepartmentService{db: db, audit: audit}
}

type DepartmentFilter struct {
	TenantUUID string
}

func (s *DepartmentService) List(ctx context.Context, filter DepartmentFilter) ([]iamm.Department, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("iam: department service unavailable")
	}
	tenantUUID := strings.ToLower(strings.TrimSpace(filter.TenantUUID))
	if tenantUUID == "" {
		return nil, errors.New("tenant_uuid required")
	}
	var deps []iamm.Department
	if err := s.db.WithContext(ctx).
		Where("tenant_uuid = ?", tenantUUID).
		Order("sort_order ASC, path ASC").
		Find(&deps).Error; err != nil {
		return nil, err
	}
	return deps, nil
}

type CreateDepartmentInput struct {
	TenantUUID  string
	Name        string
	Code        string
	ParentID    *uint64
	Description string
	SortOrder   int
	ActorID     *uint64
}

func (s *DepartmentService) Create(ctx context.Context, input CreateDepartmentInput) (*iamm.Department, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("iam: department service unavailable")
	}
	tenantUUID := strings.ToLower(strings.TrimSpace(input.TenantUUID))
	if tenantUUID == "" {
		return nil, errors.New("tenant_uuid required")
	}
	code := normalizeDeptCode(input.Code, input.Name)
	if code == "" {
		return nil, errors.New("code required")
	}
	var parent *iamm.Department
	if input.ParentID != nil {
		parent = &iamm.Department{}
		if err := s.db.WithContext(ctx).Where("id = ? AND tenant_uuid = ?", *input.ParentID, tenantUUID).First(parent).Error; err != nil {
			return nil, err
		}
	}
	dept := &iamm.Department{
		BaseModel:   iamm.Department{}.BaseModel,
		Name:        strings.TrimSpace(input.Name),
		Code:        code,
		Description: strings.TrimSpace(input.Description),
		SortOrder:   input.SortOrder,
	}
	dept.TenantUuid = tenantUUID
	if parent != nil {
		dept.ParentID = &parent.ID
		dept.Path = buildDeptPath(parent.Path, code)
	} else {
		dept.Path = buildDeptPath("", code)
	}
	if dept.Name == "" {
		dept.Name = code
	}

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var exists int64
		if err := tx.Model(&iamm.Department{}).
			Where("tenant_uuid = ? AND lower(code) = ?", tenantUUID, code).
			Count(&exists).Error; err != nil {
			return err
		}
		if exists > 0 {
			return fmt.Errorf("department code already exists")
		}
		return tx.Create(dept).Error
	})
	if err != nil {
		return nil, err
	}
	if s.audit != nil {
		_ = s.audit.Record(ctx, AuditEntry{
			TenantUUID:    tenantUUID,
			ActorMemberID: input.ActorID,
			Action:        "create",
			Resource:      "iam.department",
			Diff: map[string]any{
				"name":      dept.Name,
				"code":      dept.Code,
				"path":      dept.Path,
				"parent_id": dept.ParentID,
			},
		})
	}
	return dept, nil
}

type UpdateDepartmentInput struct {
	Name        string
	Description string
	SortOrder   *int
	ParentID    *uint64
	ActorID     *uint64
}

func (s *DepartmentService) Update(ctx context.Context, id uint64, input UpdateDepartmentInput) (*iamm.Department, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("iam: department service unavailable")
	}
	var dept iamm.Department
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&dept).Error; err != nil {
		return nil, err
	}
	changes := map[string]any{}
	if name := strings.TrimSpace(input.Name); name != "" && name != dept.Name {
		changes["name"] = name
	}
	if desc := strings.TrimSpace(input.Description); desc != "" && desc != dept.Description {
		changes["description"] = desc
	}
	if input.SortOrder != nil && dept.SortOrder != *input.SortOrder {
		changes["sort_order"] = *input.SortOrder
	}
	var newParent *iamm.Department
	var parentPathOverride *string
	if input.ParentID != nil {
		targetID := *input.ParentID
		if targetID == dept.ID {
			return nil, fmt.Errorf("department cannot be its own parent")
		}
		if dept.ParentID == nil || *dept.ParentID != targetID {
			newParent = &iamm.Department{}
			if targetID != 0 {
				if err := s.db.WithContext(ctx).Where("id = ? AND tenant_uuid = ?", targetID, dept.TenantUuid).First(newParent).Error; err != nil {
					return nil, err
				}
				if isDescendantPath(newParent.Path, dept.Path) {
					return nil, fmt.Errorf("department cannot be moved under its own child")
				}
				changes["parent_id"] = targetID
				pp := newParent.Path
				parentPathOverride = &pp
			} else {
				changes["parent_id"] = nil
				newParent = nil
				pp := ""
				parentPathOverride = &pp
			}
		}
	}
	if len(changes) == 0 && parentPathOverride == nil {
		return &dept, nil
	}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if parentPathOverride != nil {
			newPath := buildDeptPath(*parentPathOverride, dept.Code)
			if err := updateDepartmentPath(ctx, tx, dept, newPath); err != nil {
				return err
			}
			changes["path"] = newPath
		}
		if len(changes) > 0 {
			if err := tx.Model(&dept).Updates(changes).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if s.audit != nil {
		_ = s.audit.Record(ctx, AuditEntry{
			TenantUUID:    dept.TenantUuid,
			ActorMemberID: input.ActorID,
			Action:        "update",
			Resource:      "iam.department",
			Diff:          changes,
		})
	}
	return &dept, nil
}

func (s *DepartmentService) Delete(ctx context.Context, id uint64, actorID *uint64) error {
	if s == nil || s.db == nil {
		return errors.New("iam: department service unavailable")
	}
	var dept iamm.Department
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&dept).Error; err != nil {
		return err
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var subCount int64
		if err := tx.Model(&iamm.Department{}).Where("parent_id = ?", id).Count(&subCount).Error; err != nil {
			return err
		}
		if subCount > 0 {
			return fmt.Errorf("department has child nodes")
		}
		var memberCount int64
		if err := tx.Model(&iamm.Member{}).Where("department_id = ?", id).Count(&memberCount).Error; err != nil {
			return err
		}
		if memberCount > 0 {
			return fmt.Errorf("department has members")
		}
		if err := tx.Delete(&dept).Error; err != nil {
			return err
		}
		if s.audit != nil {
			_ = s.audit.Record(ctx, AuditEntry{
				TenantUUID:    dept.TenantUuid,
				ActorMemberID: actorID,
				Action:        "delete",
				Resource:      "iam.department",
				Diff: map[string]any{
					"id":   id,
					"name": dept.Name,
				},
			})
		}
		return nil
	})
}

func updateDepartmentPath(ctx context.Context, tx *gorm.DB, dept iamm.Department, newPath string) error {
	oldPath := dept.Path
	if oldPath == newPath {
		return nil
	}
	if err := tx.Model(&dept).Update("path", newPath).Error; err != nil {
		return err
	}
	var children []iamm.Department
	if err := tx.Where("tenant_uuid = ? AND path LIKE ?", dept.TenantUuid, oldPath+".%").Find(&children).Error; err != nil {
		return err
	}
	for _, child := range children {
		newChildPath := strings.Replace(child.Path, oldPath, newPath, 1)
		if err := tx.Model(&iamm.Department{}).Where("id = ?", child.ID).Update("path", newChildPath).Error; err != nil {
			return err
		}
	}
	return nil
}

func buildDeptPath(parentPath, code string) string {
	code = normalizeDeptCode(code, code)
	if parentPath == "" {
		return code
	}
	return parentPath + "." + code
}

func isDescendantPath(candidate, ancestor string) bool {
	candidate = strings.TrimSpace(candidate)
	ancestor = strings.TrimSpace(ancestor)
	if candidate == "" || ancestor == "" {
		return false
	}
	if candidate == ancestor {
		return true
	}
	return strings.HasPrefix(candidate, ancestor+".")
}

func normalizeDeptCode(code, fallback string) string {
	if trimmed := strings.TrimSpace(strings.ToLower(code)); trimmed != "" {
		return trimmed
	}
	return slugify(fallback)
}
