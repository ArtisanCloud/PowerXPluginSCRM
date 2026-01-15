package marketplace

import (
	mrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/marketplace"
	recommendationservice "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/recommendation"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
)

// RegisterRoutes wires marketplace admin endpoints.
func RegisterRoutes(admin *gin.RouterGroup, deps *app.Deps) {
	if admin == nil || deps == nil {
		return
	}
	handler := NewListingHandler(deps)
	checklist := NewChecklistGraphQLHandler(handler.Service())

	listingRepo := mrepo.NewListingRepository(deps.DB)
	metricsProvider := recommendationservice.NewListingMetricsProvider(listingRepo)
	recommendationLogger := deps.RuntimeLogger(deps.Ctx, "admin_marketplace_recommendation", nil)
	recommendationHandler := NewRecommendationHandler(deps.Config, listingRepo, metricsProvider, recommendationLogger)
	analyticsHandler := NewAnalyticsHandler(deps)

	group := admin.Group("/marketplace")
	{
		listings := group.Group("/listings")
		{
			listings.GET("", handler.List)
			listings.POST("", handler.Create)
			listings.GET("/:id", handler.Get)
			listings.PATCH("/:id", handler.Update)
			listings.POST("/:id/review", handler.SubmitForReview)
			listings.POST("/:id/publish", handler.Publish)
			listings.POST("/:id/suspend", handler.Suspend)
		}

		group.POST("/checklist/graphql", checklist.Resolve)

		recommendationGroup := group.Group("/recommendation")
		{
			recommendationGroup.GET("/config", recommendationHandler.GetConfig)
			recommendationGroup.POST("/sync", recommendationHandler.TriggerSync)
			recommendationGroup.PATCH("/experiment", recommendationHandler.UpdateExperiment)
		}

		group.POST("/usage", analyticsHandler.Ingest)
		group.GET("/usage/tenants/:tenantId/licenses/:licenseId/metrics", analyticsHandler.GetMetrics)
		group.GET("/revenue-share/reports", analyticsHandler.ListRevenueReports)
	}
}
