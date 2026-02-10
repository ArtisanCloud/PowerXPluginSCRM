package social_channel_governance

import (
	"net/http"
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/config"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/contracts"
	SocialService "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/admin/social_channel_governance"
	"github.com/gin-gonic/gin"
)

type ChannelSchemaHandler struct {
	loader *SocialService.ChannelSchemaLoader
	cfg    *config.Config
}

func NewChannelSchemaHandler(loader *SocialService.ChannelSchemaLoader, cfg *config.Config) *ChannelSchemaHandler {
	return &ChannelSchemaHandler{loader: loader, cfg: cfg}
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
	applyCallbackBaseDefault(schema, h.cfg)
	contracts.ResponseSuccess(c, schema)
}

func applyCallbackBaseDefault(schema *SocialService.ChannelSchemaDocument, cfg *config.Config) {
	if schema == nil || cfg == nil || cfg.Server == nil {
		return
	}
	base := strings.TrimSpace(cfg.Server.CallbackBaseURL)
	if base == "" {
		return
	}
	for ci := range schema.Channels {
		for ai := range schema.Channels[ci].AppTypes {
			for fi := range schema.Channels[ci].AppTypes[ai].Fields {
				field := &schema.Channels[ci].AppTypes[ai].Fields[fi]
				if field.Key == "callback_base_url" && field.DefaultValue == "" {
					field.DefaultValue = base
				}
			}
		}
	}
}
