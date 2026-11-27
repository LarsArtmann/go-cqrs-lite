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
// including endpoint label and key prop constraints.
func (s *Schema) validateEdgeRef(ref EdgeRef) error {
	edgeType := s.EdgeType(ref.Type)
	if edgeType == nil {
		return fmt.Errorf("%w: %q", errSinkUnknownEdgeType, ref.Type)
	}

	if err := s.validateEndpoint(ref.From, "from", ref.Type, edgeType.FromLabel); err != nil {
		return err
	}

	return s.validateEndpoint(ref.To, "to", ref.Type, edgeType.ToLabel)
}

// validateEndpoint checks that an edge endpoint is a well-formed node of the
// label the edge type declares: right label, and the key prop that node type
// declares. Without the key check a typo would create a phantom node.
func (s *Schema) validateEndpoint(ref NodeRef, role, typ, label string) error {
	if ref.Label != label {
		return fmt.Errorf(
			"%w: edge %q: expected %s-label %q, got %q",
			errSinkEdgeEndpointMismatch, typ, role, label, ref.Label,
		)
	}

	// An endpoint label without a node type cannot have a declared key prop.
	declared := ""
	if nodeType := s.NodeType(ref.Label); nodeType != nil {
		declared = nodeType.KeyProp
	}

	if ref.KeyProp != declared {
		return fmt.Errorf(
			"%w: edge %q: %s endpoint %q: expected key prop %q, got %q",
			errSinkWrongKeyProp, typ, role, ref.Label, declared, ref.KeyProp,
		)
	}

	return nil
}
