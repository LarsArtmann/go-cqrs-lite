package catalog

func (r *Registry) Build() *Catalog {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return &Catalog{
		Title:      r.title,
		Version:    r.version,
		Services:   sortedCopy(r.services, copyService),
		Domains:    sortedCopy(r.domains, copyDomain),
		Channels:   sortedCopy(r.channels, copyChannel),
		DataStores: sortedCopy(r.stores, copyDataStore),
		Flows:      sortedCopy(r.flows, copyFlow),
		Teams:      sortedCopy(r.teams, copyTeam),
		Users:      sortedCopy(r.users, copyUser),
	}
}
