package app

// Integration RBAC 资源常量。
const (
	RBACIntegrationDispatchInvoke  = "integration.dispatch:invoke"
	RBACIntegrationGrantRead       = "integration.grant_matrix:read"
	RBACIntegrationGrantManage     = "integration.grant_matrix:manage"
	RBACIntegrationWebhookRead     = "integration.webhooks:read"
	RBACIntegrationWebhookManage   = "integration.webhooks:manage"
	RBACIntegrationSecretsRead     = "integration.secrets:read"
	RBACIntegrationSecretsManage   = "integration.secrets:manage"
	RBACIntegrationApprovalsRead   = "integration.approvals:read"
	RBACIntegrationApprovalsManage = "integration.approvals:manage"
)
