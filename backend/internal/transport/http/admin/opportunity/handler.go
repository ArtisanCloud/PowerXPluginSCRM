package opportunity

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/contracts"
	oppmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/models/opportunity"
	opprepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/domain/repository/opportunity"
	authmw "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/middleware"
	oppsvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/opportunity"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/middleware"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler struct {
	svc *oppsvc.Service
}

func NewHandler(svc *oppsvc.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) List(c *gin.Context) {
	tenantUUID, ok := tenantUUID(c)
	if !ok {
		return
	}
	var query listOpportunityQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		contracts.ResponseBadRequest(c, "invalid query: "+err.Error())
		return
	}
	filter, ok := parseListFilter(c, query)
	if !ok {
		return
	}
	items, err := h.svc.List(c.Request.Context(), tenantUUID, filter)
	if handleServiceError(c, err) {
		return
	}
	contracts.ResponseSuccess(c, gin.H{"items": opportunityRecordsPayload(items)})
}

func (h *Handler) Dashboard(c *gin.Context) {
	tenantUUID, ok := tenantUUID(c)
	if !ok {
		return
	}
	var query listOpportunityQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		contracts.ResponseBadRequest(c, "invalid query: "+err.Error())
		return
	}
	filter, ok := parseListFilter(c, query)
	if !ok {
		return
	}
	item, err := h.svc.Dashboard(c.Request.Context(), tenantUUID, filter)
	if handleServiceError(c, err) {
		return
	}
	contracts.ResponseSuccess(c, item)
}

func (h *Handler) Create(c *gin.Context) {
	tenantUUID, ok := tenantUUID(c)
	if !ok {
		return
	}
	var req createOpportunityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid body: "+err.Error())
		return
	}
	expectedCloseAt, err := parseOptionalTime(req.ExpectedCloseAt)
	if err != nil {
		contracts.ResponseBadRequest(c, "expected_close_at must be RFC3339")
		return
	}
	item, err := h.svc.Create(c.Request.Context(), tenantUUID, oppsvc.CreateRequest{
		LeadUUID:        req.LeadUUID,
		Title:           req.Title,
		OwnerUserUUID:   firstNonEmpty(req.OwnerMemberUUID, req.OwnerUserUUID),
		Amount:          req.Amount,
		Currency:        req.Currency,
		Probability:     req.Probability,
		ExpectedCloseAt: expectedCloseAt,
		ActorUserUUID:   actorFromContext(c),
	})
	if handleServiceError(c, err) {
		return
	}
	contracts.ResponseCreated(c, opportunityRecordPayload(item))
}

func (h *Handler) Get(c *gin.Context) {
	tenantUUID, ok := tenantUUID(c)
	if !ok {
		return
	}
	item, err := h.svc.Get(c.Request.Context(), tenantUUID, c.Param("opportunity_uuid"))
	if handleServiceError(c, err) {
		return
	}
	contracts.ResponseSuccess(c, item)
}

func (h *Handler) Update(c *gin.Context) {
	tenantUUID, ok := tenantUUID(c)
	if !ok {
		return
	}
	var req updateOpportunityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid body: "+err.Error())
		return
	}
	var expectedCloseAt *time.Time
	clearExpectedCloseAt := false
	if req.ExpectedCloseAt != nil {
		if strings.TrimSpace(*req.ExpectedCloseAt) == "" {
			clearExpectedCloseAt = true
		} else {
			parsed, err := parseOptionalTime(*req.ExpectedCloseAt)
			if err != nil {
				contracts.ResponseBadRequest(c, "expected_close_at must be RFC3339")
				return
			}
			expectedCloseAt = parsed
		}
	}
	item, err := h.svc.Update(c.Request.Context(), tenantUUID, c.Param("opportunity_uuid"), oppsvc.UpdateRequest{
		Title:                req.Title,
		OwnerUserUUID:        firstNonEmptyPtr(req.OwnerMemberUUID, req.OwnerUserUUID),
		Amount:               req.Amount,
		Currency:             req.Currency,
		Probability:          req.Probability,
		ExpectedCloseAt:      expectedCloseAt,
		ClearExpectedCloseAt: clearExpectedCloseAt,
		ActorUserUUID:        actorFromContext(c),
	})
	if handleServiceError(c, err) {
		return
	}
	contracts.ResponseSuccess(c, item)
}

func (h *Handler) ListLineItems(c *gin.Context) {
	tenantUUID, ok := tenantUUID(c)
	if !ok {
		return
	}
	items, err := h.svc.ListLineItems(c.Request.Context(), tenantUUID, c.Param("opportunity_uuid"))
	if handleServiceError(c, err) {
		return
	}
	contracts.ResponseSuccess(c, gin.H{"items": items})
}

func (h *Handler) AddLineItem(c *gin.Context) {
	tenantUUID, ok := tenantUUID(c)
	if !ok {
		return
	}
	var req lineItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid body: "+err.Error())
		return
	}
	item, err := h.svc.AddLineItem(c.Request.Context(), tenantUUID, c.Param("opportunity_uuid"), oppsvc.LineItemRequest{
		Name:          req.Name,
		Quantity:      defaultQuantity(req.Quantity),
		UnitPrice:     req.UnitPrice,
		TotalAmount:   req.TotalAmount,
		Currency:      req.Currency,
		ActorUserUUID: actorFromContext(c),
	})
	if handleServiceError(c, err) {
		return
	}
	contracts.ResponseCreated(c, item)
}

func (h *Handler) UploadQuoteFile(c *gin.Context) {
	tenantUUID, ok := tenantUUID(c)
	if !ok {
		return
	}
	totalAmount, err := parseOptionalFloat(c.PostForm("total_amount"))
	if err != nil {
		contracts.ResponseBadRequest(c, "total_amount must be a number")
		return
	}
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		contracts.ResponseBadRequest(c, "quote file is required")
		return
	}
	defer file.Close()
	item, err := h.svc.AddQuoteFile(
		c.Request.Context(),
		tenantUUID,
		c.Param("opportunity_uuid"),
		actorFromContext(c),
		totalAmount,
		c.PostForm("currency"),
		file,
		header,
	)
	if handleServiceError(c, err) {
		return
	}
	contracts.ResponseCreated(c, item)
}

func (h *Handler) DownloadQuoteFile(c *gin.Context) {
	tenantUUID, ok := tenantUUID(c)
	if !ok {
		return
	}
	item, err := h.svc.GetLineItem(c.Request.Context(), tenantUUID, c.Param("opportunity_uuid"), c.Param("item_uuid"))
	if handleServiceError(c, err) {
		return
	}
	path, ok := h.svc.QuoteFilePath(item)
	if !ok {
		contracts.ResponseNotFound(c, "quote file not found")
		return
	}
	downloadName := strings.TrimSpace(item.FileName)
	if downloadName == "" {
		downloadName = item.Name
	}
	c.FileAttachment(path, downloadName)
}

func (h *Handler) DeleteLineItem(c *gin.Context) {
	tenantUUID, ok := tenantUUID(c)
	if !ok {
		return
	}
	err := h.svc.DeleteLineItem(c.Request.Context(), tenantUUID, c.Param("opportunity_uuid"), c.Param("item_uuid"), actorFromContext(c))
	if handleServiceError(c, err) {
		return
	}
	contracts.ResponseSuccess(c, gin.H{"deleted": true})
}

func (h *Handler) ListTasks(c *gin.Context) {
	tenantUUID, ok := tenantUUID(c)
	if !ok {
		return
	}
	items, err := h.svc.ListTasks(c.Request.Context(), tenantUUID, c.Param("opportunity_uuid"))
	if handleServiceError(c, err) {
		return
	}
	contracts.ResponseSuccess(c, gin.H{"items": items})
}

func (h *Handler) AddTask(c *gin.Context) {
	tenantUUID, ok := tenantUUID(c)
	if !ok {
		return
	}
	var req taskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid body: "+err.Error())
		return
	}
	dueAt, err := parseOptionalTime(req.DueAt)
	if err != nil {
		contracts.ResponseBadRequest(c, "due_at must be RFC3339")
		return
	}
	item, err := h.svc.AddTask(c.Request.Context(), tenantUUID, c.Param("opportunity_uuid"), oppsvc.TaskRequest{
		Title:         req.Title,
		DueAt:         dueAt,
		ActorUserUUID: actorFromContext(c),
	})
	if handleServiceError(c, err) {
		return
	}
	contracts.ResponseCreated(c, item)
}

func (h *Handler) UpdateTaskStatus(c *gin.Context) {
	tenantUUID, ok := tenantUUID(c)
	if !ok {
		return
	}
	var req taskStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid body: "+err.Error())
		return
	}
	item, err := h.svc.UpdateTaskStatus(c.Request.Context(), tenantUUID, c.Param("opportunity_uuid"), c.Param("task_uuid"), oppsvc.TaskStatusRequest{
		Status:        req.Status,
		ActorUserUUID: actorFromContext(c),
	})
	if handleServiceError(c, err) {
		return
	}
	contracts.ResponseSuccess(c, item)
}

func (h *Handler) Stage(c *gin.Context) {
	tenantUUID, ok := tenantUUID(c)
	if !ok {
		return
	}
	var req stageOpportunityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid body: "+err.Error())
		return
	}
	item, err := h.svc.AdvanceStage(c.Request.Context(), tenantUUID, c.Param("opportunity_uuid"), oppsvc.StageRequest{
		Stage:         req.Stage,
		ActorUserUUID: actorFromContext(c),
	})
	if handleServiceError(c, err) {
		return
	}
	contracts.ResponseSuccess(c, opportunityRecordPayload(item))
}

func (h *Handler) Close(c *gin.Context) {
	tenantUUID, ok := tenantUUID(c)
	if !ok {
		return
	}
	var req closeOpportunityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid body: "+err.Error())
		return
	}
	item, err := h.svc.Close(c.Request.Context(), tenantUUID, c.Param("opportunity_uuid"), oppsvc.CloseRequest{
		Result:        req.Result,
		LostReason:    req.LostReason,
		ActorUserUUID: actorFromContext(c),
	})
	if handleServiceError(c, err) {
		return
	}
	contracts.ResponseSuccess(c, opportunityRecordPayload(item))
}

func (h *Handler) Reopen(c *gin.Context) {
	tenantUUID, ok := tenantUUID(c)
	if !ok {
		return
	}
	item, err := h.svc.Reopen(c.Request.Context(), tenantUUID, c.Param("opportunity_uuid"), oppsvc.ReopenRequest{
		ActorUserUUID: actorFromContext(c),
	})
	if handleServiceError(c, err) {
		return
	}
	contracts.ResponseSuccess(c, opportunityRecordPayload(item))
}

func (h *Handler) MarkRisk(c *gin.Context) {
	tenantUUID, ok := tenantUUID(c)
	if !ok {
		return
	}
	var req riskOpportunityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid body: "+err.Error())
		return
	}
	item, err := h.svc.MarkRisk(c.Request.Context(), tenantUUID, c.Param("opportunity_uuid"), oppsvc.RiskRequest{
		Flag:          req.Flag,
		Payload:       req.Payload,
		ActorUserUUID: actorFromContext(c),
	})
	if handleServiceError(c, err) {
		return
	}
	contracts.ResponseSuccess(c, opportunityRecordPayload(item))
}

func (h *Handler) Activities(c *gin.Context) {
	tenantUUID, ok := tenantUUID(c)
	if !ok {
		return
	}
	items, err := h.svc.ListActivities(c.Request.Context(), tenantUUID, c.Param("opportunity_uuid"), 100)
	if handleServiceError(c, err) {
		return
	}
	contracts.ResponseSuccess(c, gin.H{"items": opportunityActivitiesPayload(items)})
}

func tenantUUID(c *gin.Context) (string, bool) {
	tenantUUID, ok := middleware.TenantUUIDFromContext(c)
	if !ok || strings.TrimSpace(tenantUUID) == "" {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return "", false
	}
	return tenantUUID, true
}

func parseListFilter(c *gin.Context, query listOpportunityQuery) (oppsvc.ListFilter, bool) {
	expectedCloseFrom, err := parseOptionalTime(query.ExpectedCloseFrom)
	if err != nil {
		contracts.ResponseBadRequest(c, "expected_close_from must be RFC3339")
		return oppsvc.ListFilter{}, false
	}
	expectedCloseTo, err := parseOptionalTime(query.ExpectedCloseTo)
	if err != nil {
		contracts.ResponseBadRequest(c, "expected_close_to must be RFC3339")
		return oppsvc.ListFilter{}, false
	}
	return oppsvc.ListFilter{
		Stage:             query.Stage,
		OwnerUserUUID:     firstNonEmpty(query.OwnerMemberUUID, query.OwnerUserUUID),
		LeadUUID:          query.LeadUUID,
		Keyword:           query.Keyword,
		SourceChannel:     query.SourceChannel,
		RiskOnly:          query.RiskOnly,
		ExpectedCloseFrom: expectedCloseFrom,
		ExpectedCloseTo:   expectedCloseTo,
		Limit:             query.Limit,
	}, true
}

func handleServiceError(c *gin.Context, err error) bool {
	if err == nil {
		return false
	}
	var conflict *oppsvc.ConflictError
	switch {
	case errors.As(err, &conflict):
		contracts.ResponseErrorWithDetails(c, http.StatusConflict, contracts.ErrCodeConflict, "lead already has active opportunity", gin.H{
			"opportunity_uuid": conflict.OpportunityUUID,
		})
	case errors.Is(err, oppsvc.ErrLeadNotFound), errors.Is(err, oppsvc.ErrOpportunityNotFound):
		contracts.ResponseNotFound(c, err.Error())
	case errors.Is(err, oppsvc.ErrLeadMustBeQualified),
		errors.Is(err, oppsvc.ErrInvalidPayload),
		errors.Is(err, oppsvc.ErrInvalidStage),
		errors.Is(err, oppsvc.ErrInvalidStageTransition),
		errors.Is(err, oppsvc.ErrTerminalOpportunity),
		errors.Is(err, oppsvc.ErrInvalidCloseResult),
		errors.Is(err, oppsvc.ErrLostReasonRequired),
		errors.Is(err, oppsvc.ErrOpportunityNotTerminal),
		errors.Is(err, oppsvc.ErrActorUserUUIDRequired),
		errors.Is(err, opprepo.ErrTenantUUIDMissing):
		contracts.ResponseError(c, http.StatusUnprocessableEntity, contracts.ErrCodeValidationFailed, err.Error())
	case errors.Is(err, opprepo.ErrDBNotReady):
		contracts.ResponseServiceUnavailable(c, "opportunity service unavailable", nil)
	default:
		contracts.ResponseInternalError(c, err)
	}
	return true
}

func parseOptionalTime(raw string) (*time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func parseOptionalFloat(raw string) (float64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, nil
	}
	return strconv.ParseFloat(raw, 64)
}

func defaultQuantity(value float64) float64 {
	if value <= 0 {
		return 1
	}
	return value
}

func actorFromContext(c *gin.Context) string {
	for _, key := range []string{"member_uuid", "actor_user_uuid", "user_uuid", "user_id"} {
		if value := strings.TrimSpace(c.GetString(key)); value != "" {
			if actor, ok := normalizeActorUUID(value); ok {
				return actor
			}
		}
	}
	if tc, ok := authmw.GetTenantContext(c); ok {
		if value := strings.TrimSpace(tc.MemberUUID); value != "" {
			if actor, ok := normalizeActorUUID(value); ok {
				return actor
			}
		}
		if value := strings.TrimSpace(tc.UserUUID); value != "" {
			if actor, ok := normalizeActorUUID(value); ok {
				return actor
			}
		}
	}
	return ""
}

func normalizeActorUUID(value string) (string, bool) {
	parsed, err := uuid.Parse(strings.TrimSpace(value))
	if err != nil || parsed == uuid.Nil {
		return "", false
	}
	return strings.ToLower(parsed.String()), true
}

func opportunityRecordsPayload(items []*oppmodel.OpportunityRecord) []gin.H {
	out := make([]gin.H, 0, len(items))
	for _, item := range items {
		out = append(out, opportunityRecordPayload(item))
	}
	return out
}

func opportunityActivitiesPayload(items []*oppmodel.OpportunityActivity) []gin.H {
	out := make([]gin.H, 0, len(items))
	for _, item := range items {
		operator := ""
		if item != nil {
			operator = strings.TrimSpace(item.OperatorUserUUID)
		}
		if item == nil {
			out = append(out, gin.H{})
			continue
		}
		out = append(out, gin.H{
			"activity_uuid":        item.ActivityUUID,
			"tenant_uuid":          item.TenantUUID,
			"opportunity_uuid":     item.OpportunityUUID,
			"activity_type":        item.ActivityType,
			"from_stage":           item.FromStage,
			"to_stage":             item.ToStage,
			"payload":              item.Payload,
			"operator_user_uuid":   operator,
			"operator_member_uuid": operator,
			"request_id":           item.RequestID,
			"created_at":           item.CreatedAt,
		})
	}
	return out
}

func opportunityRecordPayload(item *oppmodel.OpportunityRecord) gin.H {
	if item == nil {
		return gin.H{}
	}
	owner := strings.TrimSpace(item.OwnerUserUUID)
	return gin.H{
		"opportunity_uuid":       item.OpportunityUUID,
		"tenant_uuid":            item.TenantUUID,
		"lead_uuid":              item.LeadUUID,
		"title":                  item.Title,
		"stage":                  item.Stage,
		"amount":                 item.Amount,
		"currency":               item.Currency,
		"probability":            item.Probability,
		"owner_user_uuid":        owner,
		"owner_member_uuid":      owner,
		"source_channel":         item.SourceChannel,
		"source_app_type":        item.SourceAppType,
		"source_account_uuid":    item.SourceAccountUUID,
		"external_userid":        item.ExternalUserID,
		"expected_close_at":      item.ExpectedCloseAt,
		"won_at":                 item.WonAt,
		"lost_at":                item.LostAt,
		"lost_reason":            item.LostReason,
		"risk_flags":             item.RiskFlags,
		"created_by":             item.CreatedBy,
		"created_by_member_uuid": item.CreatedBy,
		"updated_by":             item.UpdatedBy,
		"updated_by_member_uuid": item.UpdatedBy,
		"created_at":             item.CreatedAt,
		"updated_at":             item.UpdatedAt,
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func firstNonEmptyPtr(values ...*string) *string {
	for _, value := range values {
		if value != nil && strings.TrimSpace(*value) != "" {
			clean := strings.TrimSpace(*value)
			return &clean
		}
	}
	return nil
}
