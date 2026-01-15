package agent

import (
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/contracts"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
)

type STSExchangeResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int32  `json:"expires_in"`
}

type STSHandler struct{ deps *app.Deps }

func RegisterSTSRoutes(rg *gin.RouterGroup, deps *app.Deps) {
	h := &STSHandler{deps: deps}
	rg.POST("/sts/exchange", h.Exchange)
}

// Exchange 主动触发 STS Exchange（调试端点）
func (h *STSHandler) Exchange(c *gin.Context) {
	if h.deps.PowerXClient == nil {
		contracts.ResponseInternalError(c, Err("powerx client not initialized"))
		return
	}
	tok, exp, err := h.deps.PowerXClient.ExchangeSTS(c.Request.Context())
	if err != nil {
		contracts.ResponseInternalError(c, err)
		return
	}
	contracts.ResponseSuccess(c, &STSExchangeResponse{AccessToken: tok, ExpiresIn: exp})
}

type simpleErr string

func (e simpleErr) Error() string { return string(e) }
func Err(s string) error          { return simpleErr(s) }
