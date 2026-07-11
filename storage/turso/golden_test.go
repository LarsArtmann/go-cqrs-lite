package turso_test

import (
	"encoding/json"
	"flag"
	"path/filepath"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/event/v4/eventtest"
	"github.com/larsartmann/go-cqrs-lite/storage/turso/v4"
)

var update = flag.Bool("update", false, "update golden files")

func TestGolden_ErrorMessages(t *testing.T) {
	errors := map[string]string{
		"ErrMemorySync": turso.ErrMemorySync.Error(),
	}

	got, err := json.MarshalIndent(errors, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	eventtest.AssertGolden(
		t,
		filepath.Join("testdata", "golden", "error-messages.json"),
		got,
		*update,
	)
}
