package social_channel_governance

import (
	"context"
	"sync"
	"testing"
	"time"

	basemodels "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	model "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/social_channel_governance"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func openCallbackTaskRepoDB(t *testing.T, name string) *gorm.DB {
	t.Helper()
	basemodels.ForceSchemaForTests("")
	db, err := gorm.Open(sqlite.Open("file:"+name+"?mode=memory&cache=shared"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	require.NoError(t, err)
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS social_wecom_callback_tasks (
			task_uuid TEXT PRIMARY KEY,
			tenant_uuid TEXT NOT NULL,
			suite_id TEXT NOT NULL,
			event_type TEXT NOT NULL,
			callback_key TEXT NOT NULL,
			event_key TEXT NOT NULL DEFAULT '',
			auth_code TEXT NOT NULL DEFAULT '',
			corp_id TEXT NOT NULL DEFAULT '',
			agent_id TEXT NOT NULL DEFAULT '',
			suite_ticket TEXT NOT NULL DEFAULT '',
			state TEXT NOT NULL DEFAULT '',
			msg_signature TEXT NOT NULL DEFAULT '',
			timestamp INTEGER NOT NULL DEFAULT 0,
			nonce TEXT NOT NULL DEFAULT '',
			event_time DATETIME,
			status TEXT NOT NULL DEFAULT 'received',
			idempotent_hit BOOLEAN NOT NULL DEFAULT FALSE,
			attempt_count INTEGER NOT NULL DEFAULT 0,
			max_attempts INTEGER NOT NULL DEFAULT 3,
			last_error TEXT NOT NULL DEFAULT '',
			next_retry_at DATETIME,
			processing_started_at DATETIME,
			finished_at DATETIME,
			payload TEXT,
			created_at DATETIME,
			updated_at DATETIME
		);`,
		`CREATE UNIQUE INDEX IF NOT EXISTS uq_social_wecom_callback_task_key
			ON social_wecom_callback_tasks (tenant_uuid, suite_id, callback_key);`,
	}
	for _, stmt := range stmts {
		require.NoError(t, db.Exec(stmt).Error)
	}
	return db
}

func TestOpenWorkFoundationRepository_EnqueueCallbackTask_Idempotent(t *testing.T) {
	db := openCallbackTaskRepoDB(t, "openwork_callback_task_enqueue")
	repo := NewOpenWorkFoundationRepository(db)

	task := &model.WeComOpenCallbackTask{
		TenantUUID:  "00000000-0000-0000-0000-000000000001",
		SuiteID:     "suite-001",
		EventType:   "create_auth",
		CallbackKey: "auth_code:abc",
		EventKey:    "event:abc",
		AuthCode:    "abc",
		Status:      model.OpenWorkCallbackTaskReceived,
	}
	first, created, err := repo.EnqueueCallbackTask(context.Background(), task)
	require.NoError(t, err)
	require.True(t, created)
	require.NotEmpty(t, first.TaskUUID)

	second, created, err := repo.EnqueueCallbackTask(context.Background(), &model.WeComOpenCallbackTask{
		TenantUUID:  task.TenantUUID,
		SuiteID:     task.SuiteID,
		EventType:   task.EventType,
		CallbackKey: task.CallbackKey,
		EventKey:    "event:dup",
		AuthCode:    "abc",
	})
	require.NoError(t, err)
	require.False(t, created)
	require.Equal(t, first.TaskUUID, second.TaskUUID)
}

func TestOpenWorkFoundationRepository_ClaimNextCallbackTask_Concurrent(t *testing.T) {
	db := openCallbackTaskRepoDB(t, "openwork_callback_task_claim")
	repo := NewOpenWorkFoundationRepository(db)
	tenantUUID := "00000000-0000-0000-0000-000000000002"

	_, _, err := repo.EnqueueCallbackTask(context.Background(), &model.WeComOpenCallbackTask{
		TenantUUID:  tenantUUID,
		SuiteID:     "suite-001",
		EventType:   "create_auth",
		CallbackKey: "auth_code:claim",
		EventKey:    "event:claim",
		AuthCode:    "claim",
		Status:      model.OpenWorkCallbackTaskReceived,
	})
	require.NoError(t, err)

	var wg sync.WaitGroup
	wg.Add(2)
	hits := make(chan string, 2)
	claim := func() {
		defer wg.Done()
		task, ok, claimErr := repo.ClaimNextCallbackTask(context.Background())
		require.NoError(t, claimErr)
		if ok && task != nil {
			hits <- task.TaskUUID
		}
	}
	go claim()
	go claim()
	wg.Wait()
	close(hits)

	var claimed []string
	for id := range hits {
		claimed = append(claimed, id)
	}
	require.Len(t, claimed, 1)
}

func TestOpenWorkFoundationRepository_ClaimFailedTaskAfterBackoff(t *testing.T) {
	db := openCallbackTaskRepoDB(t, "openwork_callback_task_retry")
	repo := NewOpenWorkFoundationRepository(db)
	tenantUUID := "00000000-0000-0000-0000-000000000003"

	task, _, err := repo.EnqueueCallbackTask(context.Background(), &model.WeComOpenCallbackTask{
		TenantUUID:  tenantUUID,
		SuiteID:     "suite-001",
		EventType:   "create_auth",
		CallbackKey: "auth_code:retry",
		EventKey:    "event:retry",
		AuthCode:    "retry",
		Status:      model.OpenWorkCallbackTaskFailed,
		NextRetryAt: ptrTime(time.Now().UTC().Add(-1 * time.Second)),
		MaxAttempts: 3,
	})
	require.NoError(t, err)

	claimed, ok, err := repo.ClaimNextCallbackTask(context.Background())
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, task.TaskUUID, claimed.TaskUUID)
	require.Equal(t, model.OpenWorkCallbackTaskProcessing, claimed.Status)
}

func ptrTime(in time.Time) *time.Time {
	return &in
}
