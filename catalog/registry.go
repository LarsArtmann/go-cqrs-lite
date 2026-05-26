package catalog

import (
	"fmt"
	"sync"

	errorfamily "github.com/larsartmann/go-error-family"
)

// ErrDomainNotFound is returned when a domain ID does not exist in the registry.
var ErrDomainNotFound = errorfamily.NewRejection("catalog.domain_not_found", "domain not found")

// Registry is a thread-safe catalog builder that accumulates services,
// domains, channels, data stores, flows, teams, and users before
// producing an immutable Catalog.
type Registry struct {
	mu       sync.RWMutex
	title    string
	version  string
	services map[ServiceID]*Service
	domains  map[DomainID]*Domain
	channels map[ChannelID]*Channel
	stores   map[DataStoreID]*DataStore
	flows    map[FlowID]*Flow
	teams    map[TeamID]*Team
	users    map[UserID]*User
}

// NewRegistry creates a new catalog registry with the given title and version.
func NewRegistry(title, version string) *Registry {
	return &Registry{
		mu:       sync.RWMutex{},
		title:    title,
		version:  version,
		services: make(map[ServiceID]*Service),
		domains:  make(map[DomainID]*Domain),
		channels: make(map[ChannelID]*Channel),
		stores:   make(map[DataStoreID]*DataStore),
		flows:    make(map[FlowID]*Flow),
		teams:    make(map[TeamID]*Team),
		users:    make(map[UserID]*User),
	}
}

// SetServiceMeta updates the metadata fields of an existing service.
// If the service does not exist, it is created with the given metadata.
func (r *Registry) SetServiceMeta(serviceID ServiceID, name, version, summary string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	svc := r.ensureServiceEntry(serviceID)
	svc.Name = name
	svc.Version = version

	if summary != "" {
		svc.Summary = summary
	}
}

// SetServiceOptions applies ServiceOption functions to an existing service.
func (r *Registry) SetServiceOptions(serviceID ServiceID, opts ...ServiceOption) {
	r.mu.Lock()
	defer r.mu.Unlock()

	svc := r.ensureServiceEntry(serviceID)

	for _, opt := range opts {
		opt(svc)
	}
}

// AddService registers a service or merges messages into an existing service.
func (r *Registry) AddService(svc Service) {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, ok := r.services[svc.ID]
	if ok {
		if len(svc.Commands) > 0 {
			existing.Commands = append(existing.Commands, svc.Commands...)
		}

		if len(svc.Events) > 0 {
			existing.Events = append(existing.Events, svc.Events...)
		}

		if len(svc.Queries) > 0 {
			existing.Queries = append(existing.Queries, svc.Queries...)
		}

		return
	}

	r.services[svc.ID] = copyServicePtr(svc)
}

// ensureServiceEntry returns a service entry for the given ID, creating it if needed.
func (r *Registry) ensureServiceEntry(serviceID ServiceID) *Service {
	svc, ok := r.services[serviceID]
	if !ok {
		svc = &Service{
			ID:       serviceID,
			Name:     string(serviceID),
			Version:  "",
			Summary:  "",
			Owners:   nil,
			Commands: []Message{},
			Events:   []Message{},
			Queries:  []Message{},
		}
		r.services[serviceID] = svc
	}

	return svc
}

func (r *Registry) addMessage(
	serviceID ServiceID,
	kind MessageKind,
	getter func(*Service) []Message,
	setter func(*Service, []Message),
	msg Message,
) {
	r.mu.Lock()
	defer r.mu.Unlock()

	msg.Kind = kind
	svc := r.ensureServiceEntry(serviceID)
	setter(svc, append(getter(svc), msg))
}

// AddCommand adds a command message to a service, creating the service if needed.
func (r *Registry) AddCommand(serviceID ServiceID, msg Message) {
	r.addMessage(
		serviceID,
		CommandMessage,
		func(s *Service) []Message { return s.Commands },
		func(s *Service, m []Message) { s.Commands = m },
		msg,
	)
}

// AddEvent adds an event message to a service, creating the service if needed.
func (r *Registry) AddEvent(serviceID ServiceID, msg Message) {
	r.addMessage(
		serviceID,
		EventMessage,
		func(s *Service) []Message { return s.Events },
		func(s *Service, m []Message) { s.Events = m },
		msg,
	)
}

// AddQuery adds a query message to a service, creating the service if needed.
func (r *Registry) AddQuery(serviceID ServiceID, msg Message) {
	r.addMessage(
		serviceID,
		QueryMessage,
		func(s *Service) []Message { return s.Queries },
		func(s *Service, m []Message) { s.Queries = m },
		msg,
	)
}

// SetDomainOptions applies DomainOption functions to an existing domain.
// If the domain does not exist, the call is silently ignored.
func (r *Registry) SetDomainOptions(domainID DomainID, opts ...DomainOption) {
	r.mu.Lock()
	defer r.mu.Unlock()

	d, ok := r.domains[domainID]
	if !ok {
		return
	}

	for _, opt := range opts {
		opt(d)
	}
}

// AddDomain registers a domain in the catalog.
func (r *Registry) AddDomain(domain Domain) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.domains[domain.ID] = copyDomainPtr(domain)
}

// AddServiceToDomain associates an existing service with an existing domain.
func (r *Registry) AddServiceToDomain(serviceID ServiceID, domainID DomainID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	d, ok := r.domains[domainID]
	if !ok {
		return fmt.Errorf("add service %q to domain %q: %w", serviceID, domainID, ErrDomainNotFound)
	}

	d.Services = append(d.Services, serviceID)

	return nil
}

// SetChannelOptions applies ChannelOption functions to an existing channel.
// If the channel does not exist, the call is silently ignored.
func (r *Registry) SetChannelOptions(channelID ChannelID, opts ...ChannelOption) {
	r.mu.Lock()
	defer r.mu.Unlock()

	ch, ok := r.channels[channelID]
	if !ok {
		return
	}

	for _, opt := range opts {
		opt(ch)
	}
}

// AddChannel registers a channel in the catalog.
func (r *Registry) AddChannel(ch Channel) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.channels[ch.ID] = copyChannelPtr(ch)
}

// AddDataStore registers a data store in the catalog.
func (r *Registry) AddDataStore(ds DataStore) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.stores[ds.ID] = copyDataStorePtr(ds)
}

// AddFlow registers a flow in the catalog.
func (r *Registry) AddFlow(f Flow) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.flows[f.ID] = copyFlowPtr(f)
}

// AddTeam registers a team in the catalog.
func (r *Registry) AddTeam(team Team) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.teams[team.ID] = copyTeamPtr(team)
}

// AddUser registers a user in the catalog.
func (r *Registry) AddUser(user User) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.users[user.ID] = copyUserPtr(user)
}

