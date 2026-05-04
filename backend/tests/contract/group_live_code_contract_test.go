package contract

import (
	"bytes"
	"encoding/json"
	"fmt"
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

type groupLiveCodeV21Record struct {
	GroupCodeUUID      string    `json:"group_code_uuid"`
	Channel            string    `json:"channel"`
	AppType            string    `json:"app_type"`
	ChannelAccountUUID string    `json:"channel_account_uuid"`
	ActivityName       string    `json:"activity_name"`
	JoinScene          int       `json:"join_scene"`
	SkipVerify         bool      `json:"skip_verify"`
	AutoCreateRoom     bool      `json:"auto_create_room"`
	State              string    `json:"state"`
	ConfigID           string    `json:"config_id,omitempty"`
	QRCode             string    `json:"qr_code,omitempty"`
	Status             string    `json:"status"`
	SyncStatus         string    `json:"sync_status"`
	LastSyncError      string    `json:"last_sync_error,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

func TestGroupLiveCodeContract_CRUDAndSyncStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tenantUUID := "00000000-0000-0000-0000-000000000251"
	r := setupGroupLiveCodeContractV21Router(tenantUUID)

	createPayload := map[string]any{
		"channel":              "wechat",
		"app_type":             "wecom",
		"channel_account_uuid": "30aa4d9f-6768-4fdf-97c8-66844422a7a5",
		"activity_name":        "群裂变活动A",
		"join_scene":           1,
		"skip_verify":          false,
		"auto_create_room":     false,
	}
	createBody, _ := json.Marshal(createPayload)
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/admin/leads/acquisition/group-codes", bytes.NewBuffer(createBody))
	createReq.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()
	r.ServeHTTP(createRec, createReq)
	require.Equal(t, http.StatusOK, createRec.Code)

	var createResp map[string]any
	require.NoError(t, json.Unmarshal(createRec.Body.Bytes(), &createResp))
	require.Equal(t, true, createResp["success"])
	created := createResp["data"].(map[string]any)
	groupCodeUUID := created["group_code_uuid"].(string)
	require.Equal(t, "pending", created["sync_status"])
	require.Equal(t, "draft", created["status"])

	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin/leads/acquisition/group-codes", nil)
	listRec := httptest.NewRecorder()
	r.ServeHTTP(listRec, listReq)
	require.Equal(t, http.StatusOK, listRec.Code)
	var listResp map[string]any
	require.NoError(t, json.Unmarshal(listRec.Body.Bytes(), &listResp))
	items := listResp["data"].(map[string]any)["items"].([]any)
	require.Len(t, items, 1)

	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin/leads/acquisition/group-codes/"+groupCodeUUID, nil)
	getRec := httptest.NewRecorder()
	r.ServeHTTP(getRec, getReq)
	require.Equal(t, http.StatusOK, getRec.Code)

	updateBody := bytes.NewBufferString(`{"activity_name":"群裂变活动B","skip_verify":true}`)
	updateReq := httptest.NewRequest(http.MethodPut, "/api/v1/admin/leads/acquisition/group-codes/"+groupCodeUUID, updateBody)
	updateReq.Header.Set("Content-Type", "application/json")
	updateRec := httptest.NewRecorder()
	r.ServeHTTP(updateRec, updateReq)
	require.Equal(t, http.StatusOK, updateRec.Code)

	syncReq := httptest.NewRequest(http.MethodPost, "/api/v1/admin/leads/acquisition/group-codes/"+groupCodeUUID+"/sync", nil)
	syncRec := httptest.NewRecorder()
	r.ServeHTTP(syncRec, syncReq)
	require.Equal(t, http.StatusOK, syncRec.Code)
	var syncResp map[string]any
	require.NoError(t, json.Unmarshal(syncRec.Body.Bytes(), &syncResp))
	syncData := syncResp["data"].(map[string]any)
	require.Equal(t, "success", syncData["sync_status"])
	require.NotEmpty(t, syncData["config_id"])
	require.NotEmpty(t, syncData["qr_code"])

	delReq := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/leads/acquisition/group-codes/"+groupCodeUUID, nil)
	delRec := httptest.NewRecorder()
	r.ServeHTTP(delRec, delReq)
	require.Equal(t, http.StatusOK, delRec.Code)

	listRec2 := httptest.NewRecorder()
	r.ServeHTTP(listRec2, listReq)
	require.Equal(t, http.StatusOK, listRec2.Code)
	var listResp2 map[string]any
	require.NoError(t, json.Unmarshal(listRec2.Body.Bytes(), &listResp2))
	items2 := listResp2["data"].(map[string]any)["items"].([]any)
	require.Len(t, items2, 0)
}

func setupGroupLiveCodeContractV21Router(tenantUUID string) *gin.Engine {
	type createReq struct {
		Channel            string `json:"channel"`
		AppType            string `json:"app_type"`
		ChannelAccountUUID string `json:"channel_account_uuid"`
		ActivityName       string `json:"activity_name"`
		JoinScene          int    `json:"join_scene"`
		SkipVerify         bool   `json:"skip_verify"`
		AutoCreateRoom     bool   `json:"auto_create_room"`
	}
	type updateReq struct {
		ActivityName *string `json:"activity_name"`
		SkipVerify   *bool   `json:"skip_verify"`
	}

	store := map[string]*groupLiveCodeV21Record{}

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("tenant_uuid", tenantUUID)
		authx.SetTenantContext(c, authx.TenantContext{TenantUUID: tenantUUID})
		c.Request = c.Request.WithContext(authx.ContextWithTenantUUID(c.Request.Context(), tenantUUID))
		c.Next()
	})

	group := r.Group("/api/v1/admin/leads/acquisition")
	group.POST("/group-codes", func(c *gin.Context) {
		var req createReq
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
			return
		}
		now := time.Now().UTC()
		id := uuid.NewString()
		item := &groupLiveCodeV21Record{
			GroupCodeUUID:      id,
			Channel:            req.Channel,
			AppType:            req.AppType,
			ChannelAccountUUID: req.ChannelAccountUUID,
			ActivityName:       req.ActivityName,
			JoinScene:          req.JoinScene,
			SkipVerify:         req.SkipVerify,
			AutoCreateRoom:     req.AutoCreateRoom,
			State:              "st-" + id[:8],
			Status:             "draft",
			SyncStatus:         "pending",
			CreatedAt:          now,
			UpdatedAt:          now,
		}
		store[id] = item
		c.JSON(http.StatusOK, gin.H{"success": true, "data": item})
	})

	group.GET("/group-codes", func(c *gin.Context) {
		items := make([]*groupLiveCodeV21Record, 0, len(store))
		for _, v := range store {
			items = append(items, v)
		}
		sort.Slice(items, func(i, j int) bool {
			return items[i].CreatedAt.After(items[j].CreatedAt)
		})
		c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"items": items}})
	})

	group.GET("/group-codes/:group_code_uuid", func(c *gin.Context) {
		id := c.Param("group_code_uuid")
		item, ok := store[id]
		if !ok {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "not found"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": item})
	})

	group.PUT("/group-codes/:group_code_uuid", func(c *gin.Context) {
		id := c.Param("group_code_uuid")
		item, ok := store[id]
		if !ok {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "not found"})
			return
		}
		var req updateReq
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
			return
		}
		if req.ActivityName != nil {
			item.ActivityName = *req.ActivityName
		}
		if req.SkipVerify != nil {
			item.SkipVerify = *req.SkipVerify
		}
		item.UpdatedAt = time.Now().UTC()
		c.JSON(http.StatusOK, gin.H{"success": true, "data": item})
	})

	group.DELETE("/group-codes/:group_code_uuid", func(c *gin.Context) {
		id := c.Param("group_code_uuid")
		delete(store, id)
		c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"deleted": true}})
	})

	group.POST("/group-codes/:group_code_uuid/sync", func(c *gin.Context) {
		id := c.Param("group_code_uuid")
		item, ok := store[id]
		if !ok {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "not found"})
			return
		}
		item.SyncStatus = "success"
		item.ConfigID = "cfg-" + id[:8]
		item.QRCode = fmt.Sprintf("https://qrcode.example/%s", item.ConfigID)
		item.LastSyncError = ""
		item.UpdatedAt = time.Now().UTC()
		c.JSON(http.StatusOK, gin.H{"success": true, "data": item})
	})

	return r
}
