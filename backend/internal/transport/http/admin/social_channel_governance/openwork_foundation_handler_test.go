package social_channel_governance

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	basemodels "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	model "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/social_channel_governance"
	socialrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/social_channel_governance"
	socialsvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/social_channel_governance"
	httpmw "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/middleware"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type httpFakeRoundTripper struct{}

func (httpFakeRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	path := req.URL.Path
	query := req.URL.RawQuery
	body := `{"errcode":40001,"errmsg":"unknown endpoint"}`
	switch {
	case strings.HasSuffix(path, "/get_provider_token"):
		body = `{"errcode":0,"errmsg":"ok","provider_access_token":"provider-token-001","expires_in":7200}`
	case strings.HasSuffix(path, "/get_customized_auth_url") && strings.Contains(query, "provider_access_token=provider-token-001"):
		body = `{"errcode":0,"errmsg":"ok","qrcode_url":"https://open.work.weixin.qq.com/3rdapp/install?suite_id=dk001"}`
	case strings.HasSuffix(path, "/get_suite_token"):
		body = `{"errcode":0,"errmsg":"ok","suite_access_token":"suite-token-001","expires_in":7200}`
	case strings.HasSuffix(path, "/get_permanent_code") && strings.Contains(query, "suite_access_token=suite-token-001"):
		body = `{
			"errcode":0,
			"errmsg":"ok",
			"corpid":"wwcorp001",
			"permanent_code":"perm-code-001",
			"auth_corp_info":{"corpid":"wwcorp001","corp_name":"Demo Corp"},
			"auth_info":{"agent":[{"agentid":1000002}]},
			"authorization_info":{"auth_user_info":{"userid":"admin001"}}
		}`
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    req,
	}, nil
}

func TestOpenWorkFoundationHandler_StartAuthorization(t *testing.T) {
	router, _ := newOpenWorkHandlerTestRouter(t, "openwork_handler_start")
	tenantUUID := "00000000-0000-0000-0000-000000000101"

	payload := map[string]any{
		"template_id":     "dk001",
		"template_secret": "suite-secret-001",
		"template_ticket": "ticket-001",
		"provider_corpid": "ww-provider-001",
		"provider_secret": "provider-secret-001",
		"state":           "state-001",
	}
	resp := doJSON(t, router, http.MethodPost, "/admin/social/openwork/wecom/authorize/start?tenant_uuid="+tenantUUID, payload)
	require.Equal(t, http.StatusOK, resp.Code)
	data := mustDataMap(t, resp.Body.Bytes())
	require.Equal(t, "delegated_template", data["auth_mode"])
	require.Equal(t, "dk001", data["template_id"])
	require.Equal(t, "https://open.work.weixin.qq.com/3rdapp/install?suite_id=dk001", data["authorize_url"])
}

func TestOpenWorkFoundationHandler_CompleteAuthorizationAndListBindings(t *testing.T) {
	router, db := newOpenWorkHandlerTestRouter(t, "openwork_handler_complete")
	tenantUUID := "00000000-0000-0000-0000-000000000102"
	accountUUID := "11111111-1111-4111-8111-111111111102"
	require.NoError(t, db.Create(&model.ChannelAccount{
		AccountUUID:     accountUUID,
		TenantUuid:      tenantUUID,
		ChannelCode:     "wechat",
		AppType:         "wecom",
		AccountID:       "1000002",
		DisplayName:     "WeCom A",
		Status:          model.ChannelAccountStatusConnected,
		OwnerMemberUUID: "owner-001",
		Capabilities:    datatypes.JSONMap{},
		Credentials:     datatypes.JSONMap{},
	}).Error)

	payload := map[string]any{
		"template_id":          "dk001",
		"template_secret":      "suite-secret-001",
		"template_ticket":      "ticket-001",
		"provider_corpid":      "ww-provider-001",
		"provider_secret":      "provider-secret-001",
		"auth_code":            "auth-code-001",
		"channel_account_uuid": accountUUID,
		"set_default":          true,
	}
	resp := doJSON(t, router, http.MethodPost, "/admin/social/openwork/wecom/authorize/complete?tenant_uuid="+tenantUUID, payload)
	require.Equal(t, http.StatusOK, resp.Code)

	listReq := httptest.NewRequest(http.MethodGet, "/admin/social/openwork/wecom/bindings?tenant_uuid="+tenantUUID, nil)
	listRec := httptest.NewRecorder()
	router.ServeHTTP(listRec, listReq)
	require.Equal(t, http.StatusOK, listRec.Code)
	data := mustDataMap(t, listRec.Body.Bytes())
	items := data["items"].([]any)
	require.Len(t, items, 1)
	item := items[0].(map[string]any)
	require.Equal(t, "wwcorp001", item["corp_id"])
	require.Equal(t, true, item["is_default"])
}

func TestOpenWorkFoundationHandler_SetDefaultAndCreateSyncJob(t *testing.T) {
	router, db := newOpenWorkHandlerTestRouter(t, "openwork_handler_default_switch")
	tenantUUID := "00000000-0000-0000-0000-000000000103"

	bindingA := &model.WeComOpenAuthBinding{
		BindingUUID: "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa",
		TenantUUID:  tenantUUID,
		ChannelCode: "wechat",
		AppType:     "wecom",
		SuiteID:     "suite-001",
		CorpID:      "corp-a",
		AgentID:     "10001",
		Status:      model.WeComAuthBindingStatusActive,
		IsDefault:   true,
	}
	bindingB := &model.WeComOpenAuthBinding{
		BindingUUID: "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb",
		TenantUUID:  tenantUUID,
		ChannelCode: "wechat",
		AppType:     "wecom",
		SuiteID:     "suite-001",
		CorpID:      "corp-b",
		AgentID:     "10002",
		Status:      model.WeComAuthBindingStatusActive,
		IsDefault:   false,
	}
	require.NoError(t, db.Create(bindingA).Error)
	require.NoError(t, db.Create(bindingB).Error)

	switchResp := doJSON(t, router, http.MethodPost, "/admin/social/openwork/wecom/bindings/"+bindingB.BindingUUID+"/default?tenant_uuid="+tenantUUID, map[string]any{})
	require.Equal(t, http.StatusOK, switchResp.Code)

	jobPayload := map[string]any{
		"domain":      "org",
		"mode":        "bootstrap",
		"max_retries": 2,
	}
	jobResp := doJSON(t, router, http.MethodPost, "/admin/social/openwork/wecom/sync/jobs?tenant_uuid="+tenantUUID, jobPayload)
	require.Equal(t, http.StatusOK, jobResp.Code)
	jobData := mustDataMap(t, jobResp.Body.Bytes())
	require.Equal(t, bindingB.BindingUUID, jobData["binding_uuid"])
}

func TestOpenWorkFoundationHandler_GetAuthorizationStatus(t *testing.T) {
	router, db := newOpenWorkHandlerTestRouter(t, "openwork_handler_auth_status")
	tenantUUID := "00000000-0000-0000-0000-000000000106"
	require.NoError(t, db.Create(&model.WeComOpenAuthBinding{
		BindingUUID: "f6f9a130-8ba5-41fb-8a64-fad26fd15f8b",
		TenantUUID:  tenantUUID,
		ChannelAccountUUID: "acc-106",
		ChannelCode: "wechat",
		AppType:     "wecom",
		SuiteID:     "suite-001",
		CorpID:      "corp-auth",
		AgentID:     "10088",
		Status:      model.WeComAuthBindingStatusActive,
		IsDefault:   true,
	}).Error)
	require.NoError(t, db.Exec(`
		INSERT INTO social_channel_accounts (
			account_uuid, tenant_uuid, channel_code, app_type, account_id, display_name, status, owner_member_uuid
		) VALUES (?, ?, 'wechat', 'wecom', ?, ?, 'connected', ?)
	`, "acc-106", tenantUUID, "corp-auth", "Corp Auth", "1001").Error)

	req := httptest.NewRequest(
		http.MethodGet,
		"/admin/social/openwork/wecom/authorize/status?tenant_uuid="+tenantUUID+"&template_id=suite-001&state=state-001&started_at="+strconv.FormatInt(time.Now().Add(-60*time.Second).Unix(), 10),
		nil,
	)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	data := mustDataMap(t, rec.Body.Bytes())
	require.Equal(t, "authorized", data["status"])
	require.Equal(t, "state-001", data["state"])
}

func TestOpenWorkFoundationHandler_ReplayConflictAndDashboard(t *testing.T) {
	router, db := newOpenWorkHandlerTestRouter(t, "openwork_handler_replay_dashboard")
	tenantUUID := "00000000-0000-0000-0000-000000000104"
	binding := &model.WeComOpenAuthBinding{
		BindingUUID: "dddddddd-dddd-4ddd-8ddd-dddddddddddd",
		TenantUUID:  tenantUUID,
		ChannelCode: "wechat",
		AppType:     "wecom",
		SuiteID:     "suite-001",
		CorpID:      "corp-d",
		AgentID:     "10003",
		Status:      model.WeComAuthBindingStatusActive,
		IsDefault:   true,
	}
	require.NoError(t, db.Create(binding).Error)

	jobResp := doJSON(t, router, http.MethodPost, "/admin/social/openwork/wecom/sync/jobs?tenant_uuid="+tenantUUID, map[string]any{
		"domain":      "external_contacts",
		"mode":        "pushback",
		"max_retries": 1,
	})
	require.Equal(t, http.StatusOK, jobResp.Code)

	conflictReq := httptest.NewRequest(http.MethodGet, "/admin/social/openwork/wecom/sync/conflicts?tenant_uuid="+tenantUUID, nil)
	conflictRec := httptest.NewRecorder()
	router.ServeHTTP(conflictRec, conflictReq)
	require.Equal(t, http.StatusOK, conflictRec.Code)
	conflictData := mustDataMap(t, conflictRec.Body.Bytes())
	conflicts := conflictData["items"].([]any)
	require.Len(t, conflicts, 1)
	conflictUUID := conflicts[0].(map[string]any)["conflict_uuid"].(string)

	replayResp := doJSON(t, router, http.MethodPost, "/admin/social/openwork/wecom/sync/conflicts/"+conflictUUID+"/replay?tenant_uuid="+tenantUUID, map[string]any{
		"note": "manual replay",
	})
	require.Equal(t, http.StatusOK, replayResp.Code)

	dashboardReq := httptest.NewRequest(http.MethodGet, "/admin/social/openwork/wecom/sync/dashboard?tenant_uuid="+tenantUUID, nil)
	dashboardRec := httptest.NewRecorder()
	router.ServeHTTP(dashboardRec, dashboardReq)
	require.Equal(t, http.StatusOK, dashboardRec.Code)
	dashboardData := mustDataMap(t, dashboardRec.Body.Bytes())
	require.EqualValues(t, 0, dashboardData["open_conflicts"])
	jobs := dashboardData["jobs"].(map[string]any)
	require.EqualValues(t, 1, jobs["dead_letter"])
}

func TestOpenWorkFoundationHandler_MissingTenantUnauthorized(t *testing.T) {
	router, _ := newOpenWorkHandlerTestRouter(t, "openwork_handler_no_tenant")
	resp := doJSON(t, router, http.MethodPost, "/admin/social/openwork/wecom/authorize/start", map[string]any{
		"template_id":     "suite-001",
		"template_secret": "suite-secret-001",
		"template_ticket": "ticket-001",
	})
	require.Equal(t, http.StatusUnauthorized, resp.Code)
}

func newOpenWorkHandlerTestRouter(t *testing.T, name string) (*gin.Engine, *gorm.DB) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	basemodels.ForceSchemaForTests("")
	db, err := gorm.Open(sqlite.Open("file:"+name+"?mode=memory&cache=shared"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	require.NoError(t, err)
	require.NoError(t, ensureOpenWorkFoundationHandlerTables(db))

	repo := socialrepo.NewOpenWorkFoundationRepository(db)
	accountRepo := socialrepo.NewAccountRepository(db)
	svc := socialsvc.NewOpenWorkFoundationService(repo, accountRepo).SetHTTPClient(&http.Client{
		Transport: httpFakeRoundTripper{},
		Timeout:   5 * time.Second,
	})
	handler := NewOpenWorkFoundationHandler(svc)

	r := gin.New()
	group := r.Group("/admin/social", httpmw.EnsureTenant())
	{
		group.POST("/openwork/wecom/authorize/start", handler.StartAuthorization)
		group.POST("/openwork/wecom/authorize/complete", handler.CompleteAuthorization)
		group.GET("/openwork/wecom/authorize/status", handler.GetAuthorizationStatus)
		group.GET("/openwork/wecom/bindings", handler.ListBindings)
		group.POST("/openwork/wecom/bindings/:binding_uuid/default", handler.SetDefaultBinding)
		group.POST("/openwork/wecom/sync/jobs", handler.CreateSyncJob)
		group.GET("/openwork/wecom/sync/conflicts", handler.ListSyncConflicts)
		group.POST("/openwork/wecom/sync/conflicts/:conflict_uuid/replay", handler.ReplaySyncConflict)
		group.GET("/openwork/wecom/sync/dashboard", handler.GetDashboard)
	}
	return r, db
}

func ensureOpenWorkFoundationHandlerTables(db *gorm.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS social_channel_accounts (
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
		);`,
		`CREATE TABLE IF NOT EXISTS social_wecom_auth_bindings (
			binding_uuid TEXT PRIMARY KEY,
			tenant_uuid TEXT NOT NULL,
			channel_account_uuid TEXT,
			channel_code TEXT NOT NULL,
			app_type TEXT NOT NULL,
			suite_id TEXT NOT NULL,
			corp_id TEXT NOT NULL,
			agent_id TEXT NOT NULL,
			corp_name TEXT NOT NULL DEFAULT '',
			permanent_code TEXT NOT NULL DEFAULT '',
			suite_access_token TEXT NOT NULL DEFAULT '',
			suite_ticket TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL,
			is_default BOOLEAN NOT NULL DEFAULT FALSE,
			default_switched_at DATETIME,
			last_event_type TEXT NOT NULL DEFAULT '',
			last_event_at DATETIME,
			auth_scope TEXT,
			metadata TEXT,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		);`,
		`CREATE UNIQUE INDEX IF NOT EXISTS uq_social_wecom_auth_binding_identity
			ON social_wecom_auth_bindings (tenant_uuid, corp_id, agent_id);`,
		`CREATE TABLE IF NOT EXISTS social_wecom_auth_events (
			event_uuid TEXT PRIMARY KEY,
			tenant_uuid TEXT NOT NULL,
			suite_id TEXT NOT NULL,
			event_type TEXT NOT NULL,
			corp_id TEXT NOT NULL DEFAULT '',
			agent_id TEXT NOT NULL DEFAULT '',
			event_time DATETIME,
			event_key TEXT NOT NULL,
			payload TEXT,
			created_at DATETIME
		);`,
		`CREATE UNIQUE INDEX IF NOT EXISTS uq_social_wecom_auth_event_key
			ON social_wecom_auth_events (event_key);`,
		`CREATE TABLE IF NOT EXISTS social_sync_baseline_jobs (
			job_uuid TEXT PRIMARY KEY,
			tenant_uuid TEXT NOT NULL,
			binding_uuid TEXT NOT NULL,
			domain TEXT NOT NULL,
			mode TEXT NOT NULL,
			status TEXT NOT NULL,
			idempotency_key TEXT NOT NULL DEFAULT '',
			resolve_source TEXT NOT NULL DEFAULT 'default',
			total_count INTEGER NOT NULL DEFAULT 0,
			success_count INTEGER NOT NULL DEFAULT 0,
			failed_count INTEGER NOT NULL DEFAULT 0,
			conflict_count INTEGER NOT NULL DEFAULT 0,
			retry_count INTEGER NOT NULL DEFAULT 0,
			max_retries INTEGER NOT NULL DEFAULT 3,
			last_error TEXT NOT NULL DEFAULT '',
			next_retry_at DATETIME,
			dead_letter_at DATETIME,
			started_at DATETIME,
			finished_at DATETIME,
			write_back_fields TEXT,
			context TEXT,
			created_at DATETIME,
			updated_at DATETIME
		);`,
		`CREATE TABLE IF NOT EXISTS social_sync_conflict_records (
			conflict_uuid TEXT PRIMARY KEY,
			tenant_uuid TEXT NOT NULL,
			job_uuid TEXT NOT NULL,
			binding_uuid TEXT NOT NULL,
			domain TEXT NOT NULL,
			conflict_key TEXT NOT NULL,
			status TEXT NOT NULL,
			reason TEXT NOT NULL DEFAULT '',
			external_version TEXT NOT NULL DEFAULT '',
			local_version TEXT NOT NULL DEFAULT '',
			resolution_note TEXT NOT NULL DEFAULT '',
			replay_count INTEGER NOT NULL DEFAULT 0,
			last_replayed_at DATETIME,
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

func doJSON(t *testing.T, router *gin.Engine, method, path string, payload map[string]any) *httptest.ResponseRecorder {
	t.Helper()
	var body io.Reader
	if payload != nil {
		raw, err := json.Marshal(payload)
		require.NoError(t, err)
		body = bytes.NewReader(raw)
	}
	req := httptest.NewRequest(method, path, body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func mustDataMap(t *testing.T, raw []byte) map[string]any {
	t.Helper()
	var envelope map[string]any
	require.NoError(t, json.Unmarshal(raw, &envelope))
	require.Equal(t, true, envelope["success"])
	data, ok := envelope["data"].(map[string]any)
	require.True(t, ok)
	return data
}
