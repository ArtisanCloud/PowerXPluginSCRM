package models

// backend/internal/entity/models/model.go

import (
	"time"

	"gorm.io/gorm"
)

// BaseModel 基础模型，包含通用字段
type BaseModel struct {
	ID         uint64         `gorm:"primaryKey;autoIncrement;comment:自增ID" json:"id"`
	TenantUuid string         `gorm:"type:uuid;not null;index;comment:租户UUID" json:"tenant_uuid"`
	CreatedAt  time.Time      `gorm:"autoCreateTime;comment:创建时间" json:"created_at"`
	UpdatedAt  time.Time      `gorm:"autoUpdateTime;comment:更新时间" json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"column:deleted_at;comment:软删除时间" json:"deleted_at,omitempty"`
}

type BaseNoTenantModel struct {
	ID        uint64         `gorm:"primaryKey;autoIncrement"`
	CreatedAt time.Time      `gorm:"autoCreateTime"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at" json:"deleted_at,omitempty"`
}

const (
	TablePluginTenantExt                 = "plugin_tenant_ext"
	TableTemplate                        = "template"
	TablePluginCredentials               = "plugin_credentials"
	TablePrivacyDataClassifications      = "privacy_data_classifications"
	TablePrivacyConsentTokens            = "privacy_consent_tokens"
	TablePrivacyLifecycleEvents          = "privacy_lifecycle_events"
	TableSecurityBaselineChecklists      = "security_baseline_checklists"
	TableSecurityAuditReports            = "security_audit_reports"
	TableSecurityVulnerabilityAdvisory   = "security_vulnerability_advisories"
	TableSecurityAdvisoryDistributions   = "security_advisory_distributions"
	TableToolGrantRevocations            = "tool_grant_revocations"
	TableToolGrantUsageEvents            = "tool_grant_usage_events"
	TableIntegrationWebhookSubscriptions = "integration_webhook_subscriptions"
	TableIntegrationWebhookAttempts      = "integration_webhook_attempts"
	TableIntegrationSecrets              = "integration_secrets"
	TableIntegrationGrantMatrixOverrides = "integration_grant_matrix_overrides"
	TableMarketplaceListings             = "marketplace_listings"
	TableMarketplaceListingAssets        = "marketplace_listing_assets"
	TableMarketplaceListingVersions      = "marketplace_listing_versions"
	TableMarketplacePricingPlans         = "marketplace_pricing_plans"
	TableMarketplacePlanTiers            = "marketplace_plan_tiers"
	TableMarketplaceChecklistRuns        = "marketplace_checklist_runs"
	TableMarketplaceChecklistItems       = "marketplace_checklist_items"
	TableMarketplaceLicenses             = "marketplace_licenses"
	TableMarketplaceLicenseEvents        = "marketplace_license_events"
	TableMarketplaceTaxTransactions      = "marketplace_tax_transactions"
	TableCustomerAccounts                = "customer_accounts"
	TableMarketplaceUsageEnvelopes       = "marketplace_usage_envelopes"
	TableMarketplaceUsageAggregates      = "marketplace_usage_aggregates"
	TableMarketplaceRevenueReports       = "marketplace_revenue_share_reports"
	TableMarketplaceNotifications        = "marketplace_notifications"
	TableOperationsSupportChannels       = "operations_support_channels"
	TableOperationsSupportTickets        = "operations_support_tickets"
	TableOperationsIncidents             = "operations_incidents"
	TableOperationsIncidentUpdates       = "operations_incident_updates"
	TableOperationsIncidentChecklist     = "operations_incident_checklist"
	TableOperationsSupportTicketEvents   = "operations_support_ticket_events"
	TableOperationsReadinessItems        = "operations_readiness_checklist_items"
	TableOperationsSLAScores             = "operations_sla_profiles"
	TableOperationsSLAAdjustments        = "operations_sla_adjustments"
	TableAdminConsoleAuditEvents         = "admin_console_audit_events"
	TableAdminConsoleConfigChanges       = "admin_console_config_changes"
	TableAdminConsoleJobRuns             = "admin_console_job_runs"
	TableIAMTenants                      = "iam_tenants"
	TableIAMUsers                        = "iam_users"
	TableIAMMembers                      = "iam_members"
	TableIAMRoles                        = "iam_roles"
	TableIAMPermissions                  = "iam_permissions"
	TableIAMDepartments                  = "iam_departments"
	TableIAMMemberRoles                  = "iam_member_roles"
	TableIAMRolePermissions              = "iam_role_permissions"
	TableIAMRefreshTokens                = "iam_refresh_tokens"
	TableIAMAuditLogs                    = "iam_audit_logs"
)
