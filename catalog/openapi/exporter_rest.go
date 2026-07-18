package openapi

import (
	"fmt"
	"log"
	"strings"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
	"github.com/larsartmann/go-cqrs-lite/catalog/v4/internal/caseutil"
)

// HTTP method constants used when mapping CQRS message types to OpenAPI operations.
const (
	httpGet    = "GET"
	httpPost   = "POST"
	httpPut    = "PUT"
	httpDelete = "DELETE"
	httpPatch  = "PATCH"
)

// resolvePath returns the URL path for a message, honouring an explicit
// [catalog.Message.Operation] path when present and falling back to the
// conventional basePath/serviceID/messageID layout otherwise.
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

// resolveMethod returns the HTTP verb for a message, using the explicit operation
// method when provided or the supplied default (POST for commands/events, GET for queries).
func resolveMethod(msg catalog.Message, defaultMethod string) string {
	if msg.Operation != nil && msg.Operation.Method != "" {
		return strings.ToUpper(string(msg.Operation.Method))
	}

	return defaultMethod
}

// buildResponses selects the response map for an operation using the precedence
// rule: explicit [catalog.Message.Responses] > operation [catalog.Operation.StatusCodes] > caller defaults.
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

// statusDescription maps a numeric HTTP status code string to its standard reason phrase.
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

// ensurePathItem returns the existing [PathItem] for a path or creates and registers a new one.
func ensurePathItem(doc *Document, path string) *PathItem {
	if existing, ok := doc.Paths[path]; ok {
		return existing
	}

	item := &PathItem{}
	doc.Paths[path] = item

	return item
}

// setOperation assigns an [Operation] to the correct HTTP-method field of a [PathItem].
// If the method slot is already occupied, the new operation replaces the old one
// and a warning is printed to stderr — this usually indicates a duplicate
// (method, path) pair in the catalog that [catalog.Catalog.Validate] would flag.
func setOperation(item *PathItem, method string, op *Operation) {
	if existing := item.operationFor(method); existing != nil {
		log.Printf("warn: openapi: duplicate operation %s %s — overwriting %q with %q",
			method, op.OperationID, existing.OperationID, op.OperationID)
	}

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

func (item *PathItem) operationFor(method string) *Operation {
	switch method {
	case httpGet:
		return item.Get
	case httpPost:
		return item.Post
	case httpPut:
		return item.Put
	case httpDelete:
		return item.Delete
	case httpPatch:
		return item.Patch
	default:
		return nil
	}
}

// resolveParameters extracts OpenAPI path/query parameters for a message, preferring
// explicit [catalog.Schema.Parameters] tags and falling back to ID-field inference.
func (e *Exporter) resolveParameters(path string, msg catalog.Message) (string, []Parameter) {
	if msg.Schema != nil && len(msg.Schema.Parameters) > 0 {
		return extractTaggedParameters(path, msg.Schema.Parameters)
	}

	return extractIDParameter(path, msg.Schema)
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

func extractTaggedParameters(path string, params []catalog.Parameter) (string, []Parameter) {
	result := make([]Parameter, 0, len(params))

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

// addSecuritySchemes copies catalog-level [catalog.SecurityScheme] entries into the
// OpenAPI Components.SecuritySchemes map.
func (e *Exporter) addSecuritySchemes(doc *Document, cat *catalog.Catalog) {
	if len(cat.SecuritySchemes) == 0 {
		return
	}

	doc.Components.SecuritySchemes = make(map[string]SecurityScheme, len(cat.SecuritySchemes))

	for _, scheme := range cat.SecuritySchemes {
		doc.Components.SecuritySchemes[scheme.ID] = SecurityScheme{
			Type:         scheme.Type,
			Scheme:       scheme.Scheme,
			BearerFormat: scheme.BearerFormat,
			In:           scheme.In,
			Name:         scheme.Name,
			Description:  scheme.Description,
		}
	}
}

// msgSecurity converts a message's security-scheme IDs into the OpenAPI
// operation-level Security requirement array format.
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
