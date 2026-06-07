package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"
)

// HealthStatus represents the health status of a component.
type HealthStatus string

const (
	HealthStatusPass HealthStatus = "pass"
	HealthStatusFail HealthStatus = "fail"
	HealthStatusWarn HealthStatus = "warn"
)

// HealthCheckResponse is the standardized health check response.
type HealthCheckResponse struct {
	Status    HealthStatus      `json:"status"`
	Version   string            `json:"version,omitempty"`
	ReleaseID string            `json:"releaseId,omitempty"`
	Notes     []string          `json:"notes,omitempty"`
	Output    string            `json:"output,omitempty"`
	Checks    map[string]Check  `json:"checks,omitempty"`
	Links     map[string]string `json:"links,omitempty"`
}

// Check represents a single health check probe.
type Check struct {
	ComponentID       string            `json:"componentId,omitempty"`
	ComponentType     string            `json:"componentType,omitempty"`
	ObservedValue     any               `json:"observedValue,omitempty"`
	ObservedUnit      string            `json:"observedUnit,omitempty"`
	Status            HealthStatus      `json:"status"`
	AffectedEndpoints []string          `json:"affectedEndpoints,omitempty"`
	Time              string            `json:"time"`
	Links             map[string]string `json:"links,omitempty"`
}

// HealthChecker is a function that checks the health of a component.
type HealthChecker func(ctx context.Context) Check

// HealthCheckHandler returns an HTTP handler for health checks.
// It supports both /health/live (liveness) and /health/ready (readiness).
func HealthCheckHandler(version string, checks ...HealthChecker) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		now := time.Now().UTC().Format(time.RFC3339)

		path := r.URL.Path
		isLive := path == "/health/live" || path == "/health"
		isReady := path == "/health/ready"

		resp := HealthCheckResponse{
			Status:  HealthStatusPass,
			Version: version,
			Checks:  make(map[string]Check),
		}

		// Liveness check: always pass if the process is running
		if isLive {
			resp.Checks["liveness"] = Check{
				ComponentType: "process",
				Status:        HealthStatusPass,
				Time:          now,
			}
			writeHealthResponse(w, resp)

			return
		}

		// Readiness check: run all provided checks
		if isReady {
			for i, check := range checks {
				result := check(ctx)
				if result.Time == "" {
					result.Time = now
				}

				name := result.ComponentID
				if name == "" {
					name = "check-" + strconv.Itoa(i)
				}

				resp.Checks[name] = result
				if result.Status == HealthStatusFail {
					resp.Status = HealthStatusFail
				} else if result.Status == HealthStatusWarn && resp.Status == HealthStatusPass {
					resp.Status = HealthStatusWarn
				}
			}

			writeHealthResponse(w, resp)

			return
		}

		// Default: full health check
		resp.Checks["liveness"] = Check{
			ComponentType: "process",
			Status:        HealthStatusPass,
			Time:          now,
		}

		for i, check := range checks {
			result := check(ctx)
			if result.Time == "" {
				result.Time = now
			}

			name := result.ComponentID
			if name == "" {
				name = "check-" + strconv.Itoa(i)
			}

			resp.Checks[name] = result
			if result.Status == HealthStatusFail {
				resp.Status = HealthStatusFail
			} else if result.Status == HealthStatusWarn && resp.Status == HealthStatusPass {
				resp.Status = HealthStatusWarn
			}
		}

		writeHealthResponse(w, resp)
	})
}

func writeHealthResponse(w http.ResponseWriter, resp HealthCheckResponse) {
	code := http.StatusOK

	switch resp.Status {
	case HealthStatusFail:
		code = http.StatusServiceUnavailable
	case HealthStatusWarn:
		code = http.StatusOK
	}

	w.Header().Set("Content-Type", "application/health+json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(resp)
}
