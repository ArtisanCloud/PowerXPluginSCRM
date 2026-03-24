package lead_capture

import (
	"context"
	"testing"
	"time"

	domainmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/models/lead_capture"
	domainrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/repository/lead_capture"
	basemodels "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	leadrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/lead_capture"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestAttributionService_FirstTouchPrimaryAndMultiMapping(t *testing.T) {
	db := openAttributionServiceDB(t, "attribution_service")
	ctx := context.Background()
	tenantUUID := "00000000-0000-0000-0000-000000000041"

	repos := domainrepo.NewBundle(db)
	leadRepository := leadrepo.NewLeadRepository(db)
	svc := NewAttributionService(repos.Attributions, leadRepository, NewLeadService(leadRepository))

	codeA := &domainmodel.ChannelCode{
		CodeUUID:           uuid.NewString(),
		TenantUUID:         tenantUUID,
		Channel:            "wechat",
		AppType:            "wecom",
		ChannelAccountUUID: "11111111-1111-4111-8111-111111111141",
		CodeKey:            "code-a",
		DisplayName:        "A",
		TargetType:         "group",
		TargetID:           "group-a",
		Status:             domainmodel.ChannelCodeStatusActive,
		CreatedBy:          "system",
		UpdatedBy:          "system",
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}
	codeB := *codeA
	codeB.CodeUUID = uuid.NewString()
	codeB.CodeKey = "code-b"
	require.NoError(t, db.Create(codeA).Error)
	require.NoError(t, db.Create(&codeB).Error)

	event1 := &domainmodel.ChannelCodeEvent{
		EventUUID:          uuid.NewString(),
		TenantUUID:         tenantUUID,
		Channel:            "wechat",
		AppType:            "wecom",
		ChannelAccountUUID: codeA.ChannelAccountUUID,
		CodeUUID:           codeA.CodeUUID,
		ExternalEventID:    "evt-attr-001",
		EventType:          "join",
		IdempotencyKey:     "k1",
		OccurredAt:         time.Now().UTC(),
		Payload:            []byte(`{"phone":"13800001111","display_name":"张三"}`),
	}
	event2 := *event1
	event2.EventUUID = uuid.NewString()
	event2.CodeUUID = codeB.CodeUUID
	event2.ExternalEventID = "evt-attr-002"
	event2.IdempotencyKey = "k2"

	leadUUID1, primary1, err := svc.AttributeByEvent(ctx, event1, codeA)
	require.NoError(t, err)
	require.NotEmpty(t, leadUUID1)
	require.True(t, primary1)

	leadUUID2, primary2, err := svc.AttributeByEvent(ctx, &event2, &codeB)
	require.NoError(t, err)
	require.Equal(t, leadUUID1, leadUUID2)
	require.False(t, primary2)

	records, err := repos.Attributions.ListByLeadUUID(ctx, tenantUUID, leadUUID1)
	require.NoError(t, err)
	require.Len(t, records, 2)
	require.True(t, records[0].IsPrimary)
	require.Equal(t, domainmodel.AttributionTypeFirstTouch, records[0].AttributionType)
	require.False(t, records[1].IsPrimary)
	require.Equal(t, domainmodel.AttributionTypeFollowTouch, records[1].AttributionType)
}

func openAttributionServiceDB(t *testing.T, name string) *gorm.DB {
	t.Helper()
	basemodels.ForceSchemaForTests("")
	db, err := gorm.Open(sqlite.Open("file:"+name+"?mode=memory&cache=shared"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	require.NoError(t, err)
	require.NoError(t, ensureAttributionServiceTables(db))
	return db
}

func ensureAttributionServiceTables(db *gorm.DB) error {
	stmts := []string{
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
