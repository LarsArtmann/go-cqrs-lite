package graph

import (
	"errors"
	"testing"
)

func testSchema() *Schema {
	return &Schema{
		Nodes: []NodeType{
			{Label: "User", KeyProp: "id", Properties: []PropertyType{
				{Name: "name"}, {Name: "email"},
			}},
			{Label: "Message", KeyProp: "id", Properties: []PropertyType{
				{Name: "content"}, {Name: "created_at"},
			}},
		},
		Edges: []EdgeType{
			{Type: "AUTHORED_BY", FromLabel: "Message", ToLabel: "User"},
			{Type: "REPLY_TO", FromLabel: "Message", ToLabel: "Message", Properties: []PropertyType{
				{Name: "at"},
			}},
		},
	}
}

func TestSchema_Validate_Valid(t *testing.T) {
	t.Parallel()

	s := testSchema()
	if err := s.Validate(); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestSchema_Validate_NoNodeTypes(t *testing.T) {
	t.Parallel()

	s := &Schema{}
	if err := s.Validate(); !errors.Is(err, errSchemaNoNodeTypes) {
		t.Fatalf("expected errSchemaNoNodeTypes, got %v", err)
	}
}

func TestSchema_Validate_DuplicateNodeLabel(t *testing.T) {
	t.Parallel()

	s := &Schema{
		Nodes: []NodeType{
			{Label: "User", KeyProp: "id"},
			{Label: "User", KeyProp: "id"},
		},
	}
	if err := s.Validate(); !errors.Is(err, errSchemaDuplicateNodeLabel) {
		t.Fatalf("expected errSchemaDuplicateNodeLabel, got %v", err)
	}
}

func TestSchema_Validate_DuplicateEdgeType(t *testing.T) {
	t.Parallel()

	s := &Schema{
		Nodes: []NodeType{
			{Label: "User", KeyProp: "id"},
		},
		Edges: []EdgeType{
			{Type: "KNOWS", FromLabel: "User", ToLabel: "User"},
			{Type: "KNOWS", FromLabel: "User", ToLabel: "User"},
		},
	}
	if err := s.Validate(); !errors.Is(err, errSchemaDuplicateEdgeType) {
		t.Fatalf("expected errSchemaDuplicateEdgeType, got %v", err)
	}
}

func TestSchema_Validate_KeyPropInProperties(t *testing.T) {
	t.Parallel()

	s := &Schema{
		Nodes: []NodeType{
			{Label: "User", KeyProp: "id", Properties: []PropertyType{{Name: "id"}}},
		},
	}
	if err := s.Validate(); !errors.Is(err, errSchemaKeyPropInProperties) {
		t.Fatalf("expected errSchemaKeyPropInProperties, got %v", err)
	}
}

func TestSchema_Validate_EdgeReferencesUnknownLabel(t *testing.T) {
	t.Parallel()

	s := &Schema{
		Nodes: []NodeType{
			{Label: "User", KeyProp: "id"},
		},
		Edges: []EdgeType{
			{Type: "AUTHORED_BY", FromLabel: "Message", ToLabel: "User"},
		},
	}
	if err := s.Validate(); !errors.Is(err, errSchemaUnknownFromLabel) {
		t.Fatalf("expected errSchemaUnknownFromLabel, got %v", err)
	}
}

func TestSchemaSink_AcceptsValidWrite(t *testing.T) {
	t.Parallel()

	driver := NewMemoryDriver(WithDriverSchema(testSchema()))

	err := driver.RunInTx(func(sink GraphSink) error {
		return sink.MergeNode(
			NodeRef{Label: "User", KeyProp: "id", KeyValue: "u1"},
			map[string]any{"name": "alice", "email": "a@b.com"},
		)
	})
	if err != nil {
		t.Fatalf("valid write rejected: %v", err)
	}
}

func TestSchemaSink_RejectsUnknownNodeLabel(t *testing.T) {
	t.Parallel()

	driver := NewMemoryDriver(WithDriverSchema(testSchema()))

	err := driver.RunInTx(func(sink GraphSink) error {
		return sink.MergeNode(
			NodeRef{Label: "Phantom", KeyProp: "id", KeyValue: "x"},
			nil,
		)
	})
	if !errors.Is(err, errSinkUnknownNodeLabel) {
		t.Fatalf("expected errSinkUnknownNodeLabel, got %v", err)
	}
}

func TestSchemaSink_RejectsUnknownProp(t *testing.T) {
	t.Parallel()

	driver := NewMemoryDriver(WithDriverSchema(testSchema()))

	err := driver.RunInTx(func(sink GraphSink) error {
		return sink.MergeNode(
			NodeRef{Label: "User", KeyProp: "id", KeyValue: "u1"},
			map[string]any{"bogus": "value"},
		)
	})
	if !errors.Is(err, errSinkUnknownProp) {
		t.Fatalf("expected errSinkUnknownProp, got %v", err)
	}
}

func TestSchemaSink_RejectsWrongKeyProp(t *testing.T) {
	t.Parallel()

	driver := NewMemoryDriver(WithDriverSchema(testSchema()))

	err := driver.RunInTx(func(sink GraphSink) error {
		return sink.MergeNode(
			NodeRef{Label: "User", KeyProp: "email", KeyValue: "a@b.com"},
			nil,
		)
	})
	if !errors.Is(err, errSinkWrongKeyProp) {
		t.Fatalf("expected errSinkWrongKeyProp, got %v", err)
	}
}

func TestSchemaSink_RejectsEdgeEndpointMismatch(t *testing.T) {
	t.Parallel()

	driver := NewMemoryDriver(WithDriverSchema(testSchema()))

	err := driver.RunInTx(func(sink GraphSink) error {
		return sink.MergeEdge(EdgeRef{
			Type: "AUTHORED_BY",
			From: NodeRef{Label: "User", KeyProp: "id", KeyValue: "u1"},
			To:   NodeRef{Label: "User", KeyProp: "id", KeyValue: "u2"},
		}, nil)
	})
	if !errors.Is(err, errSinkEdgeEndpointMismatch) {
		t.Fatalf("expected errSinkEdgeEndpointMismatch, got %v", err)
	}
}

func TestSchemaSink_RejectsUnknownEdgeType(t *testing.T) {
	t.Parallel()

	driver := NewMemoryDriver(WithDriverSchema(testSchema()))

	err := driver.RunInTx(func(sink GraphSink) error {
		return sink.MergeEdge(EdgeRef{
			Type: "KNOWS",
			From: NodeRef{Label: "User", KeyProp: "id", KeyValue: "u1"},
			To:   NodeRef{Label: "User", KeyProp: "id", KeyValue: "u2"},
		}, nil)
	})
	if !errors.Is(err, errSinkUnknownEdgeType) {
		t.Fatalf("expected errSinkUnknownEdgeType, got %v", err)
	}
}

func TestSchemaSink_RejectsSetNodePropertyUnknownProp(t *testing.T) {
	t.Parallel()

	driver := NewMemoryDriver(WithDriverSchema(testSchema()))

	_ = driver.RunInTx(func(sink GraphSink) error {
		return sink.MergeNode(
			NodeRef{Label: "User", KeyProp: "id", KeyValue: "u1"},
			map[string]any{"name": "alice"},
		)
	})

	err := driver.RunInTx(func(sink GraphSink) error {
		return sink.SetNodeProperty(
			NodeRef{Label: "User", KeyProp: "id", KeyValue: "u1"},
			"bogus", true,
		)
	})
	if !errors.Is(err, errSinkUnknownProp) {
		t.Fatalf("expected errSinkUnknownProp, got %v", err)
	}
}

func TestSchemaSink_NilSchemaBackwardCompatible(t *testing.T) {
	t.Parallel()

	driver := NewMemoryDriver()

	err := driver.RunInTx(func(sink GraphSink) error {
		return sink.MergeNode(
			NodeRef{Label: "Anything", KeyProp: "any", KeyValue: "1"},
			map[string]any{"bogus": "value"},
		)
	})
	if err != nil {
		t.Fatalf("nil schema should accept anything, got: %v", err)
	}
}

func TestSchemaSink_ValidEdgeWithProps(t *testing.T) {
	t.Parallel()

	driver := NewMemoryDriver(WithDriverSchema(testSchema()))

	err := driver.RunInTx(func(sink GraphSink) error {
		if err := sink.MergeNode(
			NodeRef{Label: "Message", KeyProp: "id", KeyValue: "m1"}, nil,
		); err != nil {
			return err
		}

		return sink.MergeEdge(EdgeRef{
			Type: "REPLY_TO",
			From: NodeRef{Label: "Message", KeyProp: "id", KeyValue: "m1"},
			To:   NodeRef{Label: "Message", KeyProp: "id", KeyValue: "m0"},
		}, map[string]any{"at": "2024-01-01"})
	})
	if err != nil {
		t.Fatalf("valid edge with props rejected: %v", err)
	}
}

func TestSchemaSink_RejectsEdgeUnknownProp(t *testing.T) {
	t.Parallel()

	driver := NewMemoryDriver(WithDriverSchema(testSchema()))

	err := driver.RunInTx(func(sink GraphSink) error {
		return sink.MergeEdge(EdgeRef{
			Type: "REPLY_TO",
			From: NodeRef{Label: "Message", KeyProp: "id", KeyValue: "m1"},
			To:   NodeRef{Label: "Message", KeyProp: "id", KeyValue: "m0"},
		}, map[string]any{"bogus": true})
	})
	if !errors.Is(err, errSinkUnknownProp) {
		t.Fatalf("expected errSinkUnknownProp, got %v", err)
	}
}

func TestSchema_ValidIndex(t *testing.T) {
	t.Parallel()

	s := testSchema()
	s.Indexes = []IndexSpec{
		{Name: "msg_content", Label: "Message", Properties: []string{"content"}},
		{Name: "msg_by_id", Label: "Message", Properties: []string{"id"}}, // key prop allowed
	}
	if err := s.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSchema_RejectsEmptyIndexName(t *testing.T) {
	t.Parallel()

	s := testSchema()
	s.Indexes = []IndexSpec{{Name: "", Label: "Message", Properties: []string{"content"}}}
	if !errors.Is(s.Validate(), errSchemaEmptyIndexName) {
		t.Fatalf("expected errSchemaEmptyIndexName, got %v", s.Validate())
	}
}

func TestSchema_RejectsIndexUnknownLabel(t *testing.T) {
	t.Parallel()

	s := testSchema()
	s.Indexes = []IndexSpec{{Name: "bad", Label: "Bogus", Properties: []string{"x"}}}
	if !errors.Is(s.Validate(), errSchemaUnknownIndexLabel) {
		t.Fatalf("expected errSchemaUnknownIndexLabel, got %v", s.Validate())
	}
}

func TestSchema_RejectsIndexUnknownProp(t *testing.T) {
	t.Parallel()

	s := testSchema()
	s.Indexes = []IndexSpec{{Name: "bad", Label: "Message", Properties: []string{"nonexistent"}}}
	if !errors.Is(s.Validate(), errSchemaUnknownIndexProp) {
		t.Fatalf("expected errSchemaUnknownIndexProp, got %v", s.Validate())
	}
}
