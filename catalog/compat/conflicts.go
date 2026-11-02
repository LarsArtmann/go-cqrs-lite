package compat

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
	"github.com/larsartmann/go-cqrs-lite/catalog/v4/schema"
)

// Conflict kinds reported by [Conflicts].
const (
	// ConflictVersionMismatch means both catalogs declare the same message
	// at different versions.
	ConflictVersionMismatch = "version-mismatch"

	// ConflictSchemaMismatch means both catalogs declare the same message
	// with different payload shapes.
	ConflictSchemaMismatch = "schema-mismatch"

	// ConflictKindMismatch means the same ID is a command in one catalog
	// and an event or query in the other.
	ConflictKindMismatch = "message-kind-mismatch"

	// ConflictServiceMismatch means both catalogs declare the same service
	// at different versions.
	ConflictServiceMismatch = "service-metadata-mismatch"
)

// Conflict is one identity declared twice with divergent content.
type Conflict struct {
	// Path locates the duplicated identity, for example
	// "services[orders].events[order_placed]".
	Path string `json:"path"`

	// Kind is one of the Conflict* constants.
	Kind string `json:"kind"`

	// Detail states both sides so the reader can decide which is true.
	Detail string `json:"detail"`
}

// String renders the conflict on one line.
func (c Conflict) String() string {
	return fmt.Sprintf("%s: %s (%s)", c.Path, c.Detail, c.Kind)
}

// messageDecl is one message together with the service that declares it.
type messageDecl struct {
	service catalog.ServiceID
	kind    catalog.MessageKind
	msg     catalog.Message
}

// Conflicts reports every message or service that BOTH catalogs declare with
// divergent identity. It is a pre-merge detector: a hub union of two valid
// catalogs can silently keep one version, one shape, or one service record,
// and no single-catalog validation can see the divergence.
//
// An idempotent union produces no conflicts: identical declarations on both
// sides are not divergence. Metadata that merges by union (owners, badges,
// labels) is deliberately not reported.
func Conflicts(a, b *catalog.Catalog) []Conflict {
	conflicts := append(messageConflicts(a, b), serviceConflicts(a, b)...)

	sort.SliceStable(conflicts, func(i, j int) bool {
		if conflicts[i].Path != conflicts[j].Path {
			return conflicts[i].Path < conflicts[j].Path
		}

		return conflicts[i].Kind < conflicts[j].Kind
	})

	return conflicts
}

func messageConflicts(a, b *catalog.Catalog) []Conflict {
	left, right := indexDecls(a), indexDecls(b)

	var conflicts []Conflict

	for key, l := range left {
		r, ok := right[key]
		if !ok {
			continue
		}

		path := string(key)

		if l.kind != r.kind {
			conflicts = append(conflicts, Conflict{
				Path:   path,
				Kind:   ConflictKindMismatch,
				Detail: fmt.Sprintf("declared as %q in %s but %q in %s", l.kind, l.service, r.kind, r.service),
			})
		}

		if l.msg.Version != r.msg.Version {
			conflicts = append(conflicts, Conflict{
				Path:   path,
				Kind:   ConflictVersionMismatch,
				Detail: fmt.Sprintf("version %q vs %q", l.msg.Version, r.msg.Version),
			})
		}

		if !schemasEqual(l.msg.Schema, r.msg.Schema) {
			conflicts = append(conflicts, Conflict{
				Path:   path,
				Kind:   ConflictSchemaMismatch,
				Detail: "payload schemas differ",
			})
		}
	}

	return conflicts
}

func serviceConflicts(a, b *catalog.Catalog) []Conflict {
	left, right := indexServices(a), indexServices(b)

	var conflicts []Conflict

	for id, l := range left {
		r, ok := right[id]
		if !ok || l.Version == r.Version {
			continue
		}

		conflicts = append(conflicts, Conflict{
			Path:   fmt.Sprintf("services[%s]", id),
			Kind:   ConflictServiceMismatch,
			Detail: fmt.Sprintf("version %q vs %q", l.Version, r.Version),
		})
	}

	return conflicts
}

func indexDecls(c *catalog.Catalog) map[catalog.MessageID]messageDecl {
	decls := make(map[catalog.MessageID]messageDecl)

	if c == nil {
		return decls
	}

	for _, svc := range c.Services {
		for _, msg := range serviceMessages(svc) {
			decls[catalog.Key(msg)] = messageDecl{service: svc.ID, kind: msg.Kind, msg: msg}
		}
	}

	return decls
}

func indexServices(c *catalog.Catalog) map[catalog.ServiceID]catalog.Service {
	services := make(map[catalog.ServiceID]catalog.Service)

	if c == nil {
		return services
	}

	for _, svc := range c.Services {
		services[svc.ID] = svc
	}

	return services
}

// schemasEqual compares two payload shapes through their canonical JSON, the
// same rendering every exporter consumes.
func schemasEqual(a, b *catalog.Schema) bool {
	left, leftErr := renderSchema(a)
	right, rightErr := renderSchema(b)

	if leftErr != nil || rightErr != nil {
		return a == nil && b == nil
	}

	return bytes.Equal(left, right)
}

func renderSchema(s *schema.Schema) ([]byte, error) {
	if s == nil {
		return nil, nil //nolint:nilerr // absence is a comparable value, not an error
	}

	return schema.ToJSON(s)
}
