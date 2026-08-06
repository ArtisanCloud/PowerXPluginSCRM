package integration

import (
	"context"
	"testing"
	"time"

	domainmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/models/lead_capture"
	domainrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/repository/lead_capture"
	basemodels "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	socialmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/social_channel_governance"
	leadrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/lead_capture"
	socialrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/social_channel_governance"
	leadsvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/lead_capture"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestChannelCodeEventIdempotencyIntegration_IngestAndTrace(t *testing.T) {
	db := openChannelCodeEventIdempotencyDB(t, "channel_code_event_idempotency")
	ctx := context.Background()
	tenantUUID := "00000000-0000-0000-0000-000000000061"
	accountUUID := "11111111-1111-4111-8111-111111111161"
	codeUUID := uuid.NewString()

	require.NoError(t, db.Create(&socialmodel.ChannelAccount{
		AccountUUID:     accountUUID,
		TenantUuid:      tenantUUID,
		ChannelCode:     "wechat",
		AppType:         "wecom",
		AccountID:       "wecom-int",
		DisplayName:     "企微集成账号",
		Status:          socialmodel.ChannelAccountStatusConnected,
		OrgSyncDefault:  true,
		OwnerMemberUUID: "00000000-0000-0000-0000-000000000151",
		CreatedAt:       time.Now().UTC(),
		UpdatedAt:       time.Now().UTC(),
	}).Error)
	require.NoError(t, db.Create(&domainmodel.ChannelCode{
		CodeUUID:           codeUUID,
		TenantUUID:         tenantUUID,
		Channel:            "wechat",
		AppType:            "wecom",
		ChannelAccountUUID: accountUUID,
		CodeKey:            "wecom-int-code",
		DisplayName:        "集成渠道码",
		TargetType:         "group",
		TargetID:           "group-int",
		Status:             domainmodel.ChannelCodeStatusActive,
		CreatedBy:          "system",
		UpdatedBy:          "system",
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}).Error)

	repos := domainrepo.NewBundle(db)
	leadRepository := leadrepo.NewLeadRepository(db)
	attributionSvc := leadsvc.NewAttributionService(repos.Attributions, leadRepository, leadsvc.NewLeadService(leadRepository))
	eventSvc := leadsvc.NewChannelCodeEventService(repos.ChannelCodeEvents, repos.ChannelCodes, socialrepo.NewAccountRepository(db), attributionSvc, nil)

	first, err := eventSvc.IngestWebhook(ctx, leadsvc.ChannelCodeEventIngestRequest{
		Channel:            "wechat",
		ChannelAccountUUID: accountUUID,
		CodeKey:            "wecom-int-code",
		ExternalEventID:    "evt-int-001",
		EventType:          "join",
		OccurredAt:         "2026-03-24T08:00:00Z",
		Payload: map[string]any{
			"phone": "13900009999",
			"name":  "王五",
		},
	})
	require.NoError(t, err)
	require.True(t, first.Created)
	require.NotEmpty(t, first.LeadUUID)

	second, err := eventSvc.IngestWebhook(ctx, leadsvc.ChannelCodeEventIngestRequest{
		Channel:            "wechat",
		ChannelAccountUUID: accountUUID,
		CodeKey:            "wecom-int-code",
		ExternalEventID:    "evt-int-001",
		EventType:          "join",
		OccurredAt:         "2026-03-24T08:00:00Z",
		Payload: map[string]any{
			"phone": "13900009999",
			"name":  "王五",
		},
	})
	require.NoError(t, err)
	require.False(t, second.Created)
	require.True(t, second.IdempotentHit)
	require.Equal(t, first.LeadUUID, second.LeadUUID)

	var eventCount int64
	require.NoError(t, db.Table("lead_capture_channel_code_events").Where("tenant_uuid = ?", tenantUUID).Count(&eventCount).Error)
	require.EqualValues(t, 1, eventCount)

	var attributionCount int64
	require.NoError(t, db.Table("lead_capture_lead_attribution_records").Where("tenant_uuid = ?", tenantUUID).Count(&attributionCount).Error)
	require.EqualValues(t, 1, attributionCount)

	var leadCount int64
	require.NoError(t, db.Table("lead_capture_leads").Where("tenant_uuid = ?", tenantUUID).Count(&leadCount).Error)
	require.EqualValues(t, 1, leadCount)
}

func openChannelCodeEventIdempotencyDB(t *testing.T, name string) *gorm.DB {
	t.Helper()
	basemodels.ForceSchemaForTests("")
	db, err := gorm.Open(sqlite.Open("file:"+name+"?mode=memory&cache=shared"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	require.NoError(t, err)
	require.NoError(t, ensureChannelCodeEventIdempotencyTables(db))
	return db
}

func ensureChannelCodeEventIdempotencyTables(db *gorm.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS social_channel_accounts (
			account_uuid TEXT PRIMARY KEY,
			tenant_uuid TEXT NOT NULL,
			channel_code TEXT NOT NULL,
			app_type TEXT NOT NULL,
			account_id TEXT NOT NULL,
			display_name TEXT NOT NULL,
			status TEXT NOT NULL,
			org_sync_default BOOLEAN NOT NULL DEFAULT FALSE,
			owner_member_uuid TEXT NOT NULL,
			member_user_uuids TEXT,
			capabilities TEXT,
			credentials TEXT,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		);`,
		`CREATE TABLE IF NOT EXISTS lead_capture_channel_codes (
			code_uuid TEXT PRIMARY KEY,
			tenant_uuid TEXT NOT NULL,
			channel TEXT NOT NULL,
			app_type TEXT NOT NULL,
			channel_account_uuid TEXT NOT NULL,
			code_key TEXT NOT NULL,
			display_name TEXT NOT NULL,
			target_type TEXT NOT NULL,
			target_id TEXT NOT NULL,
			status TEXT NOT NULL,
			created_by TEXT NOT NULL,
			updated_by TEXT NOT NULL,
			created_at DATETIME,
			updated_at DATETIME
		);`,
		`CREATE UNIQUE INDEX IF NOT EXISTS uq_lead_capture_channel_codes_tenant_channel_code
			ON lead_capture_channel_codes (tenant_uuid, channel, code_key);`,
		`CREATE TABLE IF NOT EXISTS lead_capture_channel_code_events (
			event_uuid TEXT PRIMARY KEY,
			tenant_uuid TEXT NOT NULL,
			channel TEXT NOT NULL,
			app_type TEXT NOT NULL,
			channel_account_uuid TEXT NOT NULL,
			code_uuid TEXT NOT NULL,
			external_event_id TEXT NOT NULL,
			event_type TEXT NOT NULL,
			idempotency_key TEXT NOT NULL UNIQUE,
			occurred_at DATETIME NOT NULL,
			payload TEXT NOT NULL,
			created_at DATETIME
		);`,
		`CREATE TABLE IF NOT EXISTS lead_capture_lead_attribution_records (
			attribution_uuid TEXT PRIMARY KEY,
			tenant_uuid TEXT NOT NULL,
			lead_uuid TEXT NOT NULL,
			code_uuid TEXT NOT NULL,
			event_uuid TEXT NOT NULL,
			is_primary BOOLEAN NOT NULL DEFAULT FALSE,
			attribution_type TEXT NOT NULL,
			created_at DATETIME
		);`,
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
	}
	for _, stmt := range stmts {
		if err := db.Exec(stmt).Error; err != nil {
			return err
		}
	}
	return nil
}
