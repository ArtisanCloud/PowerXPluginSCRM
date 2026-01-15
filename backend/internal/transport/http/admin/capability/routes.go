package capability

import (
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
)

// RegisterRoutes wires capability registration routes under /admin.
func RegisterRoutes(rg *gin.RouterGroup, deps *app.Deps) {
	if rg == nil || deps == nil {
		return
	}
	handler := NewRegisterHandler(deps)
	catalogHandler := NewCatalogHandler(deps)
	reviewHandler := NewReviewHandler(deps)
	exposureHandler := NewExposureHandler(deps)
	quotaHandler := NewQuotaHandler(deps)
	lifecycleHandler := NewLifecycleHandler(deps)
	if handler == nil {
		return
	}
	if catalogHandler != nil {
		rg.GET("/capabilities", catalogHandler.List)
	}
	group := rg.Group("/capabilities/register")
	{
		group.GET("/template", handler.GetTemplate)
		group.POST("/validate", handler.ValidateDraft)
		group.POST("", handler.Submit)
	}
	if reviewHandler != nil {
		reviews := rg.Group("/capabilities/reviews")
		{
			reviews.GET("/:capabilityID", reviewHandler.List)
			reviews.POST("/:capabilityID/resubmit", reviewHandler.Resubmit)
			reviews.POST("/tasks/:taskID/comments", reviewHandler.AddComment)
			reviews.POST("/tasks/:taskID/decision", reviewHandler.Decide)
		}
	}
	if exposureHandler != nil {
		exposure := rg.Group("/capabilities/exposure")
		{
			exposure.GET("/template", exposureHandler.GetTemplate)
			exposure.GET("/:capabilityID", exposureHandler.Get)
			exposure.PUT("/:capabilityID", exposureHandler.Upsert)
		}
	}
	if quotaHandler != nil {
		quotas := rg.Group("/capabilities/quotas")
		{
			quotas.GET("/:capabilityID", quotaHandler.List)
			quotas.POST("/:capabilityID", quotaHandler.Upsert)
		}
	}
	if lifecycleHandler != nil {
		lifecycle := rg.Group("/capabilities/lifecycle")
		{
			lifecycle.GET("/template", lifecycleHandler.GetTemplate)
			lifecycle.GET("", lifecycleHandler.List)
			lifecycle.POST("", lifecycleHandler.Create)
			lifecycle.POST("/:planID/status", lifecycleHandler.UpdateStatus)
		}
	}
}
