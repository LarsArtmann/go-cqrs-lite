package catalog

func (r *Registry) Build() *Catalog {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return &Catalog{
		Title:        Title(r.title),
		Version:      Version(r.version),
		Services:     sortedCopy(r.services, copyService),
		Domains:      sortedCopy(r.domains, copyDomain),
		Channels:     sortedCopy(r.channels, copyChannel),
		DataStores:   sortedCopy(r.stores, copyDataStore),
		Flows:        sortedCopy(r.flows, copyFlow),
		Teams:        sortedCopy(r.teams, copyTeam),
		Users:        sortedCopy(r.users, copyUser),
		Entities:     sortedCopy(r.entities, copyEntity),
		DataProducts: sortedCopy(r.dataProducts, copyDataProduct),
		Agents:       sortedCopy(r.agents, copyAgent),
		CustomDocs:   sortedCopy(r.customDocs, copyCustomDoc),
	}
}
