package catalog

import "fmt"

// Violation represents a single validation issue found in a Catalog.
type Violation struct {
	Path    string
	Message string
}

const pathTitle = "title"

// String returns a human-readable description of the violation.
func (v Violation) String() string {
	return fmt.Sprintf("%s: %s", v.Path, v.Message)
}

// Validate checks the catalog for common issues and returns all violations found.
// Returns nil if the catalog is valid.
func (c *Catalog) Validate() []Violation {
	var violations []Violation

	if c.Title == "" {
		violations = append(violations, Violation{
			Path:    pathTitle,
			Message: "catalog title must not be empty",
		})
	}

	if c.Version == "" {
		violations = append(violations, Violation{
			Path:    "version",
			Message: "catalog version must not be empty",
		})
	}

	seenMsgIDs := make(map[MessageID]string)

	for _, svc := range c.Services {
		violations = append(violations, validateService(seenMsgIDs, svc)...)
	}

	for _, domain := range c.Domains {
		violations = append(violations, validateDomain(domain)...)
	}

	return violations
}

func validateService(seenMsgIDs map[MessageID]string, svc Service) []Violation {
	var violations []Violation

	if svc.ID == "" {
		violations = append(violations, Violation{
			Path:    fmt.Sprintf("services[%s].id", svc.Name),
			Message: "service ID must not be empty",
		})
	}

	for _, cmd := range svc.Commands {
		violations = append(violations, validateMessage(seenMsgIDs, "command", svc.ID, cmd)...)
	}

	for _, evt := range svc.Events {
		violations = append(violations, validateMessage(seenMsgIDs, "event", svc.ID, evt)...)
	}

	for _, q := range svc.Queries {
		violations = append(violations, validateMessage(seenMsgIDs, "query", svc.ID, q)...)
	}

	return violations
}

func validateMessage(
	seenMsgIDs map[MessageID]string,
	kind string,
	svcID ServiceID,
	msg Message,
) []Violation {
	var violations []Violation

	path := fmt.Sprintf("services[%s].%s[%s]", svcID, kind, msg.ID)

	if msg.ID == "" && msg.Name == "" {
		violations = append(violations, Violation{
			Path:    path,
			Message: "message must have an ID or Name",
		})
	}

	id := GetID(msg)

	if prev, exists := seenMsgIDs[id]; exists {
		violations = append(violations, Violation{
			Path:    path,
			Message: fmt.Sprintf("duplicate message ID %q (also in %s)", id, prev),
		})
	} else {
		seenMsgIDs[id] = path
	}

	return violations
}

func validateDomain(domain Domain) []Violation {
	var violations []Violation

	if domain.ID == "" {
		violations = append(violations, Violation{
			Path:    fmt.Sprintf("domains[%s].id", domain.Name),
			Message: "domain ID must not be empty",
		})
	}

	seen := make(map[ServiceID]bool, len(domain.Services))

	for _, svcID := range domain.Services {
		if seen[svcID] {
			violations = append(violations, Violation{
				Path:    fmt.Sprintf("domains[%s].services", domain.ID),
				Message: fmt.Sprintf("duplicate service %q", svcID),
			})
		}

		seen[svcID] = true
	}

	return violations
}
