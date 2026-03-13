package integration

import (
	"context"
	"testing"
	"time"

	basemodels "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	leadmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/lead_capture"
	leadrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/lead_capture"
	leadsvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/lead_capture"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestLeadCaptureBotCommandIntegration_IdempotentReplay(t *testing.T) {
	db := openLeadCaptureBotCommandDB(t, "lead_capture_bot_command_integration")
	tenantUUID := "00000000-0000-0000-0000-000000000001"
	accountUUID := "11111111-1111-4111-8111-111111111111"
	eventID := "evt-bot-replay-001"

	leadRepository := leadrepo.NewLeadRepository(db)
	svc := leadsvc.NewBotCommandService(
		leadrepo.NewConversationEventRepository(db),
		leadrepo.NewLeadConversationBindingRepository(db),
		leadRepository,
		leadsvc.NewLeadService(leadRepository),
	)

	first, err := svc.HandleWeComLeadCreateCommand(context.Background(), leadsvc.WeComBotCommandInput{
		TenantUUID:         tenantUUID,
		ChannelAccountUUID: accountUUID,
		ExternalEventID:    eventID,
		ConversationID:     "conv-bot-001",
		OperatorID:         "op-001",
		CommandText:        "创建线索 姓名=李四 手机=13900000000",
		Permissions:        []string{"scrm.lead.create"},
		OccurredAt:         time.Now().UTC(),
		RawPayload:         map[string]any{"source": "integration"},
	})
	require.NoError(t, err)
	require.True(t, first.Created)
	require.NotEmpty(t, first.LeadUUID)

	second, err := svc.HandleWeComLeadCreateCommand(context.Background(), leadsvc.WeComBotCommandInput{
		TenantUUID:         tenantUUID,
		ChannelAccountUUID: accountUUID,
		ExternalEventID:    eventID,
		ConversationID:     "conv-bot-001",
		OperatorID:         "op-001",
		CommandText:        "创建线索 姓名=李四 手机=13900000000",
		Permissions:        []string{"scrm.lead.create"},
		OccurredAt:         time.Now().UTC(),
		RawPayload:         map[string]any{"source": "integration"},
	})
	require.NoError(t, err)
	require.False(t, second.Created)
	require.Equal(t, first.LeadUUID, second.LeadUUID)
	require.Equal(t, first.RequestID, second.RequestID)

	var leadCount int64
	require.NoError(t, db.Model(&leadmodel.Lead{}).Where("tenant_uuid = ?", tenantUUID).Count(&leadCount).Error)
	require.EqualValues(t, 1, leadCount)
}

func openLeadCaptureBotCommandDB(t *testing.T, name string) *gorm.DB {
	t.Helper()
	basemodels.ForceSchemaForTests("")
	db, err := gorm.Open(sqlite.Open("file:"+name+"?mode=memory&cache=shared"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	require.NoError(t, err)

	stmts := []string{
		`CREATE TABLE IF NOT EXISTS lead_capture_leads (
			lead_uuid TEXT PRIMARY KEY,
			tenant_uuid TEXT NOT NULL,
			display_name TEXT,
			phone TEXT,
			email TEXT,
			status TEXT NOT NULL,
			owner_user_uuid TEXT,
			source_channel TEXT,
			source_app_type TEXT,
			source_account_uuid TEXT,
			created_at DATETIME,
			updated_at DATETIME
		);`,
		`CREATE TABLE IF NOT EXISTS lead_capture_sources (
			source_uuid TEXT PRIMARY KEY,
			lead_uuid TEXT NOT NULL,
			tenant_uuid TEXT NOT NULL,
			channel_code TEXT,
			app_type TEXT,
			account_uuid TEXT,
			campaign_code TEXT,
			utm_source TEXT,
			utm_medium TEXT,
			utm_campaign TEXT,
			created_at DATETIME,
			updated_at DATETIME
		);`,
		`CREATE TABLE IF NOT EXISTS lead_capture_activities (
			activity_uuid TEXT PRIMARY KEY,
			lead_uuid TEXT NOT NULL,
			tenant_uuid TEXT NOT NULL,
			activity_type TEXT NOT NULL,
			payload TEXT,
			created_at DATETIME,
			updated_at DATETIME
		);`,
		`CREATE TABLE IF NOT EXISTS lead_capture_conversation_events (
			event_uuid TEXT PRIMARY KEY,
			tenant_uuid TEXT NOT NULL,
			channel TEXT NOT NULL,
			app_type TEXT NOT NULL,
			channel_account_uuid TEXT NOT NULL,
			external_event_id TEXT NOT NULL,
			idempotency_key TEXT NOT NULL UNIQUE,
			conversation_id TEXT NOT NULL,
			actor_type TEXT NOT NULL,
			actor_id TEXT NOT NULL,
			direction TEXT NOT NULL,
			message_type TEXT NOT NULL,
			content_text TEXT,
			raw_payload TEXT,
			occurred_at DATETIME NOT NULL,
			created_at DATETIME
		);`,
		`CREATE TABLE IF NOT EXISTS lead_capture_conversation_bindings (
			binding_uuid TEXT PRIMARY KEY,
			tenant_uuid TEXT NOT NULL,
			lead_uuid TEXT NOT NULL,
			conversation_id TEXT NOT NULL,
			channel_account_uuid TEXT NOT NULL,
			bind_source TEXT NOT NULL,
			status TEXT NOT NULL,
			created_by TEXT,
			created_at DATETIME,
			updated_at DATETIME
		);`,
		`CREATE UNIQUE INDEX IF NOT EXISTS uq_lead_capture_conv_bindings_active
			ON lead_capture_conversation_bindings (tenant_uuid, conversation_id, status);`,
	}
	for _, stmt := range stmts {
		require.NoError(t, db.Exec(stmt).Error)
	}
	return db
}
