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

func TestLeadCaptureChannelRuleIntegration_RuleOffVsOn(t *testing.T) {
	db := openLeadCaptureChannelRuleDB(t, "lead_capture_channel_rule_integration")
	tenantUUID := "00000000-0000-0000-0000-000000000001"
	accountUUID := "11111111-1111-4111-8111-111111111111"

	leadRepository := leadrepo.NewLeadRepository(db)
	svc := leadsvc.NewConversationService(
		leadrepo.NewConversationEventRepository(db),
		leadrepo.NewLeadConversationBindingRepository(db),
		leadrepo.NewLeadConversationPendingRepository(db),
		leadrepo.NewLeadRealtimeProjectionRepository(db),
		nil,
		nil,
	).WithLeadRepository(leadRepository).
		WithLeadService(leadsvc.NewLeadService(leadRepository)).
		WithChannelRuleRepository(leadrepo.NewChannelRuleRepository(db))

	// rule off -> pending
	_, created, err := svc.IngestWebhook(context.Background(), leadsvc.ConversationWebhookInput{
		TenantUUID:         tenantUUID,
		ChannelAccountUUID: accountUUID,
		ExternalEventID:    "evt-rule-off",
		ConversationID:     "conv-rule-off",
		ActorType:          "customer",
		ActorID:            "customer-off",
		Direction:          "inbound",
		MessageType:        "text",
		ContentText:        "咨询",
		OccurredAt:         time.Now().UTC(),
		RawPayload: map[string]any{
			"phone": "13800002001",
		},
	})
	require.NoError(t, err)
	require.True(t, created)
	var leadCount int64
	require.NoError(t, db.Model(&leadmodel.Lead{}).Count(&leadCount).Error)
	require.EqualValues(t, 0, leadCount)

	// rule on -> auto create + bind
	_, err = leadrepo.NewChannelRuleRepository(db).UpsertAutoCreateLeadRule(context.Background(), tenantUUID, "wechat", "wecom", true)
	require.NoError(t, err)
	_, created, err = svc.IngestWebhook(context.Background(), leadsvc.ConversationWebhookInput{
		TenantUUID:         tenantUUID,
		ChannelAccountUUID: accountUUID,
		ExternalEventID:    "evt-rule-on",
		ConversationID:     "conv-rule-on",
		ActorType:          "customer",
		ActorID:            "customer-on",
		Direction:          "inbound",
		MessageType:        "text",
		ContentText:        "咨询",
		OccurredAt:         time.Now().UTC(),
		RawPayload: map[string]any{
			"phone": "13800002002",
		},
	})
	require.NoError(t, err)
	require.True(t, created)
	require.NoError(t, db.Model(&leadmodel.Lead{}).Count(&leadCount).Error)
	require.EqualValues(t, 1, leadCount)
	var bindingCount int64
	require.NoError(t, db.Model(&leadmodel.LeadConversationBinding{}).Where("conversation_id = ?", "conv-rule-on").Count(&bindingCount).Error)
	require.EqualValues(t, 1, bindingCount)
}

func openLeadCaptureChannelRuleDB(t *testing.T, name string) *gorm.DB {
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
		`CREATE TABLE IF NOT EXISTS lead_capture_channel_rules (
			rule_uuid TEXT PRIMARY KEY,
			tenant_uuid TEXT NOT NULL,
			channel TEXT NOT NULL,
			app_type TEXT NOT NULL,
			auto_create_lead_from_customer_dm BOOLEAN NOT NULL DEFAULT FALSE,
			created_at DATETIME,
			updated_at DATETIME
		);`,
		`CREATE UNIQUE INDEX IF NOT EXISTS uq_lead_capture_channel_rules_tenant_channel_app
			ON lead_capture_channel_rules (tenant_uuid, channel, app_type);`,
	}
	for _, stmt := range stmts {
		require.NoError(t, db.Exec(stmt).Error)
	}
	return db
}
