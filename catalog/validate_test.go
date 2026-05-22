package catalog

import (
	"strings"
	"testing"
)

func TestValidate_ValidCatalog(t *testing.T) {
	t.Parallel()

	cat := &Catalog{
		Title:   "Test",
		Version: "1.0.0",
		Services: []Service{
			{
				ID:       "user-svc",
				Name:     "User Service",
				Version:  "1.0.0",
				Commands: []Message{{ID: "create.user", Kind: CommandMessage, Name: "Create User"}},
			},
		},
	}

	violations := cat.Validate()
	if len(violations) != 0 {
		t.Fatalf("expected no violations, got %d: %v", len(violations), violations)
	}
}

func expectViolation(t *testing.T, cat *Catalog, wantPath string) {
	t.Helper()

	violations := cat.Validate()
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(violations))
	}

	if violations[0].Path != wantPath {
		t.Errorf("path = %q, want %s", violations[0].Path, wantPath)
	}
}

func TestValidate_EmptyTitle(t *testing.T) {
	t.Parallel()

	cat := &Catalog{Title: "", Version: "1.0.0"}
	expectViolation(t, cat, "title")
}

func TestValidate_EmptyVersion(t *testing.T) {
	t.Parallel()

	cat := &Catalog{Title: "Test", Version: ""}
	expectViolation(t, cat, "version")
}

func TestValidate_EmptyServiceID(t *testing.T) {
	t.Parallel()

	cat := &Catalog{
		Title:   "Test",
		Version: "1.0.0",
		Services: []Service{
			{Name: "Orphan Service"},
		},
	}

	violations := cat.Validate()
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d: %v", len(violations), violations)
	}

	if !strings.Contains(violations[0].Path, "services[") {
		t.Errorf("path = %q, want service path", violations[0].Path)
	}
}

func TestValidate_MessageWithoutIDOrName(t *testing.T) {
	t.Parallel()

	cat := &Catalog{
		Title:   "Test",
		Version: "1.0.0",
		Services: []Service{
			{
				ID:       "svc",
				Commands: []Message{{Kind: CommandMessage}},
			},
		},
	}

	violations := cat.Validate()
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d: %v", len(violations), violations)
	}

	if !strings.Contains(violations[0].Message, "must have an ID or Name") {
		t.Errorf("message = %q", violations[0].Message)
	}
}

func TestValidate_DuplicateMessageID(t *testing.T) {
	t.Parallel()

	cat := &Catalog{
		Title:   "Test",
		Version: "1.0.0",
		Services: []Service{
			{
				ID:       "svc-a",
				Commands: []Message{{ID: "create.user", Kind: CommandMessage, Name: "Create User"}},
			},
			{
				ID:     "svc-b",
				Events: []Message{{ID: "create.user", Kind: EventMessage, Name: "Create User"}},
			},
		},
	}

	violations := cat.Validate()
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d: %v", len(violations), violations)
	}

	if !strings.Contains(violations[0].Message, "duplicate") {
		t.Errorf("message = %q", violations[0].Message)
	}
}

func TestValidate_DomainDuplicateService(t *testing.T) {
	t.Parallel()

	cat := &Catalog{
		Title:   "Test",
		Version: "1.0.0",
		Domains: []Domain{
			{
				ID:       "orders",
				Name:     "Orders",
				Services: []ServiceID{"svc-a", "svc-a"},
			},
		},
	}

	violations := cat.Validate()
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d: %v", len(violations), violations)
	}

	if !strings.Contains(violations[0].Message, "duplicate service") {
		t.Errorf("message = %q", violations[0].Message)
	}
}

func TestViolation_String(t *testing.T) {
	t.Parallel()

	v := Violation{Path: "title", Message: "empty"}
	if v.String() != "title: empty" {
		t.Errorf("got %q", v.String())
	}
}

func TestViolation_Error(t *testing.T) {
	t.Parallel()

	v := Violation{Path: "version", Message: "missing"}
	if v.Error() != "version: missing" {
		t.Errorf("got %q", v.Error())
	}

	var err error = v
	if err.Error() != "version: missing" {
		t.Errorf("as error: got %q", err.Error())
	}
}

func TestValidate_Channel(t *testing.T) {
	t.Parallel()

	cat := &Catalog{
		Title:   "Test",
		Version: "1.0.0",
		Channels: []Channel{
			{ID: "ch1", Name: "Channel 1", Messages: []MessageID{"msg-a", "msg-a"}},
		},
	}

	violations := cat.Validate()
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d: %v", len(violations), violations)
	}

	if !strings.Contains(violations[0].Message, "duplicate message") {
		t.Errorf("message = %q", violations[0].Message)
	}
}

func TestValidate_ChannelEmptyID(t *testing.T) {
	t.Parallel()

	cat := &Catalog{
		Title:    "Test",
		Version:  "1.0.0",
		Channels: []Channel{{Name: "No ID"}},
	}

	violations := cat.Validate()
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d: %v", len(violations), violations)
	}

	if !strings.Contains(violations[0].Message, "channel ID must not be empty") {
		t.Errorf("message = %q", violations[0].Message)
	}
}
