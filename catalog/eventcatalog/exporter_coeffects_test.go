package eventcatalog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
)

func coeffectExportFixture() *catalog.Catalog {
	return &catalog.Catalog{
		Title:   "Coeffect Export",
		Version: "1.0.0",
		Services: []catalog.Service{
			{
				ID:      "user-svc",
				Name:    "User Service",
				Version: "1.0.0",
				Events: []catalog.Message{
					{
						ID:        "user.created",
						Kind:      catalog.EventMessage,
						Name:      "User Created",
						Direction: catalog.Sends,
					},
				},
			},
			{
				ID:      "audit-svc",
				Name:    "Audit Service",
				Version: "1.0.0",
				Events: []catalog.Message{
					{
						ID:        "user.created",
						Kind:      catalog.EventMessage,
						Name:      "User Created",
						Direction: catalog.Receives,
					},
					{
						ID:        "user.creted",
						Kind:      catalog.EventMessage,
						Name:      "User Creted",
						Direction: catalog.Receives,
					},
				},
			},
		},
	}
}

// The export carries the coeffect validation summary: the graph table plus
// the dangling count from Catalog.ValidateCoeffects.
func TestExporter_WritesCoeffectSummary(t *testing.T) {
	t.Parallel()

	out := t.TempDir()
	if err := NewExporter(out).Export(coeffectExportFixture()); err != nil {
		t.Fatalf("export: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(out, "coeffects.md"))
	if err != nil {
		t.Fatalf("read coeffects.md: %v", err)
	}

	summary := string(data)

	for _, want := range []string{
		"user.created",
		"user-svc",
		"audit-svc",
		"DANGLING",
		"user.creted",
		"1 dangling subscription(s)",
	} {
		if !strings.Contains(summary, want) {
			t.Errorf("coeffects.md missing %q:\n%s", want, summary)
		}
	}
}

// A fully-connected catalog renders the clean summary.
func TestExporter_CoeffectSummaryClean(t *testing.T) {
	t.Parallel()

	cat := &catalog.Catalog{
		Title:   "Clean",
		Version: "1.0.0",
		Services: []catalog.Service{
			{
				ID:      "user-svc",
				Name:    "User Service",
				Version: "1.0.0",
				Events: []catalog.Message{
					{
						ID:        "user.created",
						Kind:      catalog.EventMessage,
						Name:      "User Created",
						Direction: catalog.Sends,
					},
				},
			},
			{
				ID:      "audit-svc",
				Name:    "Audit Service",
				Version: "1.0.0",
				Events: []catalog.Message{
					{
						ID:        "user.created",
						Kind:      catalog.EventMessage,
						Name:      "User Created",
						Direction: catalog.Receives,
					},
				},
			},
		},
	}

	out := t.TempDir()
	if err := NewExporter(out).Export(cat); err != nil {
		t.Fatalf("export: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(out, "coeffects.md"))
	if err != nil {
		t.Fatalf("read coeffects.md: %v", err)
	}

	if !strings.Contains(string(data), "no dangling subscriptions") {
		t.Errorf("expected clean summary, got:\n%s", string(data))
	}
}
