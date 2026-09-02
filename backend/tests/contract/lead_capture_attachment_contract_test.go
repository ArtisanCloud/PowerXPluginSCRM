package contract

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	leadmodel "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/models/lead_capture"
	leadrepo "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/entity/repository/lead_capture"
	authx "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/middleware"
	leadsvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/lead_capture"
	httplead "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/http/admin/lead_capture"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestLeadCaptureAttachmentContract_UploadAndListNodeAttachment(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tenantUUID := "00000000-0000-4000-8000-000000000001"
	leadUUID := "10000000-0000-4000-8000-000000000001"
	memberUUID := "30000000-0000-4000-8000-000000000001"
	db := openContractDB(t, "lead_capture_attachment_contract")
	require.NoError(t, db.Exec(`CREATE TABLE lead_capture_attachments (
		attachment_uuid TEXT PRIMARY KEY, tenant_uuid TEXT NOT NULL, lead_uuid TEXT NOT NULL,
		activity_uuid TEXT NULL, stage_key TEXT, action_key TEXT, file_name TEXT NOT NULL,
		content_type TEXT, file_size INTEGER NOT NULL DEFAULT 0, storage_provider TEXT NOT NULL,
		content BLOB NOT NULL, created_at DATETIME, updated_at DATETIME
	)`).Error)
	require.NoError(t, db.Create(&leadmodel.Lead{
		LeadUUID: leadUUID, TenantUUID: tenantUUID, DisplayName: "lead", Status: leadmodel.LeadStatusEngaging,
	}).Error)

	handler := httplead.NewLeadHandler(leadsvc.NewLeadService(leadrepo.NewLeadRepository(db)))
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("tenant_uuid", tenantUUID)
		authx.SetTenantContext(c, authx.TenantContext{TenantUUID: tenantUUID, MemberUUID: memberUUID})
		requestContext := authx.ContextWithTenantUUID(c.Request.Context(), tenantUUID)
		requestContext = authx.ContextWithMemberUUID(requestContext, memberUUID)
		c.Request = c.Request.WithContext(requestContext)
		c.Next()
	})
	router.POST("/api/v1/admin/leads/:lead_id/node-attachments", handler.UploadNodeAttachment)
	router.GET("/api/v1/admin/leads/:lead_id/node-attachments", handler.ListNodeAttachments)
	router.GET("/api/v1/admin/leads/:lead_id/activities", handler.ListActivities)
	router.DELETE("/api/v1/admin/leads/:lead_id/attachments/:attachment_id", handler.DeleteAttachment)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	require.NoError(t, writer.WriteField("stage_key", leadmodel.LeadStatusEngaging))
	require.NoError(t, writer.WriteField("action_key", "engagement"))
	part, err := writer.CreateFormFile("file", "evidence.txt")
	require.NoError(t, err)
	_, err = part.Write([]byte("evidence"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	uploadRequest := httptest.NewRequest(http.MethodPost, "/api/v1/admin/leads/"+leadUUID+"/node-attachments", body)
	uploadRequest.Header.Set("Content-Type", writer.FormDataContentType())
	uploadResponse := httptest.NewRecorder()
	router.ServeHTTP(uploadResponse, uploadRequest)
	require.Equal(t, http.StatusCreated, uploadResponse.Code)

	var uploaded map[string]any
	require.NoError(t, json.Unmarshal(uploadResponse.Body.Bytes(), &uploaded))
	require.Equal(t, true, uploaded["success"])
	uploadedData := uploaded["data"].(map[string]any)
	require.NotEmpty(t, uploadedData["attachment_uuid"])
	require.NotContains(t, uploadedData, "activity_uuid")

	listRequest := httptest.NewRequest(http.MethodGet, "/api/v1/admin/leads/"+leadUUID+"/node-attachments?stage_key=engaging&action_key=engagement", nil)
	listResponse := httptest.NewRecorder()
	router.ServeHTTP(listResponse, listRequest)
	require.Equal(t, http.StatusOK, listResponse.Code)

	var listed map[string]any
	require.NoError(t, json.Unmarshal(listResponse.Body.Bytes(), &listed))
	items := listed["data"].(map[string]any)["items"].([]any)
	require.Len(t, items, 1)
	require.Equal(t, "evidence.txt", items[0].(map[string]any)["file_name"])

	var auditActivity leadmodel.LeadActivity
	require.NoError(t, db.Where("tenant_uuid = ? AND lead_uuid = ? AND activity_type = ?", tenantUUID, leadUUID, leadmodel.LeadActivityTypeAttachmentUploaded).First(&auditActivity).Error)
	require.Equal(t, uploadedData["attachment_uuid"], auditActivity.Payload["attachment_uuid"])
	require.Equal(t, memberUUID, auditActivity.Payload["operator_member_uuid"])
	require.Equal(t, "evidence.txt", auditActivity.Payload["file_name"])

	activitiesRequest := httptest.NewRequest(http.MethodGet, "/api/v1/admin/leads/"+leadUUID+"/activities", nil)
	activitiesResponse := httptest.NewRecorder()
	router.ServeHTTP(activitiesResponse, activitiesRequest)
	require.Equal(t, http.StatusOK, activitiesResponse.Code)

	var activitiesPayload map[string]any
	require.NoError(t, json.Unmarshal(activitiesResponse.Body.Bytes(), &activitiesPayload))
	activities := activitiesPayload["data"].(map[string]any)["items"].([]any)
	require.Len(t, activities, 1)
	activity := activities[0].(map[string]any)
	require.Equal(t, leadmodel.LeadActivityTypeAttachmentUploaded, activity["activity_type"])
	require.Equal(t, "evidence.txt", activity["payload"].(map[string]any)["file_name"])

	deleteRequest := httptest.NewRequest(http.MethodDelete, "/api/v1/admin/leads/"+leadUUID+"/attachments/"+uploadedData["attachment_uuid"].(string), nil)
	deleteResponse := httptest.NewRecorder()
	router.ServeHTTP(deleteResponse, deleteRequest)
	require.Equal(t, http.StatusOK, deleteResponse.Code)

	var deletedAudit leadmodel.LeadActivity
	require.NoError(t, db.Where("tenant_uuid = ? AND lead_uuid = ? AND activity_type = ?", tenantUUID, leadUUID, leadmodel.LeadActivityTypeAttachmentDeleted).First(&deletedAudit).Error)
	require.Equal(t, uploadedData["attachment_uuid"], deletedAudit.Payload["attachment_uuid"])
	var remainingAttachments int64
	require.NoError(t, db.Model(&leadmodel.LeadAttachment{}).Where("attachment_uuid = ?", uploadedData["attachment_uuid"]).Count(&remainingAttachments).Error)
	require.Zero(t, remainingAttachments)
}

func TestLeadCaptureAttachmentContract_RejectsUnknownLifecycleStage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tenantUUID := "00000000-0000-4000-8000-000000000001"
	db := openContractDB(t, "lead_capture_attachment_invalid_stage_contract")
	require.NoError(t, db.Exec(`CREATE TABLE lead_capture_attachments (
		attachment_uuid TEXT PRIMARY KEY, tenant_uuid TEXT NOT NULL, lead_uuid TEXT NOT NULL,
		activity_uuid TEXT NULL, stage_key TEXT, action_key TEXT, file_name TEXT NOT NULL,
		content_type TEXT, file_size INTEGER NOT NULL DEFAULT 0, storage_provider TEXT NOT NULL,
		content BLOB NOT NULL, created_at DATETIME, updated_at DATETIME
	)`).Error)

	handler := httplead.NewLeadHandler(leadsvc.NewLeadService(leadrepo.NewLeadRepository(db)))
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Request = c.Request.WithContext(authx.ContextWithTenantUUID(c.Request.Context(), tenantUUID))
		c.Next()
	})
	router.GET("/api/v1/admin/leads/:lead_id/node-attachments", handler.ListNodeAttachments)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/admin/leads/10000000-0000-4000-8000-000000000001/node-attachments?stage_key=sales_won", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	require.Equal(t, http.StatusBadRequest, response.Code)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &payload))
	require.Equal(t, "LEAD_ATTACHMENT_INVALID", payload["error"].(map[string]any)["code"])
}
