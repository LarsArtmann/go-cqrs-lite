package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCoeffects_NoDanglingSubscriptionsPerSource (f33): each context's own
// catalog must pass the docs-side coeffect gate — consumed events name an
// external producer, so neither catalog dangles even though each sees only
// HALF the mesh.
func TestCoeffects_NoDanglingSubscriptionsPerSource(t *testing.T) {
	if violations := buildOrdersCatalog().ValidateCoeffects(); len(violations) != 0 {
		t.Fatalf("orders catalog has dangling coeffects: %v", violations)
	}

	if violations := buildBillingCatalog().ValidateCoeffects(); len(violations) != 0 {
		t.Fatalf("billing catalog has dangling coeffects: %v", violations)
	}
}

// manifestResource mirrors catalog.index.json's entry shape.
type manifestResource struct {
	ID      string `json:"id"`
	Kind    string `json:"kind"`
	Version string `json:"version"`
	Path    string `json:"path"`
}

type manifestDoc struct {
	SchemaVersion int                `json:"schemaVersion"`
	Resources     []manifestResource `json:"resources"`
}

func readManifest(t *testing.T, dir string) manifestDoc {
	t.Helper()

	data, err := os.ReadFile(filepath.Join(dir, "catalog.index.json"))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}

	var doc manifestDoc
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("unmarshal manifest: %v", err)
	}

	return doc
}

// TestUnionMerge_BilateralContractsCarryBothSides (f34): the dry-run hub
// merge. Both sources export; the shared events (order.placed,
// invoice.issued) are written by BOTH — the bilateral declarations mean
// either copy tells the whole relationship (producer AND consumer), so the
// hub can union them without losing a side.
func TestUnionMerge_BilateralContractsCarryBothSides(t *testing.T) {
	root := t.TempDir()
	ordersDir := filepath.Join(root, "orders")
	billingDir := filepath.Join(root, "billing")

	for _, tc := range []struct{ domain, dir string }{
		{"orders", ordersDir}, {"billing", billingDir},
	} {
		if err := exportDomain(tc.domain, tc.dir, exportOptions{skipBootstrap: true}); err != nil {
			t.Fatalf("export %s: %v", tc.domain, err)
		}
	}

	sharedEvent := func(dir, eventID string) string {
		t.Helper()

		data, err := os.ReadFile(filepath.Join(dir, "events", eventID, "index.mdx"))
		if err != nil {
			t.Fatalf("read %s from %s: %v", eventID, dir, err)
		}

		return string(data)
	}

	for _, eventID := range []string{"order.placed", "invoice.issued"} {
		for _, dir := range []string{ordersDir, billingDir} {
			frontmatter := sharedEvent(dir, eventID)
			for _, svc := range []string{"orders-svc", "billing-svc"} {
				if !strings.Contains(frontmatter, svc) {
					t.Errorf("%s's copy of %s must mention %s (bilateral contract):\n%s",
						filepath.Base(dir), eventID, svc, frontmatter)
				}
			}
		}
	}
}

// TestUnionMerge_ManifestsUnionWithoutResourceIDCollisions (f34): the union
// of both manifests must cover every resource exactly once per (kind, id)
// — except the shared events, which are the intentional bilateral overlap.
func TestUnionMerge_ManifestsUnionWithoutResourceIDCollisions(t *testing.T) {
	root := t.TempDir()

	for _, domain := range []string{"orders", "billing"} {
		if err := exportDomain(domain, filepath.Join(root, domain), exportOptions{}); err != nil {
			t.Fatalf("export %s: %v", domain, err)
		}
	}

	shared := map[string]bool{
		"events/order.placed":   true,
		"events/invoice.issued": true,
	}

	seen := make(map[string]string)

	for _, domain := range []string{"orders", "billing"} {
		for _, res := range readManifest(t, filepath.Join(root, domain)).Resources {
			key := res.Kind + "/" + res.ID

			if previous, dup := seen[key]; dup && !shared[key] {
				t.Errorf("resource %s duplicated across sources (%s vs %s)", key, previous, domain)
			}

			seen[key] = domain
		}
	}

	for _, want := range []string{
		"services/orders-svc", "services/billing-svc",
		"events/order.placed", "events/invoice.issued", "events/order.completed",
		"data-products/order-lifecycle", "data-products/billing-ledger",
		"domains/ordering", "domains/invoicing",
	} {
		if _, ok := seen[want]; !ok {
			t.Errorf("union manifest missing %s (union has %v)", want, keysOf(seen))
		}
	}
}

// TestExport_PlainRefsAndSkipBootstrapOptionsWork pins the governance
// export flags end-to-end through the example's own export command path.
func TestExport_PlainRefsAndSkipBootstrapOptionsWork(t *testing.T) {
	dir := t.TempDir()

	if err := exportDomain(
		"orders",
		dir,
		exportOptions{plain: true, skipBootstrap: true},
	); err != nil {
		t.Fatalf("export: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "package.json")); !os.IsNotExist(err) {
		t.Errorf("skip-bootstrap export must not write package.json")
	}

	eventMDX, err := os.ReadFile(filepath.Join(dir, "events", "invoice.issued", "index.mdx"))
	if err != nil {
		t.Fatalf("read consumed event: %v", err)
	}

	if !strings.Contains(string(eventMDX), "- billing-svc") {
		t.Errorf("plain export should carry bare producer refs:\n%s", eventMDX)
	}

	if strings.Contains(string(eventMDX), "billing-svc-1.0.0") {
		t.Errorf("plain export must not carry composite refs:\n%s", eventMDX)
	}
}

// TestExport_GovernanceLintContract pins the per-source hygiene the hub's
// governance lint (@eventcatalog/linter on a plain-refs re-export) demands:
// every declared data-product contract FILE must exist in the export (the
// exporter copies nothing), and every team member must resolve to a declared
// user. Cross-source refs are exempt per-source (they resolve only in the
// hub's merged union — enforced by merge.py's strict gate), so these two
// intra-source invariants are what keeps a source lint-clean.
func TestExport_GovernanceLintContract(t *testing.T) {
	for _, tc := range []struct {
		domain        string
		contractPath  string
		memberUserMDX string
	}{
		{"orders", "data-products/order-lifecycle/contracts/order-placed.yaml", "users/marta.mdx"},
		{"billing", "data-products/billing-ledger/contracts/invoice-issued.yaml", "users/juno.mdx"},
	} {
		dir := t.TempDir()
		if err := exportDomain(
			tc.domain,
			dir,
			exportOptions{plain: true, skipBootstrap: true},
		); err != nil {
			t.Fatalf("export %s: %v", tc.domain, err)
		}

		for _, path := range []string{tc.contractPath, tc.memberUserMDX} {
			if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(path))); err != nil {
				t.Errorf(
					"%s export missing %s (governance lint would fail): %v",
					tc.domain,
					path,
					err,
				)
			}
		}
	}
}

func keysOf(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}

	return out
}
