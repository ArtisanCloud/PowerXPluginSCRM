package integration

import (
	"context"
	"testing"
	"time"

	fwwsbus "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/wsbus"
	basemodels "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	leadmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/lead_capture"
	leadrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/lead_capture"
	leadsvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/lead_capture"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type wsPublishCapture struct {
	topics   []string
	payloads []any
}

func (c *wsPublishCapture) Publish(_ context.Context, topic string, payload any, _ fwwsbus.PublishOptions) fwwsbus.PublishResult {
	c.topics = append(c.topics, topic)
	c.payloads = append(c.payloads, payload)
	return fwwsbus.SuccessResult()
}

func TestLeadConversationWSIntegration_BindingAndTopicPublish(t *testing.T) {
	db := openLeadConversationWSDB(t, "lead_conversation_ws_integration")
	tenantUUID := "00000000-0000-0000-0000-000000000001"
	leadUUID := "10000000-0000-4000-8000-000000000001"
	accountUUID := "11111111-1111-4111-8111-111111111111"
	now := time.Now().UTC()

	require.NoError(t, db.Create(&leadmodel.Lead{
		LeadUUID:    leadUUID,
		TenantUUID:  tenantUUID,
		DisplayName: "Alice",
		Phone:       "13800000001",
		Status:      leadmodel.LeadStatusNew,
		CreatedAt:   now,
		UpdatedAt:   now,
	}).Error)

	capture := &wsPublishCapture{}
	realtime := leadsvc.NewConversationRealtimePublisher(capture, nil)
	svc := leadsvc.NewConversationService(
		leadrepo.NewConversationEventRepository(db),
		leadrepo.NewLeadConversationBindingRepository(db),
		leadrepo.NewLeadConversationPendingRepository(db),
		leadrepo.NewLeadRealtimeProjectionRepository(db),
		realtime,
		nil,
	).WithLeadRepository(leadrepo.NewLeadRepository(db))

	eventUUID, created, err := svc.IngestWebhook(context.Background(), leadsvc.ConversationWebhookInput{
		TenantUUID:         tenantUUID,
		ChannelAccountUUID: accountUUID,
		ExternalEventID:    "evt-1001",
		ConversationID:     "conv-001",
		ActorType:          "staff",
		ActorID:            "staff-1",
		Direction:          "inbound",
		MessageType:        "text",
		ContentText:        "客户咨询价格",
		OccurredAt:         now,
		RawPayload: map[string]any{
			"phone": "13800000001",
		},
	})
	require.NoError(t, err)
	require.True(t, created)
	require.NotEmpty(t, eventUUID)

	require.Len(t, capture.topics, 1)
	require.Equal(t, leadsvc.TopicLeadConversationUpdatedV1, capture.topics[0])

	payload, ok := capture.payloads[0].(leadsvc.LeadConversationUpdatedEvent)
	require.True(t, ok)
	require.Equal(t, leadUUID, payload.LeadUUID)
	require.Equal(t, "conv-001", payload.ConversationID)

	projections, err := leadrepo.NewLeadRealtimeProjectionRepository(db).ListByLead(context.Background(), tenantUUID, leadUUID, 10)
	require.NoError(t, err)
	require.Len(t, projections, 1)
	require.Equal(t, "客户咨询价格", projections[0].LatestMessage)
	require.Equal(t, 1, projections[0].UnreadCount)
}

func openLeadConversationWSDB(t *testing.T, name string) *gorm.DB {
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
		`CREATE TABLE IF NOT EXISTS lead_capture_conversation_pending (
			pending_uuid TEXT PRIMARY KEY,
			tenant_uuid TEXT NOT NULL,
			event_uuid TEXT NOT NULL UNIQUE,
			channel_account_uuid TEXT NOT NULL,
			conversation_id TEXT NOT NULL,
			reason TEXT NOT NULL,
			status TEXT NOT NULL,
			created_at DATETIME,
			resolved_at DATETIME
		);`,
		`CREATE TABLE IF NOT EXISTS lead_capture_realtime_projection (
			projection_uuid TEXT PRIMARY KEY,
			tenant_uuid TEXT NOT NULL,
			lead_uuid TEXT NOT NULL,
			conversation_id TEXT NOT NULL,
			latest_message TEXT,
			latest_actor_type TEXT,
			latest_at DATETIME,
			unread_count INTEGER NOT NULL DEFAULT 0,
			updated_at DATETIME,
			created_at DATETIME
		);`,
		`CREATE UNIQUE INDEX IF NOT EXISTS uq_lead_capture_realtime_projection
			ON lead_capture_realtime_projection (tenant_uuid, lead_uuid, conversation_id);`,
	}
	for _, stmt := range stmts {
		require.NoError(t, db.Exec(stmt).Error)
	}
	return db
}
