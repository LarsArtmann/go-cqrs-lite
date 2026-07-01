package catalog

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

func (r *Registry) AddChannel(ch Channel) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.channels[ch.ID] = copyPtr(copyChannel, ch)
}

func (r *Registry) AddDataStore(ds DataStore) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.stores[ds.ID] = copyPtr(copyDataStore, ds)
}

func (r *Registry) AddFlow(f Flow) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.flows[f.ID] = copyPtr(copyFlow, f)
}

func (r *Registry) AddTeam(team Team) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.teams[team.ID] = copyPtr(copyTeam, team)
}

func (r *Registry) AddUser(user User) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.users[user.ID] = copyPtr(copyUser, user)
}

func (r *Registry) AddEntity(entity Entity) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.entities[entity.ID] = copyPtr(copyEntity, entity)
}

func (r *Registry) AddDataProduct(dp DataProduct) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.dataProducts[dp.ID] = copyPtr(copyDataProduct, dp)
}

func (r *Registry) AddAgent(agent Agent) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.agents[agent.ID] = copyPtr(copyAgent, agent)
}

func (r *Registry) AddCustomDoc(doc CustomDoc) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.customDocs[doc.ID] = copyPtr(copyCustomDoc, doc)
}
