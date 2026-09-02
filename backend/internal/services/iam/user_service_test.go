package iam

import (
	"context"
	"testing"

	basemodels "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	iamm "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/iam"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestUserServiceListExposesStableMemberUUID(t *testing.T) {
	basemodels.ForceSchemaForTests("")
	db, err := gorm.Open(sqlite.Open("file:iam_member_uuid?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&iamm.User{}, &iamm.Member{}, &iamm.Role{}, &iamm.MemberRole{}))

	tenantUUID := "00000000-0000-4000-8000-000000000001"
	memberUUID := "30000000-0000-4000-8000-000000000001"
	account := &iamm.User{
		TenantUuid:  tenantUUID,
		Email:       "collector@example.test",
		DisplayName: "王采集",
		Status:      iamm.StatusActive,
		Meta:        datatypes.JSONMap{},
	}
	require.NoError(t, db.Create(account).Error)
	require.NoError(t, db.Create(&iamm.Member{
		BaseModel: basemodels.BaseModel{TenantUuid: tenantUUID},
		UserID:    account.ID,
		Username:  "collector",
		Status:    iamm.StatusActive,
		Meta:      datatypes.JSONMap{"member_uuid": memberUUID},
	}).Error)

	items, err := NewUserService(db, nil).List(context.Background(), UserFilter{TenantUUID: tenantUUID})
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, memberUUID, items[0].MemberUUID)
	require.Equal(t, "王采集", items[0].DisplayName)
}
