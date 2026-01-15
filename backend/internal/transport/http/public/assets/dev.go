package assets

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// RegisterBuildMetaRoutes wires development build meta endpoints used by the web admin shell.
// It serves both `/dev.json` and `/:buildId(.json)` requests with a lightweight JSON payload.
func RegisterBuildMetaRoutes(engine *gin.Engine, basePath string) {
	if engine == nil || basePath == "" {
		return
	}

	base := strings.TrimRight(basePath, "/")

	handler := func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept")
		c.Header("Access-Control-Allow-Methods", "GET, OPTIONS")

		if c.Request.Method == http.MethodOptions {
			c.Status(http.StatusNoContent)
			return
		}

		buildID := strings.TrimSpace(c.Param("buildId"))
		if buildID == "" {
			buildID = "dev"
		}
		buildID = strings.TrimPrefix(buildID, "/")
		buildID = strings.TrimSuffix(buildID, ".json")
		if buildID == "" {
			buildID = "dev"
		}

		c.JSON(http.StatusOK, gin.H{
			"id":          buildID,
			"timestamp":   time.Now().UnixMilli(),
			"matcher":     gin.H{"static": gin.H{}, "wildcard": gin.H{}, "dynamic": gin.H{}},
			"prerendered": []interface{}{},
		})
	}

	engine.GET(base+"/:buildId", handler)
	engine.OPTIONS(base+"/:buildId", handler)
}
