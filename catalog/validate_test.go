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

func expectViolationMessage(t *testing.T, cat *Catalog, wantSubstring string) {
	t.Helper()

	violations := cat.Validate()
	if len(violations) != 1 {
		t.Fatalf("expected 1 violation, got %d: %v", len(violations), violations)
	}

	if !strings.Contains(violations[0].Message, wantSubstring) {
		t.Errorf("message = %q", violations[0].Message)
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

	expectViolationMessage(t, cat, "must have an ID or Name")
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

	expectViolationMessage(t, cat, "duplicate")
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

	expectViolationMessage(t, cat, "duplicate service")
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

	expectViolationMessage(t, cat, "duplicate message")
}

func TestValidate_ChannelEmptyID(t *testing.T) {
	t.Parallel()

	cat := &Catalog{
		Title:    "Test",
		Version:  "1.0.0",
		Channels: []Channel{{Name: "No ID"}},
	}

	expectViolationMessage(t, cat, "channel ID must not be empty")
}

func TestValidate_EntityPropertyWithoutName(t *testing.T) {
	t.Parallel()

	cat := &Catalog{
		Title:   "Test",
		Version: "1.0.0",
		Entities: []Entity{
			{ID: "e1", Name: "E1", Properties: []EntityProperty{{Type: "string"}}},
		},
	}

	violations := cat.Validate()
	found := false
	for _, v := range violations {
		if strings.Contains(v.Message, "must have a name") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected property name violation, got: %v", violations)
	}
}

func TestValidate_EntityPropertyWithoutType(t *testing.T) {
	t.Parallel()

	cat := &Catalog{
		Title:   "Test",
		Version: "1.0.0",
		Entities: []Entity{
			{ID: "e1", Name: "E1", Properties: []EntityProperty{{Name: "id"}}},
		},
	}

	violations := cat.Validate()
	found := false
	for _, v := range violations {
		if strings.Contains(v.Message, "must have a type") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected property type violation, got: %v", violations)
	}
}

func TestValidate_EntityDuplicateProperty(t *testing.T) {
	t.Parallel()

	cat := &Catalog{
		Title:   "Test",
		Version: "1.0.0",
		Entities: []Entity{
			{ID: "e1", Name: "E1", Properties: []EntityProperty{
				{Name: "id", Type: "string"},
				{Name: "id", Type: "string"},
			}},
		},
	}

	violations := cat.Validate()
	found := false
	for _, v := range violations {
		if strings.Contains(v.Message, "duplicate") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected duplicate property violation, got: %v", violations)
	}
}

func TestValidate_DuplicateOperationPath(t *testing.T) {
	t.Parallel()

	cat := &Catalog{
		Title:   "Test",
		Version: "1.0.0",
		Services: []Service{
			{
				ID: "svc",
				Commands: []Message{
					{
						ID:        "cmd.a",
						Kind:      CommandMessage,
						Name:      "A",
						Operation: &Operation{Method: "POST", Path: "/api/items"},
					},
					{
						ID:        "cmd.b",
						Kind:      CommandMessage,
						Name:      "B",
						Operation: &Operation{Method: "POST", Path: "/api/items"},
					},
				},
			},
		},
	}

	violations := cat.Validate()
	found := false
	for _, v := range violations {
		if strings.Contains(v.Message, "duplicate operation") &&
			strings.Contains(v.Message, "POST /api/items") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected duplicate operation violation, got: %v", violations)
	}
}

func TestValidate_OperationMethodWithoutPath(t *testing.T) {
	t.Parallel()

	cat := &Catalog{
		Title:   "Test",
		Version: "1.0.0",
		Services: []Service{
			{
				ID: "svc",
				Commands: []Message{
					{
						ID:        "cmd.a",
						Kind:      CommandMessage,
						Name:      "A",
						Operation: &Operation{Method: "POST", Path: ""},
					},
				},
			},
		},
	}

	violations := cat.Validate()
	found := false
	for _, v := range violations {
		if strings.Contains(v.Message, "method is set but path is empty") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected method-without-path violation, got: %v", violations)
	}
}

func TestValidate_Response2xxWithoutSchema(t *testing.T) {
	t.Parallel()

	cat := &Catalog{
		Title:   "Test",
		Version: "1.0.0",
		Services: []Service{
			{
				ID: "svc",
				Commands: []Message{
					{
						ID:   "cmd.a",
						Kind: CommandMessage,
						Name: "A",
						Responses: []ResponseSpec{
							{StatusCode: "200", Description: "OK"},
						},
					},
				},
			},
		},
	}

	violations := cat.Validate()
	found := false
	for _, v := range violations {
		if strings.Contains(v.Message, "2xx response has no body schema") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected 2xx-without-schema violation, got: %v", violations)
	}
}

func TestValidate_Response2xxWithSchema_NoViolation(t *testing.T) {
	t.Parallel()

	cat := &Catalog{
		Title:   "Test",
		Version: "1.0.0",
		Services: []Service{
			{
				ID: "svc",
				Commands: []Message{
					{
						ID:   "cmd.a",
						Kind: CommandMessage,
						Name: "A",
						Responses: []ResponseSpec{
							{
								StatusCode:  "200",
								Description: "OK",
								Schema:      &Schema{Type: TypeObject},
							},
						},
					},
				},
			},
		},
	}

	violations := cat.Validate()
	for _, v := range violations {
		if strings.Contains(v.Message, "2xx response has no body schema") {
			t.Fatalf("did not expect 2xx-without-schema violation, got: %v", v)
		}
	}
}
