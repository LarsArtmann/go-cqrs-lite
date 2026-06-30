package graph

import "github.com/larsartmann/go-cqrs-lite/event/v3"

var (
	errNoName = event.NewRejection(
		"graph.projection.name_required",
		"graph projection: name is required",
	)
	errNilDriver = event.NewRejection(
		"graph.projection.driver_required",
		"graph projection: driver must not be nil",
	)
	errNilHandler = event.NewRejection(
		"graph.projection.handler_required",
		"graph projection: handler must not be nil",
	)
	errEmptyLabel = event.NewRejection(
		"graph.noderef.label_required",
		"graph: NodeRef.Label is required",
	)
	errEmptyKeyProp = event.NewRejection(
		"graph.noderef.key_prop_required",
		"graph: NodeRef.KeyProp is required",
	)
	errEmptyEdgeType = event.NewRejection(
		"graph.edgeref.type_required",
		"graph: EdgeRef.Type is required",
	)
)

// Schema declaration errors — returned by Schema.Validate().
var (
	errSchemaNoNodeTypes = event.NewRejection(
		"graph.schema.no_node_types",
		"graph schema: at least one node type is required",
	)
	errSchemaEmptyLabel = event.NewRejection(
		"graph.schema.label_required",
		"graph schema: node label is required",
	)
	errSchemaEmptyKeyProp = event.NewRejection(
		"graph.schema.key_prop_required",
		"graph schema: key prop is required",
	)
	errSchemaDuplicateNodeLabel = event.NewRejection(
		"graph.schema.duplicate_node_label",
		"graph schema: duplicate node label",
	)
	errSchemaKeyPropInProperties = event.NewRejection(
		"graph.schema.key_prop_in_properties",
		"graph schema: key prop must not also be listed in properties",
	)
	errSchemaEmptyEdgeType = event.NewRejection(
		"graph.schema.edge_type_required",
		"graph schema: edge type is required",
	)
	errSchemaDuplicateEdgeType = event.NewRejection(
		"graph.schema.duplicate_edge_type",
		"graph schema: duplicate edge type",
	)
	errSchemaEmptyFromLabel = event.NewRejection(
		"graph.schema.from_label_required",
		"graph schema: edge from-label is required",
	)
	errSchemaEmptyToLabel = event.NewRejection(
		"graph.schema.to_label_required",
		"graph schema: edge to-label is required",
	)
	errSchemaUnknownFromLabel = event.NewRejection(
		"graph.schema.unknown_from_label",
		"graph schema: edge from-label not declared as a node type",
	)
	errSchemaUnknownToLabel = event.NewRejection(
		"graph.schema.unknown_to_label",
		"graph schema: edge to-label not declared as a node type",
	)
	errSchemaEmptyPropName = event.NewRejection(
		"graph.schema.prop_name_required",
		"graph schema: property name is required",
	)
	errSchemaDuplicateProp = event.NewRejection(
		"graph.schema.duplicate_prop",
		"graph schema: duplicate property",
	)
	errSchemaEmptyIndexName = event.NewRejection(
		"graph.schema.index_name_required",
		"graph schema: index name is required",
	)
	errSchemaUnknownIndexLabel = event.NewRejection(
		"graph.schema.unknown_index_label",
		"graph schema: index label not declared as a node type",
	)
	errSchemaUnknownIndexProp = event.NewRejection(
		"graph.schema.unknown_index_prop",
		"graph schema: index property not declared on node type",
	)
)

// Schema enforcement errors — returned by the schema-validating sink wrapper.
var (
	errSinkUnknownNodeLabel = event.NewRejection(
		"graph.sink.unknown_node_label",
		"graph: node label not declared in schema",
	)
	errSinkUnknownEdgeType = event.NewRejection(
		"graph.sink.unknown_edge_type",
		"graph: edge type not declared in schema",
	)
	errSinkWrongKeyProp = event.NewRejection(
		"graph.sink.wrong_key_prop",
		"graph: key prop does not match schema declaration",
	)
	errSinkUnknownProp = event.NewRejection(
		"graph.sink.unknown_prop",
		"graph: property not declared in schema",
	)
	errSinkEdgeEndpointMismatch = event.NewRejection(
		"graph.sink.edge_endpoint_mismatch",
		"graph: edge endpoint label does not match schema declaration",
	)
)

// Read API errors — returned by MemoryDriver read operations.
var (
	ErrPathNotFound = event.NewRejection(
		"graph.read.path_not_found",
		"graph: no path found between nodes",
	)
)
