package runtime_ops

import (
	"context"
	"testing"
	"time"

	model "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/runtime_ops"
	authx "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/middleware"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestRecordBreachPersistsAuditEvent(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	createRuntimeOpsTables(t, db)

	svc := NewQuotaService(db, nil)

	ctx := authx.ContextWithTenantUuid(context.Background(), 42)
	if err := svc.RecordBreach(ctx, "plugin.demo", "tenant-1", "bootstrap", ActionThrottle); err != nil {
		t.Fatalf("RecordBreach failed: %v", err)
	}

	var count int64
	if err := db.Model(&model.RuntimeAuditEvent{}).Count(&count).Error; err != nil {
		t.Fatalf("count audit events: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 audit event, got %d", count)
	}
}

func TestRecordUsageAssignsID(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	createRuntimeOpsTables(t, db)

	svc := NewQuotaService(db, nil)
	entry := &model.QuotaLedger{
		ScopeType:      "tenant",
		ScopeRef:       "tenant-1",
		WindowStart:    time.Now().Add(-5 * time.Minute),
		WindowEnd:      time.Now(),
		TokensConsumed: 10,
	}

	result, err := svc.RecordUsage(context.Background(), entry)
	if err != nil {
		t.Fatalf("RecordUsage failed: %v", err)
	}
	if result.ID == "" {
		t.Fatalf("expected RecordUsage to populate ID")
	}
}

func createRuntimeOpsTables(t *testing.T, db *gorm.DB) {
	t.Helper()
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS runtime_audit_events (
      id TEXT PRIMARY KEY,
      plugin_id TEXT NOT NULL,
      tenant_uuid TEXT,
      event_type TEXT NOT NULL,
      payload TEXT,
      occurred_at DATETIME,
      created_at DATETIME
    )`,
		`CREATE TABLE IF NOT EXISTS quota_ledgers (
      id TEXT PRIMARY KEY,
      scope_type TEXT NOT NULL,
      scope_ref TEXT NOT NULL,
      window_start DATETIME NOT NULL,
      window_end DATETIME NOT NULL,
      tokens_consumed REAL DEFAULT 0,
      cpu_seconds REAL DEFAULT 0,
      bandwidth_mb REAL DEFAULT 0,
      invocations REAL DEFAULT 0,
      over_limit_action TEXT,
      reported_at DATETIME,
      created_at DATETIME
    )`,
		`CREATE TABLE IF NOT EXISTS marketplace_overages (
      id TEXT PRIMARY KEY,
      plugin_id TEXT NOT NULL,
      tenant_uuid TEXT,
      hour_window DATETIME NOT NULL,
      quota_metric TEXT NOT NULL,
      breach_count INTEGER DEFAULT 0,
      last_breach_at DATETIME,
      reported INTEGER DEFAULT 0,
      created_at DATETIME,
      updated_at DATETIME
    )`,
	}
	for _, stmt := range stmts {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("failed to prepare table: %v", err)
		}
	}
}

func TestDefaultReadinessBlueprintIncludesOperationsChecklists(t *testing.T) {
	blueprint := DefaultReadinessBlueprint()
	required := []ChecklistType{ChecklistSupportReady, ChecklistIncidentReady, ChecklistSLAReady}
	for _, key := range required {
		items, ok := blueprint[key]
		if !ok {
			t.Fatalf("missing readiness checklist %q", key)
		}
		if len(items) == 0 {
			t.Fatalf("readiness checklist %q should contain at least one item", key)
		}
	}
}
