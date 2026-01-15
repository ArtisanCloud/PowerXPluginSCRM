package customerhttp

import (
	authmw "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/middleware"
	customerobs "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/observability/customer"
	customersvc "github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/services/customer"
	"github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-plugin-scrm/backend/internal/shared/app"
	"github.com/gin-gonic/gin"
)

// Authenticate enforces customer authentication for /mini-app routes.
func Authenticate(deps *app.Deps) gin.HandlerFunc {
	factory := customersvc.NewAuthenticatorFactory(nil, nil)
	if deps != nil && deps.Config != nil {
		factory = customersvc.NewAuthenticatorFactory(deps.Config, nil)
	}
	authenticator := factory.Build()
	audit := customerobs.NewAuditLogger(nil)
	return authmw.CustomerAuth(authenticator, audit)
}
