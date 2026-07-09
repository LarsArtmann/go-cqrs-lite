package docserver

import (
	"encoding/json/v2"
	"net/http"

	"github.com/larsartmann/go-cqrs-lite/catalog/v3"
)

// HealthCheckHandler returns a simple health check handler that verifies
// the catalog has at least one service. Returns 200 with a JSON body if
// healthy, 503 if the catalog is nil or empty.
//
// This is a standalone handler — it does not require a DocsServer.
// Wire it as a liveness/readiness probe alongside your documentation endpoints.
func HealthCheckHandler(cat *catalog.Catalog) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		if cat == nil || len(cat.Services) == 0 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.MarshalWrite(w, map[string]any{
				"status":  "unhealthy",
				"message": "catalog has no services",
			})

			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.MarshalWrite(w, map[string]any{
			"status":   "healthy",
			"services": len(cat.Services),
		})
	}
}
