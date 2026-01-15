package server

import (
	"net/http"

	fwbootstrap "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/bootstrap"
	"github.com/gin-gonic/gin"
)

var methods = []string{
	http.MethodGet,
	http.MethodPost,
	http.MethodPut,
	http.MethodPatch,
	http.MethodDelete,
	http.MethodOptions,
	http.MethodHead,
}

// RegisterGinRoutes wires every HTTP verb to the underlying gin engine so that
// framework bootstrap.Router can delegate actual handling.
func RegisterGinRoutes(r fwbootstrap.Router, engine *gin.Engine) {
	if r == nil || engine == nil {
		return
	}
	handler := ginHandler(engine)
	for _, method := range methods {
		r.Handle(method, "", handler)
		r.Handle(method, "/*path", handler)
	}
}

func ginHandler(engine *gin.Engine) fwbootstrap.Handler {
	return func(ctx fwbootstrap.Context) {
		writer, req := unwrapHTTP(ctx)
		if writer == nil || req == nil {
			ctx.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to bridge request to gin"})
			return
		}
		engine.ServeHTTP(writer, req)
	}
}

func unwrapHTTP(ctx fwbootstrap.Context) (http.ResponseWriter, *http.Request) {
	type httpBridger interface {
		HTTPResponseWriter() http.ResponseWriter
		HTTPRequest() *http.Request
	}

	bridger, ok := ctx.(httpBridger)
	if !ok {
		return nil, nil
	}

	writer := bridger.HTTPResponseWriter()
	req := bridger.HTTPRequest()
	if writer == nil || req == nil {
		return nil, nil
	}
	return writer, req
}
