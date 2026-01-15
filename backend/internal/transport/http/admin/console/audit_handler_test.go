package console_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/config"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	model "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/admin_console"
	consolerepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/admin_console"
	adminmetrics "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/observability/admin_console"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
	consoletransport "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/admin/console"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func setupAuditHandler(t *testing.T) (*consoletransport.AuditHandler, *consolerepo.AuditRepository, *gorm.DB) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open("file:admin_console_audit?mode=memory&cache=shared"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
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
	deps := &app.Deps{DB: db, Config: cfg, AdminConsoleMetrics: adminmetrics.NewMetrics()}
	repo := consolerepo.NewAuditRepository(db)
	return consoletransport.NewAuditHandler(deps), repo, db
}

func seedAuditEvent(t *testing.T, repo *consolerepo.AuditRepository, evt model.AuditEvent) {
	t.Helper()
	evt.ID = uuid.NewString()
	if evt.OccurredAt.IsZero() {
		evt.OccurredAt = time.Now().UTC()
	}
	evt.CreatedAt = evt.OccurredAt
	if evt.Diff == nil {
		evt.Diff = datatypes.JSON([]byte("{}"))
	}
	require.NoError(t, repo.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&evt).Error)
}

func TestAuditHandler_ListEvents(t *testing.T) {
	handler, repo, _ := setupAuditHandler(t)
	seedAuditEvent(t, repo, model.AuditEvent{
		PluginID:       app.PluginID,
		ActorID:        "user:1",
		ActorName:      stringPtr("Alice"),
		PermissionCode: "operations.plugin.admin",
		Action:         "config.section.update",
		ResourceType:   "config.section",
		ResourceRef:    stringPtr("admin_console.retention"),
		Summary:        stringPtr("Updated retention policy"),
		OccurredAt:     time.Now().UTC().Add(-time.Hour),
	})

	r := gin.New()
	req := httptest.NewRequest(http.MethodGet, "/audit/events", nil)
	resp := httptest.NewRecorder()
	ctx := gin.CreateTestContextOnly(resp, r)
	ctx.Request = req
	handler.ListEvents(ctx)
	require.Equal(t, http.StatusOK, resp.Code)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &payload))
	require.True(t, payload["success"].(bool))
}

func TestAuditHandler_ExportEvents(t *testing.T) {
	handler, repo, _ := setupAuditHandler(t)
	seedAuditEvent(t, repo, model.AuditEvent{
		PluginID:       app.PluginID,
		ActorID:        "user:2",
		PermissionCode: "operations.plugin.admin",
		Action:         "config.section.update",
		ResourceType:   "config.section",
		OccurredAt:     time.Now().UTC(),
	})

	r := gin.New()
	req := httptest.NewRequest(http.MethodGet, "/audit/export?format=json", nil)
	resp := httptest.NewRecorder()
	ctx := gin.CreateTestContextOnly(resp, r)
	ctx.Request = req
	handler.ExportEvents(ctx)
	require.Equal(t, http.StatusOK, resp.Code)
	require.Equal(t, "application/json", resp.Header().Get("Content-Type"))
	var payload map[string]any
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &payload))
	require.True(t, payload["success"].(bool))
	data := payload["data"].(map[string]any)
	require.NotEmpty(t, data["content_base64"])
}

func stringPtr(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
