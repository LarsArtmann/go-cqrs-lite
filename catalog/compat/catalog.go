package compat

import (
	"fmt"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
)

// CompatCatalog classifies every payload-schema change between two catalogs.
//
// Messages are matched by [catalog.Key]. A message that exists only in the new
// catalog is additive; one that exists only in the old catalog is reported but
// never blocks, because deleting a message is a domain decision rather than a
// payload-shape breach.
//
// The blocking rule is deliberately narrow: a breaking shape change on a
// message whose version did NOT move yields a [KindVersionNotBumped] finding
// marked Blocking. A breaking change WITH a version bump is reported as
// [ClassNeedsVersionBump] but does not block, because only the declaration
// (schema.EventSchema) can show whether the upcast op exists; cqrs-lint E021
// and E022 own that half of the policy.
func CompatCatalog(old, next *catalog.Catalog) Report {
	var b reportBuilder

	oldMsgs, newMsgs := indexMessages(old), indexMessages(next)

	for key, oldMsg := range oldMsgs {
		newMsg, ok := newMsgs[key]
		if !ok {
			b.add(Finding{
				Path:    string(key),
				Kind:    KindMessageRemoved,
				Class:   ClassNeedsVersionBump,
				Message: fmt.Sprintf("message %q is no longer declared", key),
			})

			continue
		}

		diffMessage(&b, key, oldMsg.Version, oldMsg.Schema, newMsg.Version, newMsg.Schema)
	}

	for key := range newMsgs {
		if _, ok := oldMsgs[key]; ok {
			continue
		}

		b.add(Finding{
			Path:    string(key),
			Kind:    KindMessageAdded,
			Class:   ClassAdditive,
			Message: fmt.Sprintf("message %q is newly declared", key),
		})
	}

	return b.report()
}

// diffMessage compares one message's payload shape and applies the
// version-must-move rule.
func diffMessage(
	b *reportBuilder,
	key catalog.MessageID,
	oldVersion catalog.Version,
	oldSchema *catalog.Schema,
	newVersion catalog.Version,
	newSchema *catalog.Schema,
) {
	report := Compat(oldSchema, newSchema)
	breaking := false

	for _, f := range report.Findings {
		f.Path = joinPath(string(key), f.Path)

		if f.Class.IsBreaking() {
			breaking = true
		}

		b.add(f)
	}

	if !breaking || oldVersion != newVersion {
		return
	}

	b.add(Finding{
		Path:  string(key),
		Kind:  KindVersionNotBumped,
		Class: ClassNeedsVersionBump,
		Op:    OpTransform,
		Message: fmt.Sprintf(
			"breaking payload change to %q while its version stays %q: bump the version and declare the upcast op",
			key,
			newVersion,
		),
		Blocking: true,
	})
}

// indexMessages collects every message of a catalog by its unique key.
// Duplicate keys are already a [catalog.Catalog.Validate] violation; the last
// declaration wins here, and this helper never reports duplicates.
func indexMessages(c *catalog.Catalog) map[catalog.MessageID]catalog.Message {
	messages := make(map[catalog.MessageID]catalog.Message)

	if c == nil {
		return messages
	}

	for _, svc := range c.Services {
		for _, msg := range serviceMessages(svc) {
			messages[catalog.Key(msg)] = msg
		}
	}

	return messages
}

func serviceMessages(svc catalog.Service) []catalog.Message {
	messages := make([]catalog.Message, 0, len(svc.Commands)+len(svc.Events)+len(svc.Queries))
	messages = append(messages, svc.Commands...)
	messages = append(messages, svc.Events...)
	messages = append(messages, svc.Queries...)

	return messages
}
