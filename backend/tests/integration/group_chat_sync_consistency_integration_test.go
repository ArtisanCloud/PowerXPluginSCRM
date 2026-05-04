package integration

import (
	"testing"
	"time"

	basemodels "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type groupChatSnapshotContract struct {
	SnapshotUUID       string     `gorm:"column:snapshot_uuid;primaryKey"`
	TenantUUID         string     `gorm:"column:tenant_uuid"`
	ChannelAccountUUID string     `gorm:"column:channel_account_uuid"`
	ChatID             string     `gorm:"column:chat_id"`
	Name               string     `gorm:"column:name"`
	OwnerUserID        string     `gorm:"column:owner_userid"`
	MemberCount        int        `gorm:"column:member_count"`
	SourceConfigID     string     `gorm:"column:source_config_id"`
	LastActivityAt     *time.Time `gorm:"column:last_activity_at"`
	UpdatedAt          time.Time  `gorm:"column:updated_at"`
}

func (groupChatSnapshotContract) TableName() string { return "acquisition_group_chat_snapshots" }

func TestGroupChatSyncConsistency_PullAndWebhookEventuallyConsistent(t *testing.T) {
	db := openGroupChatSyncConsistencyDB(t, "group_chat_sync_consistency")
	tenantUUID := "00000000-0000-0000-0000-000000000254"
	accountUUID := "30aa4d9f-6768-4fdf-97c8-66844422a7a5"
	chatID := "chat-001"

	t0 := time.Now().UTC().Add(-2 * time.Minute)
	t1 := t0.Add(30 * time.Second)
	t2 := t1.Add(30 * time.Second)

	require.NoError(t, upsertGroupChatSnapshot(db, groupChatSnapshotContract{
		SnapshotUUID:       "snap-pull-1",
		TenantUUID:         tenantUUID,
		ChannelAccountUUID: accountUUID,
		ChatID:             chatID,
		Name:               "活动群A",
		OwnerUserID:        "owner-a",
		MemberCount:        10,
		SourceConfigID:     "cfg-001",
		UpdatedAt:          t0,
	}))

	require.NoError(t, upsertGroupChatSnapshot(db, groupChatSnapshotContract{
		SnapshotUUID:       "snap-webhook-1",
		TenantUUID:         tenantUUID,
		ChannelAccountUUID: accountUUID,
		ChatID:             chatID,
		Name:               "活动群A",
		OwnerUserID:        "owner-a",
		MemberCount:        12,
		SourceConfigID:     "cfg-001",
		UpdatedAt:          t1,
	}))

	require.NoError(t, upsertGroupChatSnapshot(db, groupChatSnapshotContract{
		SnapshotUUID:       "snap-pull-2",
		TenantUUID:         tenantUUID,
		ChannelAccountUUID: accountUUID,
		ChatID:             chatID,
		Name:               "活动群A-改名",
		OwnerUserID:        "owner-b",
		MemberCount:        15,
		SourceConfigID:     "cfg-001",
		UpdatedAt:          t2,
	}))

	var rows []groupChatSnapshotContract
	require.NoError(t, db.Where("tenant_uuid = ? and channel_account_uuid = ?", tenantUUID, accountUUID).Find(&rows).Error)
	require.Len(t, rows, 1)
	require.Equal(t, "活动群A-改名", rows[0].Name)
	require.Equal(t, "owner-b", rows[0].OwnerUserID)
	require.Equal(t, 15, rows[0].MemberCount)
	require.Equal(t, t2.Unix(), rows[0].UpdatedAt.Unix())
}

func openGroupChatSyncConsistencyDB(t *testing.T, name string) *gorm.DB {
	t.Helper()
	basemodels.ForceSchemaForTests("")
	db, err := gorm.Open(sqlite.Open("file:"+name+"?mode=memory&cache=shared"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE IF NOT EXISTS acquisition_group_chat_snapshots (
		snapshot_uuid TEXT PRIMARY KEY,
		tenant_uuid TEXT NOT NULL,
		channel_account_uuid TEXT NOT NULL,
		chat_id TEXT NOT NULL,
		name TEXT,
		owner_userid TEXT,
		member_count INTEGER NOT NULL DEFAULT 0,
		source_config_id TEXT,
		last_activity_at DATETIME,
		updated_at DATETIME
	);`).Error)
	require.NoError(t, db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS uq_acq_group_chat_snapshot_scope
		ON acquisition_group_chat_snapshots (tenant_uuid, channel_account_uuid, chat_id);`).Error)
	return db
}

func upsertGroupChatSnapshot(db *gorm.DB, incoming groupChatSnapshotContract) error {
	var existing groupChatSnapshotContract
	err := db.Where("tenant_uuid = ? AND channel_account_uuid = ? AND chat_id = ?", incoming.TenantUUID, incoming.ChannelAccountUUID, incoming.ChatID).First(&existing).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return db.Create(&incoming).Error
		}
		return err
	}
	return db.Model(&existing).Updates(map[string]any{
		"name":             incoming.Name,
		"owner_userid":     incoming.OwnerUserID,
		"member_count":     incoming.MemberCount,
		"source_config_id": incoming.SourceConfigID,
		"last_activity_at": incoming.LastActivityAt,
		"updated_at":       incoming.UpdatedAt,
	}).Error
}
