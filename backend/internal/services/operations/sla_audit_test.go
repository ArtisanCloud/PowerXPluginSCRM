package operations_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/config"
	basemodels "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models"
	opmodels "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/operations"
	oprepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/operations"
	opmetrics "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/observability/operations"
	operationsvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/operations"
	operationsjobs "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/operations/jobs"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestSLARecomputeJobWritesAudit(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:sla_audit?mode=memory&cache=shared"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	require.NoError(t, err)
	basemodels.ForceSchemaForTests("")
	require.NoError(t, db.AutoMigrate(
		&opmodels.SLAProfile{},
		&opmodels.SLAAdjustment{},
		&opmodels.ReadinessChecklistItem{},
	))

	repo := oprepo.NewSLARepository(db)
	svc := operationsvc.NewSLAService(repo, &config.Config{}, opmetrics.NewMetrics())

	_, err = svc.UpsertTargets(context.Background(), operationsvc.ProfileTargets{
		PlanType:              "real_time",
		UptimeTarget:          99.9,
		ResponseTargetMs:      500,
		SuccessTargetPct:      99.5,
		SupportFrtTargetHours: 4,
	})
	require.NoError(t, err)

	_, err = svc.UpdateActuals(context.Background(), "real_time", operationsvc.ActualMetrics{
		UptimeActual:          99.95,
		ResponseActualMs:      300,
		SuccessActualPct:      99.8,
		SupportFrtActualHours: 2,
	})
	require.NoError(t, err)

	buffer := &bytes.Buffer{}
	logger := logrus.New()
	logger.SetOutput(buffer)
	job := operationsjobs.NewSLARecomputeJob(svc, logger.WithField("component", "operations.sla_recompute_job"))

	require.NoError(t, job.Run(context.Background()))
	adjustments, err := repo.ListAdjustments(context.Background(), app.PluginID, "real_time", 5)
	require.NoError(t, err)
	require.NotEmpty(t, adjustments)
	require.Contains(t, buffer.String(), "operations.sla_recompute_job")
}
