package webhooks

import (
	"context"
	"encoding/json"
	"testing"

	oppmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/models/opportunity"
	entitymodels "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	leadmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/lead_capture"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestBuildOpenWorkCallbackKey_PreferAuthCode(t *testing.T) {
	key := buildOpenWorkCallbackKey("create_auth", "auth-code-001", "sig", "1", "2", "event")
	require.Equal(t, "auth_code:auth-code-001", key)
}

func TestBuildOpenWorkCallbackKey_UseSignatureTuple(t *testing.T) {
	key := buildOpenWorkCallbackKey("suite_ticket", "", "sig-001", "1775115592", "1774327042", "")
	require.Equal(t, "callback_signature:sig-001:1775115592:1774327042", key)
}

func TestBuildOpenWorkEventKey_PreferAuthCode(t *testing.T) {
	key := buildOpenWorkEventKey("tenant-1", "suite-1", "create_auth", "auth-code", "", "", "", map[string]any{"k": "v"})
	require.Equal(t, "tenant-1:suite-1:create_auth:auth_code:auth-code", key)
}

func TestBuildOpenWorkEventKey_UseSignatureTuple(t *testing.T) {
	key := buildOpenWorkEventKey("tenant-1", "suite-1", "cancel_auth", "", "sig-001", "1775115592", "1774327042", map[string]any{"k": "v"})
	require.Equal(t, "tenant-1:suite-1:cancel_auth:sig:sig-001:1775115592:1774327042", key)
}

func TestMarkLeadDisconnectedMarksActiveOpportunityRisk(t *testing.T) {
	db := openWorkCallbackOpportunityDB(t)
	tenantUUID := "00000000-0000-0000-0000-000000000901"
	channelAccountUUID := "00000000-0000-0000-0000-000000000902"
	externalUserID := "external-user-001"
	leadUUID := uuid.NewString()
	activeOpportunityUUID := uuid.NewString()
	wonOpportunityUUID := uuid.NewString()
	actorUUID := "00000000-0000-0000-0000-000000000903"

	require.NoError(t, db.Create(&leadmodel.Lead{
		LeadUUID:          leadUUID,
		TenantUUID:        tenantUUID,
		DisplayName:       "断开测试线索",
		Status:            leadmodel.LeadStatusConverted,
		OwnerUserUUID:     actorUUID,
		SourceChannel:     "wechat",
		SourceAppType:     "openwork",
		SourceAccountUUID: &channelAccountUUID,
	}).Error)
	require.NoError(t, db.Create(&leadmodel.LeadActivity{
		ActivityUUID: uuid.NewString(),
		LeadUUID:     leadUUID,
		TenantUUID:   tenantUUID,
		ActivityType: leadmodel.LeadActivityTypeSyncTrace,
		Payload: map[string]any{
			"source_account_uuid": channelAccountUUID,
			"external_wechat_id":  externalUserID,
		},
	}).Error)
	require.NoError(t, db.Create(&oppmodel.OpportunityRecord{
		OpportunityUUID: activeOpportunityUUID,
		TenantUUID:      tenantUUID,
		LeadUUID:        leadUUID,
		Title:           "活跃商机",
		Stage:           oppmodel.StageProposal,
		Currency:        "CNY",
		Probability:     30,
		OwnerUserUUID:   actorUUID,
		RiskFlags:       []byte(`[]`),
		CreatedBy:       actorUUID,
		UpdatedBy:       actorUUID,
	}).Error)
	require.NoError(t, db.Create(&oppmodel.OpportunityRecord{
		OpportunityUUID: wonOpportunityUUID,
		TenantUUID:      tenantUUID,
		LeadUUID:        leadUUID,
		Title:           "已赢单商机",
		Stage:           oppmodel.StageWon,
		Currency:        "CNY",
		Probability:     100,
		OwnerUserUUID:   actorUUID,
		RiskFlags:       []byte(`[]`),
		CreatedBy:       actorUUID,
		UpdatedBy:       actorUUID,
	}).Error)

	handler := &OpenWorkCallbackHandler{deps: &app.Deps{DB: db}}
	handler.markLeadDisconnectedByExternalContact(context.Background(), tenantUUID, channelAccountUUID, externalUserID)

	var lead leadmodel.Lead
	require.NoError(t, db.Where("tenant_uuid = ? AND lead_uuid = ?", tenantUUID, leadUUID).First(&lead).Error)
	require.Equal(t, leadmodel.LeadStatusDisconnected, lead.Status)
	require.Empty(t, lead.OwnerUserUUID)

	var active oppmodel.OpportunityRecord
	require.NoError(t, db.Where("opportunity_uuid = ?", activeOpportunityUUID).First(&active).Error)
	var flags []string
	require.NoError(t, json.Unmarshal(active.RiskFlags, &flags))
	require.Contains(t, flags, "disconnected")

	var won oppmodel.OpportunityRecord
	require.NoError(t, db.Where("opportunity_uuid = ?", wonOpportunityUUID).First(&won).Error)
	require.Equal(t, oppmodel.StageWon, won.Stage)
	require.JSONEq(t, `[]`, string(won.RiskFlags))

	var activity oppmodel.OpportunityActivity
	require.NoError(t, db.Where("opportunity_uuid = ? AND activity_type = ?", activeOpportunityUUID, oppmodel.ActivityRiskFlag).First(&activity).Error)
	require.Equal(t, actorUUID, activity.OperatorUserUUID)
	require.Contains(t, string(activity.Payload), "openwork_callback")
}

func openWorkCallbackOpportunityDB(t *testing.T) *gorm.DB {
	t.Helper()
	entitymodels.ForceSchemaForTests("")
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	require.NoError(t, err)
	for _, stmt := range []string{
		`CREATE TABLE lead_capture_leads (
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
		)`,
		`CREATE TABLE lead_capture_activities (
			activity_uuid TEXT PRIMARY KEY,
			lead_uuid TEXT NOT NULL,
			tenant_uuid TEXT NOT NULL,
			activity_type TEXT NOT NULL,
			payload TEXT,
			created_at DATETIME,
			updated_at DATETIME
		)`,
		`CREATE TABLE opportunity_records (
			opportunity_uuid TEXT PRIMARY KEY,
			tenant_uuid TEXT NOT NULL,
			lead_uuid TEXT NOT NULL,
			title TEXT NOT NULL,
			stage TEXT NOT NULL,
			amount NUMERIC,
			currency TEXT NOT NULL,
			probability INTEGER NOT NULL,
			owner_user_uuid TEXT NOT NULL,
			source_channel TEXT,
			source_app_type TEXT,
			source_account_uuid TEXT,
			external_userid TEXT,
			expected_close_at DATETIME,
			won_at DATETIME,
			lost_at DATETIME,
			lost_reason TEXT,
			risk_flags TEXT NOT NULL,
			created_by TEXT NOT NULL,
			updated_by TEXT NOT NULL,
			created_at DATETIME,
			updated_at DATETIME
		)`,
		`CREATE TABLE opportunity_activities (
			activity_uuid TEXT PRIMARY KEY,
			tenant_uuid TEXT NOT NULL,
			opportunity_uuid TEXT NOT NULL,
			activity_type TEXT NOT NULL,
			from_stage TEXT,
			to_stage TEXT,
			payload TEXT NOT NULL,
			operator_user_uuid TEXT NOT NULL,
			request_id TEXT,
			created_at DATETIME
		)`,
	} {
		require.NoError(t, db.Exec(stmt).Error)
	}
	return db
}
