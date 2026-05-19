package openapi

import (
	"fmt"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/catalog"
)

const (
	openAPIVersion = "3.0.3"
	contentType    = "application/json"
	objectType     = "object"
)

// Exporter generates an OpenAPI 3.0 document from a catalog.
type Exporter struct {
	Title       string
	Version     string
	Description string
	BasePath    string
}

// Option configures an Exporter.
type Option func(*Exporter)

// WithDescription sets the description for the OpenAPI document.
func WithDescription(desc string) Option {
	return func(e *Exporter) {
		e.Description = desc
	}
}

// WithBasePath sets the base path for all endpoints.
func WithBasePath(path string) Option {
	return func(e *Exporter) {
		e.BasePath = path
	}
}

// NewExporter creates an OpenAPI exporter with the given title and version.
func NewExporter(title, version string, opts ...Option) *Exporter {
	e := &Exporter{
		Title:       title,
		Version:     version,
		Description: "",
		BasePath:    "/api",
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
			Title:       e.Title,
			Version:     e.Version,
			Description: e.Description,
		},
		Servers: []Server{
			{URL: ".", Description: "Current server"},
		},
		Paths:      make(map[string]*PathItem),
		Components: Components{Schemas: make(map[string]any)},
		Tags:       []Tag{},
	}

	for _, svc := range cat.Services {
		tagName := svc.Name
		if tagName == "" {
			tagName = string(svc.ID)
		}

		doc.Tags = append(doc.Tags, Tag{
			Name:        tagName,
			Description: svc.Summary,
		})

		for _, cmd := range svc.Commands {
			e.addCommand(doc, string(svc.ID), tagName, cmd)
		}

		for _, qry := range svc.Queries {
			e.addQuery(doc, string(svc.ID), tagName, qry)
		}

		for _, evt := range svc.Events {
			e.addEvent(doc, string(svc.ID), tagName, evt)
		}
	}

	return doc
}

func (e *Exporter) addCommand(doc *Document, svcID, tagName string, msg catalog.Message) {
	path := fmt.Sprintf("%s/%s/%s", e.BasePath, svcID, toKebab(string(msg.ID)))
	schemaRef := e.addSchema(doc, msg)

	//nolint:exhaustruct
	doc.Paths[path] = &PathItem{
		Post: &Operation{ //nolint:exhaustruct
			Tags:        []string{tagName},
			Summary:     msg.Name,
			Description: msg.Summary,
			OperationID: "post" + toPascal(string(msg.ID)),
			RequestBody: &RequestBody{
				Description: msg.Name + " request",
				Content: map[string]MediaType{
					contentType: {Schema: schemaRef}, //nolint:exhaustruct
				},
				Required: true,
			},
			Responses: map[string]*Response{
				"200": {
					Description: "Success",
					Content: map[string]MediaType{
						contentType: {Schema: objectSchema()}, //nolint:exhaustruct
					},
				},
				"400": {
					Description: "Bad Request",
					Content: map[string]MediaType{
						contentType: {Schema: objectSchema()}, //nolint:exhaustruct
					},
				},
			},
		},
	}
}

func (e *Exporter) addQuery(doc *Document, svcID, tagName string, msg catalog.Message) {
	path := fmt.Sprintf("%s/%s/%s", e.BasePath, svcID, toKebab(string(msg.ID)))
	schemaRef := e.addSchema(doc, msg)

	op := &Operation{ //nolint:exhaustruct
		Tags:        []string{tagName},
		Summary:     msg.Name,
		Description: msg.Summary,
		OperationID: "get" + toPascal(string(msg.ID)),
		Responses: map[string]*Response{
			"200": {
				Description: "Success",
				Content: map[string]MediaType{
					contentType: {Schema: schemaRef}, //nolint:exhaustruct
				},
			},
			"404": {
				Description: "Not Found",
				Content: map[string]MediaType{
					contentType: {Schema: objectSchema()}, //nolint:exhaustruct
				},
			},
		},
	}

	if msg.Schema != nil && msg.Schema.Properties != nil {
		for fieldName, prop := range msg.Schema.Properties {
			lower := strings.ToLower(fieldName)

			if lower == "id" || lower == "aggregate_id" || strings.HasSuffix(lower, "_id") {
				path = fmt.Sprintf("%s/{%s}", path, fieldName)

				op.Parameters = append(op.Parameters, Parameter{
					Name:        fieldName,
					In:          "path",
					Description: prop.Description,
					Required:    true,
					Schema:      prop,
				})

				break
			}
		}
	}

	//nolint:exhaustruct
	doc.Paths[path] = &PathItem{
		Get: op,
	}
}

func (e *Exporter) addEvent(doc *Document, svcID, tagName string, msg catalog.Message) {
	path := fmt.Sprintf("%s/%s/events/%s", e.BasePath, svcID, toKebab(string(msg.ID)))
	schemaRef := e.addSchema(doc, msg)

	//nolint:exhaustruct
	doc.Paths[path] = &PathItem{
		Post: &Operation{ //nolint:exhaustruct
			Tags:        []string{tagName},
			Summary:     "Event: " + msg.Name,
			Description: msg.Summary,
			OperationID: "event" + toPascal(string(msg.ID)),
			RequestBody: &RequestBody{
				Description: msg.Name + " event payload",
				Content: map[string]MediaType{
					contentType: {Schema: schemaRef}, //nolint:exhaustruct
				},
				Required: true,
			},
			Responses: map[string]*Response{
				"200": {
					Description: "Event received",
				},
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
