package manifestx

import "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/manifest"

// Plugin returns the manifest definition consumed by the framework/router layer.
func Plugin() manifest.Plugin {
	return manifest.Plugin{
		ID:      "com.powerx.plugins.base",
		Name:    "PowerX Base Plugin",
		Version: "0.1.0",
		Permissions: []string{
			"iam.tenant.read",
			"iam.tenant.write",
			"iam.department.read",
			"iam.department.write",
			"iam.department.delete",
			"iam.user.read",
			"iam.user.write",
			"iam.role.read",
			"iam.role.write",
			"iam.role.delete",
			"iam.permission.read",
			"iam.audit.read",
			"iam.sts.mint",
			"base.templates.read",
			"base.templates.manage",
		},
		Menus: []manifest.Menu{
			{
				Path:  "/_p/com.powerx.plugins.base/admin/templates/intro",
				Title: "模板介绍",
			},
			{
				Path:  "/_p/com.powerx.plugins.base/admin/templates/crud",
				Title: "模板管理",
			},
			{
				Path:  "/_p/com.powerx.plugins.base/admin/iam/overview",
				Title: "组织与权限",
			},
			{
				Path:  "/_p/com.powerx.plugins.base/admin/iam/members",
				Title: "成员管理",
			},
			{
				Path:  "/_p/com.powerx.plugins.base/admin/iam/roles",
				Title: "角色与权限",
			},
		},
	}
}
