package openapi

import (
	"cmp"
	"fmt"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/catalog"
)

const (
	openAPIVersion = "3.0.3"
	contentType    = "application/json"
)

// Exporter generates an OpenAPI 3.0 document from a catalog.
type Exporter struct {
	title       string
	version     string
	description string
	basePath    string
}

// Option configures an Exporter.
type Option func(*Exporter)

// WithDescription sets the description for the OpenAPI document.
func WithDescription(desc string) Option {
	return func(e *Exporter) {
		e.description = desc
	}
}

// WithBasePath sets the base path for all endpoints.
func WithBasePath(path string) Option {
	return func(e *Exporter) {
		e.basePath = path
	}
}

// NewExporter creates an OpenAPI exporter with the given title and version.
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

// Export generates an OpenAPI 3.0 Document from the given catalog.
func (e *Exporter) Export(cat *catalog.Catalog) *Document {
	doc := &Document{
		OpenAPI: openAPIVersion,
		Info: Info{
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
		tagName := cmp.Or(svc.Name, string(svc.ID))

		doc.Tags = append(doc.Tags, Tag{
			Name:        tagName,
			Description: svc.Summary,
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

	return doc
}

func jsonContent(schema any) map[string]MediaType {
	return map[string]MediaType{
		contentType: {Schema: schema}, //nolint:exhaustruct
	}
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
	path := fmt.Sprintf("%s/%s/%s", e.basePath, serviceID, toKebab(string(msg.ID)))
	schemaRef := e.addSchema(doc, msg)

	obj := objectSchema()

	//nolint:exhaustruct
	doc.Paths[path] = &PathItem{
		Post: &Operation{ //nolint:exhaustruct
			Tags:        []string{tagName},
			Summary:     msg.Name,
			Description: msg.Summary,
			OperationID: "post" + toPascal(string(msg.ID)),
			Deprecated:  msg.Deprecated,
			RequestBody: &RequestBody{
				Description: msg.Name + " request",
				Content:     jsonContent(schemaRef),
				Required:    true,
			},
			Responses: map[string]*Response{
				"200": responseWithContent("Success", obj),
				"400": responseWithContent("Bad Request", obj),
			},
		},
	}
}

func (e *Exporter) addQuery(
	doc *Document,
	serviceID catalog.ServiceID,
	tagName string,
	msg catalog.Message,
) {
	path := fmt.Sprintf("%s/%s/%s", e.basePath, serviceID, toKebab(string(msg.ID)))
	schemaRef := e.addSchema(doc, msg)

	op := &Operation{ //nolint:exhaustruct
		Tags:        []string{tagName},
		Summary:     msg.Name,
		Description: msg.Summary,
		OperationID: "get" + toPascal(string(msg.ID)),
		Deprecated:  msg.Deprecated,
		Responses: map[string]*Response{
			"200": responseWithContent("Success", schemaRef),
			"404": responseWithContent("Not Found", objectSchema()),
		},
	}

	path, op.Parameters = extractIDParameter(path, msg.Schema)

	//nolint:exhaustruct
	doc.Paths[path] = &PathItem{
		Get: op,
	}
}

func extractIDParameter(path string, schema *catalog.Schema) (string, []Parameter) {
	if schema == nil || schema.Properties == nil {
		return path, nil
	}

	for fieldName, prop := range schema.Properties {
		lower := strings.ToLower(fieldName)

		if lower == "id" || lower == "aggregate_id" || strings.HasSuffix(lower, "_id") {
			path = fmt.Sprintf("%s/{%s}", path, fieldName)

			return path, []Parameter{{
				Name: fieldName, In: "path",
				Description: prop.Description, Required: true, Schema: prop,
			}}
		}
	}

	return path, nil
}

func (e *Exporter) addEvent(
	doc *Document,
	serviceID catalog.ServiceID,
	tagName string,
	msg catalog.Message,
) {
	path := fmt.Sprintf("%s/%s/events/%s", e.basePath, serviceID, toKebab(string(msg.ID)))
	schemaRef := e.addSchema(doc, msg)

	//nolint:exhaustruct
	doc.Paths[path] = &PathItem{
		Post: &Operation{ //nolint:exhaustruct
			Tags:        []string{tagName},
			Summary:     "Event: " + msg.Name,
			Description: msg.Summary,
			OperationID: "event" + toPascal(string(msg.ID)),
			Deprecated:  msg.Deprecated,
			RequestBody: &RequestBody{
				Description: msg.Name + " event payload",
				Content:     jsonContent(schemaRef),
				Required:    true,
			},
			Responses: map[string]*Response{
				"200": responseNoContent("Event received"),
			},
		},
	}
}

func (e *Exporter) addSchema(doc *Document, msg catalog.Message) any {
	if msg.Schema == nil {
		return objectSchema()
	}

	key := schemaKey(msg)
	doc.Components.Schemas[key] = schemaToAny(msg.Schema)

	return SchemaRef{Ref: "#/components/schemas/" + key}
}
