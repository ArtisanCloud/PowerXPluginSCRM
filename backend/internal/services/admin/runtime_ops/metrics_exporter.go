package runtime_ops

import (
	"net/http"
)

// MetricsHTTPHandler returns an HTTP handler exposing runtime ops metrics in Prometheus format.
func MetricsHTTPHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		renderMetrics(w)
	})
}
