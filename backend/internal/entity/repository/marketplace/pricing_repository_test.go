package marketplace

import (
	"context"
	"testing"

	dbm "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/marketplace"
	"github.com/stretchr/testify/require"
)

func TestPricingRepository_CreateAndUpdatePlan(t *testing.T) {
	db := setupMarketplaceTestDB(t)
	repo := NewPricingRepository(db)
	ctx := context.Background()

	amount := 19.99
	plan := &dbm.PricingPlan{
		TenantUuid:  "tenant-1",
		ListingID: "listing-1",
		PlanCode:  "standard",
		PlanType:  dbm.PricingPlanTypeSubscription,
		Currency:  "USD",
		Amount:    &amount,
	}
	tiers := []dbm.PlanTier{
		{Metric: "requests", RangeFrom: 0, RangeTo: floatPtr(1000), UnitAmount: 0.01, UnitName: "call"},
		{Metric: "requests", RangeFrom: 1000, UnitAmount: 0.008, UnitName: "call"},
	}

	require.NoError(t, repo.CreatePlan(ctx, plan, tiers))
	require.NotEmpty(t, plan.ID)

	stored, err := repo.GetPlan(ctx, "tenant-1", plan.ID)
	require.NoError(t, err)
	require.Len(t, stored.Tiers, 2)

	newAmount := 29.99
	plan.Amount = &newAmount
	updateTiers := []dbm.PlanTier{
		{Metric: "requests", RangeFrom: 0, RangeTo: floatPtr(500), UnitAmount: 0.012, UnitName: "call"},
	}
	require.NoError(t, repo.UpdatePlan(ctx, plan, updateTiers))

	stored, err = repo.GetPlan(ctx, "tenant-1", plan.ID)
	require.NoError(t, err)
	require.Equal(t, &newAmount, stored.Amount)
	require.Len(t, stored.Tiers, 1)
	require.Equal(t, 0.012, stored.Tiers[0].UnitAmount)
}

func TestPricingRepository_SetDefaultPlan(t *testing.T) {
	db := setupMarketplaceTestDB(t)
	repo := NewPricingRepository(db)
	ctx := context.Background()

	standard := &dbm.PricingPlan{
		TenantUuid:  "tenant-1",
		ListingID: "listing-1",
		PlanCode:  "standard",
		PlanType:  dbm.PricingPlanTypeSubscription,
		Currency:  "USD",
	}
	require.NoError(t, repo.CreatePlan(ctx, standard, nil))

	premium := &dbm.PricingPlan{
		TenantUuid:  "tenant-1",
		ListingID: "listing-1",
		PlanCode:  "premium",
		PlanType:  dbm.PricingPlanTypeSubscription,
		Currency:  "USD",
	}
	require.NoError(t, repo.CreatePlan(ctx, premium, nil))

	require.NoError(t, repo.SetDefaultPlan(ctx, "tenant-1", "listing-1", premium.ID))

	plans, err := repo.ListPlans(ctx, "tenant-1", "listing-1")
	require.NoError(t, err)
	var defaultCount int
	for _, p := range plans {
		if p.IsDefault {
			defaultCount++
			require.Equal(t, premium.ID, p.ID)
		}
	}
	require.Equal(t, 1, defaultCount)
}
