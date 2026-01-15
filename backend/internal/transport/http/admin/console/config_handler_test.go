package console_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/config"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	adminmetrics "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/observability/admin_console"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
	consoletransport "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/admin/console"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupHandler(t *testing.T) (*consoletransport.ConfigHandler, *gorm.DB) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open("file:admin_console_handler?mode=memory&cache=shared"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
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
	return consoletransport.NewConfigHandler(deps), db
}

func TestConfigHandler_ListSections(t *testing.T) {
	handler, _ := setupHandler(t)
	r := gin.New()
	req := httptest.NewRequest(http.MethodGet, "/config/sections", nil)
	resp := httptest.NewRecorder()
	ctx := gin.CreateTestContextOnly(resp, r)
	ctx.Request = req
	handler.ListSections(ctx)
	require.Equal(t, http.StatusOK, resp.Code)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &payload))
	require.True(t, payload["success"].(bool))
}

func TestConfigHandler_UpdateSection(t *testing.T) {
	handler, _ := setupHandler(t)
	r := gin.New()
	body := map[string]any{
		"values": map[string]any{
			"audit_retention_days":         210,
			"config_change_retention_days": 90,
			"job_history_days":             30,
		},
	}
	data, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPut, "/config/sections/admin_console.retention", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	ctx := gin.CreateTestContextOnly(resp, r)
	ctx.Params = gin.Params{gin.Param{Key: "sectionKey", Value: "admin_console.retention"}}
	ctx.Request = req
	handler.UpdateSection(ctx)
	require.Equal(t, http.StatusOK, resp.Code)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &payload))
	require.True(t, payload["success"].(bool))
}
