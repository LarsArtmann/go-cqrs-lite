package openapi

import (
	"cmp"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
	"github.com/larsartmann/go-cqrs-lite/catalog/v4/internal/caseutil"
)

const (
	openAPIVersion = "3.0.3"
	contentType    = "application/json"
)

type Exporter struct {
	title       string
	version     string
	description string
	basePath    string
}

type Option = catalog.Option[Exporter]

func WithDescription(desc string) Option {
	return func(e *Exporter) {
		e.description = desc
	}
}

func WithBasePath(path string) Option {
	return func(e *Exporter) {
		e.basePath = path
	}
}

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
	mt := MediaType{Schema: schema}

	if len(examples) > 0 {
		var val any
		if err := json.Unmarshal(examples[0], &val); err == nil {
			mt.Example = val
		}
	}

	return map[string]MediaType{contentType: mt}
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

const (
	httpGet    = "GET"
	httpPost   = "POST"
	httpPut    = "PUT"
	httpDelete = "DELETE"
	httpPatch  = "PATCH"
)

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

func (e *Exporter) resolvePath(
	serviceID catalog.ServiceID,
	msg catalog.Message,
	isEvent bool,
) string {
	if msg.Operation != nil && msg.Operation.Path != "" {
		return msg.Operation.Path
	}

	if isEvent {
		return fmt.Sprintf(
			"%s/%s/events/%s",
			e.basePath,
			serviceID,
			caseutil.ToKebab(string(msg.ID)),
		)
	}

	return fmt.Sprintf("%s/%s/%s", e.basePath, serviceID, caseutil.ToKebab(string(msg.ID)))
}

func resolveMethod(msg catalog.Message, defaultMethod string) string {
	if msg.Operation != nil && msg.Operation.Method != "" {
		return strings.ToUpper(string(msg.Operation.Method))
	}

	return defaultMethod
}

func hasExplicitOperation(msg catalog.Message) bool {
	return msg.Operation != nil && msg.Operation.Path != ""
}

func (e *Exporter) buildResponses(
	doc *Document,
	msg catalog.Message,
	defaults map[string]*Response,
) map[string]*Response {
	if len(msg.Responses) > 0 {
		return e.buildTypedResponses(doc, msg)
	}

	if msg.Operation != nil && len(msg.Operation.StatusCodes) > 0 {
		responses := make(map[string]*Response, len(msg.Operation.StatusCodes))

		for _, code := range msg.Operation.StatusCodes {
			responses[code] = responseWithContent(statusDescription(code), objectSchema())
		}

		return responses
	}

	return defaults
}

func (e *Exporter) buildTypedResponses(doc *Document, msg catalog.Message) map[string]*Response {
	responses := make(map[string]*Response, len(msg.Responses))

	for _, resp := range msg.Responses {
		openapiResp := &Response{
			Description: resp.Description,
		}

		if resp.Schema != nil {
			key := responseSchemaKey(msg, resp.StatusCode)
			doc.Components.Schemas[key] = schemaToAny(resp.Schema)
			openapiResp.Content = jsonContentWithExample(
				SchemaRef{Ref: "#/components/schemas/" + key},
				resp.Examples,
			)
		}

		responses[resp.StatusCode] = openapiResp
	}

	return responses
}

func responseSchemaKey(msg catalog.Message, statusCode string) string {
	return schemaKey(msg) + ".response." + statusCode
}

func statusDescription(code string) string {
	switch code {
	case "200":
		return "OK"
	case "201":
		return "Created"
	case "202":
		return "Accepted"
	case "204":
		return "No Content"
	case "400":
		return "Bad Request"
	case "401":
		return "Unauthorized"
	case "403":
		return "Forbidden"
	case "404":
		return "Not Found"
	case "409":
		return "Conflict"
	case "500":
		return "Internal Server Error"
	default:
		return code
	}
}

func ensurePathItem(doc *Document, path string) *PathItem {
	if existing, ok := doc.Paths[path]; ok {
		return existing
	}

	item := &PathItem{}
	doc.Paths[path] = item

	return item
}

func setOperation(item *PathItem, method string, op *Operation) {
	switch method {
	case httpGet:
		item.Get = op
	case httpPost:
		item.Post = op
	case httpPut:
		item.Put = op
	case httpDelete:
		item.Delete = op
	case httpPatch:
		item.Patch = op
	default:
		item.Post = op
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

func extractIDParameter(path string, schema *catalog.Schema) (string, []Parameter) {
	if schema == nil || schema.Properties == nil {
		return path, nil
	}

	for fieldName, prop := range schema.Properties {
		if isIDField(strings.ToLower(fieldName)) {
			path = fmt.Sprintf("%s/{%s}", path, fieldName)

			return path, []Parameter{{
				Name: fieldName, In: "path",
				Description: prop.Description, Required: true, Schema: prop,
			}}
		}
	}

	return path, nil
}

func isIDField(lower string) bool {
	return lower == "id" || strings.HasSuffix(lower, "_id")
}

func (e *Exporter) resolveParameters(path string, msg catalog.Message) (string, []Parameter) {
	if msg.Schema != nil && len(msg.Schema.Parameters) > 0 {
		return extractTaggedParameters(path, msg.Schema.Parameters)
	}

	return extractIDParameter(path, msg.Schema)
}

func extractTaggedParameters(path string, params []catalog.Parameter) (string, []Parameter) {
	var result []Parameter

	for _, p := range params {
		op := Parameter{
			Name:        p.Name,
			In:          p.In,
			Description: p.Description,
			Required:    p.Required,
		}

		if p.Schema != nil {
			op.Schema = map[string]any{
				"type":        string(p.Schema.Type),
				"description": p.Schema.Description,
				"format":      p.Schema.Format,
			}
		}

		result = append(result, op)

		if p.In == "path" && !strings.Contains(path, "{"+p.Name+"}") {
			path = fmt.Sprintf("%s/{%s}", path, p.Name)
		}
	}

	return path, result
}

func (e *Exporter) addSecuritySchemes(doc *Document, cat *catalog.Catalog) {
	if len(cat.SecuritySchemes) == 0 {
		return
	}

	doc.Components.SecuritySchemes = make(map[string]SecurityScheme, len(cat.SecuritySchemes))

	for _, ss := range cat.SecuritySchemes {
		doc.Components.SecuritySchemes[ss.ID] = SecurityScheme{
			Type:         ss.Type,
			Scheme:       ss.Scheme,
			BearerFormat: ss.BearerFormat,
			In:           ss.In,
			Name:         ss.Name,
			Description:  ss.Description,
		}
	}
}

func msgSecurity(msg catalog.Message) []map[string][]string {
	if len(msg.Security) == 0 {
		return nil
	}

	req := make(map[string][]string, len(msg.Security))

	for _, id := range msg.Security {
		req[id] = []string{}
	}

	return []map[string][]string{req}
}
