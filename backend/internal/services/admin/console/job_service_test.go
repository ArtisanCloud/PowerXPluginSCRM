package console_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	model "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/admin_console"
	consolesvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/console"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/config"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	adminmetrics "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/observability/admin_console"
)

type stubLocker struct {
	held map[string]time.Time
}

func newStubLocker() *stubLocker {
	return &stubLocker{held: make(map[string]time.Time)}
}

func (s *stubLocker) TryLock(_ context.Context, key string, ttl time.Duration) (bool, error) {
	if expiry, ok := s.held[key]; ok {
		if time.Now().Before(expiry) {
			return false, nil
		}
	}
	s.held[key] = time.Now().Add(ttl)
	return true, nil
}

func (s *stubLocker) Unlock(_ context.Context, key string) error {
	delete(s.held, key)
	return nil
}

func setupJobService(t *testing.T) (*consolesvc.JobService, *gorm.DB, *stubLocker) {
	t.Helper()
	dsn := fmt.Sprintf("file:admin_console_job_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	require.NoError(t, err)
	models.ForceSchemaForTests("")

	require.NoError(t, db.Exec(`CREATE TABLE IF NOT EXISTS admin_console_job_runs (
		id TEXT PRIMARY KEY,
		plugin_id TEXT NOT NULL,
		tenant_uuid TEXT,
		environment TEXT,
		job_type TEXT NOT NULL,
		trigger_source TEXT NOT NULL,
		status TEXT NOT NULL,
		action TEXT,
		scope_type TEXT,
		scope_ref TEXT,
		target_id TEXT,
		reason TEXT,
		dry_run BOOLEAN,
		metadata TEXT,
		started_at DATETIME,
		finished_at DATETIME,
		duration_ms INTEGER,
		message TEXT,
		retry_of TEXT,
		audit_event_id TEXT,
		created_by TEXT NOT NULL,
		created_at DATETIME,
		updated_at DATETIME
	);`).Error)

	cfg := &config.Config{
		AdminConsole: &config.AdminConsoleConfig{
			JobHistoryDays: 45,
			Troubleshooting: config.AdminConsoleTroubleshooting{
				RefreshIntervalSeconds: 300,
				CacheTTLSeconds:        120,
			},
			SafeOps: config.AdminConsoleSafeOps{
				LockTTLSeconds:   60,
				MaxConcurrentOps: 2,
			},
		},
	}
	locker := newStubLocker()
	deps := &app.Deps{
		DB:                  db,
		Config:              cfg,
		AdminConsoleMetrics: adminmetrics.NewMetrics(),
	}
	svc := consolesvc.NewJobService(deps, consolesvc.WithLocker(locker))
	return svc, db, locker
}

func TestJobService_ScheduleSafeOpPersistsRun(t *testing.T) {
	svc, db, _ := setupJobService(t)
	input := consolesvc.ScheduleSafeOpInput{
		TenantUuid:    strPtr("tenant-1"),
		Environment:   "production",
		JobType:       consolesvc.JobTypeWebhookReplay,
		TriggerSource: consolesvc.TriggerSourceManual,
		Action:        consolesvc.SafeOpActionReplay,
		ScopeType:     consolesvc.SafeOpScopeTenant,
		ScopeRef:      "tenant-1",
		TargetID:      "hook-evt-123",
		Reason:        "replay failing webhook",
		Actor: consolesvc.Actor{
			ID:             "user:1",
			Name:           "Ops Admin",
			PermissionCode: "operations.plugin.ops",
		},
	}

	run, err := svc.ScheduleSafeOp(context.Background(), input)
	require.NoError(t, err)
	require.NotNil(t, run)
	require.Equal(t, consolesvc.JobStatusPending, run.Status)
	require.Equal(t, consolesvc.SafeOpActionReplay, run.SafeOp.Action)
	require.Equal(t, "tenant-1", run.SafeOp.ScopeRef)

	var count int64
	require.NoError(t, db.Model(&model.JobRun{}).Count(&count).Error)
	require.Equal(t, int64(1), count)
}

func TestJobService_RetryEligibility(t *testing.T) {
	svc, db, _ := setupJobService(t)
	failed := &model.JobRun{
		ID:            "run-failed",
		PluginID:      app.PluginID,
		TenantUuid:    strPtr("tenant-1"),
		Environment:   "staging",
		JobType:       string(consolesvc.JobTypeWebhookReplay),
		TriggerSource: string(consolesvc.TriggerSourceManual),
		Status:        string(consolesvc.JobStatusFailed),
		Action:        string(consolesvc.SafeOpActionReplay),
		ScopeType:     string(consolesvc.SafeOpScopeTenant),
		ScopeRef:      strPtr("tenant-1"),
		CreatedBy:     "user:1",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	require.NoError(t, db.Create(failed).Error)

	ret, err := svc.RetryRun(context.Background(), consolesvc.RetryRunInput{
		RunID:      failed.ID,
		TenantUuid: strPtr("tenant-1"),
		Actor: consolesvc.Actor{
			ID:             "user:2",
			Name:           "Retry Ops",
			PermissionCode: "operations.plugin.ops",
		},
	})
	require.NoError(t, err)
	require.NotNil(t, ret)
	require.Equal(t, failed.ID, ret.RetryOf)
	require.Equal(t, consolesvc.JobStatusPending, ret.Status)

	succeeded := &model.JobRun{
		ID:            "run-success",
		PluginID:      app.PluginID,
		JobType:       string(consolesvc.JobTypeWebhookReplay),
		TriggerSource: string(consolesvc.TriggerSourceManual),
		Status:        string(consolesvc.JobStatusSucceeded),
		Action:        string(consolesvc.SafeOpActionReplay),
		ScopeType:     string(consolesvc.SafeOpScopeTenant),
		ScopeRef:      strPtr("tenant-1"),
		CreatedBy:     "user:1",
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	require.NoError(t, db.Create(succeeded).Error)

	_, err = svc.RetryRun(context.Background(), consolesvc.RetryRunInput{
		RunID: succeeded.ID,
		Actor: consolesvc.Actor{
			ID:             "user:2",
			PermissionCode: "operations.plugin.ops",
		},
	})
	require.ErrorIs(t, err, consolesvc.ErrRetryNotAllowed)
}

func TestJobService_AdvisoryLockCollision(t *testing.T) {
	svc, _, locker := setupJobService(t)
	input := consolesvc.ScheduleSafeOpInput{
		Environment:   "production",
		JobType:       consolesvc.JobTypeWebhookReplay,
		TriggerSource: consolesvc.TriggerSourceManual,
		Action:        consolesvc.SafeOpActionReplay,
		ScopeType:     consolesvc.SafeOpScopeTenant,
		ScopeRef:      "tenant-2",
		TargetID:      "hook-evt-999",
		Actor: consolesvc.Actor{
			ID:             "user:3",
			PermissionCode: "operations.plugin.ops",
		},
	}

	first, err := svc.ScheduleSafeOp(context.Background(), input)
	require.NoError(t, err)
	require.NotNil(t, first)

	_, err = svc.ScheduleSafeOp(context.Background(), input)
	require.ErrorIs(t, err, consolesvc.ErrOperationInProgress)

	require.Contains(t, locker.held, consolesvc.LockKey(input))

	require.NoError(t, svc.UpdateRunStatus(context.Background(), consolesvc.UpdateRunStatusInput{
		RunID:   first.ID,
		Status:  consolesvc.JobStatusSucceeded,
		Message: "completed",
	}))

	_, err = svc.ScheduleSafeOp(context.Background(), input)
	require.NoError(t, err)
}

func strPtr(v string) *string {
	return &v
}
