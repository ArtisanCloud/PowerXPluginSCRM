package lead_capture

import (
	"context"
	"testing"
	"time"

	fwwsbus "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/wsbus"
	basemodels "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	leadmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/lead_capture"
	leadrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/lead_capture"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type wsPublishStub struct {
	calls int
}

func (s *wsPublishStub) Publish(_ context.Context, _ string, _ any, _ fwwsbus.PublishOptions) fwwsbus.PublishResult {
	s.calls++
	return fwwsbus.SuccessResult()
}

func TestConversationService_IdempotencyAndDuplicateWebhook(t *testing.T) {
	db := openConversationServiceTestDB(t, "conversation_service_idempotency")
	tenantUUID := "00000000-0000-0000-0000-000000000001"
	leadUUID := "10000000-0000-4000-8000-000000000001"
	require.NoError(t, db.Create(&leadmodel.Lead{
		LeadUUID:    leadUUID,
		TenantUUID:  tenantUUID,
		Phone:       "13800000001",
		Status:      leadmodel.LeadStatusNew,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
		DisplayName: "Alice",
	}).Error)

	wsStub := &wsPublishStub{}
	svc := NewConversationService(
		leadrepo.NewConversationEventRepository(db),
		leadrepo.NewLeadConversationBindingRepository(db),
		leadrepo.NewLeadConversationPendingRepository(db),
		leadrepo.NewLeadRealtimeProjectionRepository(db),
		NewConversationRealtimePublisher(wsStub, nil),
		nil,
	).WithLeadRepository(leadrepo.NewLeadRepository(db))

	in := ConversationWebhookInput{
		TenantUUID:         tenantUUID,
		ChannelAccountUUID: "11111111-1111-4111-8111-111111111111",
		ExternalEventID:    "evt-001",
		ConversationID:     "conv-001",
		ActorType:          "staff",
		ActorID:            "staff-1",
		Direction:          "inbound",
		MessageType:        "text",
		ContentText:        "hello",
		OccurredAt:         time.Now().UTC(),
		RawPayload: map[string]any{
			"phone": "13800000001",
		},
	}
	firstEventUUID, created, err := svc.IngestWebhook(context.Background(), in)
	require.NoError(t, err)
	require.True(t, created)
	require.NotEmpty(t, firstEventUUID)

	secondEventUUID, createdAgain, err := svc.IngestWebhook(context.Background(), in)
	require.NoError(t, err)
	require.False(t, createdAgain)
	require.Equal(t, firstEventUUID, secondEventUUID)
	require.Equal(t, 1, wsStub.calls)

	var eventCount int64
	require.NoError(t, db.Model(&leadmodel.ConversationEvent{}).Count(&eventCount).Error)
	require.EqualValues(t, 1, eventCount)

	var bindCount int64
	require.NoError(t, db.Model(&leadmodel.LeadConversationBinding{}).
		Where("tenant_uuid = ? AND lead_uuid = ? AND conversation_id = ?", tenantUUID, leadUUID, "conv-001").
		Count(&bindCount).Error)
	require.EqualValues(t, 1, bindCount)
}

func openConversationServiceTestDB(t *testing.T, name string) *gorm.DB {
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
