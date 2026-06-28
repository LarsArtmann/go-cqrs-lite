package graph

// schemaSink wraps a [GraphSink] with schema validation. Every write is
// checked against the declared [Schema] before being forwarded to the
// underlying sink. This catches structural typos (unknown labels, unknown
// properties, edge endpoint mismatches) at the projection boundary — the
// graph-tier equivalent of relational Row column-name validation.
//
// When schema is nil, schemaSink is a transparent pass-through (no validation).
// This preserves the backward-compatible open-world default.
type schemaSink struct {
	inner  GraphSink
	schema *Schema
}

func (s *schemaSink) MergeNode(ref NodeRef, props map[string]any) error {
	if s.schema != nil {
		if err := s.schema.validateNodeRef(ref); err != nil {
			return err
		}

		if err := s.schema.validateNodeProps(ref.Label, props); err != nil {
			return err
		}
	}

	return s.inner.MergeNode(ref, props)
}

func (s *schemaSink) MergeEdge(ref EdgeRef, props map[string]any) error {
	if s.schema != nil {
		if err := s.schema.validateEdgeRef(ref); err != nil {
			return err
		}

		if err := s.schema.validateEdgeProps(ref.Type, props); err != nil {
			return err
		}
	}

	return s.inner.MergeEdge(ref, props)
}

func (s *schemaSink) RemoveNode(ref NodeRef) error {
	return s.inner.RemoveNode(ref)
}

func (s *schemaSink) RemoveEdge(ref EdgeRef) error {
	return s.inner.RemoveEdge(ref)
}

func (s *schemaSink) SetNodeProperty(ref NodeRef, prop string, value any) error {
	if s.schema != nil {
		if err := s.schema.validateNodeRef(ref); err != nil {
			return err
		}

		// Check that the property name is declared on this node type.
		nt := s.schema.NodeType(ref.Label)
		if nt == nil {
			return errSinkUnknownNodeLabel
		}

		if !hasProperty(nt.Properties, prop) {
			return errSinkUnknownProp
		}
	}

	return s.inner.SetNodeProperty(ref, prop, value)
}

// wrapWithSchema returns a sink that validates against schema before each
// write. If schema is nil, inner is returned unchanged.
func wrapWithSchema(inner GraphSink, schema *Schema) GraphSink {
	if schema == nil {
		return inner
	}

	return &schemaSink{inner: inner, schema: schema}
}
