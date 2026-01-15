package runtime_ops

import (
	"context"

	model "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/runtime_ops"
)

// UpdateQuotaMetrics publishes quota usage metrics from ledger entry.
func UpdateQuotaMetrics(pluginID string, entry *model.QuotaLedger) {
	if entry == nil {
		return
	}
	SetQuotaUsage(pluginID, entry.ScopeType, entry.ScopeRef, entry.TokensConsumed)
}

// EmitMarketplaceCost updates aggregate cost metric for reporting.
func EmitMarketplaceCost(pluginID, tenantID string, amount float64) {
	AddCost(pluginID, tenantID, amount)
}

// NotifyQuotaBreach pushes audit + metrics for breaches.
func NotifyQuotaBreach(ctx context.Context, svc *QuotaService, pluginID, tenantID, capability, action string) {
	if svc == nil {
		return
	}
	svc.HandleBreach(ctx, pluginID, tenantID, capability, action)
}
