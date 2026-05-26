package catalog

import (
	"maps"
	"slices"
)

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
