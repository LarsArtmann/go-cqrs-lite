package middleware_test

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

	assertMiddlewareGolden(
		t,
		filepath.Join("testdata", "golden", "health-check-response.json"),
		got,
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

	assertMiddlewareGolden(
		t,
		filepath.Join("testdata", "golden", "retry-config-validation.json"),
		got,
	)
}

func assertMiddlewareGolden(t *testing.T, path string, got []byte) {
	t.Helper()

	if *update {
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			t.Fatalf("mkdir: %v", err)
		}

		if err := os.WriteFile(path, append(got, '\n'), 0o644); err != nil {
			t.Fatalf("write golden: %v", err)
		}

		return
	}

	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden %s (run with -update to create): %v", path, err)
	}

	if strings.TrimSpace(string(got)) != strings.TrimSpace(string(want)) {
		t.Errorf("golden mismatch for %s (run with -update to refresh)", path)
	}
}
