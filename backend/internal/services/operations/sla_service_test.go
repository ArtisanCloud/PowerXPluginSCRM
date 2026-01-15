package operations_test

import (
	"context"
	"testing"

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

func setupSLAService(t *testing.T) (*operationsvc.SLAService, *oprepo.SLARepository) {
	db, err := gorm.Open(sqlite.Open("file:sla_service?mode=memory&cache=shared"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	require.NoError(t, err)
	basemodels.ForceSchemaForTests("")
	require.NoError(t, db.AutoMigrate(
		&opmodels.SLAProfile{},
		&opmodels.SLAAdjustment{},
		&opmodels.ReadinessChecklistItem{},
	))
	repo := oprepo.NewSLARepository(db)
	svc := operationsvc.NewSLAService(repo, &config.Config{}, opmetrics.NewMetrics())
	return svc, repo
}

func TestSLAServiceUpsertAndActuals(t *testing.T) {
	svc, repo := setupSLAService(t)
	ctx := context.Background()

	profile, err := svc.UpsertTargets(ctx, operationsvc.ProfileTargets{
		PlanType:              "real_time",
		UptimeTarget:          99.9,
		ResponseTargetMs:      500,
		SuccessTargetPct:      99.5,
		SupportFrtTargetHours: 4,
	})
	require.NoError(t, err)
	require.Equal(t, "real_time", profile.PlanType)
	require.Equal(t, 99.9, profile.UptimeTarget)

	updated, err := svc.UpdateActuals(ctx, "real_time", operationsvc.ActualMetrics{
		UptimeActual:          99.95,
		ResponseActualMs:      300,
		SuccessActualPct:      99.8,
		SupportFrtActualHours: 2,
	})
	require.NoError(t, err)
	require.GreaterOrEqual(t, updated.SLAScore, 85.0)

	items, err := svc.ChecklistSummary(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, items)

	adjustments, err := repo.ListAdjustments(ctx, updated.PluginID, updated.PlanType, 10)
	require.NoError(t, err)
	require.NotEmpty(t, adjustments)
}
