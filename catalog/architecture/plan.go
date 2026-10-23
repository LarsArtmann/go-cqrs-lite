package architecture

import (
	"maps"
	"slices"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
)

// keptPlan is the resolved export policy: which resources render, which are
// redacted, why each kept resource is in, and which selectors matched
// nothing (ADR-0160). A nil kept map means "no selection": every resource
// is kept.
type keptPlan struct {
	kept     map[selKey]bool
	banned   map[selKey]bool
	reasons  map[selKey]string
	notFound []Selector
}

// plan computes the export policy against one catalog. The closure of a
// container selector is resolved against the catalog: domain → its services
// → those services' messages. Channels, stores, entities, ADRs and the rest
// are pulled in only by their own selectors — they are infrastructure a
// consumer names deliberately. With [ReferencesInclude], one level of the
// resources a kept resource points at joins the set as well (a service
// brings its stores, entities and message channels), never overriding a
// ban. Selectors that name no resource — or an unknown kind — land in
// notFound so a redaction that matches nothing fails loudly.
func (s *Selection) plan(cat *catalog.Catalog, references ReferencesMode) keptPlan {
	if s.empty() {
		return keptPlan{
			kept:     nil,
			banned:   nil,
			reasons:  nil,
			notFound: nil,
		}
	}

	plan := keptPlan{
		kept:     make(map[selKey]bool),
		banned:   make(map[selKey]bool),
		reasons:  make(map[selKey]string),
		notFound: make([]Selector, 0),
	}
	resolve := newSelectorResolver(cat)

	ban := func(selectors []Selector, rule string) {
		for _, sel := range selectors {
			kind, ok := adrKind(sel.Kind)
			if !ok {
				plan.notFound = append(plan.notFound, sel)

				continue
			}

			key := selKey{kind: kind, id: sel.ID}

			if !resolve.exists(key) {
				plan.notFound = append(plan.notFound, sel)

				continue
			}

			plan.banned[key] = true
			plan.reasons[key] = rule
		}
	}

	ban(s.neverExport, "neverExport")
	ban(s.exclude, "exclude")

	if len(s.include) == 0 {
		// No include set: keep everything the catalog has except bans.
		for _, key := range resolve.allKeys() {
			if !plan.banned[key] {
				plan.kept[key] = true
				plan.reasons[key] = "whole catalog"
			}
		}

		return plan
	}

	var addWithClosure func(sel Selector, reason string)

	addWithClosure = func(sel Selector, reason string) {
		kind, ok := adrKind(sel.Kind)
		if !ok {
			plan.notFound = append(plan.notFound, sel)

			return
		}

		key := selKey{kind: kind, id: sel.ID}
		if plan.kept[key] || plan.banned[key] {
			return
		}

		if !resolve.exists(key) {
			plan.notFound = append(plan.notFound, sel)

			return
		}

		plan.kept[key] = true
		plan.reasons[key] = reason

		for _, member := range resolve.closure(key) {
			addWithClosure(member, closureReason(kind, key.id))
		}
	}

	for _, sel := range s.include {
		addWithClosure(sel, "included")
	}

	if references == ReferencesInclude {
		level := slices.Collect(maps.Keys(plan.kept))

		for _, key := range level {
			for _, edge := range resolve.edges(key) {
				addWithClosure(
					edge,
					"referenced by "+selectorSpelling(key.kind)+" "+key.id,
				)
			}
		}
	}

	return plan
}

// closureReason explains why a container's member is in the export.
func closureReason(kind resourceKind, id string) string {
	switch kind {
	case kindDomain:
		return "in domain " + id
	case kindService:
		return "in service " + id
	case kindOverview, kindCommand, kindEvent, kindQuery, kindChannel,
		kindDataStore, kindFlow, kindEntity, kindDataProduct, kindAgent,
		kindADR, kindTeam, kindUser, kindDoc:
		return "referenced by " + selectorSpelling(kind) + " " + id
	default:
		return "referenced by " + id
	}
}
