package lead_capture

import (
	"context"
	"testing"

	basemodels "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	orgmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/org_sync"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestAssignmentService_RejectsUnboundMember(t *testing.T) {
	db := openAssignmentGuardTestDB(t, "assignment_guard_unbound")
	svc := NewAssignmentService()

	err := svc.EnsureMemberBound(context.Background(), db, "00000000-0000-0000-0000-000000000001", 1001)
	require.ErrorIs(t, err, ErrAssigneeNotBound)
}

func TestAssignmentService_AllowsConfirmedMapping(t *testing.T) {
	db := openAssignmentGuardTestDB(t, "assignment_guard_confirmed")
	require.NoError(t, db.Create(&orgmodel.MemberBinding{
		MemberBindingUUID:  "a0000000-0000-4000-8000-000000000001",
		TenantUUID:         "00000000-0000-0000-0000-000000000001",
		ChannelAccountUUID: "c0000000-0000-4000-8000-000000000001",
		MainMemberID:       "1001",
		ExternalMemberID:   "ext-1001",
		SyncStatus:         "synced",
	}).Error)

	svc := NewAssignmentService()
	err := svc.EnsureMemberBound(context.Background(), db, "00000000-0000-0000-0000-000000000001", 1001)
	require.NoError(t, err)
}

func openAssignmentGuardTestDB(t *testing.T, name string) *gorm.DB {
	t.Helper()
	basemodels.ForceSchemaForTests("")
	db, err := gorm.Open(sqlite.Open("file:"+name+"?mode=memory&cache=shared"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE IF NOT EXISTS org_sync_member_bindings (
		member_binding_uuid TEXT PRIMARY KEY,
		tenant_uuid TEXT NOT NULL,
		channel_account_uuid TEXT NOT NULL,
		main_member_id TEXT NOT NULL,
		external_member_id TEXT NOT NULL,
		sync_status TEXT NOT NULL,
		last_pulled_at DATETIME,
		last_pushed_at DATETIME,
		created_at DATETIME,
		updated_at DATETIME
	);`).Error)
	return db
}
