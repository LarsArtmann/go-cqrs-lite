package catalog

import (
	"fmt"
	"maps"
	"slices"
	"sync"

	errorfamily "github.com/larsartmann/go-error-family"
)

// ErrDomainNotFound is returned when a domain ID does not exist in the registry.
var ErrDomainNotFound = errorfamily.NewRejection("catalog.domain_not_found", "domain not found")

// Registry is a thread-safe catalog builder that accumulates services,
// domains, channels, data stores, flows, teams, and users before
// producing an immutable Catalog.
type Registry struct {
	mu        sync.RWMutex
	title     string
	version   string
	services  map[ServiceID]*Service
	domains   map[DomainID]*Domain
	channels  map[ChannelID]*Channel
	stores    map[string]*DataStore
	flows     map[string]*Flow
	teams     map[string]*Team
	users     map[string]*User
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
		stores:   make(map[string]*DataStore),
		flows:    make(map[string]*Flow),
		teams:    make(map[string]*Team),
		users:    make(map[string]*User),
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

func copyServicePtr(s Service) *Service {
	cp := copyService(&s)

	return &cp
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

// AddDomain registers a domain in the catalog.
func (r *Registry) AddDomain(domain Domain) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.domains[domain.ID] = copyDomainPtr(domain)
}

func copyDomainPtr(d Domain) *Domain {
	cp := copyDomain(&d)

	return &cp
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

// AddChannel registers a channel in the catalog.
func (r *Registry) AddChannel(ch Channel) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.channels[ch.ID] = copyChannelPtr(ch)
}

func copyChannelPtr(ch Channel) *Channel {
	cp := copyChannel(&ch)

	return &cp
}

// AddDataStore registers a data store in the catalog.
func (r *Registry) AddDataStore(ds DataStore) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.stores[ds.ID] = copyDataStorePtr(ds)
}

func copyDataStorePtr(ds DataStore) *DataStore {
	cp := copyDataStore(&ds)

	return &cp
}

// AddFlow registers a flow in the catalog.
func (r *Registry) AddFlow(f Flow) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.flows[f.ID] = copyFlowPtr(f)
}

func copyFlowPtr(f Flow) *Flow {
	cp := copyFlow(&f)

	return &cp
}

// AddTeam registers a team in the catalog.
func (r *Registry) AddTeam(team Team) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.teams[team.ID] = copyTeamPtr(team)
}

func copyTeamPtr(t Team) *Team {
	cp := copyTeam(&t)

	return &cp
}

// AddUser registers a user in the catalog.
func (r *Registry) AddUser(user User) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.users[user.ID] = copyUserPtr(user)
}

func copyUserPtr(u User) *User {
	cp := copyUser(&u)

	return &cp
}

// Build returns an immutable Catalog with all registered entries.
func (r *Registry) Build() *Catalog {
	r.mu.RLock()
	defer r.mu.RUnlock()

	serviceKeys := slices.Sorted(maps.Keys(r.services))

	services := make([]Service, 0, len(r.services))
	for _, key := range serviceKeys {
		services = append(services, copyService(r.services[key]))
	}

	domainKeys := slices.Sorted(maps.Keys(r.domains))

	domains := make([]Domain, 0, len(r.domains))
	for _, key := range domainKeys {
		domains = append(domains, copyDomain(r.domains[key]))
	}

	channelKeys := slices.Sorted(maps.Keys(r.channels))

	channels := make([]Channel, 0, len(r.channels))
	for _, key := range channelKeys {
		channels = append(channels, copyChannel(r.channels[key]))
	}

	storeKeys := slices.Sorted(maps.Keys(r.stores))

	dataStores := make([]DataStore, 0, len(r.stores))
	for _, key := range storeKeys {
		dataStores = append(dataStores, copyDataStore(r.stores[key]))
	}

	flowKeys := slices.Sorted(maps.Keys(r.flows))

	flows := make([]Flow, 0, len(r.flows))
	for _, key := range flowKeys {
		flows = append(flows, copyFlow(r.flows[key]))
	}

	teamKeys := slices.Sorted(maps.Keys(r.teams))

	teams := make([]Team, 0, len(r.teams))
	for _, key := range teamKeys {
		teams = append(teams, copyTeam(r.teams[key]))
	}

	userKeys := slices.Sorted(maps.Keys(r.users))

	users := make([]User, 0, len(r.users))
	for _, key := range userKeys {
		users = append(users, copyUser(r.users[key]))
	}

	return &Catalog{
		Title:      r.title,
		Version:    r.version,
		Services:   services,
		Domains:    domains,
		Channels:   channels,
		DataStores: dataStores,
		Flows:      flows,
		Teams:      teams,
		Users:      users,
	}
}
