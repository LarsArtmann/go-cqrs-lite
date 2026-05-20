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
