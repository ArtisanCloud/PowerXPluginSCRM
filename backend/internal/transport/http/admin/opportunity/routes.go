package opportunity

import (
	opprepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/repository/opportunity"
	oppsvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/opportunity"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
)

// RegisterRoutes wires Opportunity admin routes.
func RegisterRoutes(rg *gin.RouterGroup, deps *app.Deps) {
	if rg == nil || deps == nil {
		return
	}

	if deps.DB == nil {
		return
	}
	service := oppsvc.NewServiceWithDB(
		deps.DB,
		opprepo.NewOpportunityRepository(deps.DB),
		opprepo.NewOpportunityActivityRepository(deps.DB),
	)
	handler := NewHandler(service)
	op := rg.Group("/opportunity")
	{
		op.GET("/dashboard", handler.Dashboard)
		op.GET("/records", handler.List)
		op.POST("/records", handler.Create)
		op.GET("/records/:opportunity_uuid", handler.Get)
		op.PUT("/records/:opportunity_uuid", handler.Update)
		op.GET("/records/:opportunity_uuid/line-items", handler.ListLineItems)
		op.POST("/records/:opportunity_uuid/line-items", handler.AddLineItem)
		op.POST("/records/:opportunity_uuid/quote-files", handler.UploadQuoteFile)
		op.GET("/records/:opportunity_uuid/line-items/:item_uuid/download", handler.DownloadQuoteFile)
		op.DELETE("/records/:opportunity_uuid/line-items/:item_uuid", handler.DeleteLineItem)
		op.GET("/records/:opportunity_uuid/tasks", handler.ListTasks)
		op.POST("/records/:opportunity_uuid/tasks", handler.AddTask)
		op.PATCH("/records/:opportunity_uuid/tasks/:task_uuid/status", handler.UpdateTaskStatus)
		op.POST("/records/:opportunity_uuid/stage", handler.Stage)
		op.POST("/records/:opportunity_uuid/close", handler.Close)
		op.POST("/records/:opportunity_uuid/reopen", handler.Reopen)
		op.POST("/records/:opportunity_uuid/risk", handler.MarkRisk)
		op.GET("/records/:opportunity_uuid/activities", handler.Activities)
	}
}
