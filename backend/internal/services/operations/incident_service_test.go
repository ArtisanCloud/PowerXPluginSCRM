package operations_test

import (
	"context"
	"testing"
	"time"

	basemodels "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	opmodels "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/operations"
	oprepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/operations"
	opmetrics "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/observability/operations"
	operationsvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/operations"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type recordingIncidentDispatcher struct {
	events []string
}

func (r *recordingIncidentDispatcher) DispatchIncidentEvent(ctx context.Context, eventType string, incident *opmodels.Incident, payload map[string]any) error {
	r.events = append(r.events, eventType)
	return nil
}

func TestIncidentService_Lifecycle(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:incident_service?mode=memory&cache=shared"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	require.NoError(t, err)
	basemodels.ForceSchemaForTests("")
	require.NoError(t, db.AutoMigrate(
		&opmodels.Incident{},
		&opmodels.IncidentTimelineEntry{},
		&opmodels.IncidentChecklistItem{},
		&opmodels.ReadinessChecklistItem{},
	))

	repo := oprepo.NewIncidentRepository(db)
	dispatcher := &recordingIncidentDispatcher{}
	svc := operationsvc.NewIncidentService(repo, nil, opmetrics.NewMetrics(), dispatcher)

	ctx := context.Background()
	incident, err := svc.CreateIncident(ctx, operationsvc.CreateIncidentRequest{
		Severity:        "sev1",
		DetectionSource: "monitoring",
		Summary:         "API latency spike",
		Impact:          map[string]any{"apis": []string{"billing"}},
		Labels:          map[string]bool{"availability": true},
	})
	require.NoError(t, err)
	require.NotEmpty(t, incident.ID)
	require.Contains(t, dispatcher.events, "operations.incident.created")

	updateAt := time.Now().Add(30 * time.Minute)
	updated, err := svc.UpdateIncident(ctx, incident.ID, operationsvc.UpdateIncidentRequest{
		Status:       ptrString(operationsvc.IncidentStatusAcknowledged),
		Mitigation:   ptrString("rerouted traffic"),
		NextUpdateAt: &updateAt,
	})
	require.NoError(t, err)
	require.Equal(t, operationsvc.IncidentStatusAcknowledged, updated.Status)
	require.Contains(t, dispatcher.events, "operations.incident.status_changed")

	timeline, err := svc.AppendTimeline(ctx, incident.ID, operationsvc.TimelineEntryRequest{
		EntryType:          "announcement",
		Message:            "Issue acknowledged",
		StakeholderChannel: "support_hub",
	})
	require.NoError(t, err)
	require.Equal(t, "announcement", timeline.EntryType)

	resp, err := svc.GetIncident(ctx, incident.ID)
	require.NoError(t, err)
	require.Len(t, resp.Timeline, 1)
	require.Equal(t, incident.ID, resp.Incident.ID)
	require.True(t, resp.ChecklistStatus.IncidentReady)
	require.False(t, resp.ChecklistStatus.SupportReady)
}

func ptrString(v string) *string {
	return &v
}
