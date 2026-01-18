package social_channel_governance

import (
	"net/http"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/contracts"
	SocialService "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/social_channel_governance"
	"github.com/gin-gonic/gin"
)

type ChannelSchemaHandler struct {
	loader *SocialService.ChannelSchemaLoader
}

func NewChannelSchemaHandler(loader *SocialService.ChannelSchemaLoader) *ChannelSchemaHandler {
	return &ChannelSchemaHandler{loader: loader}
}

func (h *ChannelSchemaHandler) GetSchema(c *gin.Context) {
	if h == nil || h.loader == nil {
		contracts.ResponseError(c, http.StatusServiceUnavailable, contracts.ErrCodeInternalError, "schema loader unavailable")
		return
	}
	schema, err := h.loader.Load(c.Request.Context())
	if err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, schema)
}
