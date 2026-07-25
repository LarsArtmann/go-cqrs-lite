package graph

import (
	"fmt"
	"slices"
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
	Nodes   []NodeType
	Edges   []EdgeType
	Indexes []IndexSpec
}

// IndexSpec declares a property index for a node label. The MemoryDriver
// ignores indexes (in-memory lookups are already O(1)), but a future Neo4j
// or Memgraph driver uses them to CREATE INDEX at startup. Mirrors
// [storage.IndexSpec] in the relational tier.
type IndexSpec struct {
	Name       string
	Label      string
	Properties []string
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
type PropertyType struct {
	Name string
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

	seenNodes, err := s.validateNodeTypes()
	if err != nil {
		return err
	}

	if err := s.validateEdgeTypes(seenNodes); err != nil {
		return err
	}

	return s.validateIndexes()
}

// validateNodeTypes checks every declared node type. Returns the set of seen
// labels so callers can validate edge endpoint references against it.
func (s *Schema) validateNodeTypes() (map[string]struct{}, error) {
	seen := make(map[string]struct{}, len(s.Nodes))

	for i := range s.Nodes {
		nodeType := s.Nodes[i]

		if nodeType.Label == "" {
			return nil, fmt.Errorf("graph schema: node type %d: %w", i, errSchemaEmptyLabel)
		}

		if nodeType.KeyProp == "" {
			return nil, fmt.Errorf(
				"graph schema: node type %q: %w",
				nodeType.Label,
				errSchemaEmptyKeyProp,
			)
		}

		if _, dup := seen[nodeType.Label]; dup {
			return nil, fmt.Errorf(
				"graph schema: %w: %q",
				errSchemaDuplicateNodeLabel,
				nodeType.Label,
			)
		}

		seen[nodeType.Label] = struct{}{}

		if err := validateProperties(nodeType.Properties, "node", nodeType.Label); err != nil {
			return nil, err
		}

		if hasProperty(nodeType.Properties, nodeType.KeyProp) {
			return nil, fmt.Errorf(
				"graph schema: node type %q: %w",
				nodeType.Label,
				errSchemaKeyPropInProperties,
			)
		}
	}

	return seen, nil
}

// validateEdgeTypes checks every declared edge type, including endpoint
// labels referring to declared node types.
func (s *Schema) validateEdgeTypes(seenNodes map[string]struct{}) error {
	seen := make(map[string]struct{}, len(s.Edges))

	for i := range s.Edges {
		edgeType := s.Edges[i]

		if edgeType.Type == "" {
			return fmt.Errorf("graph schema: edge type %d: %w", i, errSchemaEmptyEdgeType)
		}

		if _, dup := seen[edgeType.Type]; dup {
			return fmt.Errorf("graph schema: %w: %q", errSchemaDuplicateEdgeType, edgeType.Type)
		}

		seen[edgeType.Type] = struct{}{}

		if err := s.validateSingleEdge(edgeType, seenNodes); err != nil {
			return err
		}
	}

	return nil
}

// validateSingleEdge checks endpoint labels and properties for one edge type.
func (s *Schema) validateSingleEdge(edgeType EdgeType, seenNodes map[string]struct{}) error {
	if edgeType.FromLabel == "" {
		return fmt.Errorf(
			"graph schema: edge type %q: %w",
			edgeType.Type,
			errSchemaEmptyFromLabel,
		)
	}

	if edgeType.ToLabel == "" {
		return fmt.Errorf(
			"graph schema: edge type %q: %w",
			edgeType.Type,
			errSchemaEmptyToLabel,
		)
	}

	if _, ok := seenNodes[edgeType.FromLabel]; !ok {
		return fmt.Errorf(
			"graph schema: edge type %q: %w: %q",
			edgeType.Type,
			errSchemaUnknownFromLabel,
			edgeType.FromLabel,
		)
	}

	if _, ok := seenNodes[edgeType.ToLabel]; !ok {
		return fmt.Errorf(
			"graph schema: edge type %q: %w: %q",
			edgeType.Type,
			errSchemaUnknownToLabel,
			edgeType.ToLabel,
		)
	}

	return validateProperties(edgeType.Properties, "edge", edgeType.Type)
}

// validateIndexes checks every declared index for a valid label and properties.
func (s *Schema) validateIndexes() error {
	for i := range s.Indexes {
		idx := s.Indexes[i]

		if idx.Name == "" {
			return fmt.Errorf("graph schema: index %d: %w", i, errSchemaEmptyIndexName)
		}

		nodeType := s.NodeType(idx.Label)
		if nodeType == nil {
			return fmt.Errorf(
				"graph schema: index %q: %w: %q",
				idx.Name,
				errSchemaUnknownIndexLabel,
				idx.Label,
			)
		}

		for _, prop := range idx.Properties {
			if prop == nodeType.KeyProp {
				continue
			}

			if !hasProperty(nodeType.Properties, prop) {
				return fmt.Errorf(
					"graph schema: index %q: %w: %q on %q",
					idx.Name, errSchemaUnknownIndexProp, prop, idx.Label,
				)
			}
		}
	}

	return nil
}

func validateProperties(props []PropertyType, kind, owner string) error {
	seen := make(map[string]struct{}, len(props))

	for i := range props {
		p := props[i]

		if p.Name == "" {
			return fmt.Errorf(
				"graph schema: %s %q: property %d: %w",
				kind,
				owner,
				i,
				errSchemaEmptyPropName,
			)
		}

		if _, dup := seen[p.Name]; dup {
			return fmt.Errorf(
				"graph schema: %s %q: %w: %q",
				kind,
				owner,
				errSchemaDuplicateProp,
				p.Name,
			)
		}

		seen[p.Name] = struct{}{}
	}

	return nil
}

func hasProperty(props []PropertyType, name string) bool {
	return slices.ContainsFunc(props, func(p PropertyType) bool { return p.Name == name })
}

