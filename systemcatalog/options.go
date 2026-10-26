package systemcatalog

import (
	"strings"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
)

const (
	defaultTitle   = "System"
	defaultVersion = "1.0.0"
)

// config carries the resolved derivation options.
type config struct {
	title     string
	version   string
	serviceID catalog.ServiceID
	name      string
	summary   string
	owners    []string
	overrides map[string]catalog.MessageConfig
	curator   func(*catalog.Builder)
}

// Option configures catalog derivation.
type Option func(*config)

// WithTitle sets the catalog title (also the default service display name
// and, unless [WithServiceID] overrides it, the source of the service id).
func WithTitle(title string) Option {
	return func(c *config) {
		c.title = title
	}
}

// WithVersion sets the catalog and service version
// (default "1.0.0").
func WithVersion(version string) Option {
	return func(c *config) {
		c.version = version
	}
}

// WithServiceID overrides the service id (default: title lowercased and
// hyphenated).
func WithServiceID(id string) Option {
	return func(c *config) {
		c.serviceID = catalog.ServiceID(id)
	}
}

// WithServiceName overrides the service display name (default: the title).
func WithServiceName(name string) Option {
	return func(c *config) {
		c.name = name
	}
}

// WithServiceSummary sets a human-readable summary for the derived service.
func WithServiceSummary(summary string) Option {
	return func(c *config) {
		c.summary = summary
	}
}

// WithServiceOwners sets owner references on the derived service. The
// referenced users/teams themselves are registered by the curator via
// builder.AddUser/AddTeam.
func WithServiceOwners(owners ...string) Option {
	return func(c *config) {
		c.owners = append(c.owners, owners...)
	}
}

// WithCurator runs fn with the builder AFTER derivation registered the
// service and its messages — the hook for curated resources (users, domains,
// data stores, flows) that derivation must never own.
func WithCurator(fn func(*catalog.Builder)) Option {
	return func(c *config) {
		c.curator = fn
	}
}

// WithMessageOverrides supplies curated message configs BY ID: derivation
// skips deriving the given ids and registers the curated configs instead.
// Use it where a message deserves more than derivation can know — a typed
// payload schema with doc tags, a curated display name, a summary. The
// override set is deliberately explicit (map key = message id): the derived
// inventory stays the source of truth for WHAT exists, the overrides only
// enrich HOW chosen messages document.
func WithMessageOverrides(overrides map[string]catalog.MessageConfig) Option {
	return func(c *config) {
		if c.overrides == nil {
			c.overrides = make(map[string]catalog.MessageConfig, len(overrides))
		}

		for id, msg := range overrides {
			c.overrides[id] = msg
		}
	}
}

// resolveOptions applies opts over the defaults.
func resolveOptions(opts []Option) config {
	cfg := config{ //nolint:exhaustruct_v5 // option-driven: unset fields are zero by design
		title:   defaultTitle,
		version: defaultVersion,
	}

	for _, opt := range opts {
		opt(&cfg)
	}

	if cfg.name == "" {
		cfg.name = cfg.title
	}

	if cfg.serviceID == "" {
		cfg.serviceID = catalog.ServiceID(defaultServiceID(cfg.title))
	}

	return cfg
}

// defaultServiceID lowercases the title and joins its fields with hyphens
// ("Invoice Service" → "invoice-service").
func defaultServiceID(title string) string {
	return strings.ToLower(strings.Join(strings.Fields(title), "-"))
}
