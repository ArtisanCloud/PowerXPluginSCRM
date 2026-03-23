package lead_capture

import (
	"context"
	"testing"
	"time"

	basemodels "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	leadmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/lead_capture"
	leadrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/lead_capture"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestBotCommandService_PermissionAndParsing(t *testing.T) {
	db := openBotCommandServiceTestDB(t, "bot_command_service_permission")
	tenantUUID := "00000000-0000-0000-0000-000000000001"
	accountUUID := "11111111-1111-4111-8111-111111111111"

	svc := NewBotCommandService(
		leadrepo.NewConversationEventRepository(db),
		leadrepo.NewLeadConversationBindingRepository(db),
		leadrepo.NewLeadRepository(db),
		NewLeadService(leadrepo.NewLeadRepository(db)),
	)

	_, err := svc.HandleWeComLeadCreateCommand(context.Background(), WeComBotCommandInput{
		TenantUUID:         tenantUUID,
		ChannelAccountUUID: accountUUID,
		ExternalEventID:    "evt-bot-001",
		ConversationID:     "conv-bot-001",
		OperatorID:         "op-001",
		CommandText:        "创建线索 姓名=张三 手机=13800000000",
		Permissions:        []string{"scrm.leads:read"},
		OccurredAt:         time.Now().UTC(),
		RawPayload:         map[string]any{"source": "test"},
	})
	require.ErrorIs(t, err, ErrBotCommandForbidden)

	_, err = svc.HandleWeComLeadCreateCommand(context.Background(), WeComBotCommandInput{
		TenantUUID:         tenantUUID,
		ChannelAccountUUID: accountUUID,
		ExternalEventID:    "evt-bot-002",
		ConversationID:     "conv-bot-002",
		OperatorID:         "op-001",
		CommandText:        "随机指令 xxx",
		Permissions:        []string{"scrm.lead.create"},
		OccurredAt:         time.Now().UTC(),
		RawPayload:         map[string]any{"source": "test"},
	})
	require.ErrorIs(t, err, ErrBotCommandUnsupported)

	result, err := svc.HandleWeComLeadCreateCommand(context.Background(), WeComBotCommandInput{
		TenantUUID:         tenantUUID,
		ChannelAccountUUID: accountUUID,
		ExternalEventID:    "evt-bot-003",
		ConversationID:     "conv-bot-003",
		OperatorID:         "op-001",
		CommandText:        "创建线索 姓名=张三 手机=13800000000 邮箱=a@example.com",
		Permissions:        []string{"scrm.lead.create"},
		OccurredAt:         time.Now().UTC(),
		RawPayload:         map[string]any{"source": "test"},
	})
	require.NoError(t, err)
	require.True(t, result.Created)
	require.NotEmpty(t, result.RequestID)
	require.NotEmpty(t, result.LeadUUID)

	var activity leadmodel.LeadActivity
	require.NoError(t, db.Where("tenant_uuid = ? AND lead_uuid = ? AND activity_type = ?", tenantUUID, result.LeadUUID, leadmodel.LeadActivityTypeBotCommand).First(&activity).Error)
	require.Equal(t, result.RequestID, activity.Payload["request_id"])
}

func openBotCommandServiceTestDB(t *testing.T, name string) *gorm.DB {
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
