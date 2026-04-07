package social_channel_governance_test

import (
	"context"
	"testing"
	"time"

	basemodels "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	socialrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/social_channel_governance"
	socialsvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/social_channel_governance"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestOpenWorkFoundationIngestEventCreatesBinding(t *testing.T) {
	db := openOpenWorkIntegrationDB(t, "openwork_foundation_integration")
	repo := socialrepo.NewOpenWorkFoundationRepository(db)
	svc := socialsvc.NewOpenWorkFoundationService(repo, nil)

	event, binding, err := svc.IngestEvent(context.Background(), socialsvc.OpenWorkEventIngestInput{
		TenantUUID: "00000000-0000-0000-0000-000000000001",
		SuiteID:    "dk-openwork-integration",
		EventType:  "create_auth",
		CorpID:     "ww-integration",
		AgentID:    "1000048",
		EventTime:  time.Now().Unix(),
		Payload:    map[string]any{"source": "integration"},
	})
	require.NoError(t, err)
	require.NotNil(t, event)
	require.NotNil(t, binding)
	require.Equal(t, "active", binding.Status)
	require.Equal(t, "ww-integration", binding.CorpID)
}

func openOpenWorkIntegrationDB(t *testing.T, name string) *gorm.DB {
	t.Helper()
	basemodels.ForceSchemaForTests("")
	db, err := gorm.Open(sqlite.Open("file:"+name+"?mode=memory&cache=shared"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE IF NOT EXISTS social_wecom_auth_events (
		event_uuid TEXT PRIMARY KEY,
		tenant_uuid TEXT NOT NULL,
		suite_id TEXT NOT NULL,
		event_type TEXT NOT NULL,
		corp_id TEXT,
		agent_id TEXT,
		event_time DATETIME,
		event_key TEXT UNIQUE NOT NULL,
		payload TEXT,
		created_at DATETIME
	);`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE IF NOT EXISTS social_wecom_auth_bindings (
		binding_uuid TEXT PRIMARY KEY,
		tenant_uuid TEXT NOT NULL,
		channel_account_uuid TEXT,
		channel_code TEXT,
		app_type TEXT,
		suite_id TEXT NOT NULL,
		corp_id TEXT NOT NULL,
		agent_id TEXT,
		corp_name TEXT,
		permanent_code TEXT,
		suite_access_token TEXT,
		suite_ticket TEXT,
		status TEXT NOT NULL,
		is_default BOOLEAN NOT NULL DEFAULT FALSE,
		default_switched_at DATETIME,
		last_event_type TEXT,
		last_event_at DATETIME,
		auth_scope TEXT,
		metadata TEXT,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	);`).Error)
	require.NoError(t, db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS uq_social_wecom_auth_binding_identity
		ON social_wecom_auth_bindings (tenant_uuid, corp_id, agent_id);`).Error)
	return db
}
