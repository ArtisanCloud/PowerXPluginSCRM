package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/config"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	model "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/integration"
	repo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/integration"
	service "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/integration"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupWebhookService(t *testing.T) (*service.WebhookService, context.Context) {
	t.Helper()

	models.ForceSchemaForTests("")

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS integration_webhook_subscriptions (
      id TEXT PRIMARY KEY,
      tenant_uuid TEXT NOT NULL,
      event_type TEXT NOT NULL,
      target_url TEXT NOT NULL,
      secret TEXT,
      retry_policy TEXT,
      status TEXT NOT NULL,
      metadata TEXT,
      created_at DATETIME,
      updated_at DATETIME,
      UNIQUE (tenant_uuid, event_type, target_url)
    )`,
		`CREATE TABLE IF NOT EXISTS integration_webhook_attempts (
		id TEXT PRIMARY KEY,
		subscription_id TEXT NOT NULL,
      envelope_id TEXT,
      status TEXT NOT NULL,
      retry_count INTEGER DEFAULT 0,
      last_error TEXT,
      next_delivery_at DATETIME,
      payload_snapshot TEXT,
      created_at DATETIME,
      updated_at DATETIME
    )`,
	}
	for _, stmt := range stmts {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("prepare table: %v", err)
		}
	}

	cfg := &config.Config{
		Server: &config.ServerConfig{
			SecretKey: "test-secret",
		},
	}

	subRepo := repo.NewWebhookSubscriptionRepository(db)
	attemptRepo := repo.NewDeliveryAttemptRepository(db)

	return service.NewWebhookService(cfg, subRepo, attemptRepo, nil), context.Background()
}

func TestWebhookService_ReplayAttemptFlow(t *testing.T) {
	svc, ctx := setupWebhookService(t)

	sub, err := svc.CreateSubscription(ctx, service.CreateSubscriptionParams{
		TenantUuid:  "42",
		EventType: "integration.dispatch",
		TargetURL: "https://example.org/webhook",
	})
	if err != nil {
		t.Fatalf("CreateSubscription failed: %v", err)
	}
	if sub == nil || sub.ID == "" {
		t.Fatalf("CreateSubscription returned empty subscription")
	}

	attempt, err := svc.RecordDeliveryAttempt(ctx, service.DeliveryResultParams{
		SubscriptionID: sub.ID,
		TenantUuid:       sub.TenantUuid,
		Status:         model.AttemptStatusDLQ,
		RetryCount:     2,
	})
	if err != nil {
		t.Fatalf("RecordDeliveryAttempt failed: %v", err)
	}
	if attempt.Status != model.AttemptStatusDLQ {
		t.Fatalf("expected attempt status DLQ, got %s", attempt.Status)
	}

	now := time.Now().UTC()
	if err := svc.UpdateAttemptStatus(ctx, attempt.ID, model.AttemptStatusPending, attempt.RetryCount, &now, "", sub.TenantUuid); err != nil {
		t.Fatalf("UpdateAttemptStatus failed: %v", err)
	}

	updated, err := svc.GetAttempt(ctx, attempt.ID)
	if err != nil {
		t.Fatalf("GetAttempt failed: %v", err)
	}
	if updated.Status != model.AttemptStatusPending {
		t.Fatalf("expected status to be PENDING, got %s", updated.Status)
	}
	if updated.NextDeliveryAt == nil {
		t.Fatalf("expected next_delivery_at to be set")
	}
	if updated.NextDeliveryAt.Before(now) {
		t.Fatalf("expected next_delivery_at >= replay timestamp, got %v", updated.NextDeliveryAt)
	}
}
