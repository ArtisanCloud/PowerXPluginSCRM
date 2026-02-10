package websocket

import (
	"path"
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/config"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/transport/websocket/bus"
	"github.com/gin-gonic/gin"
)

func RegisterWSRoutes(r *gin.Engine, authMiddleware gin.HandlerFunc, cfg *config.Config) {
	if r == nil {
		return
	}

	apiPrefix := "/api/v1"
	if cfg != nil && cfg.Server != nil {
		if p := strings.TrimSpace(cfg.Server.APIPrefix); p != "" {
			apiPrefix = p
		}
	}
	if !strings.HasPrefix(apiPrefix, "/") {
		apiPrefix = "/" + apiPrefix
	}
	apiPrefix = strings.TrimRight(apiPrefix, "/")

	wsPath := path.Join(apiPrefix, "ws")

	prefix := wsPath
	if cfg != nil && cfg.Server != nil {
		if p := strings.TrimSpace(cfg.Server.WSPrefix); p != "" {
			prefix = p
		}
	}
	if !strings.HasPrefix(prefix, "/") {
		prefix = "/" + prefix
	}

	wsGroup := r.Group(prefix)
	wsGroup.Use(BearerShim())
	if authMiddleware != nil {
		wsGroup.Use(authMiddleware)
	}
	wsHandler := bus.NewHandler()
	wsGroup.GET("", wsHandler.ServeWS)
}
