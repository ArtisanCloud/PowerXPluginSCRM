package social_channel_governance_test

import "testing"

func TestTenantRLSIsolationPlaceholder(t *testing.T) {
	t.Skip("integration environment required: verify cross-tenant access is denied with SET LOCAL app.tenant_uuid")
}
