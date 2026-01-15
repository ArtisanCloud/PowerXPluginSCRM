package controller

import "context"

// EmitQuotaEvent notifies MCP controller about quota changes (placeholder).
func EmitQuotaEvent(ctx context.Context, pluginID, tenantID, capability, status string) {
	// TODO: integrate with MCP event pipeline
}
