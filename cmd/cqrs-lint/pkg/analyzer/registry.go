package analyzer

// CQRSRegistry holds the cross-referenced analysis of all CQRS constructs
// found in the analyzed project.
type CQRSRegistry struct {
	Commands    []CommandInfo
	Events      []EventInfo
	Folds       []FoldInfo
	Deciders    []DeciderInfo
	Projections []ProjectionInfo
	Handlers    []HandlerInfo

	// EventTypesEmitted tracks event type strings emitted via event.New/NewEvent.
	EventTypesEmitted map[string]string // event type string → file
	// EventTypesInCatalog tracks event types registered via catalog.Event.
	EventTypesInCatalog map[string]bool
	// CommandTypesRegistered tracks command types registered via RegisterTyped.
	CommandTypesRegistered map[string]bool
}

// NewCQRSRegistry creates an empty registry.
func NewCQRSRegistry() *CQRSRegistry {
	return &CQRSRegistry{
		EventTypesEmitted:      make(map[string]string),
		EventTypesInCatalog:    make(map[string]bool),
		CommandTypesRegistered: make(map[string]bool),
	}
}

// CommandByName finds a command by struct type name.
func (r *CQRSRegistry) CommandByName(name string) *CommandInfo {
	for i := range r.Commands {
		if r.Commands[i].Name == name {
			return &r.Commands[i]
		}
	}

	return nil
}

// IsCommandRegistered returns true if a command type has been registered
// via RegisterTyped or similar.
func (r *CQRSRegistry) IsCommandRegistered(cmdType string) bool {
	return r.CommandTypesRegistered[cmdType]
}

// IsEventInCatalog returns true if an event type has been cataloged.
func (r *CQRSRegistry) IsEventInCatalog(eventType string) bool {
	return r.EventTypesInCatalog[eventType]
}

// IsEventEmitted returns true if an event type string was found in event.New/NewEvent calls.
func (r *CQRSRegistry) IsEventEmitted(eventType string) bool {
	_, ok := r.EventTypesEmitted[eventType]

	return ok
}
