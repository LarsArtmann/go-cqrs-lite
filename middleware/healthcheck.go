package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"
)

// ReleaseID identifies a specific deployment release.
type ReleaseID string

func (r ReleaseID) String() string { return string(r) }

func (r ReleaseID) IsZero() bool { return r == "" }

// ComponentID identifies a specific infrastructure component.
type ComponentID string

func (c ComponentID) String() string { return string(c) }

func (c ComponentID) IsZero() bool { return c == "" }

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
	ReleaseID    ReleaseID         `json:"releaseId,omitempty"`
	Notes     []string          `json:"notes,omitempty"`
	Output    string            `json:"output,omitempty"`
	Checks    map[string]Check  `json:"checks,omitempty"`
	Links     map[string]string `json:"links,omitempty"`
}

// Check represents a single health check probe.
type Check struct {
	ComponentID       ComponentID       `json:"componentId,omitempty"`
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

		resp := HealthCheckResponse{ //nolint:exhaustruct // optional fields omitted by design
			Status:  HealthStatusPass,
			Version: version,
			Checks:  make(map[string]Check),
		}

		resp.Checks["liveness"] = Check{ //nolint:exhaustruct // optional fields omitted by design
			ComponentType: "process",
			Status:        HealthStatusPass,
			Time:          now,
		}

		if isLive {
			writeHealthResponse(w, resp)

			return
		}

		if isReady {
			runChecks(ctx, now, checks, &resp)

			writeHealthResponse(w, resp)

			return
		}

		runChecks(ctx, now, checks, &resp)

		writeHealthResponse(w, resp)
	})
}

func runChecks(ctx context.Context, now string, checks []HealthChecker, resp *HealthCheckResponse) {
	for i, check := range checks {
		result := check(ctx)

		if result.Time == "" {
			result.Time = now
		}

		name := string(result.ComponentID)

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
}

func writeHealthResponse(w http.ResponseWriter, resp HealthCheckResponse) {
	code := http.StatusOK

	if resp.Status == HealthStatusFail {
		code = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/health+json; charset=utf-8")
	w.WriteHeader(code)

	err := json.NewEncoder(w).Encode(resp)
	if err != nil {
		// Error encoding response; client likely disconnected.
		_ = err
	}
}
