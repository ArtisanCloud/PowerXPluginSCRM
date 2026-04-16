package contract

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
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

func TestTagSyncConflictContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tenantUUID := "00000000-0000-0000-0000-000000000001"
	db := openTagSyncContractDB(t, "tag_sync_contract")
	seedTagConflict(t, db, tenantUUID, "tags", "tag-001")

	syncRepo := socialrepo.NewSyncFoundationRepository(db)
	jobSvc := socialsvc.NewSyncJobService(
		syncRepo,
		socialsvc.NewIdempotencyService(),
		socialsvc.NewSyncScheduler(),
		socialsvc.NewCapabilityService(socialsvc.NewChannelFactory()),
	)
	handler := socialhttp.NewSyncJobHandler(
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
	r.POST("/api/v1/admin/social/openwork/foundation/sync/jobs", handler.Create)
	r.GET("/api/v1/admin/social/openwork/foundation/sync/conflicts", conflictHandler.List)
	r.POST("/api/v1/admin/social/openwork/foundation/sync/conflicts/:conflict_uuid/replay", conflictHandler.Replay)

	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/admin/social/openwork/foundation/sync/jobs", mustBody(t, map[string]any{
		"domain":    "tags",
		"direction": "pull",
		"mode":      "incremental",
		"channel":   "wechat",
		"app_type":  "wecom",
	}))
	createReq.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()
	r.ServeHTTP(createRec, createReq)
	require.Equal(t, http.StatusOK, createRec.Code)

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin/social/openwork/foundation/sync/conflicts?domain=tags", nil)
	listRec := httptest.NewRecorder()
	r.ServeHTTP(listRec, listReq)
	require.Equal(t, http.StatusOK, listRec.Code)

	var listPayload map[string]any
	require.NoError(t, json.Unmarshal(listRec.Body.Bytes(), &listPayload))
	require.Equal(t, true, listPayload["success"])
	data := listPayload["data"].(map[string]any)
	items := data["items"].([]any)
	require.Len(t, items, 1)

	item := items[0].(map[string]any)
	conflictUUID := item["conflict_uuid"].(string)
	replayReq := httptest.NewRequest(http.MethodPost, "/api/v1/admin/social/openwork/foundation/sync/conflicts/"+conflictUUID+"/replay", mustBody(t, map[string]any{
		"resolved_by": "contract_tester",
	}))
	replayReq.Header.Set("Content-Type", "application/json")
	replayRec := httptest.NewRecorder()
	r.ServeHTTP(replayRec, replayReq)
	require.Equal(t, http.StatusOK, replayRec.Code)
}

func openTagSyncContractDB(t *testing.T, name string) *gorm.DB {
	t.Helper()
	basemodels.ForceSchemaForTests("")
	db, err := gorm.Open(sqlite.Open("file:"+name+"?mode=memory&cache=shared"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE IF NOT EXISTS social_sync_jobs (
		job_uuid TEXT PRIMARY KEY,
		tenant_uuid TEXT NOT NULL,
		domain TEXT NOT NULL,
		direction TEXT NOT NULL,
		mode TEXT NOT NULL,
		status TEXT NOT NULL,
		attempt_no INTEGER NOT NULL DEFAULT 0,
		max_attempts INTEGER NOT NULL DEFAULT 3,
		idempotency_key TEXT NOT NULL,
		payload TEXT,
		error_code TEXT,
		error_message TEXT,
		started_at DATETIME,
		finished_at DATETIME,
		capability_status TEXT,
		created_at DATETIME,
		updated_at DATETIME
	);`).Error)
	require.NoError(t, db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS uq_social_sync_jobs_idempotency
		ON social_sync_jobs (tenant_uuid, idempotency_key);`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE IF NOT EXISTS social_sync_conflicts (
		conflict_uuid TEXT PRIMARY KEY,
		tenant_uuid TEXT NOT NULL,
		domain TEXT NOT NULL,
		entity_type TEXT,
		entity_key TEXT,
		resolution_strategy TEXT,
		status TEXT NOT NULL,
		local_value TEXT,
		remote_value TEXT,
		resolved_by TEXT,
		resolved_at DATETIME,
		created_at DATETIME,
		updated_at DATETIME
	);`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE IF NOT EXISTS social_sync_dead_letters (
		dead_letter_uuid TEXT PRIMARY KEY,
		tenant_uuid TEXT NOT NULL,
		job_uuid TEXT NOT NULL,
		domain TEXT NOT NULL,
		direction TEXT,
		last_error_code TEXT,
		last_error_message TEXT,
		retry_exhausted_at DATETIME,
		replay_status TEXT,
		replayed_by TEXT,
		replayed_at DATETIME,
		payload TEXT,
		created_at DATETIME,
		updated_at DATETIME
	);`).Error)
	return db
}

func seedTagConflict(t *testing.T, db *gorm.DB, tenantUUID, domain, entityKey string) {
	t.Helper()
	now := time.Now().UTC()
	require.NoError(t, db.WithContext(context.Background()).Exec(`INSERT INTO social_sync_conflicts(
		conflict_uuid, tenant_uuid, domain, entity_type, entity_key, resolution_strategy, status, local_value, remote_value, created_at, updated_at
	) VALUES(?,?,?,?,?,?,?,?,?,?,?)`,
		"11111111-1111-1111-1111-111111111111",
		tenantUUID,
		domain,
		"tag",
		entityKey,
		"remote_first",
		"open",
		`{"name":"本地标签"}`,
		`{"name":"远端标签"}`,
		now,
		now,
	).Error)
}

func mustBody(t *testing.T, payload map[string]any) io.Reader {
	t.Helper()
	buf := &bytes.Buffer{}
	require.NoError(t, json.NewEncoder(buf).Encode(payload))
	return buf
}
