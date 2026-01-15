package operations_test

import (
	"bytes"
	"context"
	"testing"

	opmodels "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/operations"
	operationsvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/operations"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestLoggingIncidentDispatcher_RecordsEvent(t *testing.T) {
	buffer := &bytes.Buffer{}
	logger := logrus.New()
	logger.SetOutput(buffer)
	dispatcher := operationsvc.NewLoggingIncidentDispatcher(logrus.NewEntry(logger))

	err := dispatcher.DispatchIncidentEvent(context.Background(), "operations.incident.created", &opmodels.Incident{
		ID:       "inc-123",
		Severity: operationsvc.IncidentSeveritySev1,
	}, map[string]any{"status": "detected"})
	require.NoError(t, err)

	output := buffer.String()
	require.Contains(t, output, "incident_id=inc-123")
	require.Contains(t, output, "event_type=operations.incident.created")
}
