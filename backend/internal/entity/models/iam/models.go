package iam

import (
	"strings"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	StatusActive    = "active"
	StatusDisabled  = "disabled"
	StatusSuspended = "suspended"
	StatusLocked    = "locked"

	RoleScopeSystem = "system"
	RoleScopeTenant = "tenant"
)

type Tenant struct {
	ID        uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	UUID      string         `gorm:"type:uuid;not null;uniqueIndex:idx_iam_tenants_uuid" json:"uuid"`
	Key       string         `gorm:"size:64;not null;uniqueIndex:idx_iam_tenants_key" json:"key"`
	Name      string         `gorm:"size:128;not null" json:"name"`
	Status    string         `gorm:"size:32;not null;default:'active'" json:"status"`
	Plan      string         `gorm:"size:64;not null;default:'free'" json:"plan"`
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at" json:"deleted_at,omitempty"`
}

func (Tenant) TableName() string { return models.S(models.TableIAMTenants) }

func (t *Tenant) BeforeCreate(tx *gorm.DB) error {
	if strings.TrimSpace(t.UUID) == "" {
		t.UUID = uuid.NewString()
	}
	return nil
}

type User struct {
	ID           uint64            `gorm:"primaryKey;autoIncrement" json:"id"`
	TenantUuid   string            `gorm:"column:tenant_uuid;type:uuid;not null;index:idx_iam_users_tenant;uniqueIndex:idx_iam_users_tenant_email,priority:1" json:"tenant_uuid"`
	Email        string            `gorm:"size:255;uniqueIndex:idx_iam_users_tenant_email,priority:2" json:"email"`
	Phone        string            `gorm:"size:32;index" json:"phone"`
	DisplayName  string            `gorm:"size:128" json:"display_name"`
	AvatarURL    string            `gorm:"size:255" json:"avatar_url"`
	Status       string            `gorm:"size:32;not null;default:'active'" json:"status"`
	IsRoot       bool              `gorm:"column:is_root;not null;default:false;index:idx_iam_users_is_root" json:"is_root"`
	PasswordHash string            `gorm:"size:255;not null" json:"-"`
	Meta         datatypes.JSONMap `gorm:"type:jsonb" json:"meta"`
	CreatedAt    time.Time         `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time         `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt    gorm.DeletedAt    `gorm:"column:deleted_at" json:"deleted_at,omitempty"`
}

func (User) TableName() string { return models.S(models.TableIAMUsers) }

type Member struct {
	models.BaseModel
	UserID       uint64            `gorm:"column:user_id;not null;index:idx_iam_users_account" json:"user_id"`
	Username     string            `gorm:"size:64;not null;index:idx_iam_users_username" json:"username"`
	DisplayName  string            `gorm:"size:128" json:"display_name"`
	AvatarURL    string            `gorm:"size:255" json:"avatar_url"`
	Status       string            `gorm:"size:32;not null;default:'active';index:idx_iam_users_status" json:"status"`
	DepartmentID *uint64           `gorm:"index:idx_iam_users_department" json:"department_id"`
	Meta         datatypes.JSONMap `gorm:"type:jsonb" json:"meta"`
	LastLoginAt  *time.Time        `gorm:"column:last_login_at;index:idx_iam_users_last_login" json:"last_login_at,omitempty"`
}

func (Member) TableName() string { return models.S(models.TableIAMMembers) }

type Role struct {
	models.BaseModel
	Code          string `gorm:"size:64;not null;index:idx_iam_roles_code" json:"code"`
	Name          string `gorm:"size:128;not null" json:"name"`
	Description   string `gorm:"size:255" json:"description"`
	ScopeType     string `gorm:"size:32;not null;default:'tenant';index:idx_iam_roles_scope" json:"scope_type"`
	PolicyVersion string `gorm:"size:64;not null;default:'v1'" json:"policy_version"`
}

func (Role) TableName() string { return models.S(models.TableIAMRoles) }

type Permission struct {
	ID          uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	Resource    string    `gorm:"size:128;not null;uniqueIndex:idx_iam_permissions_resource_action,priority:1" json:"resource"`
	Action      string    `gorm:"size:64;not null;uniqueIndex:idx_iam_permissions_resource_action,priority:2" json:"action"`
	Description string    `gorm:"size:255" json:"description"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (Permission) TableName() string { return models.S(models.TableIAMPermissions) }

type Department struct {
	models.BaseModel
	Name        string  `gorm:"size:128;not null;index:idx_iam_departments_name" json:"name"`
	Code        string  `gorm:"size:64;not null;index:idx_iam_departments_code" json:"code"`
	ParentID    *uint64 `gorm:"index:idx_iam_departments_parent" json:"parent_id"`
	Description string  `gorm:"size:255" json:"description"`
	Path        string  `gorm:"size:512;not null;index:idx_iam_departments_path" json:"path"`
	SortOrder   int     `gorm:"default:0;index:idx_iam_departments_sort" json:"sort_order"`
}

func (Department) TableName() string { return models.S(models.TableIAMDepartments) }

type MemberRole struct {
	UserID    uint64    `gorm:"column:member_id;primaryKey" json:"user_id"`
	RoleID    uint64    `gorm:"primaryKey" json:"role_id"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (MemberRole) TableName() string { return models.S(models.TableIAMMemberRoles) }

type RolePermission struct {
	RoleID        uint64    `gorm:"primaryKey" json:"role_id"`
	PermissionID  uint64    `gorm:"primaryKey" json:"permission_id"`
	TenantUuid    string    `gorm:"type:uuid;not null;index:idx_iam_role_permissions_tenant" json:"tenant_uuid"`
	PolicyVersion string    `gorm:"size:64;not null;default:'v1'" json:"policy_version"`
	CreatedAt     time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (RolePermission) TableName() string { return models.S(models.TableIAMRolePermissions) }

type RefreshToken struct {
	ID         uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	TokenHash  string    `gorm:"size:128;uniqueIndex" json:"token_hash"`
	UserID     uint64    `gorm:"index" json:"user_id"`
	TenantUuid string    `gorm:"type:uuid;index" json:"tenant_uuid"`
	UserRecord uint64    `gorm:"column:member_id;index" json:"user_record_id"`
	ExpiresAt  time.Time `gorm:"index" json:"expires_at"`
	Revoked    bool      `gorm:"default:false" json:"revoked"`
	CreatedAt  time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (RefreshToken) TableName() string { return models.S(models.TableIAMRefreshTokens) }

type AuditLog struct {
	models.BaseModel
	ActorUserID *uint64           `gorm:"column:actor_member_id;index:idx_iam_audit_actor" json:"actor_user_id"`
	Action      string            `gorm:"size:128;not null;index:idx_iam_audit_action" json:"action"`
	Resource    string            `gorm:"size:128;not null;index:idx_iam_audit_resource" json:"resource"`
	Diff        datatypes.JSONMap `gorm:"type:jsonb" json:"diff"`
}

func (AuditLog) TableName() string { return models.S(models.TableIAMAuditLogs) }
