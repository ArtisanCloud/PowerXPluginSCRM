package social_channel_governance_test

import (
	"context"
	"testing"

	basemodels "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	model "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/social_channel_governance"
	socialrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/social_channel_governance"
	socialsvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/social_channel_governance"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestManualToDelegatedRollbackIntegration(t *testing.T) {
	db := openAccountIntegrationDB(t, "manual_to_delegated_rollback")
	repo := socialrepo.NewAccountRepository(db)
	svc := socialsvc.NewChannelAccountService(repo, nil, nil, nil, nil, nil)

	account := &model.ChannelAccount{
		AccountUUID:     "22222222-2222-4222-8222-222222222222",
		TenantUuid:      "00000000-0000-0000-0000-000000000001",
		ChannelCode:     "wechat",
		AppType:         "wecom",
		AccountID:       "manual-account",
		DisplayName:     "manual account",
		Status:          model.ChannelAccountStatusConnected,
		OwnerMemberUUID: "1",
		Credentials:     datatypes.JSONMap{},
	}
	require.NoError(t, db.Create(account).Error)

	updated, err := svc.MigrateManualToDelegated(context.Background(), account.TenantUuid, account.AccountUUID, "binding-001")
	require.NoError(t, err)
	require.Equal(t, "manual_to_delegated", updated.Credentials["migration_state"])
	require.Equal(t, "binding-001", updated.Credentials["foundation_binding_uuid"])

	rolledBack, err := svc.RollbackDelegatedToManual(context.Background(), account.TenantUuid, account.AccountUUID)
	require.NoError(t, err)
	require.Equal(t, "delegated_rollback_manual", rolledBack.Credentials["migration_state"])
	_, hasBinding := rolledBack.Credentials["foundation_binding_uuid"]
	require.False(t, hasBinding)
}

func openAccountIntegrationDB(t *testing.T, name string) *gorm.DB {
	t.Helper()
	basemodels.ForceSchemaForTests("")
	db, err := gorm.Open(sqlite.Open("file:"+name+"?mode=memory&cache=shared"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE IF NOT EXISTS social_channel_accounts (
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
	);`).Error)
	return db
}
