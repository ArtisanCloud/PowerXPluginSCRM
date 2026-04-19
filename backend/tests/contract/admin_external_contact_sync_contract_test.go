package contract

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	leadrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/lead_capture"
	socialrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/social_channel_governance"
	authx "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/middleware"
	leadsvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/lead_capture"
	httplead "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/admin/lead_capture"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type contractExternalContactLeadAdapter struct{}

func (contractExternalContactLeadAdapter) FetchLeads(_ context.Context, _ leadsvc.TriggerSyncRequest, _ string) ([]leadsvc.WeComLeadRecord, error) {
	return []leadsvc.WeComLeadRecord{{
		ExternalLeadID: "ext-contact-001",
		CorpID:         "corp-contract-001",
		DisplayName:    "合同测试客户",
		Phone:          "13800000011",
		Email:          "contract-contact@example.com",
	}}, nil
}

func TestExternalContactSyncContract_CreateLeadAndCheckpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tenantUUID := "00000000-0000-0000-0000-000000000001"
	defaultAccountUUID := "11111111-1111-4111-8111-111111111111"

	db := openContractDB(t, "external_contact_sync_contract")
	require.NoError(t, ensureContractSyncFoundationTables(db))
	require.NoError(t, db.Exec(`
		INSERT INTO social_channel_accounts (
			account_uuid, tenant_uuid, channel_code, app_type, account_id, display_name,
			status, org_sync_default, owner_member_uuid, created_at, updated_at
		) VALUES (?, ?, 'wechat', 'wecom', 'contract-wecom', '合同测试账号', 'connected', 1, 'owner-001', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
	`, defaultAccountUUID, tenantUUID).Error)

	taskRepo := leadrepo.NewLeadSyncTaskRepository(db)
	leadRepo := leadrepo.NewLeadRepository(db)
	syncRepo := socialrepo.NewSyncFoundationRepository(db)
	svc := leadsvc.NewWeComSyncService(taskRepo, nil, nil).
		WithLeadIngestion(leadRepo, contractExternalContactLeadAdapter{}).
		WithLeadService(leadsvc.NewLeadService(leadRepo)).
		WithSyncFoundation(syncRepo)
	handler := httplead.NewWeComSyncHandler(svc)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("tenant_uuid", tenantUUID)
		authx.SetTenantContext(c, authx.TenantContext{TenantUUID: tenantUUID})
		c.Request = c.Request.WithContext(authx.ContextWithTenantUUID(c.Request.Context(), tenantUUID))
		c.Next()
	})
	r.POST("/api/v1/admin/leads/wecom/sync", handler.TriggerSync)

	reqBody := bytes.NewBufferString(`{
		"trace_id":"contract-us3-external-contact",
		"domain":"external_contacts",
		"direction":"pull",
		"mode":"incremental",
		"checkpoint_cursor":"cursor-contract-001"
	}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/leads/wecom/sync", reqBody)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var response map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	require.Equal(t, true, response["success"])

	require.Eventually(t, func() bool {
		cp, err := syncRepo.GetCheckpoint(context.Background(), tenantUUID, "external_contacts", "pull")
		return err == nil && cp != nil && cp.Cursor == "cursor-contract-001"
	}, 3*time.Second, 80*time.Millisecond)

	leads, err := leadRepo.List(context.Background(), tenantUUID)
	require.NoError(t, err)
	require.Len(t, leads, 1)
	require.Equal(t, "合同测试客户", leads[0].DisplayName)
}

func ensureContractSyncFoundationTables(db *gorm.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS social_sync_checkpoints (
			checkpoint_uuid TEXT PRIMARY KEY,
			tenant_uuid TEXT NOT NULL,
			domain TEXT NOT NULL,
			direction TEXT NOT NULL,
			cursor TEXT,
			snapshot_version TEXT,
			last_event_time DATETIME,
			created_at DATETIME,
			updated_at DATETIME
		);`,
		`CREATE UNIQUE INDEX IF NOT EXISTS uq_social_sync_checkpoints_scope
			ON social_sync_checkpoints (tenant_uuid, domain, direction);`,
		`CREATE TABLE IF NOT EXISTS social_sync_writeback_policies (
			policy_uuid TEXT PRIMARY KEY,
			tenant_uuid TEXT NOT NULL,
			domain TEXT NOT NULL,
			mapping_rules TEXT,
			protected_fields TEXT,
			overwrite_mode TEXT,
			enabled BOOLEAN NOT NULL DEFAULT 1,
			capability_status TEXT,
			created_at DATETIME,
			updated_at DATETIME
		);`,
		`CREATE UNIQUE INDEX IF NOT EXISTS uq_social_sync_writeback_policies_scope
			ON social_sync_writeback_policies (tenant_uuid, domain);`,
		`CREATE TABLE IF NOT EXISTS social_sync_jobs (
			job_uuid TEXT PRIMARY KEY,
			tenant_uuid TEXT NOT NULL,
			domain TEXT NOT NULL,
			direction TEXT,
			mode TEXT,
			status TEXT,
			attempt_no INTEGER,
			max_attempts INTEGER,
			idempotency_key TEXT,
			payload TEXT,
			error_code TEXT,
			error_message TEXT,
			started_at DATETIME,
			finished_at DATETIME,
			capability_status TEXT,
			created_at DATETIME,
			updated_at DATETIME
		);`,
		`CREATE TABLE IF NOT EXISTS social_sync_dead_letters (
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
		);`,
	}
	for _, stmt := range stmts {
		if err := db.Exec(stmt).Error; err != nil {
			return err
		}
	}
	return nil
}
