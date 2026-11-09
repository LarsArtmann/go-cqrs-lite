package middleware_test

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"flag"
	"path/filepath"
	"testing"
	"time"

	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/middleware/v4"
)

var update = flag.Bool("update", false, "update golden files")

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

	got, err := json.Marshal(cases, jsontext.WithIndentPrefix(""), jsontext.WithIndent("  "))
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
