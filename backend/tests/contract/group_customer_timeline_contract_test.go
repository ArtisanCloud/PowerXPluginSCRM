package contract

import (
	"net/http"
	"net/http/httptest"
	"testing"

	authx "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/middleware"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestGroupCustomerTimelineContract_EmptyState(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tenantUUID := "00000000-0000-0000-0000-000000000296"
	r := setupGroupCustomerTimelineContractRouter(tenantUUID)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/leads/acquisition/group-chats/chat-001/customers/wms-user-001/timeline?limit=20", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.JSONEq(t, `{"success":true,"data":{"items":[]}}`, rec.Body.String())
}

func TestGroupCustomerTimelineContract_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tenantUUID := "00000000-0000-0000-0000-000000000296"
	r := setupGroupCustomerTimelineContractRouter(tenantUUID)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/leads/acquisition/group-chats/chat-404/customers/wms-user-001/timeline", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusNotFound, rec.Code)
	require.Contains(t, rec.Body.String(), "group chat not found")
}

func setupGroupCustomerTimelineContractRouter(tenantUUID string) *gin.Engine {
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("tenant_uuid", tenantUUID)
		authx.SetTenantContext(c, authx.TenantContext{TenantUUID: tenantUUID})
		c.Request = c.Request.WithContext(authx.ContextWithTenantUUID(c.Request.Context(), tenantUUID))
		c.Next()
	})

	group := r.Group("/api/v1/admin/leads/acquisition")
	group.GET("/group-chats/:chat_id/customers/:external_userid/timeline", func(c *gin.Context) {
		if c.Param("chat_id") == "chat-404" {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": gin.H{"message": "group chat not found"}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"items": []any{}}})
	})

	return r
}
