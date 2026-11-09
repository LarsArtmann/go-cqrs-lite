package graph

import "fmt"

// validateNodeRef checks that ref conforms to the schema's declared node type.
func (s *Schema) validateNodeRef(ref NodeRef) error {
	nodeType := s.NodeType(ref.Label)
	if nodeType == nil {
		return fmt.Errorf("%w: %q", errSinkUnknownNodeLabel, ref.Label)
	}

	if ref.KeyProp != nodeType.KeyProp {
		return fmt.Errorf(
			"%w: node %q: expected key prop %q, got %q",
			errSinkWrongKeyProp, ref.Label, nodeType.KeyProp, ref.KeyProp,
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
	edgeType := s.EdgeType(ref.Type)
	if edgeType == nil {
		return fmt.Errorf("%w: %q", errSinkUnknownEdgeType, ref.Type)
	}

	if ref.From.Label != edgeType.FromLabel {
		return fmt.Errorf(
			"%w: edge %q: expected from-label %q, got %q",
			errSinkEdgeEndpointMismatch, ref.Type, edgeType.FromLabel, ref.From.Label,
		)
	}

	if ref.To.Label != edgeType.ToLabel {
		return fmt.Errorf(
			"%w: edge %q: expected to-label %q, got %q",
			errSinkEdgeEndpointMismatch, ref.Type, edgeType.ToLabel, ref.To.Label,
		)
	}

	return nil
}
