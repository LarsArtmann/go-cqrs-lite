package analyzer

import "go/ast"

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
	// EventTypesInSchemaDecl tracks event types declared via schema.Event /
	// schema.EventOf / the system.Schemas() builder's Event method — the
	// declaration list that drives upcasting and projection decoding
	// (DomainConfig.Schema). E021 consumes it as the declared-set.
	EventTypesInSchemaDecl map[string]EventEmission // event type string → declaration location
	// CommandTypesRegistered tracks command types registered via RegisterTyped.
	// Keys MUST be struct type names — constructor-call registrations are kept
	// in ConstructorHandlers instead (T20-4).
	CommandTypesRegistered map[string]bool
	// ConstructorHandlers records the call text of RegisterTyped/RegisterQuery
	// handlers passed as constructor calls (e.g. `NewMyCommand(bus)`). The
	// concrete command type is not nameable from the call site alone; these
	// are kept OUT of CommandTypesRegistered (whose keys must be type names
	// that lookups can match) and surfaced separately for dumps and future
	// constructor-aware rules (T20-4).
	ConstructorHandlers map[string]bool
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

	// constAliasExprs records const declarations whose value is a reference
	// to another constant (bare identifier, selector expression, or
	// string(...) conversion) rather than a literal — including type-inherited
	// aliases like `eventUserRegistered = identitymodel.EventUserRegistered`.
	// Resolved into TypeConstValues by ResolveEmittedEventTypeConsts, because
	// the referenced constant may live in another file or package scanned
	// later. See cqrs-htmx feedback (C040 phantoms on alias-emitted events).
	constAliasExprs map[string]ast.Expr

	// pendingEmittedEventTypeRefs and pendingCatalogEventTypeRefs record
	// event-type arguments passed to event.New/event.NewEvent/catalog.Event
	// that could not be resolved to a string at the call site (constant
	// identifiers). Resolved against TypeConstValues by
	// ResolveEmittedEventTypeConsts after all files are scanned — the const
	// declaration and the emission site may live in different files/packages,
	// and alias chains only resolve once every declaration is scanned.
	pendingEmittedEventTypeRefs []pendingEventTypeRef
	pendingCatalogEventTypeRefs []pendingEventTypeRef

	// pendingSchemaEventTypeRefs records event-type arguments passed to
	// schema declarations (schema.Event/EventOf, builder .Event) that could
	// not be resolved to a string at the call site. Resolved alongside the
	// other pending refs by ResolveEmittedEventTypeConsts.
	pendingSchemaEventTypeRefs []pendingEventTypeRef

	// schemaBuilderIdents records per-file local variables initialized from
	// system.Schemas() ("schemas := system.Schemas()"), so builder-method
	// .Event calls through the variable resolve. Same-file only — the fluent
	// API is declared and consumed in one place.
	schemaBuilderIdents map[string]map[string]bool // file path → ident → true

	// emitHelperParams records constructor helpers whose event.New call
	// passes one of the helper's own PARAMETERS as the event type
	// (func newRoomEvent(t event.Type, ...) { event.New(t, ...) }). The
	// parameter cannot be resolved at the emission site; call sites of the
	// helper carry the constant. Maps helper function name → parameter
	// index. Name-matched like StrictApplyFolds (cross-package collisions
	// are a documented limitation). See nsfw-classifier feedback (2026-10-03):
	// helper indirection made soft-delete detection blind.
	emitHelperParams map[string]int

	// TypesWithTypeMethod records struct type names that have a Type() method.
	// Used by E007 to distinguish real CQRS query types (which implement
	// query.Query's Type() method) from DTOs whose name happens to end in "Query".
	TypesWithTypeMethod map[string]bool

	// MetaengineQueries records every metaengine.Query[I,R](collection, ...)
	// declaration with its pushdown capabilities — the utilization rules
	// (F022/F023) coach Go-side sort/filter sites over a registered R type
	// whose declaration lacks SortOnField/FilterOnField (nsfw-classifier
	// feedback, 2026-10-03).
	MetaengineQueries []QueryDeclInfo

	// DataProducts records every catalog.AddDataProduct declaration site
	// with its contract completeness, for the E019 advisory.
	DataProducts []DataProductInfo
}

// QueryDeclInfo is one metaengine.Query declaration as seen at its call site.
type QueryDeclInfo struct {
	Collection string // first argument (string literal or constant name)
	ResultType string // R type name from the generic instantiation
	Volume     int    // metaengine.Volume(n) literal; 0 when absent
	HasVolume  bool
	// HasFilterOnField/HasSortOnField report declarative pushdown options on
	// the declaration.
	HasFilterOnField bool
	HasSortOnField   bool
	File             string
	Line             int
}

// DataProductInfo is one catalog data-product declaration as seen at its
// AddDataProduct call site.
type DataProductInfo struct {
	Name         string
	File         string
	Line, Column int
	OutputCount  int
	Contracted   int // outputs carrying an explicit Contract
}

// NewCQRSRegistry creates an empty registry.
func NewCQRSRegistry() *CQRSRegistry {
	return &CQRSRegistry{
		EventTypesEmitted:      make(map[string]EventEmission),
		EventTypesInCatalog:    make(map[string]bool),
		EventTypesInSchemaDecl: make(map[string]EventEmission),
		CommandTypesRegistered: make(map[string]bool),
		ConstructorHandlers:    make(map[string]bool),
		EventPayloadTypes:      make(map[string]bool),
		TypeConstValues:        make(map[string]string),
		StrictApplyFolds:       make(map[string]bool),
		pendingHandlerMethods:  make(map[string]bool),
		constAliasExprs:        make(map[string]ast.Expr),
		TypesWithTypeMethod:    make(map[string]bool),
		emitHelperParams:       make(map[string]int),
	}
}

// pendingEventTypeRef is one event-type argument that could not be resolved
// to a string at its call site and awaits post-pass const resolution.
type pendingEventTypeRef struct {
	constName string
	file      string
	line      int
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

// IsEventSchemaDeclared returns true if an event type carries a schema
// declaration (schema.Event/EventOf or the system.Schemas() builder).
func (r *CQRSRegistry) IsEventSchemaDeclared(eventType string) bool {
	_, ok := r.EventTypesInSchemaDecl[eventType]
	return ok
}
