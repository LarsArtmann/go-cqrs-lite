package catalog

// Builder accumulates services and domains, then produces an immutable Catalog.
// It is the primary entry point for the zero-cost catalog API.
//
// Schemas, names, and directions are auto-derived from the generic type
// parameters passed to Command[T](), Event[T](), and Query[T].
//
// Example:
//
//	builder := catalog.NewBuilder("User Service", "1.0.0")
//	builder.AddService("user-svc", "User Service", "1.0.0", "Manages users",
//	    catalog.Command[CreateUserCmd]("user.create"),
//	    catalog.Event[UserCreatedEvent]("user.created", catalog.Sends),
//	    catalog.Query[GetUserQuery]("user.get"),
//	)
//	builder.AddDomain("identity", "Identity", "1.0.0",
//	    "User identity management", "user-svc")
//	cat := builder.Build()
type Builder struct {
	registry *Registry
}

// NewBuilder creates a catalog builder with the given title and version.
func NewBuilder(title, version string) *Builder {
	return &Builder{
		registry: NewRegistry(title, version),
	}
}

// AddService registers a service with messages. Messages are created via
// Command[T](), Event[T](), and Query[T]().
func (b *Builder) AddService(
	id, name, version, summary string,
	msgs ...MessageConfig,
) {
	b.registry.SetServiceMeta(ServiceID(id), name, version, summary)

	for _, m := range msgs {
		m.apply(ServiceID(id), b.registry)
	}
}

// AddDomain registers a domain and associates it with services.
func (b *Builder) AddDomain(
	id, name, version, summary string,
	serviceIDs ...string,
) {
	sids := make([]ServiceID, len(serviceIDs))
	for i, s := range serviceIDs {
		sids[i] = ServiceID(s)
	}

	b.registry.AddDomain(Domain{
		ID:       DomainID(id),
		Name:     name,
		Version:  version,
		Summary:  summary,
		Services: sids,
	})
}

// Registry returns the underlying registry for advanced use cases.
// Most consumers should use AddService and Build instead.
func (b *Builder) Registry() *Registry {
	return b.registry
}

// Build returns the immutable Catalog with all registered entries.
func (b *Builder) Build() *Catalog {
	return b.registry.Build()
}
