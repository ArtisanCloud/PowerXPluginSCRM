package marketplace

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/contracts"
	svc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/marketplace"
	httpmw "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/middleware"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ChecklistGraphQLHandler provides a lightweight GraphQL bridge for checklist operations.
type ChecklistGraphQLHandler struct {
	service *svc.ListingService
}

// NewChecklistGraphQLHandler constructs the resolver.
func NewChecklistGraphQLHandler(service *svc.ListingService) *ChecklistGraphQLHandler {
	return &ChecklistGraphQLHandler{service: service}
}

type graphQLRequest struct {
	OperationName string         `json:"operationName"`
	Query         string         `json:"query"`
	Variables     map[string]any `json:"variables"`
}

// Resolve handles GraphQL queries and mutations for checklist workflows.
func (h *ChecklistGraphQLHandler) Resolve(c *gin.Context) {
	if h == nil || h.service == nil {
		contracts.ResponseServiceUnavailable(c, "checklist service not available", nil)
		return
	}

	var req graphQLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		contracts.ResponseBadRequest(c, "invalid GraphQL request: "+err.Error())
		return
	}

	tenantID, ok := httpmw.TenantUuidString(c)
	if !ok {
		contracts.ResponseUnauthorized(c, "tenant context missing")
		return
	}

	op := strings.TrimSpace(req.OperationName)
	if op == "" {
		op = inferOperationName(req.Query)
	}
	switch strings.ToLower(op) {
	case "checklistruns":
		h.resolveListRuns(c, tenantID, req.Variables)
	case "latestchecklistrun":
		h.resolveLatestRun(c, tenantID, req.Variables)
	case "triggerchecklist":
		h.resolveTriggerChecklist(c, tenantID, req.Variables)
	default:
		contracts.ResponseBadRequest(c, "unsupported GraphQL operation")
	}
}

func (h *ChecklistGraphQLHandler) resolveListRuns(c *gin.Context, tenantID string, vars map[string]any) {
	listingID := stringVariable(vars, "listingId")
	if strings.TrimSpace(listingID) == "" {
		contracts.ResponseBadRequest(c, "listingId is required")
		return
	}
	limit := intVariable(vars, "limit", 10)
	runs, err := h.service.ListChecklistRuns(c.Request.Context(), tenantID, listingID, limit)
	if err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"checklistRuns": NewChecklistRunListResponse(runs),
		},
	})
}

func (h *ChecklistGraphQLHandler) resolveLatestRun(c *gin.Context, tenantID string, vars map[string]any) {
	listingID := stringVariable(vars, "listingId")
	if strings.TrimSpace(listingID) == "" {
		contracts.ResponseBadRequest(c, "listingId is required")
		return
	}
	run, err := h.service.LatestChecklistRun(c.Request.Context(), tenantID, listingID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusOK, gin.H{"data": gin.H{"latestChecklistRun": nil}})
			return
		}
		contracts.ResponseInternalError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"latestChecklistRun": NewChecklistRunResponse(run),
		},
	})
}

func (h *ChecklistGraphQLHandler) resolveTriggerChecklist(c *gin.Context, tenantID string, vars map[string]any) {
	inputRaw, ok := vars["input"]
	if !ok {
		contracts.ResponseBadRequest(c, "input payload is required")
		return
	}
	payloadBytes, err := json.Marshal(inputRaw)
	if err != nil {
		contracts.ResponseBadRequest(c, "invalid input payload")
		return
	}
	var payload struct {
		ListingID string `json:"listingId"`
		checklistPayload
	}
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		contracts.ResponseBadRequest(c, "invalid checklist payload")
		return
	}
	if strings.TrimSpace(payload.ListingID) == "" {
		contracts.ResponseBadRequest(c, "listingId is required in input")
		return
	}
	runInput := convertChecklistPayload(payload.checklistPayload)
	run, err := h.service.RecordChecklistRun(c.Request.Context(), tenantID, payload.ListingID, *runInput)
	if err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"triggerChecklist": NewChecklistRunResponse(run),
		},
	})
}

func stringVariable(vars map[string]any, key string) string {
	if vars == nil {
		return ""
	}
	if value, ok := vars[key]; ok {
		switch v := value.(type) {
		case string:
			return v
		case json.Number:
			return v.String()
		}
	}
	return ""
}

func intVariable(vars map[string]any, key string, fallback int) int {
	if vars == nil {
		return fallback
	}
	if value, ok := vars[key]; ok {
		switch v := value.(type) {
		case int:
			return v
		case float64:
			return int(v)
		case json.Number:
			if parsed, err := v.Int64(); err == nil {
				return int(parsed)
			}
		case string:
			if parsed, err := strconv.Atoi(v); err == nil {
				return parsed
			}
		}
	}
	return fallback
}

func inferOperationName(query string) string {
	query = strings.ToLower(strings.TrimSpace(query))
	switch {
	case strings.Contains(query, "checklistruns"):
		return "ChecklistRuns"
	case strings.Contains(query, "latestchecklistrun"):
		return "LatestChecklistRun"
	case strings.Contains(query, "triggerchecklist"):
		return "TriggerChecklist"
	default:
		return ""
	}
}
