package catalog

import (
	"fmt"
	"strings"
	"testing"
)

// stubT records Errorf calls so drift messages can be asserted without a
// real testing.T.
type stubT struct {
	helperCalls int
	errors      []string
}

func (s *stubT) Helper() { s.helperCalls++ }

func (s *stubT) Errorf(format string, args ...any) {
	s.errors = append(s.errors, strings.TrimSpace(fmt.Sprintf(format, args...)))
}

func fixtureInventoryCatalog() *Catalog {
	b := NewBuilder("Fixture", "1.0.0")
	b.AddService("fixture", "Fixture", "1.0.0", "",
		CommandFromSchema("cmd.one", nil),
		EventFromSchema("evt.one", Sends, nil),
		QueryFromSchema("qry.one", nil),
	)

	return b.Build()
}

func TestAssertInventory_Pass(t *testing.T) {
	cat := fixtureInventoryCatalog()

	AssertInventory(t, cat, "cmd.one", "evt.one", "qry.one")
	AssertCommandInventory(t, cat, "cmd.one")
	AssertEventInventory(t, cat, "evt.one")
	AssertQueryInventory(t, cat, "qry.one")
}

func TestAssertInventory_DriftReportsMissingAndUnexpected(t *testing.T) {
	stub := &stubT{}

	AssertInventory(stub, fixtureInventoryCatalog(), "cmd.one", "evt.one", "cmd.gone")

	if len(stub.errors) != 1 {
		t.Fatalf("errors = %d (%v), want exactly 1 drift report", len(stub.errors), stub.errors)
	}

	report := stub.errors[0]

	for _, want := range []string{
		"inventory drifted",
		"missing",
		"cmd.gone",
		"unexpected",
		"qry.one",
		"fix:",
	} {
		if !strings.Contains(report, want) {
			t.Errorf("drift report missing %q:\n%s", want, report)
		}
	}
}

func TestAssertInventory_EmptyCatalogEmptyWant(t *testing.T) {
	empty := NewBuilder("Empty", "1.0.0").Build()
	AssertInventory(t, empty)
}

func TestAssertInventory_NilCatalog(t *testing.T) {
	stub := &stubT{}
	AssertInventory(stub, nil, "anything")

	if len(stub.errors) != 1 || !strings.Contains(stub.errors[0], "nil") {
		t.Errorf("nil catalog must error clearly, got %v", stub.errors)
	}
}

func TestAssertInventory_KindVariantsAreScoped(t *testing.T) {
	stub := &stubT{}
	AssertCommandInventory(stub, fixtureInventoryCatalog(), "cmd.one", "evt.one")

	if len(stub.errors) != 1 || !strings.Contains(stub.errors[0], "evt.one") {
		t.Errorf("command variant must not accept event ids, got %v", stub.errors)
	}
}
