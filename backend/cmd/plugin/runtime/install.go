package runtime

import (
	"context"
	"fmt"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/capabilities"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/logger"
	capmetrics "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/observability/capability"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
)

// SyncCapabilities validates capability catalog and optionally registers it with host runtime.
func SyncCapabilities(ctx context.Context, mgr capabilities.Manager, client capabilities.HostSyncClient, metrics *capmetrics.Metrics) error {
	if mgr == nil {
		return nil
	}
	mode := "warmup"
	if client != nil {
		mode = "register"
	}
	record := func(status string) {
		if metrics != nil {
			metrics.ObserveCatalogSync(app.PluginID, mode, status)
		}
	}
	if client != nil {
		if err := mgr.RegisterWithHost(ctx, client); err != nil {
			record("failure")
			return fmt.Errorf("capability sync failed: %w", err)
		}
		record("success")
		return nil
	}
	if err := capabilities.EnsureManager(ctx, mgr, logger.WithField("component", "capabilities_sync")); err != nil {
		record("failure")
		return fmt.Errorf("capability warmup failed: %w", err)
	}
	record("success")
	return nil
}
