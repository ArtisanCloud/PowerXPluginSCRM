package contract

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"
	"time"

	authx "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/middleware"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type groupTagRecord struct {
	GroupTagUUID string    `json:"group_tag_uuid"`
	TagName      string    `json:"tag_name"`
	RuleMode     string    `json:"rule_mode"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type groupTagBindingRecord struct {
	GroupTagUUID string `json:"group_tag_uuid"`
	ChatID       string `json:"chat_id"`
	BindSource   string `json:"bind_source"`
}

func TestGroupTagContract_CRUDBindAndQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tenantUUID := "00000000-0000-0000-0000-000000000253"
	r := setupGroupTagContractRouter(tenantUUID)

	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/admin/leads/acquisition/group-tags", bytes.NewBufferString(`{"tag_name":"高意向","rule_mode":"manual"}`))
	createReq.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()
	r.ServeHTTP(createRec, createReq)
	require.Equal(t, http.StatusOK, createRec.Code)

	var createResp map[string]any
	require.NoError(t, json.Unmarshal(createRec.Body.Bytes(), &createResp))
	tag := createResp["data"].(map[string]any)
	tagUUID := tag["group_tag_uuid"].(string)
	require.Equal(t, "manual", tag["rule_mode"])

	bindPayload := `{"chat_ids":["chat-001","chat-001","chat-002"],"bind_source":"manual"}`
	bindReq := httptest.NewRequest(http.MethodPost, "/api/v1/admin/leads/acquisition/group-tags/"+tagUUID+"/bindings", bytes.NewBufferString(bindPayload))
	bindReq.Header.Set("Content-Type", "application/json")
	bindRec := httptest.NewRecorder()
	r.ServeHTTP(bindRec, bindReq)
	require.Equal(t, http.StatusOK, bindRec.Code)

	bindingsReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin/leads/acquisition/group-tags/"+tagUUID+"/bindings", nil)
	bindingsRec := httptest.NewRecorder()
	r.ServeHTTP(bindingsRec, bindingsReq)
	require.Equal(t, http.StatusOK, bindingsRec.Code)

	var bindingsResp map[string]any
	require.NoError(t, json.Unmarshal(bindingsRec.Body.Bytes(), &bindingsResp))
	items := bindingsResp["data"].(map[string]any)["items"].([]any)
	require.Len(t, items, 2, "重复 chat_id 只应绑定一次")

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin/leads/acquisition/group-tags", nil)
	listRec := httptest.NewRecorder()
	r.ServeHTTP(listRec, listReq)
	require.Equal(t, http.StatusOK, listRec.Code)
}

func setupGroupTagContractRouter(tenantUUID string) *gin.Engine {
	tags := map[string]*groupTagRecord{}
	bindings := map[string]map[string]*groupTagBindingRecord{}

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("tenant_uuid", tenantUUID)
		authx.SetTenantContext(c, authx.TenantContext{TenantUUID: tenantUUID})
		c.Request = c.Request.WithContext(authx.ContextWithTenantUUID(c.Request.Context(), tenantUUID))
		c.Next()
	})

	group := r.Group("/api/v1/admin/leads/acquisition")
	group.POST("/group-tags", func(c *gin.Context) {
		var req struct {
			TagName  string `json:"tag_name"`
			RuleMode string `json:"rule_mode"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
			return
		}
		now := time.Now().UTC()
		id := uuid.NewString()
		item := &groupTagRecord{GroupTagUUID: id, TagName: req.TagName, RuleMode: req.RuleMode, Status: "active", CreatedAt: now, UpdatedAt: now}
		tags[id] = item
		c.JSON(http.StatusOK, gin.H{"success": true, "data": item})
	})

	group.GET("/group-tags", func(c *gin.Context) {
		items := make([]*groupTagRecord, 0, len(tags))
		for _, v := range tags {
			items = append(items, v)
		}
		sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
		c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"items": items}})
	})

	group.POST("/group-tags/:group_tag_uuid/bindings", func(c *gin.Context) {
		tagUUID := c.Param("group_tag_uuid")
		if _, ok := tags[tagUUID]; !ok {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "tag not found"})
			return
		}
		var req struct {
			ChatIDs    []string `json:"chat_ids"`
			BindSource string   `json:"bind_source"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
			return
		}
		if bindings[tagUUID] == nil {
			bindings[tagUUID] = map[string]*groupTagBindingRecord{}
		}
		for _, chatID := range req.ChatIDs {
			if chatID == "" {
				continue
			}
			if _, ok := bindings[tagUUID][chatID]; ok {
				continue
			}
			bindings[tagUUID][chatID] = &groupTagBindingRecord{GroupTagUUID: tagUUID, ChatID: chatID, BindSource: req.BindSource}
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"bound": len(bindings[tagUUID])}})
	})

	group.GET("/group-tags/:group_tag_uuid/bindings", func(c *gin.Context) {
		tagUUID := c.Param("group_tag_uuid")
		tagBindings := bindings[tagUUID]
		items := make([]*groupTagBindingRecord, 0, len(tagBindings))
		for _, v := range tagBindings {
			items = append(items, v)
		}
		sort.Slice(items, func(i, j int) bool { return items[i].ChatID < items[j].ChatID })
		c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"items": items}})
	})

	return r
}
