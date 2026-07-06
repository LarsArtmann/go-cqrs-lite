package graph

import (
	errorfamily "github.com/larsartmann/go-error-family"
)

var (
	errNoName = errorfamily.NewRejection(
		"graph.projection.name_required",
		"graph projection: name is required",
	)
	errNilDriver = errorfamily.NewRejection(
		"graph.projection.driver_required",
		"graph projection: driver must not be nil",
	)
	errNilHandler = errorfamily.NewRejection(
		"graph.projection.handler_required",
		"graph projection: handler must not be nil",
	)
	errEmptyLabel = errorfamily.NewRejection(
		"graph.noderef.label_required",
		"graph: NodeRef.Label is required",
	)
	errEmptyKeyProp = errorfamily.NewRejection(
		"graph.noderef.key_prop_required",
		"graph: NodeRef.KeyProp is required",
	)
	errEmptyEdgeType = errorfamily.NewRejection(
		"graph.edgeref.type_required",
		"graph: EdgeRef.Type is required",
	)
)

// Schema declaration errors — returned by Schema.Validate().
var (
	errSchemaNoNodeTypes = errorfamily.NewRejection(
		"graph.schema.no_node_types",
		"graph schema: at least one node type is required",
	)
	errSchemaEmptyLabel = errorfamily.NewRejection(
		"graph.schema.label_required",
		"graph schema: node label is required",
	)
	errSchemaEmptyKeyProp = errorfamily.NewRejection(
		"graph.schema.key_prop_required",
		"graph schema: key prop is required",
	)
	errSchemaDuplicateNodeLabel = errorfamily.NewRejection(
		"graph.schema.duplicate_node_label",
		"graph schema: duplicate node label",
	)
	errSchemaKeyPropInProperties = errorfamily.NewRejection(
		"graph.schema.key_prop_in_properties",
		"graph schema: key prop must not also be listed in properties",
	)
	errSchemaEmptyEdgeType = errorfamily.NewRejection(
		"graph.schema.edge_type_required",
		"graph schema: edge type is required",
	)
	errSchemaDuplicateEdgeType = errorfamily.NewRejection(
		"graph.schema.duplicate_edge_type",
		"graph schema: duplicate edge type",
	)
	errSchemaEmptyFromLabel = errorfamily.NewRejection(
		"graph.schema.from_label_required",
		"graph schema: edge from-label is required",
	)
	errSchemaEmptyToLabel = errorfamily.NewRejection(
		"graph.schema.to_label_required",
		"graph schema: edge to-label is required",
	)
	errSchemaUnknownFromLabel = errorfamily.NewRejection(
		"graph.schema.unknown_from_label",
		"graph schema: edge from-label not declared as a node type",
	)
	errSchemaUnknownToLabel = errorfamily.NewRejection(
		"graph.schema.unknown_to_label",
		"graph schema: edge to-label not declared as a node type",
	)
	errSchemaEmptyPropName = errorfamily.NewRejection(
		"graph.schema.prop_name_required",
		"graph schema: property name is required",
	)
	errSchemaDuplicateProp = errorfamily.NewRejection(
		"graph.schema.duplicate_prop",
		"graph schema: duplicate property",
	)
	errSchemaEmptyIndexName = errorfamily.NewRejection(
		"graph.schema.index_name_required",
		"graph schema: index name is required",
	)
	errSchemaUnknownIndexLabel = errorfamily.NewRejection(
		"graph.schema.unknown_index_label",
		"graph schema: index label not declared as a node type",
	)
	errSchemaUnknownIndexProp = errorfamily.NewRejection(
		"graph.schema.unknown_index_prop",
		"graph schema: index property not declared on node type",
	)
)

// Schema enforcement errors — returned by the schema-validating sink wrapper.
var (
	errSinkUnknownNodeLabel = errorfamily.NewRejection(
		"graph.sink.unknown_node_label",
		"graph: node label not declared in schema",
	)
	errSinkUnknownEdgeType = errorfamily.NewRejection(
		"graph.sink.unknown_edge_type",
		"graph: edge type not declared in schema",
	)
	errSinkWrongKeyProp = errorfamily.NewRejection(
		"graph.sink.wrong_key_prop",
		"graph: key prop does not match schema declaration",
	)
	errSinkUnknownProp = errorfamily.NewRejection(
		"graph.sink.unknown_prop",
		"graph: property not declared in schema",
	)
	errSinkEdgeEndpointMismatch = errorfamily.NewRejection(
		"graph.sink.edge_endpoint_mismatch",
		"graph: edge endpoint label does not match schema declaration",
	)
)

// Read API errors — returned by MemoryDriver read operations.
var (
	ErrPathNotFound = errorfamily.NewRejection(
		"graph.read.path_not_found",
		"graph: no path found between nodes",
	)
)
