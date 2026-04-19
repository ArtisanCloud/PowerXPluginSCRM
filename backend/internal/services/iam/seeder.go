package iam

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/config"
	basemodels "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	iamm "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/iam"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type SeedOptions struct {
	TenantKey  string
	TenantName string
	AdminEmail string
	AdminPwd   string
	AdminName  string
}

type defaultRoleSeed struct {
	Code        string
	Name        string
	Description string
	ScopeType   string
}

func SeedLocalAdmin(ctx context.Context, db *gorm.DB, cfg *config.Config, mode IAMMode) error {
	if mode != IAMModeLocal {
		log.Printf("[iam] skip local admin seed (mode=%s)", mode)
		return nil
	}
	if reason, ok := delegatedModeOverride(); ok {
		log.Printf("[iam] %s indicates delegated mode, skip local admin seed", reason)
		return nil
	}
	if db == nil {
		return errors.New("iam: db is nil")
	}

	opts, warnings := loadSeedOptionsFromEnv()
	for _, msg := range warnings {
		log.Printf("[iam] %s", msg)
	}
	if err := opts.Validate(); err != nil {
		return err
	}
	log.Printf("[iam] seeding local admin tenant=%s admin=%s", opts.TenantKey, opts.AdminEmail)

	hashed, err := bcrypt.GenerateFromPassword([]byte(opts.AdminPwd), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash admin password: %w", err)
	}

	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var tenant iamm.Tenant
		if err := tx.Where("key = ?", strings.ToLower(opts.TenantKey)).First(&tenant).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				u := strings.ToLower(strings.TrimSpace(opts.TenantKey))
				if _, err := uuid.Parse(u); err != nil {
					u = strings.ToLower(uuid.NewString())
				}
				tenant = iamm.Tenant{
					UUID:   u,
					Key:    strings.ToLower(opts.TenantKey),
					Name:   opts.TenantName,
					Status: iamm.StatusActive,
					Plan:   "free",
				}
				if err := tx.Create(&tenant).Error; err != nil {
					return err
				}
			} else {
				return err
			}
		} else {
			updates := map[string]any{"name": opts.TenantName}
			if strings.TrimSpace(tenant.UUID) == "" {
				u := strings.ToLower(strings.TrimSpace(opts.TenantKey))
				if _, err := uuid.Parse(u); err != nil {
					u = strings.ToLower(uuid.NewString())
				}
				updates["uuid"] = u
			}
			if err := tx.Model(&tenant).Updates(updates).Error; err != nil {
				return err
			}
		}

		var account iamm.User
		if err := tx.Where("email = ?", opts.AdminEmail).First(&account).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				account = iamm.User{
					Email:        strings.ToLower(opts.AdminEmail),
					DisplayName:  opts.AdminName,
					IsRoot:       true,
					Status:       iamm.StatusActive,
					PasswordHash: string(hashed),
				}
				if err := tx.Create(&account).Error; err != nil {
					return err
				}
			} else {
				return err
			}
		} else {
			update := map[string]any{
				"display_name":  opts.AdminName,
				"status":        iamm.StatusActive,
				"password_hash": string(hashed),
				"is_root":       true,
			}
			if err := tx.Model(&account).Updates(update).Error; err != nil {
				return err
			}
		}

		username := strings.Split(opts.AdminEmail, "@")[0]
		username = strings.ToLower(username)
		if username == "" {
			username = fmt.Sprintf("admin-%d", tenant.ID)
		}

		tenantUUID := strings.TrimSpace(strings.ToLower(tenant.UUID))
		if tenantUUID == "" {
			tenantUUID = strings.TrimSpace(strings.ToLower(tenant.Key))
		}
		if tenantUUID == "" {
			tenantUUID = fmt.Sprintf("%d", tenant.ID)
		}

		var member iamm.Member
		memberWhere := "tenant_uuid = ? AND user_id = ?"
		if err := tx.Where(memberWhere, tenantUUID, account.ID).First(&member).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				member = iamm.Member{
					BaseModel:   basemodels.BaseModel{TenantUuid: tenantUUID},
					UserID:      account.ID,
					Username:    username,
					DisplayName: opts.AdminName,
					Status:      iamm.StatusActive,
				}
				if err := tx.Create(&member).Error; err != nil {
					return err
				}
			} else {
				return err
			}
		} else {
			updates := map[string]any{
				"status":       iamm.StatusActive,
				"display_name": opts.AdminName,
			}
			if err := tx.Model(&member).Updates(updates).Error; err != nil {
				return err
			}
		}

		rolesByCode, err := ensureDefaultRoles(tx, tenantUUID)
		if err != nil {
			return err
		}
		adminRole, ok := rolesByCode["system.admin"]
		if !ok {
			return fmt.Errorf("system.admin role missing after seed")
		}

		if err := ensureMemberRoleBinding(tx, member.ID, adminRole.ID); err != nil {
			return err
		}
		if tenantAdminRole, exists := rolesByCode["role_admin"]; exists {
			if err := ensureMemberRoleBinding(tx, member.ID, tenantAdminRole.ID); err != nil {
				return err
			}
		}

		deptID, err := ensureDefaultDepartment(tx, tenantUUID)
		if err != nil {
			return err
		}
		if deptID != nil && (member.DepartmentID == nil || *deptID != *member.DepartmentID) {
			if err := tx.Model(&member).Update("department_id", deptID).Error; err != nil {
				return err
			}
		}
		if err := seedDefaultPermissions(tx, adminRole.ID, tenantUUID); err != nil {
			return err
		}
		if tenantAdminRole, exists := rolesByCode["role_admin"]; exists {
			if err := seedDefaultPermissions(tx, tenantAdminRole.ID, tenantUUID); err != nil {
				return err
			}
		} else if tenantAdminRole, exists := rolesByCode["tenant.admin"]; exists {
			if err := seedDefaultPermissions(tx, tenantAdminRole.ID, tenantUUID); err != nil {
				return err
			}
		}
		return nil
	})
}

func delegatedModeOverride() (string, bool) {
	if mode := strings.ToLower(strings.TrimSpace(resolveConfigValue(os.Getenv("IAM_MODE"), os.Getenv("IAMMode")))); mode != "" {
		switch mode {
		case "delegated":
			return "IAMMode", true
		case "local":
			return "", false
		}
	}
	if strings.TrimSpace(os.Getenv("POWERX_PROXY")) == "1" {
		return "POWERX_PROXY", true
	}
	return "", false
}

func loadSeedOptionsFromEnv() (SeedOptions, []string) {
	const (
		defaultTenantKey  = "00000000-0000-0000-0000-000000000001"
		defaultTenantName = "Local Tenant"
		defaultAdminEmail = "admin@local.test"
		defaultAdminPwd   = "S3cret!!"
		defaultAdminName  = "Local Admin"
	)

	opts := SeedOptions{
		TenantKey:  strings.ToLower(strings.TrimSpace(os.Getenv("PLUGIN_IAM_TENANT_KEY"))),
		TenantName: strings.TrimSpace(os.Getenv("PLUGIN_IAM_TENANT_NAME")),
		AdminEmail: strings.TrimSpace(os.Getenv("PLUGIN_IAM_ADMIN_EMAIL")),
		AdminPwd:   strings.TrimSpace(os.Getenv("PLUGIN_IAM_ADMIN_PASSWORD")),
		AdminName:  strings.TrimSpace(os.Getenv("PLUGIN_IAM_ADMIN_NAME")),
	}
	warnings := make([]string, 0, 4)
	if opts.TenantKey == "" {
		opts.TenantKey = defaultTenantKey
		warnings = append(warnings, "PLUGIN_IAM_TENANT_KEY not set, using default tenant UUID")
	}
	if opts.TenantName == "" {
		opts.TenantName = defaultTenantName
		warnings = append(warnings, "PLUGIN_IAM_TENANT_NAME not set, using default tenant name")
	}
	if opts.AdminEmail == "" {
		opts.AdminEmail = defaultAdminEmail
		warnings = append(warnings, "PLUGIN_IAM_ADMIN_EMAIL not set, using default admin email")
	} else {
		opts.AdminEmail = strings.ToLower(opts.AdminEmail)
	}
	if opts.AdminPwd == "" {
		opts.AdminPwd = defaultAdminPwd
		warnings = append(warnings, "PLUGIN_IAM_ADMIN_PASSWORD not set, using default password")
	}
	if opts.AdminName == "" {
		if idx := strings.Index(opts.AdminEmail, "@"); idx > 0 {
			opts.AdminName = opts.AdminEmail[:idx]
		} else {
			opts.AdminName = defaultAdminName
		}
	}
	return opts, warnings
}

func (o SeedOptions) Validate() error {
	if len(o.AdminPwd) < 6 {
		return fmt.Errorf("PLUGIN_IAM_ADMIN_PASSWORD must be at least 6 characters")
	}
	return nil
}

func resolveConfigValue(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func ensureDefaultDepartment(tx *gorm.DB, tenantUUID string) (*uint64, error) {
	var dept iamm.Department
	if err := tx.Where("tenant_uuid = ? AND code = ?", tenantUUID, "general").First(&dept).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			dept = iamm.Department{
				BaseModel:   basemodels.BaseModel{TenantUuid: tenantUUID},
				Name:        "General",
				Code:        "general",
				Description: "Default department",
			}
			if err := tx.Create(&dept).Error; err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	return &dept.ID, nil
}

func ensureMemberRoleBinding(tx *gorm.DB, memberID, roleID uint64) error {
	var rel iamm.MemberRole
	if err := tx.Where("member_id = ? AND role_id = ?", memberID, roleID).First(&rel).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		rel = iamm.MemberRole{UserID: memberID, RoleID: roleID}
		if err := tx.Create(&rel).Error; err != nil {
			return err
		}
	}
	return nil
}

func ensureDefaultRoles(tx *gorm.DB, tenantUUID string) (map[string]iamm.Role, error) {
	definitions := []defaultRoleSeed{
		{
			Code:        "system.admin",
			Name:        "System Admin",
			Description: "Default administrator role",
			ScopeType:   iamm.RoleScopeSystem,
		},
		{
			Code:        "role_admin",
			Name:        "Tenant Admin",
			Description: "Tenant administrator role",
			ScopeType:   iamm.RoleScopeTenant,
		},
		{
			Code:        "role_user",
			Name:        "Tenant User",
			Description: "Tenant member role",
			ScopeType:   iamm.RoleScopeTenant,
		},
		{
			Code:        "role_owner",
			Name:        "Tenant Owner",
			Description: "Tenant owner role (optional)",
			ScopeType:   iamm.RoleScopeTenant,
		},
	}

	result := make(map[string]iamm.Role, len(definitions))
	for _, def := range definitions {
		var role iamm.Role
		err := tx.Where("tenant_uuid = ? AND code = ?", tenantUUID, def.Code).First(&role).Error
		if err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, err
			}
			role = iamm.Role{
				BaseModel:     basemodels.BaseModel{TenantUuid: tenantUUID},
				Code:          def.Code,
				Name:          def.Name,
				Description:   def.Description,
				ScopeType:     def.ScopeType,
				PolicyVersion: defaultPolicyVersion,
			}
			if err := tx.Create(&role).Error; err != nil {
				return nil, err
			}
			result[def.Code] = role
			continue
		}

		updates := map[string]any{}
		if strings.TrimSpace(role.Name) == "" {
			updates["name"] = def.Name
		}
		if strings.TrimSpace(role.Description) == "" {
			updates["description"] = def.Description
		}
		if strings.TrimSpace(role.ScopeType) == "" {
			updates["scope_type"] = def.ScopeType
		}
		if strings.TrimSpace(role.PolicyVersion) == "" {
			updates["policy_version"] = defaultPolicyVersion
		}
		if len(updates) > 0 {
			if err := tx.Model(&role).Updates(updates).Error; err != nil {
				return nil, err
			}
			if err := tx.Where("id = ?", role.ID).First(&role).Error; err != nil {
				return nil, err
			}
		}
		result[def.Code] = role
	}
	return result, nil
}

func seedDefaultPermissions(tx *gorm.DB, roleID uint64, tenantUUID string) error {
	tenantUUID = strings.ToLower(strings.TrimSpace(tenantUUID))
	perms := []struct {
		Resource string
		Action   string
		Desc     string
	}{
		{"*", "*", "Full access"},
		{"iam.user", "read", "Read IAM users"},
		{"iam.role", "read", "Read IAM roles"},
		{"iam.tenant", "read", "Read IAM tenants"},
		{"iam.department", "read", "Read IAM departments"},
	}
	for _, p := range perms {
		var perm iamm.Permission
		if err := tx.Where("resource = ? AND action = ?", p.Resource, p.Action).First(&perm).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				perm = iamm.Permission{Resource: p.Resource, Action: p.Action, Description: p.Desc}
				if err := tx.Create(&perm).Error; err != nil {
					return err
				}
			} else {
				return err
			}
		}
		rp := iamm.RolePermission{
			RoleID:        roleID,
			PermissionID:  perm.ID,
			TenantUuid:    tenantUUID,
			PolicyVersion: defaultPolicyVersion,
		}
		if err := tx.Where("role_id = ? AND permission_id = ?", roleID, perm.ID).FirstOrCreate(&rp).Error; err != nil {
			return err
		}
	}
	return nil
}
