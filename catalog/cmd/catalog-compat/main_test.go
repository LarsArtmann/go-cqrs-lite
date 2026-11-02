package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
	"github.com/larsartmann/go-cqrs-lite/catalog/v4/schema"
)

func eventSchema(props ...string) *schema.Schema {
	properties := make(map[string]schema.Property, len(props))
	for _, name := range props {
		properties[name] = schema.Property{Type: schema.TypeString}
	}

	return &schema.Schema{Type: schema.TypeObject, Properties: properties}
}

func testCatalog(version string, props ...string) *catalog.Catalog {
	return &catalog.Catalog{
		Title:   "orders",
		Version: "1.0.0",
		Services: []catalog.Service{{
			ID:      "orders",
			Name:    "Orders",
			Version: "1.0.0",
			Events: []catalog.Message{{
				Kind:      catalog.EventMessage,
				ID:        "order_placed",
				Name:      "order_placed",
				Version:   catalog.Version(version),
				Direction: catalog.Sends,
				Schema:    eventSchema(props...),
			}},
		}},
	}
}

func writeJSON(t *testing.T, dir, name string, value any) string {
	t.Helper()

	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal %s: %v", name, err)
	}

	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}

	return path
}

func TestRunUsageAndHelp(t *testing.T) {
	t.Parallel()

	var out, errOut bytes.Buffer

	if code := run(nil, &out, &errOut); code != exitUsage {
		t.Fatalf("no args: exit = %d, want %d", code, exitUsage)
	}

	if !strings.Contains(errOut.String(), "catalog-compat") {
		t.Fatalf("usage not printed: %q", errOut.String())
	}

	out.Reset()
	errOut.Reset()

	if code := run([]string{"help"}, &out, &errOut); code != exitOK {
		t.Fatalf("help: exit = %d, want %d", code, exitOK)
	}

	if !strings.Contains(out.String(), "baseline") {
		t.Fatalf("help text missing subcommands: %q", out.String())
	}

	if code := run([]string{"nope"}, &out, &errOut); code != exitUsage {
		t.Fatalf("unknown command: exit = %d, want %d", code, exitUsage)
	}
}

func TestRunBaselineThenCheck(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	catalogPath := writeJSON(t, dir, "catalog.json", testCatalog("1.0.0", "a"))
	baselinePath := filepath.Join(dir, "compat-baseline.json")

	var out, errOut bytes.Buffer

	code := run(
		[]string{"baseline", "--catalog", catalogPath, "--out", baselinePath},
		&out,
		&errOut,
	)
	if code != exitOK {
		t.Fatalf("baseline: exit = %d (%s)", code, errOut.String())
	}

	if _, err := os.Stat(baselinePath); err != nil {
		t.Fatalf("baseline not written: %v", err)
	}

	out.Reset()
	errOut.Reset()

	code = run(
		[]string{"check", "--catalog", catalogPath, "--baseline", baselinePath},
		&out,
		&errOut,
	)
	if code != exitOK {
		t.Fatalf("clean check: exit = %d (%s)", code, errOut.String())
	}

	if !strings.Contains(out.String(), "unchanged") {
		t.Fatalf("check output should report the class: %q", out.String())
	}
}

func TestRunCheckFailsOnBreakingChangeWithoutVersionBump(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	baselinePath := filepath.Join(dir, "compat-baseline.json")

	var out, errOut bytes.Buffer

	if code := run(
		[]string{
			"baseline",
			"--catalog",
			writeJSON(t, dir, "old.json", testCatalog("1.0.0", "a", "b")),
			"--out",
			baselinePath,
		},
		&out,
		&errOut,
	); code != exitOK {
		t.Fatalf("baseline: exit = %d (%s)", code, errOut.String())
	}

	out.Reset()
	errOut.Reset()

	code := run(
		[]string{
			"check",
			"--catalog",
			writeJSON(t, dir, "new.json", testCatalog("1.0.0", "a")),
			"--baseline",
			baselinePath,
		},
		&out,
		&errOut,
	)

	if code != exitFail {
		t.Fatalf("breaking check: exit = %d, want %d", code, exitFail)
	}

	if !strings.Contains(errOut.String(), "::error::order_placed") {
		t.Fatalf("stderr must carry an actionable ::error:: line: %q", errOut.String())
	}
}

func TestRunCheckMissingBaseline(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	catalogPath := writeJSON(t, dir, "catalog.json", testCatalog("1.0.0", "a"))

	var out, errOut bytes.Buffer

	code := run(
		[]string{
			"check",
			"--catalog",
			catalogPath,
			"--baseline",
			filepath.Join(dir, "absent.json"),
		},
		&out,
		&errOut,
	)

	if code != exitUsage {
		t.Fatalf("missing baseline: exit = %d, want %d", code, exitUsage)
	}

	if !strings.Contains(errOut.String(), "absent.json") {
		t.Fatalf("error must name the file: %q", errOut.String())
	}
}

func TestRunValidate(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	var out, errOut bytes.Buffer

	cleanPath := writeJSON(t, dir, "clean.json", testCatalog("1.0.0", "a"))

	if code := run([]string{"validate", "--catalog", cleanPath}, &out, &errOut); code != exitOK {
		t.Fatalf("clean validate: exit = %d (%s)", code, errOut.String())
	}

	dirty := testCatalog("1.0.0", "a")
	dirty.Services[0].Events[0].Channels = []catalog.ChannelID{"ghost"}

	out.Reset()
	errOut.Reset()

	if code := run(
		[]string{"validate", "--catalog", writeJSON(t, dir, "dirty.json", dirty)},
		&out,
		&errOut,
	); code != exitFail {
		t.Fatalf("dirty validate: exit = %d, want %d", code, exitFail)
	}

	if !strings.Contains(out.String(), "ghost") {
		t.Fatalf("violation not reported: %q", out.String())
	}
}

func TestRunConflicts(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	left := writeJSON(t, dir, "left.json", testCatalog("1.0.0", "a"))
	right := writeJSON(t, dir, "right.json", testCatalog("2.0.0", "a"))

	var out, errOut bytes.Buffer

	if code := run(
		[]string{"conflicts", "--left", left, "--right", right},
		&out,
		&errOut,
	); code != exitFail {
		t.Fatalf("conflicting catalogs: exit = %d, want %d", code, exitFail)
	}

	if !strings.Contains(out.String(), "version-mismatch") {
		t.Fatalf("conflict not reported: %q", out.String())
	}

	out.Reset()
	errOut.Reset()

	if code := run(
		[]string{"conflicts", "--left", left, "--right", left},
		&out,
		&errOut,
	); code != exitOK {
		t.Fatalf("self-union: exit = %d (%s)", code, errOut.String())
	}
}

func TestRunRequiresCatalogFlag(t *testing.T) {
	t.Parallel()

	var out, errOut bytes.Buffer

	if code := run([]string{"baseline"}, &out, &errOut); code != exitUsage {
		t.Fatalf("missing --catalog: exit = %d, want %d", code, exitUsage)
	}

	if !strings.Contains(errOut.String(), "--catalog") {
		t.Fatalf("error must name the flag: %q", errOut.String())
	}
}

func TestRunUnreadableCatalog(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	bad := filepath.Join(dir, "bad.json")

	if err := os.WriteFile(bad, []byte("{not json"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	var out, errOut bytes.Buffer

	if code := run([]string{"validate", "--catalog", bad}, &out, &errOut); code != exitUsage {
		t.Fatalf("malformed catalog: exit = %d, want %d", code, exitUsage)
	}
}
