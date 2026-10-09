package schema

import (
	"sort"
	"strconv"

	errorfamily "github.com/larsartmann/go-error-family"
	"github.com/larsartmann/go-codec"

	"github.com/larsartmann/go-cqrs-lite/event/v4"
)

// Chain is a validated, immutable set of [Op] declarations compiled by
// [Compile]. It exposes the read-side transforms:
//
//   - [Chain.SourceTransform] — full semantics including [Split] and [Drop]
//     (batch-level, since they change the event count); compose with
//     [event.DecorateStore] or [event.DecorateJournal].
//   - [Chain.Upcasters] — the subset of ops that maps 1:1 onto the classic
//     [Upcaster] interface, for use with [UpcastSourceTransform].
type Chain struct {
	exact   map[chainKey]Op
	byType  map[event.Type]Op
	renames map[event.Type]event.Type
	maxHops int
}

type chainKey struct {
	eventType event.Type
	version   event.SchemaVersion
}

// Compile validates the op set and returns an immutable Chain.
//
// Rejected at build time: duplicate exact (type, schema version)
// registrations, duplicate type-only ops ([RenameType]/[Drop] on a type
// that already has one), ambiguous renames (duplicate source or target,
// self-rename, rename cycles), and invalid op parameters. Declaration order
// is irrelevant to the result — matching is by specificity.
func Compile(ops ...Op) (*Chain, error) {
	chain := &Chain{
		exact:   make(map[chainKey]Op, len(ops)),
		byType:  make(map[event.Type]Op),
		renames: make(map[event.Type]event.Type),
		maxHops: len(ops) + 1,
	}

	for _, op := range ops {
		if err := validateOpParams(op); err != nil {
			return nil, err
		}

		switch typed := op.(type) {
		case *renameTypeOp:
			if err := chain.addRename(typed); err != nil {
				return nil, err
			}
		case *dropOp:
			if _, exists := chain.byType[typed.sourceType]; exists {
				return nil, duplicateOpErr(typed.sourceType, 0)
			}

			chain.byType[typed.sourceType] = typed
		case *fieldOp:
			if err := chain.addExact(typed.sourceType, typed.sourceVersion, op); err != nil {
				return nil, err
			}
		case *transformOp:
			if err := chain.addExact(typed.sourceType, typed.sourceVersion, op); err != nil {
				return nil, err
			}
		case *splitOp:
			if err := chain.addExact(typed.sourceType, typed.sourceVersion, op); err != nil {
				return nil, err
			}
		}
	}

	if err := chain.detectRenameCycles(); err != nil {
		return nil, err
	}

	return chain, nil
}

func (c *Chain) addExact(
	sourceType event.Type,
	sourceVersion event.SchemaVersion,
	op Op,
) error {
	if _, exists := c.exact[chainKey{sourceType, sourceVersion}]; exists {
		return duplicateOpErr(sourceType, sourceVersion)
	}

	c.exact[chainKey{sourceType, sourceVersion}] = op

	return nil
}

func (c *Chain) addRename(op *renameTypeOp) error {
	if _, exists := c.byType[op.from]; exists {
		return errorfamily.WrapRejection(
			ErrDuplicateOp, "schema.duplicate_rename",
			"multiple type-only ops for event type "+string(op.from),
		)
	}

	if _, taken := c.renames[op.to]; taken {
		return errorfamily.WrapRejection(
			ErrDuplicateOp, "schema.duplicate_rename_target",
			"rename target "+string(op.to)+" already produced by another RenameType",
		)
	}

	c.byType[op.from] = op
	c.renames[op.from] = op.to

	return nil
}

func (c *Chain) detectRenameCycles() error {
	visited := make(map[event.Type]int, len(c.renames))

	var froms []event.Type

	for from := range c.renames {
		froms = append(froms, from)
	}

	sort.Slice(froms, func(i, j int) bool { return froms[i] < froms[j] })

	for _, start := range froms {
		if err := walkRenames(c.renames, start, visited); err != nil {
			return err
		}
	}

	return nil
}

func walkRenames(
	renames map[event.Type]event.Type,
	start event.Type,
	visited map[event.Type]int,
) error {
	// visited: 1 = on the current walk (stack), 2 = fully walked.
	node := start

	for {
		switch visited[node] {
		case 1:
			return errorfamily.WrapRejection(
				ErrRenameCycle, "schema.rename_cycle",
				"RenameType cycle reached at "+string(node),
			)
		case 2:
			return nil
		}

		next, ok := renames[node]
		if !ok {
			visited[node] = 2

			return nil
		}

		visited[node] = 1
		node = next
	}
}

func duplicateOpErr(sourceType event.Type, sourceVersion event.SchemaVersion) error {
	return errorfamily.WrapRejection(
		ErrDuplicateOp, "schema.duplicate_op",
		"duplicate op for "+string(sourceType)+" at schema version "+
			strconv.Itoa(int(sourceVersion)),
	)
}

// SourceTransform returns the batch-level [event.SourceTransform] applying
// every compiled op, including [Split] and [Drop]. Compose it with
// [event.DecorateStore] or [event.DecorateJournal] — read paths only,
// upcasting never rewrites stored events.
func (c *Chain) SourceTransform() event.SourceTransform {
	return c.upcastAll
}

// Upcasters converts the chain's 1:1 ops ([RenameField], [AddField],
// [RemoveField], [Transform]) into classic [Upcaster] values for
// [UpcastSourceTransform]. Chains containing [RenameType], [Split], or
// [Drop] — or a non-failing decode policy that drops — cannot map onto the
// single-event interface and are rejected.
func (c *Chain) Upcasters() ([]Upcaster, error) {
	upcasters := make([]Upcaster, 0, len(c.exact))

	for key, op := range c.exact {
		converted, err := upcasterFor(key, op)
		if err != nil {
			return nil, err
		}

		upcasters = append(upcasters, converted)
	}

	return upcasters, nil
}

func upcasterFor(key chainKey, op Op) (Upcaster, error) {
	if _, batch := op.(*splitOp); batch {
		return nil, batchOpErr("Split")
	}

	if op.(decodeOp).policy() == DropOnDecodeError {
		return nil, batchOpErr("DropOnDecodeError policy")
	}

	return NewUpcaster(key.eventType, key.version, func(evt event.Event) (event.Event, error) {
		next, outcome, err := applyDecodeOp(op.(decodeOp), evt)
		if err != nil {
			return nil, err
		}

		if outcome == opDone { // passthrough: must still return a NEW instance
			return evt.Clone(), nil
		}

		return next, nil
	}), nil
}

func batchOpErr(name string) error {
	return errorfamily.WrapRejection(
		ErrBatchOpNotConvertible, "schema.op_not_convertible",
		name+" needs batch semantics; use Chain.SourceTransform()",
	)
}
