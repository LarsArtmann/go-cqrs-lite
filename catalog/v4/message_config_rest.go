package catalog

// DELETE creates a command message pre-tagged with an HTTP DELETE operation.
// The schema is auto-derived from T. Equivalent to:
//
//	catalog.Command[T](id, catalog.MsgOperation("DELETE", path), opts...)
func DELETE[T any](id MessageID, path string, opts ...MessageOption) MessageConfig {
	return Command[T](id, append([]MessageOption{MsgOperation("DELETE", path)}, opts...)...)
}

// PUT creates a command message pre-tagged with an HTTP PUT operation.
func PUT[T any](id MessageID, path string, opts ...MessageOption) MessageConfig {
	return Command[T](id, append([]MessageOption{MsgOperation("PUT", path)}, opts...)...)
}

// PATCH creates a command message pre-tagged with an HTTP PATCH operation.
func PATCH[T any](id MessageID, path string, opts ...MessageOption) MessageConfig {
	return Command[T](id, append([]MessageOption{MsgOperation("PATCH", path)}, opts...)...)
}

// WithOperation is a composite MessageOption that sets the HTTP operation
// (method + path) and a typed success response in a single call, reducing
// boilerplate when registering REST endpoints:
//
//	catalog.Command[CreateUserCmd]("user.create",
//	    catalog.WithOperation[UserDTO]("POST", "/api/users", "201"),
//	)
//
// The response's description is derived from successCode via
// HttpStatusDescription, so the generated OpenAPI always has a spec-compliant
// non-empty description. Pass an explicit Response[T] option afterward to
// override it with custom copy.
func WithOperation[T any](method Method, path, successCode string) MessageOption {
	return func(m *messageBuilder) {
		MsgOperation(string(method), path)(m)
		Response[T](successCode, HttpStatusDescription(successCode))(m)
	}
}

// WithTypedParam adds an HTTP parameter with an explicit schema, so consumer
// contract docs can express non-string parameters (integer limits, boolean
// flags, enums) instead of the untyped string [WithParam] declares. A nil
// prop degrades to the untyped-string behavior of [WithParam].
func WithTypedParam(
	name, location, description string,
	required bool,
	prop *Property,
) MessageOption {
	return func(m *messageBuilder) {
		if m.schema == nil {
			m.schema = &Schema{Type: TypeObject, Properties: map[string]Property{}}
		}

		if m.schema.Parameters == nil {
			m.schema.Parameters = []Parameter{}
		}

		m.schema.Parameters = append(m.schema.Parameters, Parameter{
			Name: name, In: location, Description: description, Required: required, Schema: prop,
		})
	}
}

// WithResponseHeader declares a header on a response, so header contracts
// such as 429 + Retry-After become typed fields instead of prose inside the
// response description (which codegen tools cannot see). When no response
// with the given status code exists yet, a bare one is created so the header
// is never silently dropped.
func WithResponseHeader(statusCode, name, description string) MessageOption {
	return func(m *messageBuilder) {
		for i := range m.responses {
			if m.responses[i].StatusCode == statusCode {
				m.responses[i].Headers = append(m.responses[i].Headers, ResponseHeader{
					Name: name, Description: description,
				})

				return
			}
		}

		m.responses = append(m.responses, ResponseSpec{
			StatusCode: statusCode,
			Headers:    []ResponseHeader{{Name: name, Description: description}},
		})
	}
}

// WithRequestBodyContent overrides the request-body media type exporters
// assume (application/json) — e.g. "multipart/form-data" for file-upload
// endpoints, which previously could not be represented at all. A nil sch
// keeps the auto-derived message schema; for multipart bodies the schema
// describes the parts.
func WithRequestBodyContent(mediaType string, sch *Schema) MessageOption {
	return func(m *messageBuilder) {
		m.bodyMediaType = mediaType
		m.bodySchema = sch
	}
}

// HttpStatusDescription returns a human-readable description for the given HTTP
// status code, suitable for use as an OpenAPI Response.description. It covers
// the common 2xx success codes; unknown codes fall back to "Success" so the
// returned value is never empty (OpenAPI 3.0 requires a non-empty description
// on every Response object).
func HttpStatusDescription(code string) string {
	switch code {
	case "200":
		return "OK"
	case "201":
		return "Created"
	case "202":
		return "Accepted"
	case "203":
		return "Non-Authoritative Information"
	case "204":
		return "No Content"
	case "205":
		return "Reset Content"
	case "206":
		return "Partial Content"
	default:
		return "Success"
	}
}
