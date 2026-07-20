package analyzer

// CQRSRegistry holds the cross-referenced analysis of all CQRS constructs
// found in the analyzed project.
type CQRSRegistry struct {
	Commands    []CommandInfo
	Events      []EventInfo
	Folds       []FoldInfo
	Deciders    []DeciderInfo
	Projections []ProjectionInfo

	// EventTypesEmitted tracks event type strings emitted via event.New/NewEvent.
	EventTypesEmitted map[string]EventEmission // event type string → emission location
	// EventTypesInCatalog tracks event types registered via catalog.Event.
	EventTypesInCatalog map[string]bool
	// CommandTypesRegistered tracks command types registered via RegisterTyped.
	CommandTypesRegistered map[string]bool
	// EventPayloadTypes tracks struct type names used as payload args to event.New().
	EventPayloadTypes map[string]bool

	// TypeConstValues maps a command.Type/query.Type constant name to its
	// string value, e.g. "GetVisitQueryType" → "GetVisitQuery". Populated by
	// scanning const declarations whose type is command.Type or query.Type.
	// Used to resolve type-constant arguments passed to Register/RegisterTyped
	// when the handler type cannot be extracted directly (method values, bare
	// identifiers). See browser-history feedback (E005/E007 false positives).
	TypeConstValues map[string]string

	// registeredTypeConsts records const names (bare identifier or selector's
	// Sel.Name) passed to Register/RegisterTyped whose target struct could not
	// be resolved at the call site. Resolved against TypeConstValues in a
	// post-pass after all files are scanned (the const decl may be in another
	// file/package).
	registeredTypeConsts []string

	// StrictApplyFolds records the set of fold function names that have been
	// wrapped in a decider.StrictApply call. B005 consults this to suppress the
	// "use decider.StrictApply" suggestion when it has already been adopted.
	// Keys are matched by the LAST identifier segment of the function name
	// (e.g. "foldCounter" for "(Decider).foldCounter" and bare "foldCounter"),
	// so that the fold's FuncName and the StrictApply arg resolve to the same
	// key regardless of qualification. See browser-history feedback (B005).
	StrictApplyFolds map[string]bool

	// pendingHandlerMethods records method names passed as handler arguments to
	// RegisterTyped/RegisterQuery whose target type could not be extracted at
	// the call site (method values like `h.handleCreateGame`). Resolved in a
	// post-pass by finding the method's FuncDecl and extracting the command/
	// query type from its parameter list. See SEC consumer feedback.
	pendingHandlerMethods map[string]bool
}

// NewCQRSRegistry creates an empty registry.
func NewCQRSRegistry() *CQRSRegistry {
	return &CQRSRegistry{
		EventTypesEmitted:      make(map[string]EventEmission),
		EventTypesInCatalog:    make(map[string]bool),
		CommandTypesRegistered: make(map[string]bool),
		EventPayloadTypes:      make(map[string]bool),
		TypeConstValues:        make(map[string]string),
		StrictApplyFolds:       make(map[string]bool),
		pendingHandlerMethods:  make(map[string]bool),
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
