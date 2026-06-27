package graph

import "errors"

var (
	errNoName        = errors.New("graph projection: name is required")
	errNilDriver     = errors.New("graph projection: driver must not be nil")
	errNilHandler    = errors.New("graph projection: handler must not be nil")
	errEmptyLabel    = errors.New("graph: NodeRef.Label is required")
	errEmptyKeyProp  = errors.New("graph: NodeRef.KeyProp is required")
	errEmptyEdgeType = errors.New("graph: EdgeRef.Type is required")
)
