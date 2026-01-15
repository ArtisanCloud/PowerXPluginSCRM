package iam

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	basemodels "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	iamm "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/iam"
	authobs "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/observability/auth"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrRoleNotFound     = errors.New("iam: role not found")
	ErrTenantRequired   = errors.New("tenant_uuid required")
	ErrPermissionDenied = errors.New("iam: permission denied")
)

type RoleService struct {
	db       *gorm.DB
	audit    *AuditService
	pluginID string
}

func NewRoleService(db *gorm.DB, audit *AuditService, pluginID string) *RoleService {
	return &RoleService{db: db, audit: audit, pluginID: pluginID}
}

type RoleFilter struct {
	TenantUUID string
	Query      string
	ScopeType  string
}

type RoleView struct {
	ID            uint64    `json:"id"`
	TenantUUID    string    `json:"tenant_uuid"`
	Code          string    `json:"code"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	ScopeType     string    `json:"scope_type"`
	PolicyVersion string    `json:"policy_version"`
	PermissionIDs []uint64  `json:"permission_ids,omitempty"`
	MemberIDs     []uint64  `json:"member_ids,omitempty"`
	MemberCount   int64     `json:"member_count"`
	CreatedAt     time.Time `json:"created_at"`
}

type PermissionView struct {
	ID          uint64 `json:"id"`
	Resource    string `json:"resource"`
	Action      string `json:"action"`
	Description string `json:"description"`
}

type CreateRoleInput struct {
	TenantUUID    string
	Code          string
	Name          string
	Description   string
	ScopeType     string
	CloneRoleID   *uint64
	PermissionIDs []uint64
	MemberIDs     []uint64
	ActorID       *uint64
}

type UpdateRoleInput struct {
	Name        string
	Description string
	ScopeType   string
	ActorID     *uint64
}

type ReplaceRolePermissionsInput struct {
	RoleID        uint64
	TenantUUID    string
	PermissionIDs []uint64
	ActorID       *uint64
}

type RoleMembersInput struct {
	RoleID     uint64
	TenantUUID string
	MemberIDs  []uint64
	ActorID    *uint64
}

func (s *RoleService) List(ctx context.Context, filter RoleFilter) ([]RoleView, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("iam: role service unavailable")
	}
	tenantUUID := strings.ToLower(strings.TrimSpace(filter.TenantUUID))
	if tenantUUID == "" {
		return nil, ErrTenantRequired
	}
	query := s.db.WithContext(ctx).Model(&iamm.Role{}).Where("tenant_uuid = ?", tenantUUID)
	if q := strings.TrimSpace(filter.Query); q != "" {
		like := "%" + strings.ToLower(q) + "%"
		query = query.Where("(lower(name) LIKE ? OR lower(code) LIKE ?)", like, like)
	}
	if scope := normalizeScopeType(filter.ScopeType); scope != "" {
		query = query.Where("scope_type = ?", scope)
	}
	var roles []iamm.Role
	if err := query.Order("created_at DESC").Find(&roles).Error; err != nil {
		return nil, err
	}
	if len(roles) == 0 {
		return []RoleView{}, nil
	}
	roleIDs := make([]uint64, len(roles))
	for i, r := range roles {
		roleIDs[i] = r.ID
	}
	memberCounts, err := s.roleMemberCounts(ctx, roleIDs)
	if err != nil {
		return nil, err
	}
	view := make([]RoleView, 0, len(roles))
	for _, role := range roles {
		view = append(view, RoleView{
			ID:            role.ID,
			TenantUUID:    role.TenantUuid,
			Code:          role.Code,
			Name:          role.Name,
			Description:   role.Description,
			ScopeType:     role.ScopeType,
			PolicyVersion: role.PolicyVersion,
			MemberCount:   memberCounts[role.ID],
			CreatedAt:     role.CreatedAt,
		})
	}
	return view, nil
}

func (s *RoleService) Get(ctx context.Context, id uint64) (*RoleView, error) {
	role, err := s.loadRole(ctx, id)
	if err != nil {
		return nil, err
	}
	perms, err := s.rolePermissionIDs(ctx, []uint64{id})
	if err != nil {
		return nil, err
	}
	memberIDs, err := s.roleMemberIDs(ctx, role.ID)
	if err != nil {
		return nil, err
	}
	return &RoleView{
		ID:            role.ID,
		TenantUUID:    role.TenantUuid,
		Code:          role.Code,
		Name:          role.Name,
		Description:   role.Description,
		ScopeType:     role.ScopeType,
		PolicyVersion: role.PolicyVersion,
		PermissionIDs: perms[role.ID],
		MemberIDs:     memberIDs,
		MemberCount:   int64(len(memberIDs)),
		CreatedAt:     role.CreatedAt,
	}, nil
}

func (s *RoleService) Create(ctx context.Context, input CreateRoleInput) (*RoleView, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("iam: role service unavailable")
	}
	tenantUUID := normalizeTenant(input.TenantUUID)
	if tenantUUID == "" {
		return nil, ErrTenantRequired
	}
	code := strings.TrimSpace(input.Code)
	if code == "" {
		return nil, errors.New("code required")
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, errors.New("name required")
	}
	scope := normalizeScopeType(input.ScopeType)
	if scope == "" {
		scope = iamm.RoleScopeTenant
	}
	permIDs := uniqueUint64(input.PermissionIDs)
	var memberIDs []uint64
	if len(input.MemberIDs) > 0 {
		memberIDs = uniqueUint64(input.MemberIDs)
	}
	var createdRole *iamm.Role
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&iamm.Role{}).
			Where("tenant_uuid = ? AND lower(code) = ?", tenantUUID, strings.ToLower(code)).
			Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return fmt.Errorf("role code already exists")
		}
		role := &iamm.Role{
			BaseModel:     basemodels.BaseModel{TenantUuid: tenantUUID},
			Code:          code,
			Name:          name,
			Description:   strings.TrimSpace(input.Description),
			ScopeType:     scope,
			PolicyVersion: nextPolicyVersion(),
		}
		if err := tx.Create(role).Error; err != nil {
			return err
		}
		if input.CloneRoleID != nil && *input.CloneRoleID > 0 && len(permIDs) == 0 {
			clonedPerms, err := s.rolePermissionIDs(ctx, []uint64{*input.CloneRoleID})
			if err != nil {
				return err
			}
			permIDs = append(permIDs, clonedPerms[*input.CloneRoleID]...)
		}
		if err := replaceRolePermissionsTx(ctx, tx, role, permIDs, role.PolicyVersion); err != nil {
			return err
		}
		if len(memberIDs) > 0 {
			if err := addMembersToRoleTx(ctx, tx, role, memberIDs, false); err != nil {
				return err
			}
		}
		createdRole = role
		return nil
	})
	if err != nil {
		return nil, err
	}
	authobs.RecordRoleChange(s.pluginID, createdRole.TenantUuid, "create", createdRole.ScopeType)
	if len(memberIDs) > 0 {
		_ = revokeMembersSessions(ctx, s.db, memberIDs)
	}
	if s.audit != nil {
		_ = s.audit.Record(ctx, AuditEntry{
			TenantUUID:    createdRole.TenantUuid,
			ActorMemberID: input.ActorID,
			Action:        "create",
			Resource:      "iam.role",
			Diff: map[string]any{
				"role_id":    createdRole.ID,
				"code":       createdRole.Code,
				"name":       createdRole.Name,
				"scope":      createdRole.ScopeType,
				"perm_ids":   permIDs,
				"member_ids": memberIDs,
			},
		})
	}
	return s.Get(ctx, createdRole.ID)
}

func (s *RoleService) Update(ctx context.Context, id uint64, input UpdateRoleInput) (*RoleView, error) {
	role, err := s.loadRole(ctx, id)
	if err != nil {
		return nil, err
	}
	updates := map[string]any{}
	if name := strings.TrimSpace(input.Name); name != "" && name != role.Name {
		updates["name"] = name
	}
	if desc := strings.TrimSpace(input.Description); desc != "" && desc != role.Description {
		updates["description"] = desc
	}
	if sch := normalizeScopeType(input.ScopeType); sch != "" && sch != role.ScopeType {
		updates["scope_type"] = sch
	}
	if len(updates) == 0 {
		return s.Get(ctx, id)
	}
	updates["policy_version"] = nextPolicyVersion()
	if err := s.db.WithContext(ctx).Model(role).Updates(updates).Error; err != nil {
		return nil, err
	}
	authobs.RecordRoleChange(s.pluginID, role.TenantUuid, "update", role.ScopeType)
	if s.audit != nil {
		_ = s.audit.Record(ctx, AuditEntry{
			TenantUUID:    role.TenantUuid,
			ActorMemberID: input.ActorID,
			Action:        "update",
			Resource:      "iam.role",
			Diff:          updates,
		})
	}
	return s.Get(ctx, id)
}

func (s *RoleService) Delete(ctx context.Context, id uint64, actorID *uint64) error {
	role, err := s.loadRole(ctx, id)
	if err != nil {
		return err
	}
	if strings.EqualFold(role.Code, "system.admin") {
		return errors.New("system admin role cannot be deleted")
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&iamm.Role{}).
			Where("tenant_uuid = ? AND deleted_at IS NULL", role.TenantUuid).
			Count(&count).Error; err != nil {
			return err
		}
		if count <= 1 {
			return fmt.Errorf("at least one role must remain for tenant %s", role.TenantUuid)
		}
		if err := tx.Delete(&iamm.Role{}, id).Error; err != nil {
			return err
		}
		if err := tx.Where("role_id = ?", id).Delete(&iamm.RolePermission{}).Error; err != nil {
			return err
		}
		if err := tx.Where("role_id = ?", id).Delete(&iamm.MemberRole{}).Error; err != nil {
			return err
		}
		authobs.RecordRoleChange(s.pluginID, role.TenantUuid, "delete", role.ScopeType)
		if s.audit != nil {
			_ = s.audit.Record(ctx, AuditEntry{
				TenantUUID:    role.TenantUuid,
				ActorMemberID: actorID,
				Action:        "delete",
				Resource:      "iam.role",
				Diff: map[string]any{
					"role_id": id,
					"code":    role.Code,
				},
			})
		}
		return nil
	})
}

func (s *RoleService) ReplacePermissions(ctx context.Context, input ReplaceRolePermissionsInput) (*RoleView, error) {
	role, err := s.loadRole(ctx, input.RoleID)
	if err != nil {
		return nil, err
	}
	if role.TenantUuid != normalizeTenant(input.TenantUUID) {
		return nil, ErrPermissionDenied
	}
	newVersion := nextPolicyVersion()
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := replaceRolePermissionsTx(ctx, tx, role, uniqueUint64(input.PermissionIDs), newVersion); err != nil {
			return err
		}
		return tx.Model(role).Update("policy_version", newVersion).Error
	})
	if err != nil {
		return nil, err
	}
	authobs.RecordRoleChange(s.pluginID, role.TenantUuid, "permissions", role.ScopeType)
	memberIDs, _ := s.roleMemberIDs(ctx, role.ID)
	if len(memberIDs) > 0 {
		_ = revokeMembersSessions(ctx, s.db, memberIDs)
	}
	if s.audit != nil {
		_ = s.audit.Record(ctx, AuditEntry{
			TenantUUID:    role.TenantUuid,
			ActorMemberID: input.ActorID,
			Action:        "update",
			Resource:      "iam.role.permissions",
			Diff: map[string]any{
				"role_id":  role.ID,
				"perm_ids": uniqueUint64(input.PermissionIDs),
			},
		})
	}
	return s.Get(ctx, role.ID)
}

func (s *RoleService) AddMembers(ctx context.Context, input RoleMembersInput) error {
	role, err := s.loadRole(ctx, input.RoleID)
	if err != nil {
		return err
	}
	if role.TenantUuid != normalizeTenant(input.TenantUUID) {
		return ErrPermissionDenied
	}
	memberIDs := uniqueUint64(input.MemberIDs)
	if len(memberIDs) == 0 {
		return nil
	}
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return addMembersToRoleTx(ctx, tx, role, memberIDs, false)
	}); err != nil {
		return err
	}
	authobs.RecordRoleChange(s.pluginID, role.TenantUuid, "member_assign", role.ScopeType)
	_ = revokeMembersSessions(ctx, s.db, memberIDs)
	if s.audit != nil {
		_ = s.audit.Record(ctx, AuditEntry{
			TenantUUID:    role.TenantUuid,
			ActorMemberID: input.ActorID,
			Action:        "update",
			Resource:      "iam.role.members",
			Diff: map[string]any{
				"role_id":    role.ID,
				"member_ids": memberIDs,
				"operation":  "add",
			},
		})
	}
	return nil
}

func (s *RoleService) RemoveMembers(ctx context.Context, input RoleMembersInput) error {
	role, err := s.loadRole(ctx, input.RoleID)
	if err != nil {
		return err
	}
	if role.TenantUuid != normalizeTenant(input.TenantUUID) {
		return ErrPermissionDenied
	}
	memberIDs := uniqueUint64(input.MemberIDs)
	if len(memberIDs) == 0 {
		return nil
	}
	if err := s.db.WithContext(ctx).WithContext(ctx).Where("role_id = ? AND member_id IN ?", role.ID, memberIDs).Delete(&iamm.MemberRole{}).Error; err != nil {
		return err
	}
	authobs.RecordRoleChange(s.pluginID, role.TenantUuid, "member_remove", role.ScopeType)
	_ = revokeMembersSessions(ctx, s.db, memberIDs)
	if s.audit != nil {
		_ = s.audit.Record(ctx, AuditEntry{
			TenantUUID:    role.TenantUuid,
			ActorMemberID: input.ActorID,
			Action:        "update",
			Resource:      "iam.role.members",
			Diff: map[string]any{
				"role_id":    role.ID,
				"member_ids": memberIDs,
				"operation":  "remove",
			},
		})
	}
	return nil
}

func (s *RoleService) ListPermissions(ctx context.Context) ([]PermissionView, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("iam: role service unavailable")
	}
	var perms []iamm.Permission
	if err := s.db.WithContext(ctx).Order("resource ASC, action ASC").Find(&perms).Error; err != nil {
		return nil, err
	}
	result := make([]PermissionView, 0, len(perms))
	for _, p := range perms {
		result = append(result, PermissionView{
			ID:          p.ID,
			Resource:    p.Resource,
			Action:      p.Action,
			Description: p.Description,
		})
	}
	return result, nil
}

func (s *RoleService) loadRole(ctx context.Context, id uint64) (*iamm.Role, error) {
	if id == 0 {
		return nil, ErrRoleNotFound
	}
	var role iamm.Role
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&role).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRoleNotFound
		}
		return nil, err
	}
	return &role, nil
}

func (s *RoleService) rolePermissionIDs(ctx context.Context, roleIDs []uint64) (map[uint64][]uint64, error) {
	result := make(map[uint64][]uint64, len(roleIDs))
	if len(roleIDs) == 0 {
		return result, nil
	}
	var rows []struct {
		RoleID       uint64
		PermissionID uint64
	}
	if err := s.db.WithContext(ctx).
		Model(&iamm.RolePermission{}).
		Where("role_id IN ?", roleIDs).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		result[row.RoleID] = append(result[row.RoleID], row.PermissionID)
	}
	return result, nil
}

func (s *RoleService) roleMemberCounts(ctx context.Context, roleIDs []uint64) (map[uint64]int64, error) {
	counts := make(map[uint64]int64, len(roleIDs))
	if len(roleIDs) == 0 {
		return counts, nil
	}
	var rows []struct {
		RoleID uint64
		Count  int64
	}
	if err := s.db.WithContext(ctx).
		Table(iamm.MemberRole{}.TableName()).
		Select("role_id, COUNT(*) AS count").
		Where("role_id IN ?", roleIDs).
		Group("role_id").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		counts[row.RoleID] = row.Count
	}
	return counts, nil
}

func (s *RoleService) roleMemberIDs(ctx context.Context, roleID uint64) ([]uint64, error) {
	var rows []struct {
		MemberID uint64
	}
	if err := s.db.WithContext(ctx).
		Model(&iamm.MemberRole{}).
		Where("role_id = ?", roleID).
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	memberIDs := make([]uint64, 0, len(rows))
	for _, row := range rows {
		memberIDs = append(memberIDs, row.MemberID)
	}
	return memberIDs, nil
}

func addMembersToRoleTx(ctx context.Context, tx *gorm.DB, role *iamm.Role, memberIDs []uint64, replace bool) error {
	if tx == nil || role == nil || len(memberIDs) == 0 {
		return nil
	}
	memberIDs = uniqueUint64(memberIDs)
	var validMembers []iamm.Member
	if err := tx.WithContext(ctx).Where("tenant_uuid = ? AND id IN ?", role.TenantUuid, memberIDs).Find(&validMembers).Error; err != nil {
		return err
	}
	if len(validMembers) != len(memberIDs) {
		return fmt.Errorf("one or more members do not belong to tenant %s", role.TenantUuid)
	}
	if replace {
		if err := tx.WithContext(ctx).Where("role_id = ?", role.ID).Delete(&iamm.MemberRole{}).Error; err != nil {
			return err
		}
	}
	for _, member := range memberIDs {
		rel := &iamm.MemberRole{UserID: member, RoleID: role.ID}
		if err := tx.Clauses(clauseOnConflictDoNothing()).Create(rel).Error; err != nil {
			return err
		}
	}
	return nil
}

func replaceRolePermissionsTx(ctx context.Context, tx *gorm.DB, role *iamm.Role, permissionIDs []uint64, policyVersion string) error {
	if tx == nil || role == nil {
		return errors.New("iam: db unavailable")
	}
	if err := tx.WithContext(ctx).Where("role_id = ?", role.ID).Delete(&iamm.RolePermission{}).Error; err != nil {
		return err
	}
	if len(permissionIDs) == 0 {
		return nil
	}
	for _, pid := range uniqueUint64(permissionIDs) {
		rp := &iamm.RolePermission{
			RoleID:        role.ID,
			PermissionID:  pid,
			TenantUuid:    strings.TrimSpace(role.TenantUuid),
			PolicyVersion: policyVersion,
		}
		if err := tx.Create(rp).Error; err != nil {
			return err
		}
	}
	return nil
}

func clauseOnConflictDoNothing() clause.OnConflict {
	return clause.OnConflict{
		Columns: []clause.Column{
			{Name: "member_id"},
			{Name: "role_id"},
		},
		DoNothing: true,
	}
}

func normalizeScopeType(scope string) string {
	switch strings.ToLower(strings.TrimSpace(scope)) {
	case iamm.RoleScopeSystem:
		return iamm.RoleScopeSystem
	case iamm.RoleScopeTenant, "":
		return iamm.RoleScopeTenant
	default:
		return ""
	}
}

func normalizeTenant(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func nextPolicyVersion() string {
	return fmt.Sprintf("pv-%d", time.Now().UnixNano())
}
