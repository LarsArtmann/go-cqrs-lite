package catalog

import (
	"fmt"
	"sync"
)

func copySlice[T any](s []T) []T {
	if s == nil {
		return nil
	}

	cp := make([]T, len(s))
	copy(cp, s)

	return cp
}

func copyService(s *Service) Service {
	return Service{
		ID:       s.ID,
		Name:     s.Name,
		Version:  s.Version,
		Summary:  s.Summary,
		Owners:   copySlice(s.Owners),
		Commands: copySlice(s.Commands),
		Events:   copySlice(s.Events),
		Queries:  copySlice(s.Queries),
	}
}

func copyDomain(d *Domain) Domain {
	return Domain{
		ID:       d.ID,
		Name:     d.Name,
		Version:  d.Version,
		Summary:  d.Summary,
		Services: copySlice(d.Services),
	}
}

func copyChannel(ch *Channel) Channel {
	return Channel{
		ID:        ch.ID,
		Name:      ch.Name,
		Version:   ch.Version,
		Summary:   ch.Summary,
		Address:   ch.Address,
		Protocols: copySlice(ch.Protocols),
		Messages:  copySlice(ch.Messages),
	}
}

type Registry struct {
	mu       sync.RWMutex
	title    string
	version  string
	services map[string]*Service
	domains  map[string]*Domain
	channels map[string]*Channel
}

func NewRegistry(title, version string) *Registry {
	return &Registry{
		title:    title,
		version:  version,
		services: make(map[string]*Service),
		domains:  make(map[string]*Domain),
		channels: make(map[string]*Channel),
	}
}

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

	r.services[svc.ID] = &svc
}

func (r *Registry) AddCommand(serviceID string, msg Message) {
	r.addMessage(
		serviceID,
		CommandMessage,
		func(s *Service) []Message { return s.Commands },
		func(s *Service, m []Message) { s.Commands = m },
		msg,
	)
}

func (r *Registry) AddEvent(serviceID string, msg Message) {
	r.addMessage(
		serviceID,
		EventMessage,
		func(s *Service) []Message { return s.Events },
		func(s *Service, m []Message) { s.Events = m },
		msg,
	)
}

func (r *Registry) AddQuery(serviceID string, msg Message) {
	r.addMessage(
		serviceID,
		QueryMessage,
		func(s *Service) []Message { return s.Queries },
		func(s *Service, m []Message) { s.Queries = m },
		msg,
	)
}

func (r *Registry) addMessage(
	serviceID string,
	kind MessageKind,
	getter func(*Service) []Message,
	setter func(*Service, []Message),
	msg Message,
) {
	r.mu.Lock()
	defer r.mu.Unlock()

	msg.Kind = kind

	svc, ok := r.services[serviceID]
	if !ok {
		svc = &Service{ID: serviceID, Name: serviceID}
		r.services[serviceID] = svc
	}

	setter(svc, append(getter(svc), msg))
}

func (r *Registry) AddDomain(domain Domain) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.domains[domain.ID] = &domain
}

func (r *Registry) AddServiceToDomain(serviceID, domainID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	d, ok := r.domains[domainID]
	if !ok {
		//nolint:err113 // dynamic error required to include domainID in message
		return fmt.Errorf("domain %q not found", domainID)
	}

	d.Services = append(d.Services, serviceID)

	return nil
}

func (r *Registry) AddChannel(ch Channel) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.channels[ch.ID] = &ch
}

func (r *Registry) Build() *Catalog {
	r.mu.RLock()
	defer r.mu.RUnlock()

	services := make([]Service, 0, len(r.services))
	for _, svc := range r.services {
		services = append(services, copyService(svc))
	}

	domains := make([]Domain, 0, len(r.domains))
	for _, d := range r.domains {
		domains = append(domains, copyDomain(d))
	}

	channels := make([]Channel, 0, len(r.channels))
	for _, ch := range r.channels {
		channels = append(channels, copyChannel(ch))
	}

	return &Catalog{
		Title:    r.title,
		Version:  r.version,
		Services: services,
		Domains:  domains,
		Channels: channels,
	}
}
