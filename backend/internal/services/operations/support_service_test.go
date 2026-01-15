package operations_test

import (
	"context"
	"testing"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/config"
	basemodels "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	opmodels "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/operations"
	oprepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/operations"
	opmetrics "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/observability/operations"
	operationsvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/operations"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type recordingDispatcher struct {
	events []string
}

func (r *recordingDispatcher) DispatchSupportEvent(ctx context.Context, tenantID, eventType string, payload map[string]any) error {
	r.events = append(r.events, eventType)
	return nil
}

func TestSupportService_ConfigurePlaybookAndMetrics(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)
	basemodels.ForceSchemaForTests("")
	require.NoError(t, db.AutoMigrate(
		&opmodels.SupportChannel{},
		&opmodels.SupportTicket{},
		&opmodels.SupportTicketEvent{},
		&opmodels.ReadinessChecklistItem{},
	))

	repo := oprepo.NewSupportRepository(db)
	dispatcher := &recordingDispatcher{}
	metrics := opmetrics.NewMetrics()
	svc := operationsvc.NewSupportService(repo, &config.Config{}, metrics, dispatcher)

	ctx := context.Background()
	input := operationsvc.ConfigurePlaybookInput{
		Channels: []operationsvc.SupportChannelInput{
			{Channel: "marketplace_ticket", Address: "https://support.local/tickets", Escalates: []string{"agent", "engineer"}},
			{Channel: "vendor_email", Address: "vendor@example.com", Escalates: []string{"agent"}},
		},
		KnowledgeBase: []operationsvc.KnowledgeBaseDoc{
			{
				Label: "README",
				URL:   "https://docs.local/readme",
			},
		},
	}

	payload, err := svc.ConfigurePlaybook(ctx, input)
	require.NoError(t, err)
	require.Len(t, payload.Channels, 2)
	require.Len(t, payload.KnowledgeBase, 1)
	require.Equal(t, "README", payload.KnowledgeBase[0].Label)

	// Ensure readiness items are marked completed.
	var completed int
	for _, item := range payload.Readiness {
		if item.Completed {
			completed++
		}
	}
	require.GreaterOrEqual(t, completed, 1)

	// Create support ticket and ensure webhook dispatch + metrics
	ticket, err := svc.CreateTicket(ctx, operationsvc.CreateTicketRequest{
		TenantUuid: "tenant-1",
		Subject:  "Login issue",
		Priority: "P1",
		RequestedBy: map[string]any{
			"name": "Alice",
		},
	})
	require.NoError(t, err)
	require.NotEmpty(t, ticket.ID)
	require.Contains(t, dispatcher.events, "operations.support.ticket.created")

	// Simulate resolution to compute metrics
	now := time.Now().Add(2 * time.Hour)
	ticket.FirstResponseAt = ptrTime(ticket.CreatedAt.Add(1 * time.Hour))
	ticket.ResolvedAt = &now
	score := 4.5
	ticket.CSATScore = &score
	require.NoError(t, repo.UpdateTicket(ctx, ticket))

	metricsResp, err := svc.ComputeMetrics(ctx)
	require.NoError(t, err)
	require.InEpsilon(t, 1.0, metricsResp.FirstResponseHours, 0.01)
	require.InEpsilon(t, 2.0, metricsResp.ResolutionHours, 0.01)
	require.InEpsilon(t, 4.5, metricsResp.CSATAverage, 0.01)
	require.InEpsilon(t, 1.0, metricsResp.ResolutionRate, 0.01)
}

func ptrTime(t time.Time) *time.Time {
	return &t
}
