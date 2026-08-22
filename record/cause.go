package record

// CauseKind discriminates what kind of thing caused a record. The zero value
// is CauseNone, so "no cause recorded" is an explicit state rather than a
// missing ID (review P4).
type CauseKind uint8

const (
	// CauseNone is the zero value: no cause was recorded. Direct user actions
	// carry no cause — the Actor covers the "who".
	CauseNone CauseKind = iota

	// CauseCommand: the record was produced while handling a command. This is
	// the strongest signal — it comes from event.Causation, which is typed at
	// the source rather than inferred from an ID format.
	CauseCommand

	// CauseTimer: the record was produced by a scheduled timer firing.
	CauseTimer

	// CauseEvent: the record was produced in reaction to another event
	// (saga-style derivation).
	CauseEvent

	// CauseUnknown: the causer's ID is known but its kind was not
	// discriminated at the source (a bare Tracing.CausationID). Mirrors
	// id.ActorUnknown: honest about what was — and was not — recorded.
	CauseUnknown
)

// String returns the lowercase kind name, used as the prefix in Cause.String
// (e.g. "command", "timer").
func (k CauseKind) String() string {
	switch k {
	case CauseCommand:
		return "command"
	case CauseTimer:
		return "timer"
	case CauseEvent:
		return "event"
	case CauseUnknown:
		return "unknown"
	default:
		return "none"
	}
}

// Cause identifies what produced a record, with the causer's kind stated
// explicitly instead of implied by stringly ID conventions. It is the single
// causation home that replaces CommonMetadata.CausationID at v5: today one
// has to know the precedence rules (typed Causation vs Tracing.CausationID)
// and the ID format to recover what caused a record.
//
// The zero value means "no cause recorded". Construct with a composite
// literal, e.g. record.Cause{Kind: record.CauseCommand, ID: cmdID.String()}.
type Cause struct {
	// Kind says what the causer was: a command, a timer, an event, or an
	// ID-only tracing chain (CauseUnknown).
	Kind CauseKind

	// ID is the causer's identifier: command ID, timer ID, or event ID.
	// Empty when Kind is CauseNone.
	ID string
}

// IsZero reports whether no cause is recorded. A Cause with a non-none Kind
// but an empty ID is NOT zero — the kind alone is information.
func (c Cause) IsZero() bool { return c.Kind == CauseNone }

// String returns the self-describing "kind:id" wire form (e.g.
// "command:01J..."), matching the id.ActorID "kind:raw" convention. The zero
// Cause returns "".
func (c Cause) String() string {
	if c.Kind == CauseNone {
		return ""
	}

	return c.Kind.String() + ":" + c.ID
}
