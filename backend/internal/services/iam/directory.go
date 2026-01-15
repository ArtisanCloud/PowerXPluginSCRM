package iam

import (
	"context"
	"errors"
	"time"
)

type IAMMode string

const (
	IAMModeDelegated IAMMode = "delegated"
	IAMModeLocal     IAMMode = "local"
)

func (m IAMMode) String() string { return string(m) }

var (
	ErrUnsupportedMode  = errors.New("iam: unsupported mode")
	ErrAuthUnavailable  = errors.New("iam: auth service unavailable")
	ErrUnauthorized     = errors.New("iam: unauthorized")
	ErrInvalidArguments = errors.New("iam: invalid arguments")
)

type LoginRequest struct {
	Tenant     string
	Identifier string
	Password   string
	Remember   bool
}

type AuthTokens struct {
	TokenType     string
	AccessToken   string
	RefreshToken  string
	ExpiresIn     int64
	Scope         string
	ExpiresAt     time.Time
	PluginID      string
	PolicyVersion string
}

type UserContext struct {
	TenantUUID    string
	TenantUuid    string
	TenantKey     string
	TenantName    string
	IsRoot        bool
	MemberID      uint64
	UserID        uint64
	Username      string
	Email         string
	DisplayName   string
	AvatarURL     string
	Roles         []string
	Permissions   []string
	DepartmentIDs []uint64
	PolicyVersion string
	PluginID      string
	IssuedAt      time.Time
}

type RoleInfo struct {
	ID          uint64
	TenantUUID  string
	TenantUuid  string
	Code        string
	Name        string
	Description string
}

type DepartmentInfo struct {
	ID          uint64
	TenantUUID  string
	TenantUuid  string
	Name        string
	Code        string
	ParentID    *uint64
	Description string
}

type TenantContext struct {
	TenantUUID  string
	TenantUuid  string
	UserID      uint64
	Roles       []string
	Permissions []string
	PolicyVer   string
}

type IAMDirectory interface {
	Mode() IAMMode
	Login(ctx context.Context, req LoginRequest) (*AuthTokens, *UserContext, error)
	Refresh(ctx context.Context, refreshToken string) (*AuthTokens, error)
	Logout(ctx context.Context, refreshToken string) error
	CurrentUser(ctx context.Context) (*UserContext, error)
	ListRoles(ctx context.Context, tenantUUID string) ([]RoleInfo, error)
	ListDepartments(ctx context.Context, tenantUUID string) ([]DepartmentInfo, error)
	CheckPermission(ctx context.Context, tc TenantContext, resource, action string) error
}
