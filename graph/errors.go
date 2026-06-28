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

// Schema declaration errors — returned by Schema.Validate().
var (
	errSchemaNoNodeTypes         = errors.New("graph schema: at least one node type is required")
	errSchemaEmptyLabel          = errors.New("graph schema: node label is required")
	errSchemaEmptyKeyProp        = errors.New("graph schema: key prop is required")
	errSchemaDuplicateNodeLabel  = errors.New("graph schema: duplicate node label")
	errSchemaKeyPropInProperties = errors.New("graph schema: key prop must not also be listed in properties")
	errSchemaEmptyEdgeType       = errors.New("graph schema: edge type is required")
	errSchemaDuplicateEdgeType   = errors.New("graph schema: duplicate edge type")
	errSchemaEmptyFromLabel      = errors.New("graph schema: edge from-label is required")
	errSchemaEmptyToLabel        = errors.New("graph schema: edge to-label is required")
	errSchemaUnknownFromLabel    = errors.New("graph schema: edge from-label not declared as a node type")
	errSchemaUnknownToLabel      = errors.New("graph schema: edge to-label not declared as a node type")
	errSchemaEmptyPropName       = errors.New("graph schema: property name is required")
	errSchemaDuplicateProp       = errors.New("graph schema: duplicate property")
)

// Schema enforcement errors — returned by the schema-validating sink wrapper.
var (
	errSinkUnknownNodeLabel     = errors.New("graph: node label not declared in schema")
	errSinkUnknownEdgeType      = errors.New("graph: edge type not declared in schema")
	errSinkWrongKeyProp         = errors.New("graph: key prop does not match schema declaration")
	errSinkUnknownProp          = errors.New("graph: property not declared in schema")
	errSinkEdgeEndpointMismatch = errors.New("graph: edge endpoint label does not match schema declaration")
)

// Read API errors — returned by MemoryDriver read operations.
var (
	ErrPathNotFound = errors.New("graph: no path found between nodes")
)
