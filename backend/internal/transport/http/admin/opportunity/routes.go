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
		op.GET("/forecast", handler.Forecast)
		op.GET("/pipeline-templates", handler.ListPipelineTemplates)
		op.GET("/pipeline-groups", handler.ListPipelineGroups)
		op.POST("/pipeline-groups", handler.CreatePipelineGroup)
		op.GET("/pipeline-groups/:group_uuid", handler.GetPipelineGroup)
		op.GET("/pipeline/default", handler.DefaultPipeline)
		op.GET("/stage-configs", handler.ListStageConfigs)
		op.PUT("/stage-configs/:stage_key", handler.SaveStageConfig)
		op.GET("/records", handler.List)
		op.POST("/records", handler.Create)
		op.GET("/records/:opportunity_uuid", handler.Get)
		op.PUT("/records/:opportunity_uuid", handler.Update)
		op.GET("/records/:opportunity_uuid/duplicates", handler.DetectDuplicates)
		op.POST("/records/:opportunity_uuid/merge", handler.MergeOpportunity)
		op.GET("/records/:opportunity_uuid/line-items", handler.ListLineItems)
		op.POST("/records/:opportunity_uuid/line-items", handler.AddLineItem)
		op.POST("/records/:opportunity_uuid/quote-files", handler.UploadQuoteFile)
		op.GET("/records/:opportunity_uuid/line-items/:item_uuid/download", handler.DownloadQuoteFile)
		op.PATCH("/records/:opportunity_uuid/line-items/:item_uuid/approval", handler.UpdateQuoteApproval)
		op.DELETE("/records/:opportunity_uuid/line-items/:item_uuid", handler.DeleteLineItem)
		op.GET("/records/:opportunity_uuid/contracts", handler.ListContracts)
		op.POST("/records/:opportunity_uuid/contracts", handler.AddContract)
		op.PATCH("/records/:opportunity_uuid/contracts/:contract_uuid/status", handler.UpdateContractStatus)
		op.GET("/records/:opportunity_uuid/payments", handler.ListPayments)
		op.POST("/records/:opportunity_uuid/payments", handler.AddPayment)
		op.PATCH("/records/:opportunity_uuid/payments/:payment_uuid/status", handler.UpdatePaymentStatus)
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
