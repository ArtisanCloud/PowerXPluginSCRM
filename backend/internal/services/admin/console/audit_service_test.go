package console_test

import (
	"context"
	"testing"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/config"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	model "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/admin_console"
	consolerepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/admin_console"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/observability/admin_console"
	consolesvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/console"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupAuditService(t *testing.T) (*consolesvc.AuditService, *consolerepo.AuditRepository, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:admin_console_audit_service?mode=memory&cache=shared"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	require.NoError(t, err)
	models.ForceSchemaForTests("")
	require.NoError(t, db.Exec(`CREATE TABLE IF NOT EXISTS admin_console_audit_events (
		id TEXT PRIMARY KEY,
		plugin_id TEXT NOT NULL,
		tenant_uuid TEXT,
		actor_id TEXT NOT NULL,
		actor_name TEXT,
		actor_email TEXT,
		permission_code TEXT NOT NULL,
		action TEXT NOT NULL,
		resource_type TEXT NOT NULL,
		resource_ref TEXT,
		summary TEXT,
		diff TEXT,
		occurred_at DATETIME,
		created_at DATETIME
	);`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE IF NOT EXISTS admin_console_config_changes (
		id TEXT PRIMARY KEY,
		plugin_id TEXT NOT NULL,
		tenant_uuid TEXT,
		section_key TEXT NOT NULL,
		change_type TEXT NOT NULL,
		previous_snapshot TEXT,
		next_snapshot TEXT,
		validation_summary TEXT,
		audit_event_id TEXT,
		applied_at DATETIME
	);`).Error)
	cfg := &config.Config{}
	deps := &app.Deps{DB: db, Config: cfg, AdminConsoleMetrics: admin_console.NewMetrics()}
	repo := consolerepo.NewAuditRepository(db)
	svc := consolesvc.NewAuditService(deps)
	return svc, repo, db
}

func TestAuditService_ListEvents(t *testing.T) {
	svc, repo, _ := setupAuditService(t)
	now := time.Now().UTC()
	seedAuditEvent(t, repo, model.AuditEvent{
		PluginID:       app.PluginID,
		ActorID:        "user:1",
		PermissionCode: "operations.plugin.admin",
		Action:         "config.section.update",
		ResourceType:   "config.section",
		OccurredAt:     now,
	})
	result, err := svc.ListEvents(context.Background(), consolesvc.ListAuditInput{Limit: 10})
	require.NoError(t, err)
	require.NotEmpty(t, result.Events)
}

func TestAuditService_ExportEventsCSV(t *testing.T) {
	svc, repo, _ := setupAuditService(t)
	seedAuditEvent(t, repo, model.AuditEvent{
		PluginID:       app.PluginID,
		ActorID:        "user:2",
		PermissionCode: "operations.plugin.admin",
		Action:         "config.section.update",
		ResourceType:   "config.section",
		OccurredAt:     time.Now().UTC(),
	})
	result, err := svc.ExportEvents(context.Background(), consolesvc.ExportAuditInput{Format: "csv"})
	require.NoError(t, err)
	require.Equal(t, "text/csv", result.ContentType)
	require.NotEmpty(t, result.Content)
}

func seedAuditEvent(t *testing.T, repo *consolerepo.AuditRepository, evt model.AuditEvent) {
	t.Helper()
	if evt.ID == "" {
		evt.ID = uuid.NewString()
	}
	if evt.OccurredAt.IsZero() {
		evt.OccurredAt = time.Now().UTC()
	}
	if evt.CreatedAt.IsZero() {
		evt.CreatedAt = evt.OccurredAt
	}
	if len(evt.Diff) == 0 {
		evt.Diff = datatypes.JSON([]byte("{}"))
	}
	require.NoError(t, repo.Create(context.Background(), &evt))
}
