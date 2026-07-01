package catalog

// WalkMessageFn is called for each message in the catalog.
// Return false to stop walking.
type WalkMessageFn func(svc Service, msg Message) bool

// WalkMessages iterates all messages across all services in the catalog.
// Commands, events, and queries are visited in order per service.
// Stops early if fn returns false.
func WalkMessages(cat *Catalog, fn WalkMessageFn) {
	for _, svc := range cat.Services {
		for _, cmd := range svc.Commands {
			if !fn(svc, cmd) {
				return
			}
		}

		for _, evt := range svc.Events {
			if !fn(svc, evt) {
				return
			}
		}

		for _, q := range svc.Queries {
			if !fn(svc, q) {
				return
			}
		}
	}
}

// WalkEntityFn is called for each entity in the catalog.
type WalkEntityFn func(entity Entity) bool

// WalkEntities iterates all entities in the catalog.
// Stops early if fn returns false.
func WalkEntities(cat *Catalog, fn WalkEntityFn) {
	for _, entity := range cat.Entities {
		if !fn(entity) {
			return
		}
	}
}

// WalkDataProductFn is called for each data product in the catalog.
type WalkDataProductFn func(dp DataProduct) bool

// WalkDataProducts iterates all data products in the catalog.
// Stops early if fn returns false.
func WalkDataProducts(cat *Catalog, fn WalkDataProductFn) {
	for _, dp := range cat.DataProducts {
		if !fn(dp) {
			return
		}
	}
}

// WalkAgentFn is called for each agent in the catalog.
type WalkAgentFn func(agent Agent) bool

// WalkAgents iterates all agents in the catalog.
// Stops early if fn returns false.
func WalkAgents(cat *Catalog, fn WalkAgentFn) {
	for _, agent := range cat.Agents {
		if !fn(agent) {
			return
		}
	}
}

// WalkCustomDocFn is called for each custom doc in the catalog.
type WalkCustomDocFn func(doc CustomDoc) bool

// WalkCustomDocs iterates all custom docs in the catalog.
// Stops early if fn returns false.
func WalkCustomDocs(cat *Catalog, fn WalkCustomDocFn) {
	for _, doc := range cat.CustomDocs {
		if !fn(doc) {
			return
		}
	}
}
