package architecture

import "slices"

// selKey identifies one resource for kept-set bookkeeping.
type selKey struct {
	kind resourceKind
	id   string
}

// Selector addresses exactly one resource of the catalog by kind and id
// (ADR-0160). Kind names are the spellings [adrKind] accepts: "domain",
// "service", "command", "event", "query", "channel", "container" (data
// store), "flow", "entity", "data-product", "agent", "decision" (ADR),
// "team", "person" (user) and "doc".
type Selector struct {
	Kind string
	ID   string
}

// Selection is the kept-set policy of an export. The zero value keeps every
// resource, which is the whole-catalog reference of ADR-0159.
//
// Resolution rules, pinned by tests (ADR-0160):
//   - NeverExporting beats everything: a never-exported resource is absent
//     even when it is inside an included domain or explicitly included.
//   - Excluding beats including: an excluded resource drops even inside an
//     included container. Excludes name exactly the listed resources; they
//     do not cascade to members.
//   - Including a container brings its natural closure: a domain brings its
//     services and those services' messages; a service brings its messages.
type Selection struct {
	include     []Selector
	exclude     []Selector
	neverExport []Selector
}

// Including adds selectors to the include set. An empty include set keeps
// the whole catalog (minus excludes and never-exports).
func (s *Selection) Including(selectors ...Selector) *Selection {
	s.include = append(s.include, selectors...)

	return s
}

// Excluding adds selectors to the exclude set; excludes always win.
func (s *Selection) Excluding(selectors ...Selector) *Selection {
	s.exclude = append(s.exclude, selectors...)

	return s
}

// NeverExporting adds selectors that are absent from any export, even when
// explicitly included.
func (s *Selection) NeverExporting(selectors ...Selector) *Selection {
	s.neverExport = append(s.neverExport, selectors...)

	return s
}

// empty reports whether no rule restricts the export, so the whole catalog
// is kept and every render path keeps today's exact behaviour.
func (s *Selection) empty() bool {
	return s == nil || (len(s.include) == 0 && len(s.exclude) == 0 && len(s.neverExport) == 0)
}

// including returns a copy with the selectors added; nil-safe so options can
// be applied in any order without pre-seeding the exporter.
func (s *Selection) including(selectors ...Selector) *Selection {
	out := s.copy()
	out.include = append(out.include, selectors...)

	return out
}

// excluding returns a copy with the selectors added; nil-safe.
func (s *Selection) excluding(selectors ...Selector) *Selection {
	out := s.copy()
	out.exclude = append(out.exclude, selectors...)

	return out
}

// neverExporting returns a copy with the selectors added; nil-safe.
func (s *Selection) neverExporting(selectors ...Selector) *Selection {
	out := s.copy()
	out.neverExport = append(out.neverExport, selectors...)

	return out
}

// copy returns a fresh selection carrying s's rules (nil-safe).
func (s *Selection) copy() *Selection {
	if s == nil {
		return &Selection{include: nil, exclude: nil, neverExport: nil}
	}

	return &Selection{
		include:     slices.Clone(s.include),
		exclude:     slices.Clone(s.exclude),
		neverExport: slices.Clone(s.neverExport),
	}
}
