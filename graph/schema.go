package graph

import (
	"fmt"
)

// Schema declares the set of node and edge types a graph projection owns.
//
// It is the graph-tier counterpart to [storage.RelationalSchema]: a
// declaration of the read-model shape that validates every write at the sink
// boundary. When a Schema is attached to a GraphProjection (via WithSchema)
// or a MemoryDriver (via WithSchema), the sink rejects writes with unknown
// node labels, unknown property names, or edges whose endpoints do not match
// the declared FromLabel and ToLabel.
//
// A nil Schema means "no validation" — the backward-compatible default.
// This preserves open-world semantics for consumers who want raw property-graph
// behaviour (like Neo4j without schema constraints). Setting a Schema makes
// the graph tier closed-world at the projection boundary, catching the most
// common graph-projection bug: a typo in a label or property name that
// silently creates a phantom node no query will ever find.
//
// The schema is deliberately minimal. It declares types, key properties, and
// property names — not value constraints (@regex, @values, cardinality ranges).
// Those are database-engine concerns, not sink-validation concerns. The goal
// is catching structural typos, not enforcing business rules.
type Schema struct {
	Nodes []NodeType
	Edges []EdgeType
}

// NodeType declares one node label in a [Schema].
//
// KeyProp is the property name that uniquely identifies a node of this label
// within the graph (e.g. "id", "email"). It must match the KeyProp used in
// [NodeRef] values written by projection handlers.
//
// Properties lists the non-key properties this node type may own. A property
// name not in this list (and not the KeyProp) is rejected by the schema.
type NodeType struct {
	Label      string
	KeyProp    string
	Properties []PropertyType
}

// EdgeType declares one edge type in a [Schema].
//
// FromLabel and ToLabel constrain which node labels may serve as edge
// endpoints. An edge whose From [NodeRef] has a different Label is rejected.
// This catches wiring bugs where a handler connects nodes of the wrong type.
//
// Properties lists the properties this edge type may own.
type EdgeType struct {
	Type       string
	FromLabel  string
	ToLabel    string
	Properties []PropertyType
}

// PropertyType declares one property on a [NodeType] or [EdgeType].
//
// Required controls whether the sink rejects writes that omit this property.
// When false, the property is optional.
type PropertyType struct {
	Name     string
	Required bool
}

// NodeType returns the declaration for the named label, or nil if no such
// node type is declared.
func (s *Schema) NodeType(label string) *NodeType {
	for i := range s.Nodes {
		if s.Nodes[i].Label == label {
			return &s.Nodes[i]
		}
	}

	return nil
}

// EdgeType returns the declaration for the named edge type, or nil.
func (s *Schema) EdgeType(typ string) *EdgeType {
	for i := range s.Edges {
		if s.Edges[i].Type == typ {
			return &s.Edges[i]
		}
	}

	return nil
}

// Validate checks the schema for structural errors: empty names, duplicate
// labels, duplicate edge types, and key properties that are not declared.
func (s *Schema) Validate() error {
	if len(s.Nodes) == 0 {
		return errSchemaNoNodeTypes
	}

	seenNodes := make(map[string]struct{}, len(s.Nodes))

	for i := range s.Nodes {
		nt := s.Nodes[i]

		if nt.Label == "" {
			return fmt.Errorf("graph schema: node type %d: %w", i, errSchemaEmptyLabel)
		}

		if nt.KeyProp == "" {
			return fmt.Errorf("graph schema: node type %q: %w", nt.Label, errSchemaEmptyKeyProp)
		}

		if _, dup := seenNodes[nt.Label]; dup {
			return fmt.Errorf("graph schema: %w: %q", errSchemaDuplicateNodeLabel, nt.Label)
		}

		seenNodes[nt.Label] = struct{}{}

		if err := validateProperties(nt.Properties, "node", nt.Label); err != nil {
			return err
		}

		if hasProperty(nt.Properties, nt.KeyProp) {
			return fmt.Errorf("graph schema: node type %q: %w", nt.Label, errSchemaKeyPropInProperties)
		}
	}

	seenEdges := make(map[string]struct{}, len(s.Edges))

	for i := range s.Edges {
		et := s.Edges[i]

		if et.Type == "" {
			return fmt.Errorf("graph schema: edge type %d: %w", i, errSchemaEmptyEdgeType)
		}

		if _, dup := seenEdges[et.Type]; dup {
			return fmt.Errorf("graph schema: %w: %q", errSchemaDuplicateEdgeType, et.Type)
		}

		seenEdges[et.Type] = struct{}{}

		if et.FromLabel == "" {
			return fmt.Errorf("graph schema: edge type %q: %w", et.Type, errSchemaEmptyFromLabel)
		}

		if et.ToLabel == "" {
			return fmt.Errorf("graph schema: edge type %q: %w", et.Type, errSchemaEmptyToLabel)
		}

		if _, ok := seenNodes[et.FromLabel]; !ok {
			return fmt.Errorf("graph schema: edge type %q: %w: %q", et.Type, errSchemaUnknownFromLabel, et.FromLabel)
		}

		if _, ok := seenNodes[et.ToLabel]; !ok {
			return fmt.Errorf("graph schema: edge type %q: %w: %q", et.Type, errSchemaUnknownToLabel, et.ToLabel)
		}

		if err := validateProperties(et.Properties, "edge", et.Type); err != nil {
			return err
		}
	}

	return nil
}

func validateProperties(props []PropertyType, kind, owner string) error {
	seen := make(map[string]struct{}, len(props))

	for i := range props {
		p := props[i]

		if p.Name == "" {
			return fmt.Errorf("graph schema: %s %q: property %d: %w", kind, owner, i, errSchemaEmptyPropName)
		}

		if _, dup := seen[p.Name]; dup {
			return fmt.Errorf("graph schema: %s %q: %w: %q", kind, owner, errSchemaDuplicateProp, p.Name)
		}

		seen[p.Name] = struct{}{}
	}

	return nil
}

func hasProperty(props []PropertyType, name string) bool {
	for i := range props {
		if props[i].Name == name {
			return true
		}
	}

	return false
}

// validateNodeRef checks that ref conforms to the schema's declared node type.
func (s *Schema) validateNodeRef(ref NodeRef) error {
	nt := s.NodeType(ref.Label)
	if nt == nil {
		return fmt.Errorf("%w: %q", errSinkUnknownNodeLabel, ref.Label)
	}

	if ref.KeyProp != nt.KeyProp {
		return fmt.Errorf(
			"%w: node %q: expected key prop %q, got %q",
			errSinkWrongKeyProp, ref.Label, nt.KeyProp, ref.KeyProp,
		)
	}

	return nil
}

// validateNodeProps checks that all prop names are declared on the node type.
func (s *Schema) validateNodeProps(label string, props map[string]any) error {
	return s.validateProps(s.NodeType(label).Properties, label, props)
}

// validateEdgeProps checks that all prop names are declared on the edge type.
func (s *Schema) validateEdgeProps(typ string, props map[string]any) error {
	return s.validateProps(s.EdgeType(typ).Properties, typ, props)
}

func (s *Schema) validateProps(declared []PropertyType, owner string, props map[string]any) error {
	allowed := make(map[string]struct{}, len(declared)+1)

	for i := range declared {
		allowed[declared[i].Name] = struct{}{}
	}

	for name := range props {
		if _, ok := allowed[name]; !ok {
			return fmt.Errorf("%w: %q on %q", errSinkUnknownProp, name, owner)
		}
	}

	return nil
}

// validateEdgeRef checks that ref conforms to the schema's declared edge type,
// including endpoint label constraints.
func (s *Schema) validateEdgeRef(ref EdgeRef) error {
	et := s.EdgeType(ref.Type)
	if et == nil {
		return fmt.Errorf("%w: %q", errSinkUnknownEdgeType, ref.Type)
	}

	if ref.From.Label != et.FromLabel {
		return fmt.Errorf(
			"%w: edge %q: expected from-label %q, got %q",
			errSinkEdgeEndpointMismatch, ref.Type, et.FromLabel, ref.From.Label,
		)
	}

	if ref.To.Label != et.ToLabel {
		return fmt.Errorf(
			"%w: edge %q: expected to-label %q, got %q",
			errSinkEdgeEndpointMismatch, ref.Type, et.ToLabel, ref.To.Label,
		)
	}

	return nil
}
