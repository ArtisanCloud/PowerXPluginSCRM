package iam

type TenantListQuery struct {
	Status   string `form:"status"`
	Query    string `form:"q"`
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
}

type CreateTenantRequest struct {
	Key    string `json:"key" binding:"required"`
	Name   string `json:"name" binding:"required"`
	Status string `json:"status"`
	Plan   string `json:"plan"`
}

type UpdateTenantRequest struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Plan   string `json:"plan"`
}

type DepartmentListQuery struct {
	TenantUUID string `form:"tenant_uuid" binding:"required"`
}

type CreateDepartmentRequest struct {
	TenantUUID  string  `json:"tenant_uuid" binding:"required"`
	Name        string  `json:"name" binding:"required"`
	Code        string  `json:"code"`
	ParentID    *uint64 `json:"parent_id"`
	Description string  `json:"description"`
	SortOrder   int     `json:"sort_order"`
}

type UpdateDepartmentRequest struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	SortOrder   *int    `json:"sort_order"`
	ParentID    *uint64 `json:"parent_id"`
}

type MemberListQuery struct {
	TenantUUID string `form:"tenant_uuid" binding:"required"`
	Status     string `form:"status"`
	Query      string `form:"q"`
	Page       int    `form:"page"`
	PageSize   int    `form:"page_size"`
}

type CreateMemberRequest struct {
	TenantUUID   string   `json:"tenant_uuid" binding:"required"`
	Email        string   `json:"email" binding:"required"`
	DisplayName  string   `json:"display_name"`
	Username     string   `json:"username"`
	Phone        string   `json:"phone"`
	DepartmentID *uint64  `json:"department_id"`
	Status       string   `json:"status"`
	Roles        []uint64 `json:"roles"`
}

type UpdateMemberRequest struct {
	DisplayName  string   `json:"display_name"`
	Status       string   `json:"status"`
	DepartmentID *uint64  `json:"department_id"`
	Roles        []uint64 `json:"roles"`
	ReplaceRoles bool     `json:"replace_roles"`
}

type BulkImportMemberEntry struct {
	Email        string   `json:"email" binding:"required,email"`
	DisplayName  string   `json:"display_name"`
	Username     string   `json:"username"`
	Phone        string   `json:"phone"`
	DepartmentID *uint64  `json:"department_id"`
	Status       string   `json:"status"`
	Roles        []uint64 `json:"roles"`
}

type BulkImportMembersRequest struct {
	TenantUUID string                  `json:"tenant_uuid" binding:"required"`
	Users      []BulkImportMemberEntry `json:"users" binding:"required,min=1"`
}

type RoleListQuery struct {
	TenantUUID string `form:"tenant_uuid" binding:"required"`
	Query      string `form:"q"`
	ScopeType  string `form:"scope_type"`
}

type CreateRoleRequest struct {
	TenantUUID    string   `json:"tenant_uuid" binding:"required"`
	Code          string   `json:"code" binding:"required"`
	Name          string   `json:"name" binding:"required"`
	Description   string   `json:"description"`
	ScopeType     string   `json:"scope_type"`
	CloneRoleID   *uint64  `json:"clone_role_id"`
	PermissionIDs []uint64 `json:"permission_ids"`
	MemberIDs     []uint64 `json:"member_ids"`
}

type UpdateRoleRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	ScopeType   string `json:"scope_type"`
}

type ReplaceRolePermissionsRequest struct {
	TenantUUID    string   `json:"tenant_uuid" binding:"required"`
	PermissionIDs []uint64 `json:"permission_ids"`
}

type RoleMembersRequest struct {
	TenantUUID string   `json:"tenant_uuid" binding:"required"`
	MemberIDs  []uint64 `json:"member_ids"`
}
