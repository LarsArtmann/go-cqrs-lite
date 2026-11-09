package record

// Type is a domain message type identifier: the event type ("user.created"),
// command type ("create_user"), or query type ("get_user").
//
// event.Type, command.Type, and query.Type are aliases of this type
// (ADR-0111): one definition, so the per-module copies cannot drift.
type Type string

// String returns the type as a plain string.
func (t Type) String() string { return string(t) }

// IsZero reports whether the type is empty.
func (t Type) IsZero() bool { return t == "" }

// ParseType validates s and returns a Type. Returns emptyErr when s is
// empty — the sentinel is parametrized so each calling module preserves its
// own error identity (event.ErrEmptyEventType, command.ErrEmptyCommandType,
// query.ErrEmptyQueryType).
func ParseType(s string, emptyErr error) (Type, error) {
	if s == "" {
		return "", emptyErr
	}

	return Type(s), nil
}
