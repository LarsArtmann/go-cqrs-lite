package openapi

import (
	"cmp"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
	"github.com/larsartmann/go-cqrs-lite/catalog/v4/internal/caseutil"
)

const (
	openAPIVersion = "3.0.3"
	contentType    = "application/json"
)

// Exporter renders a [catalog.Catalog] as an OpenAPI 3.0.3 document.
// Construct one with [NewExporter] and customise it with [WithDescription]
// and [WithBasePath] options.
type Exporter struct {
	title       string
	version     string
	description string
	basePath    string
}

// Option configures an [Exporter] using the functional-options pattern.
type Option = catalog.Option[Exporter]

// WithDescription sets the Info.Description field of the generated document.
func WithDescription(desc string) Option {
	return func(e *Exporter) {
		e.description = desc
	}
}

// WithBasePath overrides the default "/api" prefix used for auto-generated paths.
func WithBasePath(path string) Option {
	return func(e *Exporter) {
		e.basePath = path
	}
}

// NewExporter creates an [Exporter] with the given document title and version,
// applying any provided options. The default base path is "/api".
func NewExporter(title, version string, opts ...Option) *Exporter {
	e := &Exporter{
		title:       title,
		version:     version,
		description: "",
		basePath:    "/api",
	}

	for _, opt := range opts {
		opt(e)
	}

	return e
}

// Export transforms a [catalog.Catalog] into an OpenAPI [Document], emitting one
// path operation per command/query/event and one component schema per entity.
func (e *Exporter) Export(cat *catalog.Catalog) *Document {
	doc := &Document{
		OpenAPI: openAPIVersion,
		Info: catalog.DocumentInfo{
			Title:       e.title,
			Version:     e.version,
			Description: e.description,
		},
		Servers: []Server{
			{URL: ".", Description: "Current server"},
		},
		Paths:      make(map[string]*PathItem),
		Components: Components{Schemas: make(map[string]any)},
		Tags:       []Tag{},
	}

	for _, svc := range cat.Services {
		tagName := cmp.Or(string(svc.Name), string(svc.ID))

		doc.Tags = append(doc.Tags, Tag{
			Name:        tagName,
			Description: string(svc.Summary),
		})

		for _, cmd := range svc.Commands {
			e.addCommand(doc, svc.ID, tagName, cmd)
		}

		for _, qry := range svc.Queries {
			e.addQuery(doc, svc.ID, tagName, qry)
		}

		for _, evt := range svc.Events {
			e.addEvent(doc, svc.ID, tagName, evt)
		}
	}

	for _, entity := range cat.Entities {
		e.addEntitySchema(doc, entity)
	}

	e.addSecuritySchemes(doc, cat)

	return doc
}

func (e *Exporter) addEntitySchema(doc *Document, entity catalog.Entity) {
	if entity.Schema == nil {
		return
	}

	key := "entity." + string(entity.ID)
	doc.Components.Schemas[key] = schemaToAny(entity.Schema)
}

func jsonContent(schema any) map[string]MediaType {
	return map[string]MediaType{
		contentType: {Schema: schema},
	}
}

func jsonContentWithExample(schema any, examples []jsontext.Value) map[string]MediaType {
	mediaType := MediaType{Schema: schema}

	if len(examples) > 0 {
		var val any
		if err := json.Unmarshal(examples[0], &val); err == nil {
			mediaType.Example = val
		}
	}

	return map[string]MediaType{contentType: mediaType}
}

func responseWithContent(desc string, schema any) *Response {
	return &Response{
		Description: desc,
		Content:     jsonContent(schema),
	}
}

func responseNoContent(desc string) *Response {
	return &Response{Description: desc, Content: nil}
}

func (e *Exporter) addCommand(
	doc *Document,
	serviceID catalog.ServiceID,
	tagName string,
	msg catalog.Message,
) {
	path := e.resolvePath(serviceID, msg, false)
	schemaRef := e.addSchema(doc, msg)
	method := resolveMethod(msg, httpPost)

	op := &Operation{
		Tags:        []string{tagName},
		Summary:     string(msg.Name),
		Description: string(msg.Summary),
		OperationID: strings.ToLower(method) + caseutil.ToPascal(string(msg.ID)),
		Deprecated:  msg.Deprecated,
		RequestBody: &RequestBody{
			Description: string(msg.Name) + " request",
			Content:     jsonContentWithExample(schemaRef, msg.Examples),
			Required:    true,
		},
		Responses: e.buildResponses(
			doc, msg,
			map[string]*Response{
				"200": responseWithContent("Success", objectSchema()),
				"400": responseWithContent("Bad Request", objectSchema()),
			},
		),
		Security: msgSecurity(msg),
	}

	item := ensurePathItem(doc, path)
	setOperation(item, method, op)
}

func (e *Exporter) addQuery(
	doc *Document,
	serviceID catalog.ServiceID,
	tagName string,
	msg catalog.Message,
) {
	path := e.resolvePath(serviceID, msg, false)
	schemaRef := e.addSchema(doc, msg)
	method := resolveMethod(msg, httpGet)

	op := &Operation{
		Tags:        []string{tagName},
		Summary:     string(msg.Name),
		Description: string(msg.Summary),
		OperationID: strings.ToLower(method) + caseutil.ToPascal(string(msg.ID)),
		Deprecated:  msg.Deprecated,
		Responses: e.buildResponses(
			doc, msg,
			map[string]*Response{
				"200": responseWithContent("Success", schemaRef),
				"404": responseWithContent("Not Found", objectSchema()),
			},
		),
	}

	path, op.Parameters = e.resolveParameters(path, msg)

	item := ensurePathItem(doc, path)
	setOperation(item, method, op)
}

func (e *Exporter) addEvent(
	doc *Document,
	serviceID catalog.ServiceID,
	tagName string,
	msg catalog.Message,
) {
	path := e.resolvePath(serviceID, msg, true)
	schemaRef := e.addSchema(doc, msg)
	method := resolveMethod(msg, httpPost)

	op := &Operation{
		Tags:        []string{tagName},
		Summary:     "Event: " + string(msg.Name),
		Description: string(msg.Summary),
		OperationID: strings.ToLower(method) + caseutil.ToPascal(string(msg.ID)),
		Deprecated:  msg.Deprecated,
		RequestBody: &RequestBody{
			Description: string(msg.Name) + " event payload",
			Content:     jsonContentWithExample(schemaRef, msg.Examples),
			Required:    true,
		},
		Responses: e.buildResponses(
			doc, msg,
			map[string]*Response{
				"200": responseNoContent("Event received"),
			},
		),
		Security: msgSecurity(msg),
	}

	item := ensurePathItem(doc, path)
	setOperation(item, method, op)
}

func (e *Exporter) addSchema(doc *Document, msg catalog.Message) any {
	if msg.Schema == nil {
		return objectSchema()
	}

	key := schemaKey(msg)
	doc.Components.Schemas[key] = schemaToAny(msg.Schema)

	return SchemaRef{Ref: "#/components/schemas/" + key}
}
