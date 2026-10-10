package schema

import (
	"slices"
	"strconv"

	errorfamily "github.com/larsartmann/go-error-family"

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
	targets map[event.Type]struct{}
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
		targets: make(map[event.Type]struct{}),
		maxHops: len(ops) + 1,
	}

	for _, op := range ops {
		if err := chain.add(op); err != nil {
			return nil, err
		}
	}

	if err := chain.detectRenameCycles(); err != nil {
		return nil, err
	}

	return chain, nil
}

func (c *Chain) add(op Op) error {
	if err := validateOpParams(op); err != nil {
		return err
	}

	switch typed := op.(type) {
	case *renameTypeOp:
		return c.addRename(typed)
	case *dropOp:
		return c.addDrop(typed)
	case *fieldOp:
		return c.addExact(typed.sourceType, typed.sourceVersion, op)
	case *transformOp:
		return c.addExact(typed.sourceType, typed.sourceVersion, op)
	case *splitOp:
		return c.addExact(typed.sourceType, typed.sourceVersion, op)
	default:
		return invalidOpErr("unknown op (schema.Op is sealed; only package constructors are valid)")
	}
}

func (c *Chain) addDrop(op *dropOp) error {
	if _, exists := c.byType[op.sourceType]; exists {
		return duplicateOpErr(op.sourceType, 0)
	}

	c.byType[op.sourceType] = op

	return nil
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

	if _, taken := c.targets[op.target]; taken {
		return errorfamily.WrapRejection(
			ErrDuplicateOp, "schema.duplicate_rename_target",
			"rename target "+string(op.target)+" already produced by another RenameType",
		)
	}

	c.byType[op.from] = op
	c.renames[op.from] = op.target
	c.targets[op.target] = struct{}{}

	return nil
}

func (c *Chain) detectRenameCycles() error {
	visited := make(map[event.Type]int, len(c.renames))

	froms := make([]event.Type, 0, len(c.renames))

	for from := range c.renames {
		froms = append(froms, from)
	}

	slices.Sort(froms)

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
// [Drop] — or a decode policy that drops — cannot map onto the single-event
// interface and are rejected.
func (c *Chain) Upcasters() ([]Upcaster, error) {
	if len(c.byType) > 0 {
		return nil, batchOpErr("RenameType/Drop (type-only) ops")
	}

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

	decode, ok := op.(decodeOp)
	if !ok {
		return nil, invalidOpErr("op cannot convert to Upcaster")
	}

	if decode.policy() == DropOnDecodeError {
		return nil, batchOpErr("DropOnDecodeError policy")
	}

	return NewUpcaster(key.eventType, key.version, func(evt event.Event) (event.Event, error) {
		next, outcome, err := applyDecodeOp(decode, evt)
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

func validateOpParams(op Op) error {
	switch typed := op.(type) {
	case *renameTypeOp:
		return validateRename(typed)
	case *dropOp:
		if typed.sourceType == "" {
			return invalidOpErr("Drop requires a non-empty event type")
		}

		return nil
	case *fieldOp:
		return validateFieldOp(typed)
	case *transformOp:
		if err := validateSource(typed.sourceType, typed.sourceVersion); err != nil {
			return err
		}

		if typed.transform == nil {
			return invalidOpErr("Transform requires a non-nil function")
		}

		return nil
	case *splitOp:
		return validateSplit(typed)
	default:
		return invalidOpErr("unknown op (schema.Op is sealed; only package constructors are valid)")
	}
}

func validateRename(op *renameTypeOp) error {
	if op.from == "" || op.target == "" {
		return invalidOpErr("RenameType requires non-empty from and to event types")
	}

	if op.from == op.target {
		return invalidOpErr("RenameType from == to (" + string(op.from) + ")")
	}

	return nil
}

func validateFieldOp(op *fieldOp) error {
	if err := validateSource(op.sourceType, op.sourceVersion); err != nil {
		return err
	}

	if op.field == "" || (op.kind == fieldRename && op.renamedTo == "") {
		return invalidOpErr("field ops require non-empty field names")
	}

	return nil
}

func validateSplit(op *splitOp) error {
	if err := validateSource(op.sourceType, op.sourceVersion); err != nil {
		return err
	}

	if len(op.outputs) == 0 {
		return invalidOpErr("Split requires at least one Producing output")
	}

	for _, output := range op.outputs {
		if output.eventType == "" || output.payload == nil {
			return invalidOpErr(
				"Split outputs require a non-empty type and a non-nil payload function",
			)
		}
	}

	return nil
}

func validateSource(sourceType event.Type, sourceVersion event.SchemaVersion) error {
	if sourceType == "" {
		return invalidOpErr("ops require a non-empty event type")
	}

	if !sourceVersion.IsPositive() {
		return invalidOpErr("ops require a positive schema version, got " + sourceVersion.String())
	}

	return nil
}

func invalidOpErr(msg string) error {
	return errorfamily.WrapRejection(ErrInvalidOp, "schema.invalid_op", msg)
}
