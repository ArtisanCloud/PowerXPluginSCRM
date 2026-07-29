package contract

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	basemodels "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	iamentity "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/iam"
	orgmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/org_sync"
	socialmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/social_channel_governance"
	orgrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/org_sync"
	socialrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/social_channel_governance"
	authx "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/middleware"
	orgsvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/org_sync"
	socialsvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/social_channel_governance"
	orghttp "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/admin/org_sync"
	socialhttp "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/admin/social_channel_governance"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestOrgSyncConflictContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tenantUUID := "00000000-0000-0000-0000-000000000001"
	db := openTagSyncContractDB(t, "org_sync_contract")
	seedTagConflict(t, db, tenantUUID, "org", "dept-001")

	syncRepo := socialrepo.NewSyncFoundationRepository(db)
	jobSvc := socialsvc.NewSyncJobService(
		syncRepo,
		socialsvc.NewIdempotencyService(),
		socialsvc.NewSyncScheduler(),
		socialsvc.NewCapabilityService(socialsvc.NewChannelFactory()),
	)
	jobHandler := socialhttp.NewSyncJobHandler(
		jobSvc,
		socialsvc.NewSyncOrchestrator(nil, socialsvc.NewSyncScheduler(), jobSvc),
		socialsvc.NewCapabilityService(socialsvc.NewChannelFactory()),
		socialsvc.NewConflictResolutionService(syncRepo),
		nil,
		nil,
	)
	conflictHandler := socialhttp.NewConflictHandler(socialsvc.NewConflictResolutionService(syncRepo), socialsvc.NewRetryDeadletterService(syncRepo))

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("tenant_uuid", tenantUUID)
		authx.SetTenantContext(c, authx.TenantContext{TenantUUID: tenantUUID})
		c.Request = c.Request.WithContext(authx.ContextWithTenantUUID(c.Request.Context(), tenantUUID))
		c.Next()
	})
	r.POST("/api/v1/admin/social/openwork/foundation/sync/jobs", jobHandler.Create)
	r.GET("/api/v1/admin/social/openwork/foundation/sync/conflicts", conflictHandler.List)

	createPayload := map[string]any{
		"domain":    "org",
		"direction": "push",
		"mode":      "pushback",
		"channel":   "wechat",
		"app_type":  "wecom",
		"payload": map[string]any{
			"changes": []map[string]any{
				{"entity_type": "department", "entity_id": "dept-001", "action": "move"},
			},
		},
	}
	raw, err := json.Marshal(createPayload)
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/social/openwork/foundation/sync/jobs", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin/social/openwork/foundation/sync/conflicts?domain=org", nil)
	listRec := httptest.NewRecorder()
	r.ServeHTTP(listRec, listReq)
	require.Equal(t, http.StatusOK, listRec.Code)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(listRec.Body.Bytes(), &payload))
	require.Equal(t, true, payload["success"])
	data := payload["data"].(map[string]any)
	items := data["items"].([]any)
	require.GreaterOrEqual(t, len(items), 1)
}

func TestOrgPushPreviewContract_LocalUnmappedCreateCandidates(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tenantUUID := "00000000-0000-0000-0000-000000000001"
	channelAccountUUID := "30aa4d9f-6768-4fdf-97c8-66844422a7a5"
	db := openOrgSyncPushPreviewContractDB(t, "org_sync_push_preview_contract")
	require.NoError(t, db.WithContext(context.Background()).Create(&socialmodel.ChannelAccount{
		AccountUUID:     channelAccountUUID,
		TenantUuid:      tenantUUID,
		ChannelCode:     "wechat",
		AppType:         "wecom",
		AccountID:       "corp-001",
		DisplayName:     "default-wecom",
		Status:          socialmodel.ChannelAccountStatusConnected,
		OrgSyncDefault:  true,
		OwnerMemberUUID: "owner-001",
	}).Error)
	require.NoError(t, db.WithContext(context.Background()).Create(&orgmodel.SourceAccount{
		SourceAccountUUID:  channelAccountUUID,
		TenantUUID:         tenantUUID,
		Provider:           "wechat",
		AppType:            "wecom",
		ChannelAccountUUID: &channelAccountUUID,
		DisplayName:        "default-wecom",
		Status:             orgmodel.SourceAccountStatusActive,
	}).Error)
	require.NoError(t, db.WithContext(context.Background()).Create(&iamentity.Department{
		BaseModel: basemodels.BaseModel{
			TenantUuid: tenantUUID,
		},
		Name:      "销售部",
		Code:      "sales",
		Path:      "sales",
		SortOrder: 1,
	}).Error)
	var dept iamentity.Department
	require.NoError(t, db.WithContext(context.Background()).Where("tenant_uuid = ?", tenantUUID).First(&dept).Error)
	require.NoError(t, db.WithContext(context.Background()).Create(&iamentity.User{
		Email:        "org-preview@example.com",
		Phone:        "13800002222",
		DisplayName:  "李四",
		Status:       iamentity.StatusActive,
		PasswordHash: "preview_hash",
	}).Error)
	var user iamentity.User
	require.NoError(t, db.WithContext(context.Background()).Where("email = ?", "org-preview@example.com").First(&user).Error)
	require.NoError(t, db.WithContext(context.Background()).Create(&iamentity.Member{
		BaseModel: basemodels.BaseModel{
			TenantUuid: tenantUUID,
		},
		UserID:       user.ID,
		Username:     "lisi",
		DisplayName:  "李四",
		Status:       iamentity.StatusActive,
		DepartmentID: &dept.ID,
	}).Error)

	syncSvc := orgsvc.NewSyncService(
		orgrepo.NewSourceAccountRepository(db),
		orgrepo.NewSourceUnitRepository(db),
		orgrepo.NewSourceMemberRepository(db),
		orgrepo.NewSyncLogRepository(db),
		nil,
		nil,
		nil,
	)
	handler := orghttp.NewOrgSyncHandler(syncSvc, nil, nil, nil, nil)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("tenant_uuid", tenantUUID)
		authx.SetTenantContext(c, authx.TenantContext{TenantUUID: tenantUUID})
		c.Request = c.Request.WithContext(authx.ContextWithTenantUUID(c.Request.Context(), tenantUUID))
		c.Next()
	})
	r.GET("/api/v1/admin/org-sync/source-accounts/:source_account_uuid/push-preview", handler.PreviewPushSync)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/org-sync/source-accounts/"+channelAccountUUID+"/push-preview", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	require.Equal(t, true, payload["success"])
	data := payload["data"].(map[string]any)
	items := data["items"].([]any)
	require.GreaterOrEqual(t, len(items), 2)
	createCount := 0
	for _, raw := range items {
		item := raw.(map[string]any)
		if item["action"] == "create" {
			createCount++
		}
	}
	require.GreaterOrEqual(t, createCount, 2)
}

func openOrgSyncPushPreviewContractDB(t *testing.T, name string) *gorm.DB {
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
		org_sync_default BOOLEAN NOT NULL DEFAULT 0,
		owner_member_uuid TEXT NOT NULL,
		member_user_uuids TEXT,
		capabilities TEXT,
		credentials TEXT,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	);`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE IF NOT EXISTS org_sync_source_accounts (
		source_account_uuid TEXT PRIMARY KEY,
		tenant_uuid TEXT NOT NULL,
		provider TEXT NOT NULL,
		app_type TEXT NOT NULL,
		channel_account_uuid TEXT,
		display_name TEXT NOT NULL,
		status TEXT NOT NULL,
		last_sync_at DATETIME,
		last_sync_status TEXT,
		last_sync_message TEXT,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	);`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE IF NOT EXISTS org_sync_source_units (
		source_unit_uuid TEXT PRIMARY KEY,
		tenant_uuid TEXT NOT NULL,
		source_account_uuid TEXT NOT NULL,
		channel_account_uuid TEXT NOT NULL,
		external_unit_id TEXT NOT NULL,
		parent_external_unit_id TEXT,
		name TEXT NOT NULL,
		"order" INTEGER NOT NULL DEFAULT 0,
		status TEXT NOT NULL,
		created_at DATETIME,
		updated_at DATETIME
	);`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE IF NOT EXISTS org_sync_source_members (
		source_member_uuid TEXT PRIMARY KEY,
		tenant_uuid TEXT NOT NULL,
		source_account_uuid TEXT NOT NULL,
		channel_account_uuid TEXT NOT NULL,
		external_member_id TEXT NOT NULL,
		name TEXT NOT NULL,
		phone TEXT,
		email TEXT,
		profile_status TEXT,
		status TEXT NOT NULL,
		created_at DATETIME,
		updated_at DATETIME
	);`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE IF NOT EXISTS org_sync_source_member_units (
		source_member_unit_uuid TEXT PRIMARY KEY,
		tenant_uuid TEXT NOT NULL,
		source_account_uuid TEXT NOT NULL,
		channel_account_uuid TEXT NOT NULL,
		source_member_uuid TEXT NOT NULL,
		source_unit_uuid TEXT NOT NULL,
		external_member_id TEXT NOT NULL,
		external_unit_id TEXT NOT NULL,
		"order" INTEGER NOT NULL DEFAULT 0,
		created_at DATETIME,
		updated_at DATETIME
	);`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE IF NOT EXISTS org_sync_unit_bindings (
		unit_binding_uuid TEXT PRIMARY KEY,
		tenant_uuid TEXT NOT NULL,
		channel_account_uuid TEXT NOT NULL,
		main_unit_id TEXT NOT NULL,
		external_unit_id TEXT NOT NULL,
		parent_external_unit_id TEXT,
		sync_status TEXT,
		last_pulled_at DATETIME,
		last_pushed_at DATETIME,
		created_at DATETIME,
		updated_at DATETIME
	);`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE IF NOT EXISTS org_sync_member_bindings (
		member_binding_uuid TEXT PRIMARY KEY,
		tenant_uuid TEXT NOT NULL,
		channel_account_uuid TEXT NOT NULL,
		main_member_id TEXT NOT NULL,
		external_member_id TEXT NOT NULL,
		sync_status TEXT,
		last_pulled_at DATETIME,
		last_pushed_at DATETIME,
		created_at DATETIME,
		updated_at DATETIME
	);`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE IF NOT EXISTS iam_departments (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		tenant_uuid TEXT NOT NULL,
		name TEXT NOT NULL,
		code TEXT NOT NULL,
		parent_id INTEGER,
		description TEXT,
		path TEXT NOT NULL,
		sort_order INTEGER,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	);`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE IF NOT EXISTS iam_users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		email TEXT,
		phone TEXT,
		display_name TEXT,
		avatar_url TEXT,
		status TEXT NOT NULL,
		is_root BOOLEAN NOT NULL DEFAULT 0,
		password_hash TEXT NOT NULL,
		meta TEXT,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	);`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE IF NOT EXISTS iam_members (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		tenant_uuid TEXT NOT NULL,
		user_id INTEGER NOT NULL,
		username TEXT NOT NULL,
		display_name TEXT,
		avatar_url TEXT,
		status TEXT NOT NULL,
		department_id INTEGER,
		meta TEXT,
		last_login_at DATETIME,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	);`).Error)
	return db
}
