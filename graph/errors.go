package graph

import (
	errorfamily "github.com/larsartmann/go-error-family"
)

var (
	errNoName error = errorfamily.NewRejection(
		"graph.projection.name_required",
		"graph projection: name is required",
	)
	errNilDriver error = errorfamily.NewRejection(
		"graph.projection.driver_required",
		"graph projection: driver must not be nil",
	)
	errNilHandler error = errorfamily.NewRejection(
		"graph.projection.handler_required",
		"graph projection: handler must not be nil",
	)
	errEmptyLabel error = errorfamily.NewRejection(
		"graph.noderef.label_required",
		"graph: NodeRef.Label is required",
	)
	errEmptyKeyProp error = errorfamily.NewRejection(
		"graph.noderef.key_prop_required",
		"graph: NodeRef.KeyProp is required",
	)
	errEmptyEdgeType error = errorfamily.NewRejection(
		"graph.edgeref.type_required",
		"graph: EdgeRef.Type is required",
	)
)

// Schema declaration errors — returned by Schema.Validate().
var (
	errSchemaNoNodeTypes error = errorfamily.NewRejection(
		"graph.schema.no_node_types",
		"graph schema: at least one node type is required",
	)
	errSchemaEmptyLabel error = errorfamily.NewRejection(
		"graph.schema.label_required",
		"graph schema: node label is required",
	)
	errSchemaEmptyKeyProp error = errorfamily.NewRejection(
		"graph.schema.key_prop_required",
		"graph schema: key prop is required",
	)
	errSchemaDuplicateNodeLabel error = errorfamily.NewRejection(
		"graph.schema.duplicate_node_label",
		"graph schema: duplicate node label",
	)
	errSchemaKeyPropInProperties error = errorfamily.NewRejection(
		"graph.schema.key_prop_in_properties",
		"graph schema: key prop must not also be listed in properties",
	)
	errSchemaEmptyEdgeType error = errorfamily.NewRejection(
		"graph.schema.edge_type_required",
		"graph schema: edge type is required",
	)
	errSchemaDuplicateEdgeType error = errorfamily.NewRejection(
		"graph.schema.duplicate_edge_type",
		"graph schema: duplicate edge type",
	)
	errSchemaEmptyFromLabel error = errorfamily.NewRejection(
		"graph.schema.from_label_required",
		"graph schema: edge from-label is required",
	)
	errSchemaEmptyToLabel error = errorfamily.NewRejection(
		"graph.schema.to_label_required",
		"graph schema: edge to-label is required",
	)
	errSchemaUnknownFromLabel error = errorfamily.NewRejection(
		"graph.schema.unknown_from_label",
		"graph schema: edge from-label not declared as a node type",
	)
	errSchemaUnknownToLabel error = errorfamily.NewRejection(
		"graph.schema.unknown_to_label",
		"graph schema: edge to-label not declared as a node type",
	)
	errSchemaEmptyPropName error = errorfamily.NewRejection(
		"graph.schema.prop_name_required",
		"graph schema: property name is required",
	)
	errSchemaDuplicateProp error = errorfamily.NewRejection(
		"graph.schema.duplicate_prop",
		"graph schema: duplicate property",
	)
	errSchemaEmptyIndexName error = errorfamily.NewRejection(
		"graph.schema.index_name_required",
		"graph schema: index name is required",
	)
	errSchemaUnknownIndexLabel error = errorfamily.NewRejection(
		"graph.schema.unknown_index_label",
		"graph schema: index label not declared as a node type",
	)
	errSchemaUnknownIndexProp error = errorfamily.NewRejection(
		"graph.schema.unknown_index_prop",
		"graph schema: index property not declared on node type",
	)
)

// Schema enforcement errors — returned by the schema-validating sink wrapper.
var (
	errSinkUnknownNodeLabel error = errorfamily.NewRejection(
		"graph.sink.unknown_node_label",
		"graph: node label not declared in schema",
	)
	errSinkUnknownEdgeType error = errorfamily.NewRejection(
		"graph.sink.unknown_edge_type",
		"graph: edge type not declared in schema",
	)
	errSinkWrongKeyProp error = errorfamily.NewRejection(
		"graph.sink.wrong_key_prop",
		"graph: key prop does not match schema declaration",
	)
	errSinkUnknownProp error = errorfamily.NewRejection(
		"graph.sink.unknown_prop",
		"graph: property not declared in schema",
	)
	errSinkEdgeEndpointMismatch error = errorfamily.NewRejection(
		"graph.sink.edge_endpoint_mismatch",
		"graph: edge endpoint label does not match schema declaration",
	)
)

// Read API errors — returned by MemoryDriver read operations.
var (
	ErrPathNotFound error = errorfamily.NewRejection(
		"graph.read.path_not_found",
		"graph: no path found between nodes",
	)
)
