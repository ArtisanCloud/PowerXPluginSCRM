package lead_capture

import (
	"context"
	"testing"
	"time"

	basemodels "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	leadmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/lead_capture"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestDedupService_PhoneHasHigherPriorityThanEmail(t *testing.T) {
	db := openMergePolicyTestDB(t, "merge_policy_phone_priority")
	now := time.Now().UTC()
	require.NoError(t, db.Create(&leadmodel.Lead{
		LeadUUID:   "10000000-0000-4000-8000-000000000001",
		TenantUUID: "00000000-0000-0000-0000-000000000001",
		Phone:      "13800000001",
		Email:      "phone@example.com",
		Status:     leadmodel.LeadStatusNew,
		CreatedAt:  now.Add(-2 * time.Minute),
		UpdatedAt:  now.Add(-2 * time.Minute),
	}).Error)
	require.NoError(t, db.Create(&leadmodel.Lead{
		LeadUUID:   "10000000-0000-4000-8000-000000000002",
		TenantUUID: "00000000-0000-0000-0000-000000000001",
		Phone:      "",
		Email:      "same@example.com",
		Status:     leadmodel.LeadStatusNew,
		CreatedAt:  now.Add(-1 * time.Minute),
		UpdatedAt:  now.Add(-1 * time.Minute),
	}).Error)

	svc := NewDedupService()
	lead, matchOn, err := svc.FindExistingLead(context.Background(), db, "00000000-0000-0000-0000-000000000001", "13800000001", "same@example.com")
	require.NoError(t, err)
	require.NotNil(t, lead)
	require.Equal(t, "phone", matchOn)
	require.Equal(t, "10000000-0000-4000-8000-000000000001", lead.LeadUUID)
}

func TestDedupService_BuildMergeUpdatesOnlyFillsEmptyFields(t *testing.T) {
	existing := &leadmodel.Lead{
		DisplayName:   "",
		Phone:         "13800000001",
		Email:         "",
		SourceChannel: "",
		SourceAppType: "",
	}
	svc := NewDedupService()
	updates, merged := svc.BuildMergeUpdates(existing, NormalizedLeadInput{
		DisplayName:       "Alice",
		Phone:             "13800000099",
		Email:             "alice@example.com",
		SourceChannel:     "wechat",
		SourceAppType:     "wecom",
		SourceAccountUUID: "11111111-1111-4111-8111-111111111111",
	})
	require.NotContains(t, updates, "phone")
	require.Contains(t, updates, "display_name")
	require.Contains(t, updates, "email")
	require.Contains(t, updates, "source_channel")
	require.Contains(t, updates, "source_app_type")
	require.Contains(t, updates, "source_account_uuid")
	require.ElementsMatch(t, []string{"display_name", "email", "source_channel", "source_app_type", "source_account_uuid"}, merged)
}

func openMergePolicyTestDB(t *testing.T, name string) *gorm.DB {
	t.Helper()
	basemodels.ForceSchemaForTests("")
	db, err := gorm.Open(sqlite.Open("file:"+name+"?mode=memory&cache=shared"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE IF NOT EXISTS lead_capture_leads (
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
	);`).Error)
	return db
}
