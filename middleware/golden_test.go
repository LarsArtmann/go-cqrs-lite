package middleware_test

import (
	"encoding/json"
	"flag"
	"path/filepath"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v2/eventtest"
	"github.com/larsartmann/go-cqrs-lite/middleware/v2"
)

var update = flag.Bool("update", false, "update golden files")

func TestGolden_HealthCheckResponse(t *testing.T) {
	resp := middleware.HealthCheckResponse{
		Status:    middleware.HealthStatusPass,
		Version:   "2.2.0",
		ReleaseID: "2026-06-08",
		Notes:     []string{"All systems nominal"},
		Checks: map[string]middleware.Check{
			"event-store": {
				ComponentType: "datastore",
				Status:        middleware.HealthStatusPass,
				Time:          "2026-06-08T06:00:00Z",
				ObservedValue: 3,
				ObservedUnit:  "ms",
			},
			"projection-runner": {
				ComponentType: "worker",
				Status:        middleware.HealthStatusWarn,
				Time:          "2026-06-08T06:00:00Z",
				ObservedValue: "lagging by 50 events",
			},
		},
	}

	got, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	eventtest.AssertGolden(
		t,
		filepath.Join("testdata", "golden", "health-check-response.json"),
		got,
		*update,
	)
}

func TestGolden_RetryConfigValidation(t *testing.T) {
	cases := []struct {
		Name  string `json:"name"`
		Error string `json:"error"`
	}{
		{
			"max_attempts_zero",
			middleware.RetryConfig{
				MaxAttempts:  0,
				InitialDelay: time.Second,
				Multiplier:   2,
			}.Validate().
				Error(),
		},
		{
			"initial_delay_negative",
			middleware.RetryConfig{
				MaxAttempts:  3,
				InitialDelay: -1,
				Multiplier:   2,
			}.Validate().
				Error(),
		},
		{
			"multiplier_one",
			middleware.RetryConfig{
				MaxAttempts:  3,
				InitialDelay: time.Second,
				Multiplier:   1,
			}.Validate().
				Error(),
		},
	}

	got, err := json.MarshalIndent(cases, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	eventtest.AssertGolden(
		t,
		filepath.Join("testdata", "golden", "retry-config-validation.json"),
		got,
		*update,
	)
}
