package system

import (
	"cmp"
	"reflect"
	"slices"

	"github.com/larsartmann/go-cqrs-lite/command/v4"
	"github.com/larsartmann/go-cqrs-lite/query/v4"
)

// CommandRegistration describes one command handler registered on the System
// via [RegisterCommand]: its wire type and the Go payload type the handler
// decodes. PayloadType is the static Cmd type parameter — for handlers
// registered with an interface type parameter it is that interface type.
type CommandRegistration struct {
	Type        command.Type
	PayloadType reflect.Type
}

// QueryRegistration describes one query handler registered on the System via
// [RegisterQuery]: its wire type and the Go query type the handler decodes.
type QueryRegistration struct {
	Type        query.Type
	PayloadType reflect.Type
}

// CommandRegistrations returns the command handlers registered on this
// System, sorted by command type. It is the runtime-inventory surface:
// documentation bridges (catalogs, audit logs) derive the command universe
// from it instead of hand-maintained lists.
func (s *System) CommandRegistrations() []CommandRegistration {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]CommandRegistration, 0, len(s.cmdTypes))
	for cmdType, payloadType := range s.cmdTypes {
		out = append(out, CommandRegistration{Type: cmdType, PayloadType: payloadType})
	}

	slices.SortFunc(out, func(a, b CommandRegistration) int {
		return cmp.Compare(string(a.Type), string(b.Type))
	})

	return out
}

// QueryRegistrations returns the query handlers registered on this System,
// sorted by query type. See [CommandRegistrations] for the intent.
func (s *System) QueryRegistrations() []QueryRegistration {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]QueryRegistration, 0, len(s.qryTypes))
	for qryType, payloadType := range s.qryTypes {
		out = append(out, QueryRegistration{Type: qryType, PayloadType: payloadType})
	}

	slices.SortFunc(out, func(a, b QueryRegistration) int {
		return cmp.Compare(string(a.Type), string(b.Type))
	})

	return out
}
