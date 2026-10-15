package eventcatalog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
	"github.com/larsartmann/go-cqrs-lite/catalog/v4/internal/cattest"
)

func fixtureADR() catalog.ADR {
	return catalog.ADR{
		ID:      "sqlite-single-writer",
		Name:    "SQLite single-writer event store",
		Version: "1.0.0",
		Status:  catalog.ADRAccepted,
		Date:    "2026-10-15",
		Summary: "One writer per database file keeps the sync window simple",
		Owners:  []string{"lars"},
		DecisionMakers: []string{
			"lars",
		},
		AppliesTo: []catalog.ADRResourceRef{
			{Type: catalog.ADRAppliesToService, ID: "bank-sync", Version: "1.0.0"},
			{Type: catalog.ADRAppliesToEvent, ID: "BalanceUpdated"},
		},
		Supersedes:   []catalog.ADRRef{{ID: "sqlite-async-replication", Version: "0.9.0"}},
		SupersededBy: nil,
		Amends:       []catalog.ADRRef{{ID: "event-store-baseline"}},
		AmendedBy:    []catalog.ADRRef{{ID: "snapshot-every-100"}},
		Related:      []catalog.ADRRef{{ID: "wise-qonto-providers"}},
		Badges: []catalog.Badge{
			{Content: "Accepted", BackgroundColor: "green", TextColor: "white"},
		},
		Body: "## Context\n\nOne sync process writes balances.\n\n## Decision\n\nSingle writer, no row locks.\n\n## Consequences\n\nNo lock contention.",
	}
}

func TestExporter_ADR(t *testing.T) {
	t.Parallel()

	reg := cattest.NewTestRegistry()
	reg.AddADR(fixtureADR())

	tmpDir := exportCatalog(t, reg)

	content := readExported(t, tmpDir, "adrs", "sqlite-single-writer", "index.mdx")
	cattest.AssertContentContains(
		t, content, "adr mdx",
		"id: sqlite-single-writer",
		"name: SQLite single-writer event store",
		"version: 1.0.0",
		"status: accepted",
		"2026-10-15",
		"One writer per database file",
		"owners:",
		"lars",
		"decisionMakers:",
		"appliesTo:",
		"type: service",
		"id: bank-sync",
		"version: 1.0.0",
		"type: event",
		"id: BalanceUpdated",
		"supersedes:",
		"id: sqlite-async-replication",
		"amends:",
		"amendedBy:",
		"related:",
		"content: Accepted",
		"# SQLite single-writer event store",
		"## Context",
		"## Decision",
		"## Consequences",
	)
}

func TestExporter_ADRPlainRefsDropVersions(t *testing.T) {
	t.Parallel()

	reg := cattest.NewTestRegistry()
	reg.AddADR(fixtureADR())

	tmpDir := t.TempDir()
	cat := reg.Build()
	exp := NewExporter(tmpDir, WithPlainRefIDs())
	if err := exp.Export(cat); err != nil {
		t.Fatal(err)
	}

	content := readExported(t, tmpDir, "adrs", "sqlite-single-writer", "index.mdx")

	if !strings.Contains(content, "appliesTo:") {
		t.Fatalf("plain export should keep appliesTo list\n%s", content)
	}

	if strings.Contains(content, "version: 0.9.0") {
		t.Errorf("plain export should drop version pins from ADR pointers\n%s", content)
	}

	bankSyncIdx := strings.Index(content, "id: bank-sync")
	if bankSyncIdx < 0 {
		t.Fatalf("plain export should keep pointer IDs\n%s", content)
	}

	if strings.Contains(content[bankSyncIdx:bankSyncIdx+40], "version:") {
		t.Errorf("plain export appliesTo pointer should not carry a version pin\n%s", content)
	}
}

func TestExporter_ADRRejectsInvalidFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		mutate  func(*catalog.ADR)
		wantErr string
	}{
		{
			name:    "empty id",
			mutate:  func(a *catalog.ADR) { a.ID = "" },
			wantErr: "id is required",
		},
		{
			name:    "empty name",
			mutate:  func(a *catalog.ADR) { a.Name = "" },
			wantErr: "name is required",
		},
		{
			name:    "empty version",
			mutate:  func(a *catalog.ADR) { a.Version = "" },
			wantErr: "version is required",
		},
		{
			name:    "unknown status",
			mutate:  func(a *catalog.ADR) { a.Status = "decided" },
			wantErr: `status "decided" is not one of`,
		},
		{
			name:    "timestamp date",
			mutate:  func(a *catalog.ADR) { a.Date = "2026-10-15T10:00:00Z" },
			wantErr: "must be a calendar date in YYYY-MM-DD form",
		},
		{
			name: "unknown appliesTo type",
			mutate: func(a *catalog.ADR) {
				a.AppliesTo = []catalog.ADRResourceRef{{Type: "message", ID: "BalanceUpdated"}}
			},
			wantErr: `appliesTo type "message" is not an EventCatalog resource kind`,
		},
	}

	for _, tt := range tests { //nolint:dupl // table bodies differ only in mutation
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			adr := fixtureADR()
			tt.mutate(&adr)

			reg := cattest.NewTestRegistry()
			reg.AddADR(adr)

			exp := NewExporter(t.TempDir())

			err := exp.Export(reg.Build())
			if err == nil {
				t.Fatalf("expected export error containing %q, got nil", tt.wantErr)
			}

			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error %q does not contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestExporter_ADRInManifestAndLLMsTxt(t *testing.T) {
	t.Parallel()

	reg := cattest.NewTestRegistry()
	reg.AddADR(fixtureADR())

	tmpDir := exportCatalog(t, reg)

	manifest, err := os.ReadFile(filepath.Join(tmpDir, "catalog.index.json"))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}

	if !strings.Contains(string(manifest), `"kind": "adrs"`) {
		t.Errorf("manifest should list the ADR under kind adrs\n%s", string(manifest))
	}

	if !strings.Contains(string(manifest), `"id": "sqlite-single-writer"`) {
		t.Errorf("manifest should carry the ADR id\n%s", string(manifest))
	}

	llms, err := os.ReadFile(filepath.Join(tmpDir, "llms.txt"))
	if err != nil {
		t.Fatalf("read llms.txt: %v", err)
	}

	if !strings.Contains(string(llms), "ADR: SQLite single-writer event store (sqlite-single-writer)") {
		t.Errorf("llms.txt should carry the ADR section\n%s", string(llms))
	}

	if !strings.Contains(string(llms), "Status: accepted | Date: 2026-10-15") {
		t.Errorf("llms.txt should carry the ADR status and date\n%s", string(llms))
	}
}
