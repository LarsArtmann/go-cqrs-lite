package turso_test

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/turso/v2"
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

	assertTursoGolden(t, filepath.Join("testdata", "golden", "error-messages.json"), got)
}

func assertTursoGolden(t *testing.T, path string, got []byte) {
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
