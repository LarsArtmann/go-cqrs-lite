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
