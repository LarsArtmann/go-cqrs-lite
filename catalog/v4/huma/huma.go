// Package huma provides an optional adapter for converting Huma router
// operations into catalog message registrations.
//
// EXPERIMENTAL. This adapter does NOT depend on Huma directly. It accepts
// a minimal interface (HumaOperation) that matches the shape of Huma's
// operation metadata. Users bridge their Huma router to this interface.
//
// Usage:
//
//	ops := []huma.HumaOperation{
//	    {Method: "GET", Path: "/api/users/{id}", OperationID: "getUser"},
//	    {Method: "POST", Path: "/api/users", OperationID: "createUser"},
//	}
//	msgs := huma.ToMessages(ops)
//	builder.AddService("api", "API", "1.0.0", "REST API", msgs...)
package huma

import (
	"net/http"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
)

// HumaOperation is a minimal interface matching Huma's operation metadata.
// Users extract this from their Huma router and pass it to ToMessages.
type HumaOperation struct {
	Method      string
	Path        string
	OperationID string
	Summary     string
	Description string
	Tags        []string
}

// ToMessages converts a list of Huma operations to catalog MessageConfig entries.
// GET operations become Query messages; POST/PUT/PATCH/DELETE become Commands.
func ToMessages(ops []HumaOperation) []catalog.MessageConfig {
	msgs := make([]catalog.MessageConfig, 0, len(ops))

	for _, op := range ops {
		msgs = append(msgs, toMessage(op))
	}

	return msgs
}

func toMessage(op HumaOperation) catalog.MessageConfig {
	opts := []catalog.MessageOption{
		catalog.MsgOperation(op.Method, op.Path),
	}

	if op.Summary != "" {
		opts = append(opts, catalog.WithSummary(op.Summary))
	}

	id := catalog.MessageID(op.OperationID)

	if op.Method == http.MethodGet {
		return catalog.Query[struct{}](id, opts...)
	}

	return catalog.Command[struct{}](id, opts...)
}
