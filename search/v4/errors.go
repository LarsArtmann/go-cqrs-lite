package search

import (
	"strconv"

	errorfamily "github.com/larsartmann/go-error-family"
)

// MaxQueryRunes is the longest query the package accepts when a [Spec] does
// not set its own limit.
const MaxQueryRunes = 256

// DefaultScanLimit bounds how many records the scan tier reads from a
// [Source] before it reports the result as truncated.
const DefaultScanLimit = 10000

// ErrQueryTooLong is returned for a query longer than the spec allows.
var ErrQueryTooLong error = errorfamily.NewRejection(
	"search.query_too_long",
	"the search text is longer than the allowed number of characters; shorten it and retry",
)

// ErrQueryInvalid is returned for a query that carries control characters.
var ErrQueryInvalid error = errorfamily.NewRejection(
	"search.query_invalid",
	"the search text contains control characters; remove them and retry",
)

// ErrNoFields is returned by [New] for a spec without searchable fields.
var ErrNoFields error = errorfamily.NewRejection(
	"search.no_fields",
	"a search needs at least one searchable field; declare one in the spec",
)

// ErrDuplicateField is returned by [New] for two fields with the same name.
var ErrDuplicateField error = errorfamily.NewRejection(
	"search.duplicate_field",
	"two searchable fields share a name; field names must be unique",
)

// ErrInvalidWeight is returned by [New] for a field whose weight is not positive.
var ErrInvalidWeight error = errorfamily.NewRejection(
	"search.invalid_weight",
	"a searchable field weight must be greater than zero",
)

// ErrInvalidField is returned by [New] for a field without a name or a value function.
var ErrInvalidField error = errorfamily.NewRejection(
	"search.invalid_field",
	"a searchable field needs a name and a value function",
)

// ErrInvalidLimit is returned by [New] for a negative limit in the spec.
var ErrInvalidLimit error = errorfamily.NewRejection(
	"search.invalid_limit",
	"the query length limit and the scan limit cannot be negative",
)

// ErrModeUnavailable is returned when an explicit [Mode] asks for a kind of
// search the deployment cannot serve. Auto never returns it; Auto steps down
// the ladder instead.
var ErrModeUnavailable error = errorfamily.NewRejection(
	"search.mode_unavailable",
	"the requested kind of search is not available in this deployment; use ModeAuto or ModeLexical, or configure the engine",
)

// ErrUnknownSort is returned for a sort key the spec does not declare.
var ErrUnknownSort error = errorfamily.NewRejection(
	"search.unknown_sort",
	"the sort key is not declared by the spec; sort by relevance or by a declared key",
)

// rejectionWith appends a detail to a sentinel's message while keeping its code.
func rejectionWith(sentinel error, detail string) error {
	if code := errorfamily.Code(sentinel); code != "" {
		return errorfamily.NewRejection(code, detail)
	}
	return sentinel
}

// limitOrDefault returns def when n is zero and rejects negative values.
func limitOrDefault(n, def int) (int, error) {
	switch {
	case n < 0:
		return 0, rejectionWith(ErrInvalidLimit, "the limit is "+strconv.Itoa(n)+"; it cannot be negative")
	case n == 0:
		return def, nil
	default:
		return n, nil
	}
}
