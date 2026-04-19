package contract

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	basemodels "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	socialrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/social_channel_governance"
	authx "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/middleware"
	socialsvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/social_channel_governance"
	socialhttp "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/admin/social_channel_governance"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestFoundationAccessStatusContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tenantUUID := "00000000-0000-0000-0000-000000000001"
	templateID := "dk-contract-template"

	db := openOpenWorkContractDB(t, "social_foundation_contract")
	require.NoError(t, insertContractBinding(context.Background(), db, tenantUUID, templateID))

	repo := socialrepo.NewOpenWorkFoundationRepository(db)
	svc := socialsvc.NewOpenWorkFoundationService(repo, nil)
	handler := socialhttp.NewOpenWorkFoundationHandler(svc)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("tenant_uuid", tenantUUID)
		authx.SetTenantContext(c, authx.TenantContext{TenantUUID: tenantUUID})
		c.Request = c.Request.WithContext(authx.ContextWithTenantUUID(c.Request.Context(), tenantUUID))
		c.Next()
	})
	r.GET("/api/v1/admin/social/openwork/wecom/foundation/access/status", handler.GetFoundationAccessStatus)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/social/openwork/wecom/foundation/access/status?template_id="+templateID, nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	require.Equal(t, true, payload["success"])
	data, ok := payload["data"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "authorized", data["auth_status"])
	require.Equal(t, "valid", data["token_status"])
	require.Equal(t, "ok", data["callback_status"])
}

func openOpenWorkContractDB(t *testing.T, name string) *gorm.DB {
	t.Helper()
	basemodels.ForceSchemaForTests("")
	db, err := gorm.Open(sqlite.Open("file:"+name+"?mode=memory&cache=shared"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE IF NOT EXISTS social_wecom_auth_bindings (
		binding_uuid TEXT PRIMARY KEY,
		tenant_uuid TEXT NOT NULL,
		channel_account_uuid TEXT,
		channel_code TEXT,
		app_type TEXT,
		suite_id TEXT NOT NULL,
		corp_id TEXT,
		corp_name TEXT,
		agent_id TEXT,
		permanent_code TEXT,
		suite_access_token TEXT,
		suite_ticket TEXT,
		status TEXT NOT NULL,
		is_default BOOLEAN NOT NULL DEFAULT FALSE,
		last_event_type TEXT,
		last_event_at DATETIME,
		auth_scope TEXT,
		metadata TEXT,
		created_at DATETIME,
		updated_at DATETIME,
		deleted_at DATETIME
	);`).Error)
	return db
}

func insertContractBinding(ctx context.Context, db *gorm.DB, tenantUUID, suiteID string) error {
	now := time.Now().UTC()
	return db.WithContext(ctx).Exec(`INSERT INTO social_wecom_auth_bindings(
		binding_uuid, tenant_uuid, suite_id, corp_id, corp_name, agent_id, channel_account_uuid,
		status, suite_access_token, last_event_type, last_event_at, created_at, updated_at
	) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		"11111111-1111-1111-1111-111111111111",
		tenantUUID,
		suiteID,
		"ww-contract",
		"Contract Corp",
		"1000001",
		"22222222-2222-2222-2222-222222222222",
		"active",
		"suite-token",
		"auth_complete",
		now,
		now,
		now,
	).Error
}
