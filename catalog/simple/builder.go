// Package simple provides a streamlined single-service facade over the
// lower-level catalog.Builder.
//
// Most services document a single application. This package reduces the
// ceremony to a few generic calls, while the underlying catalog.Builder
// remains accessible via InnerBuilder() for multi-service, multi-domain,
// or multi-channel catalogs.
//
// Example:
//
//	b := simple.New("User Service", "1.0.0")
//	simple.Command[RegisterUserCmd](b, "register-user",
//	    simple.WithOperation("POST", "/api/users"))
//	simple.Event[UserRegisteredEvent](b, "user.registered", catalog.Sends)
//	cat := b.Build()
package simple

import (
	"fmt"

	errorfamily "github.com/larsartmann/go-error-family"

	"github.com/larsartmann/go-cqrs-lite/catalog/v4"
	"github.com/larsartmann/go-cqrs-lite/catalog/v4/internal/caseutil"
)

var ErrCatalogValidation = errorfamily.NewRejection(
	"catalog.simple.validation_failed",
	"simple: catalog validation failed",
)

// Builder accumulates messages and produces an immutable catalog.Catalog.
// It wraps catalog.Builder with a streamlined single-service API.
//
// Create one with New, then register messages using the generic
// Command[T], Query[T], or Event[T] package-level functions, or via
// AddMessage with catalog.Command[T] etc. Call Build to get the final catalog.
type Builder struct {
	inner      *catalog.Builder
	serviceID  catalog.ServiceID
	serviceCfg serviceConfig
	msgs       []catalog.MessageConfig
}

type serviceConfig struct {
	name    string
	version string
	summary string
}

// Option configures a Builder.
type Option func(*Builder)

// WithServiceName overrides the service display name (defaults to the title).
func WithServiceName(name string) Option {
	return func(b *Builder) {
		b.serviceCfg.name = name
	}
}

// WithServiceSummary sets a human-readable summary for the service.
func WithServiceSummary(summary string) Option {
	return func(b *Builder) {
		b.serviceCfg.summary = summary
	}
}

// WithServiceID overrides the service ID (defaults to kebab-case of title).
func WithServiceID(id string) Option {
	return func(b *Builder) {
		b.serviceID = catalog.ServiceID(id)
	}
}

// New creates a Builder for a single-service catalog.
// The title becomes both the catalog title and the default service name.
// The version applies to both the catalog and the service.
func New(title, version string, opts ...Option) *Builder {
	b := &Builder{ //nolint:exhaustruct // fields set below
		inner: catalog.NewBuilder(title, version),
		serviceCfg: serviceConfig{ //nolint:exhaustruct // summary is optional
			name:    title,
			version: version,
		},
	}

	for _, opt := range opts {
		opt(b)
	}

	if b.serviceID == "" {
		b.serviceID = catalog.ServiceID(caseutil.ToKebab(title))
	}

	return b
}

// Command registers a command message on the builder.
// The schema is auto-derived from T via reflection on its struct fields
// and tags. The name is auto-derived from T's type name
// (e.g., RegisterUserCmd → "Register User").
//
// Returns the builder for potential chaining.
func Command[T any](b *Builder, id string, opts ...catalog.MessageOption) *Builder {
	b.msgs = append(b.msgs, catalog.Command[T](catalog.MessageID(id), opts...))

	return b
}

// Query registers a query message on the builder.
// The schema is auto-derived from T. Direction defaults to Receives.
//
// Returns the builder for potential chaining.
func Query[T any](b *Builder, id string, opts ...catalog.MessageOption) *Builder {
	b.msgs = append(b.msgs, catalog.Query[T](catalog.MessageID(id), opts...))

	return b
}

// Event registers an event message on the builder.
// The schema is auto-derived from T. The direction must be explicit
// (catalog.Sends or catalog.Receives).
//
// Returns the builder for potential chaining.
func Event[T any](
	b *Builder,
	id string,
	direction catalog.Direction,
	opts ...catalog.MessageOption,
) *Builder {
	b.msgs = append(b.msgs, catalog.Event[T](catalog.MessageID(id), direction, opts...))

	return b
}

// AddMessage adds a pre-built MessageConfig (from catalog.Command[T],
// catalog.Event[T], or catalog.Query[T]). This is an alternative to the
// generic Command[T]/Query[T]/Event[T] package-level functions.
func (b *Builder) AddMessage(msg catalog.MessageConfig) *Builder {
	b.msgs = append(b.msgs, msg)

	return b
}

// addConfiguredService registers the single service with all accumulated
// messages on the underlying catalog builder. Shared by Build (panics on
// validation errors) and BuildValid (returns them).
func (b *Builder) addConfiguredService() {
	b.inner.AddService(
		b.serviceID,
		b.serviceCfg.name,
		b.serviceCfg.version,
		b.serviceCfg.summary,
		b.msgs...,
	)
}

// buildInner finalises the configured service and returns the built catalog.
func (b *Builder) buildInner() *catalog.Catalog {
	b.addConfiguredService()

	return b.inner.Build()
}

// Build creates the immutable catalog with all registered messages.
// Panics if catalog validation fails (empty service ID, duplicate message IDs, etc.).
// This is a Must-style convenience — use [Builder.BuildE] for an error-returning variant.
func (b *Builder) Build() *catalog.Catalog {
	cat := b.buildInner()

	if violations := cat.Validate(); len(violations) > 0 {
		panic(fmt.Errorf("%w: %v", ErrCatalogValidation, violations))
	}

	return cat
}

// BuildE creates the immutable catalog and returns validation violations as an
// error instead of panicking. Returns nil if the catalog is valid.
func (b *Builder) BuildE() (*catalog.Catalog, error) {
	cat := b.buildInner()

	if violations := cat.Validate(); len(violations) > 0 {
		return cat, fmt.Errorf("%w: %v", ErrCatalogValidation, violations)
	}

	return cat, nil
}

// BuildValid creates the immutable catalog and returns validation violations
// instead of panicking. Returns nil violations if the catalog is valid.
func (b *Builder) BuildValid() (*catalog.Catalog, []catalog.Violation) {
	cat := b.buildInner()

	return cat, cat.Validate()
}

// Registry returns the underlying catalog.Registry for advanced use cases.
func (b *Builder) Registry() *catalog.Registry {
	return b.inner.Registry()
}

// InnerBuilder returns the underlying catalog.Builder.
// Use this for multi-service catalogs, domains, channels, or other
// catalog features not exposed by this wrapper.
func (b *Builder) InnerBuilder() *catalog.Builder {
	return b.inner
}

// AddEntity registers a domain entity on the underlying builder.
func (b *Builder) AddEntity(entity catalog.Entity) *Builder {
	b.inner.AddEntity(entity)

	return b
}

// Entity registers a typed entity with auto-derived schema on the builder.
func Entity[T any](b *Builder, id string) *Builder {
	b.inner.AddEntity(catalog.Entity{ //nolint:exhaustruct // optional fields default to zero
		ID:      catalog.EntityID(id),
		Name:    catalog.Name(id),
		Version: catalog.Version(b.serviceCfg.version),
		Schema:  catalog.SchemaFromType[T](),
	})

	return b
}

// AddDataProduct registers a data product on the underlying builder.
func (b *Builder) AddDataProduct(dp catalog.DataProduct) *Builder {
	b.inner.AddDataProduct(dp)

	return b
}

// AddAgent registers an AI agent on the underlying builder.
func (b *Builder) AddAgent(agent catalog.Agent) *Builder {
	b.inner.AddAgent(agent)

	return b
}

// AddDomain registers a domain on the underlying builder.
func (b *Builder) AddDomain(
	id catalog.DomainID,
	name, version, summary string,
	services ...catalog.ServiceID,
) *Builder {
	b.inner.AddDomain(id, name, version, summary, services...)

	return b
}

// WithOperation attaches HTTP endpoint metadata to a message.
// The OpenAPI exporter uses this to generate accurate paths.
//
// This is a re-export of catalog.MsgOperation for convenience.
func WithOperation(method, path string) catalog.MessageOption {
	return catalog.MsgOperation(method, path)
}
